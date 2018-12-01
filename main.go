//
// Application site-map
//
// Overview:
// Crawls a website starting at the supplied url and generates a site map of all the internal links. Only links on
// the same domain are displayed.
//
// Program Description:
// Write a simple web crawler limited to one domain - so when you start with https://monzo.com/, it would crawl all
// pages within monzo.com, but not follow external links, for example to the Facebook and Twitter accounts. Given a
// URL, it should print a simple site map, showing the links between pages.
//
// Usage:
// 			Usage of go-sitemap
//				-delay int
//					minimum separation (in ms) between initiating loads from the server (default 100)
//				-depth int
//					maximum depth to crawl to, 0 means no limit (default 3)
//				-out string
//					site map destination file, with none meaning write to console (default: None)
//				-pages int
//					maximum number pages to load, 0 means no limit (default 2000)
//				-s string
//					site to crawl (default "en.wikipedia.org")
//				-t int
//					maximum number of concurrent loads from the server (default 10)
//				-verbose
//					set to show extra logging
//
// Known Issues / Missing Features
//		1. 	Add support for robots.txt (load and parse for the domain then use any filters requested)
//		2. 	Improve display of the site map. For example, it may be useful to see the structure based on the URL path
//			rather than based on the links present in each page
//		3.	Add retry logic on HTTP requests where appropriate (e.g. 503 response code returned)
//
package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"
)

//
// Defaults
//
const (
	DftSite         string = "en.wikipedia.org"
	DftNumLoaders   int    = 10    // number of page loading and parsing threads
	DftMinLoadDelay int    = 100   // minimum delay, in milliseconds, between each load
	DftMaxPages     int    = 2000  // number of pages to load
	DftMaxDepth     int    = 3     // max depth to crawl site to
	DftVerbose      bool   = false // true to add extra logging
)

func main() {

	//
	// Configuration
	//
	startURLStr := flag.String("s", DftSite, "site to crawl")
	fileName := flag.String("out", "", "site map destination file, with none meaning write to console")
	minLoadDelay := flag.Int("delay", DftMinLoadDelay, "minimum separation (in ms) between initiating loads from the server")
	numLoaders := flag.Int("t", DftNumLoaders, "maximum number of concurrent loads from the server")
	maxPages := flag.Int("pages", DftMaxPages, "maximum number pages to load, 0 means no limit")
	maxDepth := flag.Int("depth", DftMaxDepth, "maximum depth to crawl to, 0 means no limit")
	verbose := flag.Bool("verbose", DftVerbose, "set to show extra logging")
	flag.Parse()
	if flag.NArg() > 0 || *numLoaders < 0 || *maxPages < 0 || *maxDepth < 0 || *minLoadDelay < 0 {
		flag.Usage()
		return
	}

	//
	// Starting URL
	//
	startURL, err := url.Parse(*startURLStr)
	if err != nil {
		log.Fatalf("Invalid starting URL supplied: %s", *startURLStr)
	}
	if len(startURL.Scheme) == 0 {
		startURL.Scheme = "http"
	}

	//
	// Create and setup the site map and crawler
	//
	siteMap := CreateSiteMap(startURL)
	crawler := CreateCrawler(startURL, CreateDocumentLoader(CreateDocumentParser()), siteMap)
	crawler.minLoadDelay = *minLoadDelay
	crawler.numLoaders = *numLoaders
	crawler.maxPagesToLoad = *maxPages
	crawler.maxCrawlDepth = *maxDepth
	crawler.verbose = *verbose

	//
	// Crawl the website (this will block until crawling is complete)
	//
	start := time.Now()
	if err := crawler.crawl(); err != nil {
		log.Fatalf("FATAL: Failed to crawl website: %v", err)
	}
	crawlTime := time.Since(start).Seconds()
	log.Printf("INFO: Crawled %d pages from %s in %v seconds", len(siteMap.Pages), siteMap.Domain, crawlTime)

	//
	// Write the site map to the screen
	//
	PrintSite(*fileName, startURL.String(), siteMap)
}

// PrintSite writes the SiteMap contents to a file (or console if no file name is provided)
func PrintSite(fileName string, domain string, site *SiteMap) {

	file := os.Stdout
	if len(fileName) != 0 {
		log.Printf("INFO: Writing Site Map to file %s....\n", fileName)
		var err error
		file, err = os.Create(fileName)
		if err != nil {
			log.Fatalf("Failed to create file %s: %v", fileName, err)
		}
		defer file.Close()
	}

	// create a channel for the site map contents and a goroutine to populate it
	mapChan := make(chan MapTraversalNode, 20)
	go site.TraverseSiteMap(mapChan)

	// Write out the results
	if _, err := fmt.Fprintf(file, "\n\n ----- Site Map for website  %s -----\n", domain); err != nil {
		log.Fatalf("Failed to write to file %s: %v", fileName, err)
	}
	for page := range mapChan {
		if _, err := fmt.Fprintf(file, "%s %s [%s]\n", strings.Repeat("    ", page.Depth), page.Page.URL, page.Page.Title); err != nil {
			log.Fatalf("Failed to write to file %s: %v", fileName, err)
		}
	}

	if len(fileName) > 0 {
		log.Print("INFO: Done\n")
	}

}
