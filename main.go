//
// Application site-map
//
// **************************************************************************************************************
// PLEASE NOTE: THIS IS MY FIRST ATTEMPT AT A GOLANG APPLICATION. SOME CODING STYLES, USE OF LIBRARIES AND USE OF
// STANDARD PATTERNS (OR LACK THERE OF) MAY NOT NOT BE IDEAL!
// **************************************************************************************************************
//
// Overview:
// Crawls a website starting at the supplied url or domain name and generates a site map of all the internal links.
// Only links on the same domain are displayed.
//
// The site map is shown in a heirarchical view, with each page listing it's outgoing links.
// Note that links can form cycles and individual pages are linked to from many pages. To reduce the size of the
// display and avoid loops, the display is limited in 3 ways:
//		1. 	No links to a page at a higher depth are displayed (where depth is defined as the the shortest number of
//			links to a page from the root page). For example, a link from a page to it's parent would not be displayed,
//			but links to its children or to another page at the same depth are.
//		2. 	Links from a page back to itself are ignored
//		3.	Each page is only expanded once, and at the highest level at which it occurs. This means if a page appears
//			multiple times at the same level, its children will only be displayed the first time it appears
//
// By default, some throttling is done to avoid loading pages from a website too quickly. This is purely to limit
// any problems with the site. In addition, a limit to the number of simultaneous requests is also
// set. Both of these are controllable with command lime switches.
//
// Limits can also be set on how many pages will be loaded in total and/or the depth to crawl the website. By default
// no limits are applied.
//
// Usage:
// 			Usage of go-sitemap
//				-delay int
//					minimum separation (in ms) between initiating loads from the server (default 100)
//				-depth int
//					maximum depth to crawl to, 0 means no limit (default 0)
//				-out string
//					site map destination file, with none meaning write to console (default: None)
//				-pages int
//					maximum number pages to load, 0 means no limit (default 0)
//				-s string
//					site to crawl (default "en.wikipedia.org")
//				-t int
//					maximum number of concurrent loads from the server (default 10)
//				-verbose
//					set to show extra logging
//
// 	Example:
//  			./go-sitemap -out monzo.txt -s monzo.com -delay 250
//						Maps whole monzo.com domain, with a minimum 250 ms delay between starting each page load
//						and a maximum of 10 concurrent loads. Resultong site map is written to mozo.txt file.
//
// Build Instructions:
//		1. One external dependency is required. Please install (golang.org/x/net/html)
//			 > go get golang.org/x/net/html
//		2. Run unit tests
//			 > go test
//		3. Build / Install
//			 > go install
//
// Design Notes:
//		The application consists of the following main types:
//			SiteMap 		- stores a sites pages and hyperlinks in a tree structure and iterates over the site map.
//			DocumentParser	- interface (with DocParser implementation) to convert a HTML document it into a WebPage
//			DocumentLoader	- interface (with DocLoader implementation) to load URLs then parse the documents returned
//							  using a supplied DocumentParser
//			Crawler			- Web crawler type used to build the processing pipeline used to crawl the website and
//							  ingest the loaded WebPage documents into the SiteMap.
//
// 		The following shows the structure of the processing pipeline. Note this forms a loop which continues until
//		all pages are crawled, the maximum number of pages are loaded, or we have crawled all pages to the maximum
//		depth. Numbers in [] indicate number of concurrent goroutines processing
//
//   |---> urlLoadChan[1] --> DocumentLoader (plus DocumentParser)[>=1] |-------- pagesChan ----> SiteMap[1]
//   |                                                                  |---- linksChan ->|
//	 |	  	                                                                              |
//   |<-------------------Crawler (URL Filtering & queuing)[1] <--------------------------|
//
// The following channels are used
//		pagesChan:			pages to be ingested into the Site Map
//		urlLoadChan:		URLs to be loaded by our pool of page loading workers
//		linksChan:			all internal links read off processed pages
//
// In addition , the following channels are used to monitor progress to detect and signal completion:
//		pendingItemsChan:	tracks total number of items queued or being processed across all channels
//		finishedEventChan:	used to signal that crawling is complete
//
// An in-memory queue is used to store the urls waiting to be loaded (inside the Crawler)
//
// Known Issues / Missing Features
//		1. 	Add support for robots.txt (load and parse for the domain then use any filters requested)
//		2. 	Improve display of the site map. For example, it may be useful to see the structure based on the URL path
//			rather than based on the links present in each page
//		3.	Add retry logic on HTTP requests where appropriate (e.g. 503 response code returned)
//		4.  Add support for the <BASE> tag on a page
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
	DftNumLoaders   int    = 10    	// number of page loading and parsing threads
	DftMinLoadDelay int    = 100   	// minimum delay, in milliseconds, between each load
	DftMaxPages     int    = 0		// number of pages to load
	DftMaxDepth     int    = 0     	// max depth to crawl site to
	DftVerbose      bool   = false 	// true to add extra logging
)

func main() {

	//
	// Configuration
	//
	startURLStr := flag.String("s", DftSite, "site to crawl")
	fileName := flag.String("out", "", "site map destination file, with none meaning write to console")
	minLoadDelay := flag.Int("delay", DftMinLoadDelay, "minimum separation (in ms) between initiating loads from the server")
	numLoaders := flag.Int("t", DftNumLoaders, "maximum number of concurrent loads from the server")
	maxPages := flag.Int("pages", DftMaxPages, "maximum number pages to load, 0 means no limit (default: 0)")
	maxDepth := flag.Int("depth", DftMaxDepth, "maximum depth to crawl to, 0 means no limit (default: 0)")
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
