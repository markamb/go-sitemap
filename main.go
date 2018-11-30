package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"strings"
)

//
// Program site-map
//
// Known Issues / Missing Features
//		1. 	Add support for robots.txt (load and parse for the domain then use any filters requested)
//		2. 	Improve display of the site map. For example, it may be useful to see the structure based on the URL path
//			rather than based on the links present in each page
//		3.	Add retry logic on HTTP requests where appropriate (e.g. 503 response code returned)
//

//
// Defaults
//
const (
	DftDomain 		string 	= "en.wikipedia.org"
	DftNumLoaders 	int 	= 5			// number of page loading and parsing threads
	DftMinLoadDelay	int		= 1000		// minimum delay, in milliseconds, between each load
	DftMaxPages		int		= 10		// number of pages to load
	DftMaxDepth		int		= 0			// max depth to crawl site to
)

func main() {

	//
	// Configuration
	//
	startUrlStr := flag.String("d", DftDomain, "domain to crawl")
	minLoadDelay := flag.Int("delay", DftMinLoadDelay, "minimum separation between initiating loads from the server")
	numLoaders := flag.Int("loaders", DftNumLoaders, "maximum number of concurrent loads from the server")
	maxPages := flag.Int("pages", DftMaxPages, "maximum number pages to load (0 for None)")
	maxDepth := flag.Int("depth", DftMaxDepth, "maximum depth to crawl to (0 for None)")
	flag.Parse()
	if flag.NArg() > 0 {
		flag.Usage()
		return
	}

	//
	// Starting URL
	//
	startUrl, err := url.Parse(*startUrlStr)
	if err != nil {
		log.Fatalf("Invalid starting URL supplied: %s", *startUrlStr)
	}
	if len(startUrl.Scheme) == 0 {
		startUrl.Scheme = "http"
	}

	//
	// Create and setup the site map and crawler
	//
	siteMap := CreateSiteMap(startUrl)
	crawler := CreateCrawler(startUrl,CreateDocumentLoader(CreateDocumentParser()), siteMap)
	crawler.minLoadDelay = *minLoadDelay
	crawler.numLoaders = *numLoaders
	crawler.maxPagesToLoad = *maxPages
	crawler.maxCrawlDepth = *maxDepth

	//
	// Crawl the website (this will block until crawling is complete)
	//
	if err := crawler.crawl(); err != nil {
		log.Fatalf("FATAL: Failed to crawl website: %v", err)
	}

	//
	// Write the site map to the screen
	//
	PrintSite(startUrl.String(), siteMap)
}

// PrintSite writes the SiteMap contents to the console
func PrintSite(domain string, site *SiteMap) {

	// create a channel for the site map contents and a goroutine to populate it
	mapChan := make(chan MapTraversalNode, 20)
	go site.TraverseSiteMap(mapChan)

	// Write out the results
	fmt.Printf("\n\n ----- Site Map for website  %s -----\n", domain)
	for page := range mapChan {
		fmt.Printf("%s %s [%s]\n", strings.Repeat("    ", page.Depth), page.Page.Url, page.Page.Title)
	}
}







