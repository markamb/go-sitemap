package main

import (
	"log"
	"net/url"
	"sync"
	"time"
)

//
// Crawler Type stores a domain to be crawled and the results of doing so.
// Initialised with a DocumentLoader interface for retrieving and parsing URLs
//
type Crawler struct {

	// Interfaces used to load documents
	docLoader 		DocumentLoader

	// Site Map used to store results
	siteMap			SiteMapper

	// url to start crawling from
	startUrl		*url.URL

	// configuration
	minLoadDelay 	int		// default minimum delay between starting each load
	numLoaders 		int		// number of goroutines used for loading (= maximum number of concurrent requests)
	maxPagesToLoad	int		// Limits the number of pages loaded for testing on large sites. 0 to load all available pages.

	// an in-memory queue for storing our URLs to be crawled
	urlQueue		StringQueue

	// channels
	pagesChan 	chan *WebPage	// pages to be ingested into the Site Map
	urlLoadChan chan string 	// URLs to be loaded by our pool of page loading workers
	linksChan 	chan string		// Internal links read off processed pages
	pendingItemsChan chan int	// Track total number of items queued, or being processed across all channels
	finishedEventChan chan bool	// used to signal that crawling is complete
}

func CreateCrawler(start *url.URL, l DocumentLoader, mapper SiteMapper) *Crawler {
	return &Crawler {
		docLoader:		l,
		startUrl:		start,
		siteMap:		mapper,
		minLoadDelay:	1000,
		numLoaders:		5,
		maxPagesToLoad:	10,

		pagesChan:			make(chan *WebPage, 20),
		urlLoadChan:		make(chan string, 20),
		linksChan:			make(chan string),
		pendingItemsChan:	make(chan int),
		finishedEventChan:	make(chan bool),
	}
}

//
// Starts concurrent crawling process. This method will block until crawling is complete
//
func (c *Crawler) crawl() error {

	var wg sync.WaitGroup

	//
	// Kick off routines to load required pages, parse them, then add
	// Note we optionally throttle how quickly we load pages using a ticker to make sure
	// we're not blacklisted or unpopular with the site owner
	//
	var loadTicker *time.Ticker = nil
	if c.minLoadDelay != 0 {
		loadTicker = time.NewTicker(time.Duration(c.minLoadDelay) * time.Millisecond)
		defer loadTicker.Stop()
	}
	for i := 0; i < c.numLoaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.loadPages(loadTicker)
		}()
	}

	//
	// Kick of a single goroutine to read the pages into our Site Map
	// We must do this in a single thread as the SiteMap is not thread safe
	//
	wg.Add(1)
	go func () {
		defer wg.Done()
		c.populateSiteMap()
	}()

	//
	// start a single goroutine to read the parsed urls and test if they have already been seen.
	// URLs to be loaded are added to our internal "unbounded" queue
	//
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.enqueueNewUrls()
	}()

	//
	// a goroutine to dequeue items from the internal queue and place them on a channel
	// to be processed by our page loading worker threads
	//
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.dequeueUrls()
	}()

	//
	// Start a goroutine to track the number of items of work in progress or pendinf accross all channels and the
	// internal queue and to stop processing once this reaches zero
	//
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.monitorProgress()
	}()

	//
	// Add our start URL to start the crawling process
	//
	log.Printf("INFO: Starting crawl process...")
	c.pendingItemsChan <- 1
	c.linksChan <- c.startUrl.String() // linksChan to ensure it ends up in the visited cache

	// Wait for the crawling to complete
	wg.Wait()
	close(c.pendingItemsChan)
	return nil
}

//
// monitorProgress: keep track of the number of items being processed or queued across all
// the channels. When this count reaches zero we have completed the crawling process and should
// close the channels so the crawling goroutines will complete. This is needed because our channels
// form a loop so none can detect running out of work in isolation
//
func (c *Crawler) monitorProgress()  {
	itemCount := 0
	for delta := range c.pendingItemsChan {
		itemCount += delta
		if itemCount <= 0 {
			// All channels are empty, and no work is in progress
			log.Printf("INFO: Total number of queued items = %d, closing channels\n", itemCount)
			c.finishedEventChan <- true
			close(c.pagesChan)
			close(c.urlLoadChan)
			close(c.linksChan)
			close(c.finishedEventChan)
			return
		}
	}
}

//
// Read urls to be loaded from urlLoadChan, load and parse them, then send results to
// output channels.
// If loadTicker is supplied (not nil) we only load a new page after reading a tick (used
// to throttle our rate of loading)
//
func (c *Crawler) loadPages(loadTicker *time.Ticker)  {
	for urlStr := range c.urlLoadChan {
		page, err := c.docLoader.LoadUrl(urlStr)
		if page != nil {
			for link := range page.InternalLinks {
				c.pendingItemsChan <- 1
				c.linksChan <- link // send the links back to the crawler to keep going
			}
			c.pagesChan <- page // send page details to be ingested into site map
		} else {
			log.Printf("TRACE : Ignoring Url : %v", err)
			c.pendingItemsChan <- -1
		}
		if loadTicker != nil {
			<-loadTicker.C 		// make sure we have required delay between last load starting
		}
	}
}

//
// enqueueNewUrls: reads URLS extracted from web pages (from linksChan) and add them into the
// queue after checking for duplicates
//
func (c *Crawler) enqueueNewUrls() {
	count := 0
	seen := make(map[string]bool)
	for link := range c.linksChan {
		// if we have seen this url before skip it otherwise add it to channel to be loaded
		if _, skip := seen[link]; skip {
			// already seen this url - ignore it
			c.pendingItemsChan <- -1
		} else if c.maxPagesToLoad > 0 && count >= c.maxPagesToLoad {
			// stop crawling as we've reached our limit
			seen[link] = true
			c.pendingItemsChan <- -1
		} else {
			// add url it to our in-memory queue to be crawled
			log.Printf("TRACE: Queuing up URL %v\n", link)
			seen[link] = true
			count++
			c.urlQueue.Push(link)
		}
	}
}

//
// populateSiteMap: reads pages off the pagesChan and add them to the site map
//
func (c *Crawler) populateSiteMap() {
	for page := range c.pagesChan {
		if err := c.siteMap.AddPage(page); err != nil {
			log.Printf("WARN: %v\n", err)
		}
		c.pendingItemsChan <- -1
	}
}

//
// dequeuUrls: removes urls to be crawled from the internal queue and sends them to the urlLoadChan
//
func (c *Crawler) dequeueUrls() {
	for {
		next, ok := c.urlQueue.Pop()
		if ok {
			// block until channel accepts next url
			c.urlLoadChan <- next
		} else {
			select {
			case <-c.finishedEventChan:
				// crawling complete, exit
				return
			default:
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}




