package main

import (
	"testing"
	"net/url"
)
func TestEmptySiteMap(t *testing.T) {

	Url, err := url.Parse("https://bbc.co.uk")
	if  err != nil {
		t.Fatal(err)
	}
	site := CreateSiteMap(Url)

	if site.Domain != "bbc.co.uk" {
		t.Fail()
	}
	if site.RootPage != Url.String() {
		t.Fail()
	}
	if len(site.Pages) != 0 {
		t.Fail()
	}
}


func TestSmallSiteMap(t *testing.T) {

	Url, err := url.Parse("https://test.com")
	if  err != nil {
		t.Fatal(err)
	}
	site := CreateSiteMap(Url)

	urlBase := Url.String()
	titleBase := urlBase + ":TITLE"
	p1 := CreateWebPage(urlBase, titleBase)
	for i:=0; i < 10; i++ {
		next := urlBase + "/" + string(i)
		p1.InternalLinks[next] = true
		nextPage := CreateWebPage(next, string(i))
		if i == 1 {
			// add a sub-child off the 2nd item
			nextPage.InternalLinks[next + "/leaf"] = true
			if err := site.AddPage(CreateWebPage(next + "/leaf", "Leaf Page")); err != nil {
				t.Fatalf("Failed to add child of child page: %v", err )
			}
		}
		if err := site.AddPage(nextPage); err != nil {
			t.Fatalf("Failed to add child page: %d: %v", i, err )
		}
	}

	if err := site.AddPage(p1); err != nil {
		t.Fatalf("Failed to add page: %v", err)
	}

	// fill the channel with nodes in (hopefully) correct order
	ch := make(chan MapTraversalNode, 100)
	site.TraverseSiteMap(ch)

	next := <-ch
	if next.Depth != 0 {
		t.Fatalf("First page not correct height: %d\n", next.Depth)
	}
	if next.Page == nil || next.Page.Title != titleBase {
		t.Fatalf("First page not correct: %v\n", next.Page)
	}
	// ensure children returned in correct order
	for i:=0; i < 10; i++ {
		next = <-ch
		if next.Depth != 1 {
			t.Fatalf("Child page %d not correct height: %d\n", i, next.Depth)
		}
		if next.Page == nil || next.Page.Title != string(i) || next.Page.Url != (urlBase + "/" + string(i)) {
			t.Fatalf("Child page %d not correct contents: %v\n", i, next)
		}
		if i == 1 {
			// this child has a child of its own which should be iterated over first
			leaf := <-ch
			if leaf.Depth != 2 {
				t.Fatalf("Leaf page not correct height: %d\n", leaf.Depth)
			}
			if leaf.Page == nil || leaf.Page.Title != "Leaf Page" || leaf.Page.Url != (urlBase + "/" + string(1) + "/leaf") {
				t.Fatalf("Leaf page not correct contents: %v\n", leaf.Page)
			}
		}
	}

	if _, ok := <-ch; ok {
		t.Fatal("Channel not closed")
	}
}

