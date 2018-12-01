package main

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
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

// WebPage represents a single page in the website
// We only store internal links and the page title however this could easily be extended to add any
// other useful information we want to crawl (list of all external links, page size etc)
type WebPage struct {
	URL           *url.URL        // absolute URL for this page
	Title         string          // HTML title of this page
	InternalLinks map[string]bool // set of internal links out of this page (set as we only want each item once)
}

// CreateWebPage creates a new WebPage with a given URL and page title
func CreateWebPage(newURL *url.URL, title string) *WebPage {
	page := &WebPage{
		URL:           newURL,
		Title:         title,
		InternalLinks: make(map[string]bool),
	}
	// Normalise the URL so equivilent ones match
	page.URL.Path = strings.TrimSuffix(page.URL.Path, "/")
	return page
}

// MapTraversalNode is a structure returned for each node when traversing the site map
type MapTraversalNode struct {
	Page  *WebPage // the page details
	Depth int      // the depth of the page at this point
}

// SiteMapper is an interface used to capture the structure of a website and traverse its
// pages in a logic order.
type SiteMapper interface {

	// AddPage adds a page to the site map. If the page is already present it is ignored and we return false.
	// If the page is invalid returns an error.
	// Note that 2 pages are considered equivilent if they refer to the same resource, even though the actual
	// URL string may differ
	AddPage(page *WebPage) (bool, error)

	// TraverseSiteMap adds the pages in the site map to the supplied channel in depth first order suitable
	// for rendering a site map.
	//
	// Note that all links are returned (so a page will be returned multiple times), however the children
	// for any page are only traversed once (at the highest level at which the page appears). See main.go comments
	// for more details.
	TraverseSiteMap(ch chan<- MapTraversalNode)
}

// SiteMap type implements the SiteMapper interface
type SiteMap struct {
	Domain   string              // name of the domain/website represented
	RootPage string              // top of the website
	Pages    map[string]*WebPage // URL for all web pages on the site
}

// CreateSiteMap creates a new, empty SiteMap for the given domain
func CreateSiteMap(start *url.URL) *SiteMap {
	return &SiteMap{Domain: start.Host,
		RootPage: start.String(),
		Pages:    make(map[string]*WebPage),
	}
}

// AddPage adds a new web page. See SiteMapper interface for details.
func (site *SiteMap) AddPage(page *WebPage) (bool, error) {
	if page == nil {
		return false, fmt.Errorf("SiteMap: Attempt to add empty page or url to site map")
	}
	if _, found := site.Pages[page.URL.String()]; found {
		return false, nil
	}
	site.Pages[page.URL.String()] = page
	return true, nil
}

// TraverseSiteMap adds all pages to the supplied channel in depth first order suitable for rendering
// See SiteMapper interface for details
func (site *SiteMap) TraverseSiteMap(ch chan<- MapTraversalNode) {
	// First we need to determine lowest height for each page (i.e the minimum number of steps from the sites
	// root to a page along any path). This is used to determine at which point we traverse the pages children
	defer close(ch)
	expanded := make(map[*WebPage]bool)
	minPageHeights := site.getMinimumHeights()
	// now do the depth first traversal
	site.doDepthFirstTraversal(ch, minPageHeights, expanded, 0, site.RootPage)
}

func (site *SiteMap) doDepthFirstTraversal(
	ch chan<- MapTraversalNode, 		// channel to send pages in order
	minPageHeights map[string]int, 		// shortest number of links to this page by any path
	expanded map[*WebPage]bool, 		// pages already expanded
	height int, 						// current traversal depth
	url string) { 						// current page
	if len(url) == 0 {
		return
	}
	page, found := site.Pages[url]
	if !found {
		return
	}

	minHeight, found := minPageHeights[url]
	if minHeight < height {
		return // don't display links to higher up pages
	}

	// add the current page then traverse down the graph in a DF manner
	ch <- MapTraversalNode{page, height}

	// expand the children if this is the first time we've seen this page
	if len(page.InternalLinks) != 0 {
		if _, found := expanded[page]; !found {
			// Iterating the InternalLinks 'set' will return a random order
			// This is may be ok but we'll be a bit more deterministic and return the children
			// in alphabetical order based on url
			expanded[page] = true
			sorted := make([]string, 0, len(page.InternalLinks))
			for nextURL := range page.InternalLinks {
				// ignore links back to same page
				if nextURL != url {
					sorted = append(sorted, nextURL)
				}
			}
			sort.Strings(sorted)
			for _, next := range sorted {
				site.doDepthFirstTraversal(ch, minPageHeights, expanded, height+1, next)
			}
		}
	}
}

type heightQueueEntry struct {
	url    string
	height int
}
type heightQueue []heightQueueEntry

// Return a map of URL to the minimum height at which that page appears. Height is defined as the
// number of links from the root page to this one along the shortest path.
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
	// queue at any point in time. This could be improved using a ring buffer queue implementation however its
	// lifetime is very short.
	//
	queue := make(heightQueue, 0)
	queue = append(queue, heightQueueEntry{site.RootPage, 0})
	for len(queue) != 0 {
		next := queue[0]  // top item from queue
		queue = queue[1:] // pop top item

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
