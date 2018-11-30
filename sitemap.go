package main

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
)

//
// SiteMap is a graph representing all the links in a given website. The nodes of the graph are pages
// and the edges represent hyperlinks.
// Note that there may be cycles in the graph (in fact this would be very common). For
// example, a link from a child back up to its parent or the sites root page would be common. We capture
// all this information while crawling however we choose not to display up-ward facing links when rendering
// the site map.
//
// We store the graph nodes (pages) in a hash map of urls, to allow fast lookup, and the edges in the
// nodes themselves (as a list of urls)
//
// No locking is done on this structure and it is assumed no concurrent access will be be used.
//

//
// WebPage represents a single page in the website
// We only store internal links and the page title however this could easily be extended to add any
// other useful information we want to crawl (list of all external links, page size etc)
//
type WebPage struct {
	Url  			string			// absolute URL for this page
	Title			string			// HTML title of this page
	InternalLinks	map[string]bool	// set of internal links out of this page (set as we only want each item once)
}

// CreateWebPage creates a new WebPage with a given URL and page title
func CreateWebPage(urlStr string, title string) *WebPage {
	return &WebPage{
		Url:  urlStr,
		Title: title,
		InternalLinks:  make(map[string]bool),
	}
}

//
// MapTraversalNode is a structure returned for each node when traversing the site map
//
type MapTraversalNode struct {
	Page 	*WebPage	// the page details
	Depth	int			// the depth of the page at this point
}

//
// SiteMapper is an interface used to store capture the structure of a website and traverse its
// pages in a logic order.
//
type SiteMapper interface {

	//
	// Add page adds a page to the site map. If the page is already present an error is returned
	// and no changes are made.
	//
	AddPage(page *WebPage) error

	//
	// TraverseSiteMap adds the pages in the site map to the supplied channel in depth first order suitable
	// for rendering a site map.
	//
	// Note that all links are returned (so a page will be returned multiple times), however the children
	// for any page are only traversed once (at the highest level at which the page appears)
	//
	// The channel will be closed on completion
	//
	TraverseSiteMap(ch chan<- MapTraversalNode)
}


type SiteMap struct {
	Domain		string					// name of the domain/website represented
	RootPage	string					// top of the website
	Pages		map[string]*WebPage		// URL for all web pages on the site
}

//
// NewSiteMap creates a new, empty SiteMap for the given domain
//
func CreateSiteMap(start *url.URL) *SiteMap {
	return &SiteMap { 	Domain:		start.Host,
						RootPage:	start.String(),
						Pages:		make(map[string]*WebPage),
	}
}

// AddPage adds a new web page. See SiteMapper interface for details.
func (site *SiteMap) AddPage(page *WebPage) error {
	if page == nil || len(page.Url) == 0 {
		return errors.New(fmt.Sprintf("SiteMap: Attempt to add empty page or url to site map"))
	}
	if page, found := site.Pages[page.Url]; found {
		return errors.New(fmt.Sprintf("SiteMap: Attempt to add duplicate page already in site map (%v)", page.Url))
	}
	site.Pages[page.Url] = page
	return nil
}

//
// DepthFirstTraversal adds the pages in the site map to the supplied channel in depth first order suitable
// for rendering a site map.
//
// Note that all links are returned (so a page will be returned multiple times), however the children
// for any page are only traversed once (at the highest level the page appears)
//
// The channel will be closed on completion
//
// TraverseSiteMap adds all pages to the supplied channel in depth first order suitable for rendering
// See SiteMapper interface for details
func (site *SiteMap) TraverseSiteMap(ch chan<- MapTraversalNode) {
	//
	// First we need to determine lowest height for each page (i.e the minimum number of steps from the sites
	// root to a page along any path). This is used to determine at which point we traverse the pages children
	//
	defer close(ch)
	minPageHeights := site.getMinimumHeights()
	site.doDepthFirstTraversal(ch, minPageHeights, 0, site.RootPage)
}

func (site *SiteMap) doDepthFirstTraversal(
	ch chan<- MapTraversalNode,			// channel to send pages in order
	minPageHeights map[string]int,		// shortest number of links to this page by any path
	depth int, 							// current traversal depth
	url string) {							// current page
		if len(url) == 0 {
			return
		}
		page, found := site.Pages[url]
		if !found {
			return;
		}

		// add the current page then traverse down the graph in a DF manner
		ch <- MapTraversalNode{page, depth}

		if minHeight, found := minPageHeights[url]; found && minHeight == depth {
			delete(minPageHeights, url)			// delete entry to ensure we only expand once
			if len(page.InternalLinks) != 0 {
				// Iterating the InternalLinks 'set' will return a random order
				// This is may be ok but we'll be a bit more deterministic and return the children
				// in alphabetical order based on url
				sorted := make([]string, 0, len(page.InternalLinks))
				for nextUrl := range page.InternalLinks {
					sorted = append(sorted, nextUrl)
				}
				sort.Strings(sorted)
				for _, next := range sorted {
					site.doDepthFirstTraversal(ch, minPageHeights, depth+1, next)
				}
			}
		}
}

type heightQueueEntry struct {
	url 	string
	height	int
}
type heightQueue []heightQueueEntry

//
// Return a map of URL to the minimum height at which that page appears. Height is defined as the
// number of links from the root page to this one along the shortest path.
//
func (site *SiteMap) getMinimumHeights() map[string]int {

	//
	// We do a breadth-first traversal of the tree, storing the height at which we see each url for the first time
	//
	heights := make(map[string]int)
	if len(site.RootPage) == 0 {
		return heights
	}

	//
	// Note we are using a very simple queue structure here with memory usage equal to number of items pushed
	// onto the queue (in this case, the number of pages) rather than the maximum number of elements on the
	// queue at any point in time.
	// This could be improved using a ring buffer queue implementation
	//
	queue := make(heightQueue, 0)
	queue = append(queue, heightQueueEntry{site.RootPage, 0})
	for len(queue) != 0 {
		next := queue[0]		// top item from queue
		queue = queue[1:]		// pop top item

		// get the page details and process the links
		page, found := site.Pages[next.url]
		if !found {
			// This page was never loaded. Could happen if we're only crawling part of the site
			continue
		}

		// store current height for this page
		heights[next.url] = next.height

		//
		// Add any children we haven't already seen to the end of the queue for processing
		//
		newHeight := next.height + 1
		for child := range page.InternalLinks {
			if _, found := heights[child]; !found {
				queue = append(queue, heightQueueEntry{child, newHeight})
			}
		}
	}

	return heights
}


