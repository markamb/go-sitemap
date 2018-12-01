package main

import (
	"net/url"
	"testing"
)

func TestEmptySiteMap(t *testing.T) {

	URL, err := url.Parse("https://bbc.co.uk")
	if err != nil {
		t.Fatal(err)
	}
	site := CreateSiteMap(URL)

	if site.Domain != "bbc.co.uk" {
		t.Fail()
	}
	if site.RootPage != URL.String() {
		t.Fail()
	}
	if len(site.Pages) != 0 {
		t.Fail()
	}
}

// Test that pages are only displayed at the top level at which they appear
func TestSiteMap(t *testing.T) {

	URL, err := url.Parse("https://test.com")
	if err != nil {
		t.Fatal(err)
	}
	site := CreateSiteMap(URL)

	urlBase := URL.String()

	//
	// level 1 (root page)
	//
	level1 := addPage(t, site, true, urlBase, "1")

	//
	// level 2
	// 3 child pages, plus a link back to itself which should be ignored)
	//
	level2_1_1 := addPage(t, site, true, urlBase+"/1/1", "1_1")
	level2_1_2 := addPage(t, site, true, urlBase+"/1/2", "1_2")
	level2_1_3 := addPage(t, site, true, urlBase+"/1/3", "1_3")
	level1.InternalLinks[level2_1_1.URL.String()] = true
	level1.InternalLinks[level2_1_2.URL.String()] = true
	level1.InternalLinks[level2_1_3.URL.String()] = true
	level1.InternalLinks[level1.URL.String()] = true

	// add some duplicate pages - these should fail to add
	addPage(t, site, false, urlBase+"/1/2", "Duplicate")
	addPage(t, site, false, urlBase+"/1/2/", "Duplicate")

	// level 3
	//
	level3_1_1_1 := addPage(t, site, true, urlBase+"/1/1/1", "1_1_1")
	level3_1_1_2 := addPage(t, site, true, urlBase+"/1/1/2", "1_1_2")
	level3_1_3_1 := addPage(t, site, true, urlBase+"/1/3/1", "1_3_2")
	level2_1_1.InternalLinks[level3_1_1_1.URL.String()] = true
	level2_1_1.InternalLinks[level3_1_1_2.URL.String()] = true
	level2_1_3.InternalLinks[level3_1_3_1.URL.String()] = true
	level2_1_3.InternalLinks[level3_1_1_1.URL.String()] = true // duplicate at same level
	level2_1_3.InternalLinks[level1.URL.String()] = true       // link back to higher level (should be skipped)
	level2_1_3.InternalLinks[level3_1_1_1.URL.String()] = true // link to same level (should be displayed)

	// level 4
	// Add a child under 1_1_1 which should only appear once (as 1_1_1 should only be expanded once)
	level4_1_1_1_1 := addPage(t, site, true, urlBase+"/1/1/1/1", "1_1_1_1")
	level3_1_1_1.InternalLinks[level4_1_1_1_1.URL.String()] = true

	// last level 5 which should be ignored (links back to parent level)
	level4_1_1_1_1.InternalLinks[level3_1_3_1.URL.String()] = true

	// write structure if test fails for debugging
	//	PrintSite("", urlBase, site)

	// traverse the site map, fill the channel with nodes in (hopefully) correct order
	ch := make(chan MapTraversalNode, 100)
	site.TraverseSiteMap(ch)

	// assert pages coming pack in correct order
	assertPage(t, level1, 0, <-ch)
	assertPage(t, level2_1_1, 1, <-ch)
	assertPage(t, level3_1_1_1, 2, <-ch)
	assertPage(t, level4_1_1_1_1, 3, <-ch)
	assertPage(t, level3_1_1_2, 2, <-ch)
	assertPage(t, level2_1_2, 1, <-ch)
	assertPage(t, level2_1_3, 1, <-ch)
	assertPage(t, level3_1_1_1, 2, <-ch)
	assertPage(t, level3_1_3_1, 2, <-ch)

	// check no more pages coming back and channel is closed
	if _, ok := <-ch; ok {
		t.Fatal("Channel not closed")
	}
}

func createWebPage(t *testing.T, rawurl string, title string) *WebPage {
	URL, err := url.Parse(rawurl)
	if err != nil {
		t.Fatalf("Invalid URL supplied in test case: %v", err)
	}
	return CreateWebPage(URL, title)
}

func addPage(t *testing.T, site *SiteMap, expectSuccess bool, urlStr string, title string) *WebPage {
	URL, err := url.Parse(urlStr)
	if err != nil {
		t.Fatalf("Invalid URL string in test case %v. Duplicate URL?", err)
	}
	page := CreateWebPage(URL, title)
	if page == nil {
		t.Fatalf("Failed to create page %s. Duplicate URL?", page.URL)
	}
	added, err := site.AddPage(page)
	if err != nil {
		t.Fatalf("Exception adding page %s: %v", page.URL, err)
	}
	if expectSuccess != added {
		t.Fatalf("Failed adding page to site: expected %v, got %v for page %s", expectSuccess, added, title)
	}
	return page
}

// validate the 2 pages are the same
func assertPage(t *testing.T, expectedPage *WebPage, expectedDepth int, got MapTraversalNode) {
	if got.Depth != expectedDepth {
		t.Fatalf("Next page not correct height (%s): expected %d, got %d\n", expectedPage.URL, expectedDepth, got.Depth)
	}
	if expectedPage == nil && got.Page == nil {
		return
	}
	if got.Page != expectedPage {
		t.Fatalf("Next page not correct (%s): expected %v, got %v\n", expectedPage.URL, expectedPage, got.Page)
	}
}
