package main

import (
	"flag"
	"log"
	"net/url"
)

//
// Program site-map
//
// Known Issues / Missing Features
//		1. 	Add support for robots.txt (load and parse for the domain then use any filters requested)
//		2. 	Improve display of the site map. For example, it may be useful to see the structure based on the URL path
//			rather than based on the links present in each page
//		3.
//

//
// Defaults
//
const (
	DftDomain 		string 	= "en.wikipedia.org"
	DftNumLoaders 	int 	= 5			// number of page loading and parsing threads
	DftMinLoadDelay	int		= 1000		// minimum delay, in milliseconds, between each load
	DftMaxPages		int		= 10		// number of pages to load
)

func main() {

	//
	// Configuration
	//
	startUrlStr := flag.String("d", DftDomain, "domain to crawl")
	minLoadDelay := flag.Int("delay", DftMinLoadDelay, "minimum separation between initiating loads from the server")
	numLoaders := flag.Int("loaders", DftNumLoaders, "maximum number of concurrent loads from the server")
	maxPages := flag.Int("pages", DftMaxPages, "maximum number pages to load (0 for None)")
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
	// Create and setup the crawlier
	//
	crawler := CreateCrawler(startUrl, CreateDocumentLoader(CreateDocumentParser()))
	crawler.minLoadDelay = *minLoadDelay
	crawler.numLoaders = *numLoaders
	crawler.maxPagesToLoad = *maxPages

	//
	// Crawl the website (this will block until crawling is complete)
	//
	if err := crawler.crawl(); err != nil {
		log.Fatalf("FATAL: Failed to crawl website: %v", err)
	}

	//
	// Write the site map to the screen
	//
	crawler.dump()
}



