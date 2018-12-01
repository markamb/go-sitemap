package main

import (
	"net/url"
	"strings"
	"testing"
)

// validatePage helper function which validates a parsed page is correct
func validatePage(t *testing.T, err error, page *WebPage, expectedURL string, expectedTitle string, expectedLinks []string) {
	if err != nil || page == nil {
		t.Fatalf("Failed to parse valid HTML: %v", err)
	}
	if page.URL.String() != expectedURL {
		t.Fatalf("Page returned with incorrect url: Got %s, expected %s", page.URL, expectedURL)
	}
	if page.Title != expectedTitle {
		t.Fatalf("Page returned with incorrect title: Got %s, Expected %s", page.Title, expectedTitle)
	}

	if expectedLinks != nil {
		expectedCount := len(expectedLinks)
		for _, expected := range expectedLinks {
			if _, found := page.InternalLinks[expected]; !found {
				t.Fatalf("Failed to find expected link %s in page, have %v", expected, page.InternalLinks)
			}
			expectedCount--
		}
		if expectedCount != 0 {
			t.Fatalf("Unexpected extra links in page: %v", page.InternalLinks)
		}
	}
}

func TestParseDocument(t *testing.T) {

	URL := "https://example.com"
	html := `
<HTML>
	<HEAD>
		<TITLE>Page Title</TITLE>
		<SCRIPT></SCRIPT>
	</HEAD>
	<BODY>
		<H1>Something Big</H1>
		<a href="https://example.com">HTTPS Link</a>
		<a title="stuff" href="http://example.com">HTTP Link</a>
		<a title="stuff" href="http://example.com/1">Abs Link</a>
		<a href="/2">Relative Link</a>
		<a href="/2">Duplicate Link</a>
		<a href="/3">New Relative Link</a>
		<a href="https://example.com/3">Absolute Duplicate</a>
		<a href="http://anotherdomain.com/1">Different Domain</a>
		<a href="https://example.com:8080">Different Port</a>
		<img src="picture.jpg">
		
		<P>An unsupported <B>link type</B>
		Send me mail at <a href="mailto:support@yourcompany.com">

		<BR>More Stuff
	</BODY>
</HTML>`

	var parser DocumentParser
	parser = CreateDocumentParser()
	expectedLinks := []string{"http://example.com/1",
		"https://example.com/3",
		"https://example.com/2"}
	page, err := parser.ParseDocument(URL, strings.NewReader(html))
	validatePage(t, err, page, URL, "Page Title", expectedLinks)
}

func TestParseDocumentNoLinks(t *testing.T) {

	URL := "http://example2.com"
	html := `
<HTML>
	<HEAD>
		<TITLE>Page Title 2</TITLE>
		<SCRIPT></SCRIPT>
	</HEAD>
	<BODY>
		<H1>Something Big</H1>
		<img src="picture.jpg">

		<a href="http://anotherdomain.com/1">Link Name</a>

		<P>An unsupported <B>link type</B>
		Send me mail at <a href="mailto:support@yourcompany.com">

		<BR>More Stuff
	</BODY>
</HTML>`

	var parser DocumentParser
	parser = CreateDocumentParser()
	page, err := parser.ParseDocument(URL, strings.NewReader(html))
	validatePage(t, err, page, URL, "Page Title 2", nil)
}

func TestParseMultiLineTitle(t *testing.T) {

	URL := "http://example2.com"
	html := `
<HTML>
	<HEAD>
		<TITLE>
			Page Title 2
			with some extra stuff added

		</TITLE>
		<SCRIPT></SCRIPT>
	</HEAD>
	<BODY>
		<H1>Something Big</H1>
	</BODY>
</HTML>`

	var parser DocumentParser
	parser = CreateDocumentParser()
	page, err := parser.ParseDocument(URL, strings.NewReader(html))
	validatePage(t, err, page, URL, "Page Title 2", nil)
}

func doTestURLParsing(t *testing.T, parser *DocParser, parent *url.URL, testURL string, expectedInternal bool, expectedURL string) {

	internal, newURL, err := parser.parseURL(parent, testURL)
	if err != nil {
		t.Fatalf("Unexpecyted error parsing URL: %v", err)
	}
	if internal != expectedInternal {
		t.Fatalf("Internal lookup incorrect for url %s: expected %v, got %v", testURL, expectedInternal, internal)
	}
	if newURL != expectedURL {
		t.Fatalf("Resulting URL incorrect for url %s: expected %v, got %v", testURL, expectedURL, newURL)
	}
}

func TestURLParser(t *testing.T) {

	parser := CreateDocumentParser()

	parent, _ := url.Parse("http://en.wikipedia.com")
	doTestURLParsing(t, parser, parent, "http://www.wikimediafoundation.org", false, "")
	doTestURLParsing(t, parser, parent, "http://www.wikimediafoundation.org/path", false, "")
	doTestURLParsing(t, parser, parent, "www.wikimediafoundation.org", false, "")
	doTestURLParsing(t, parser, parent, "www.wikimediafoundation.org/path", false, "")
	doTestURLParsing(t, parser, parent, "wikimediafoundation.org", false, "")
	doTestURLParsing(t, parser, parent, "wikimediafoundation.org/path", false, "")

	parent, _ = url.Parse("http://en.wikipedia.com/a/path")
	doTestURLParsing(t, parser, parent, "http://www.wikimediafoundation.org", false, "")
	doTestURLParsing(t, parser, parent, "http://www.wikimediafoundation.org/path", false, "")
	doTestURLParsing(t, parser, parent, "www.wikimediafoundation.org", false, "")
	doTestURLParsing(t, parser, parent, "www.wikimediafoundation.org/path", false, "")
	doTestURLParsing(t, parser, parent, "wikimediafoundation.org", false, "")
	doTestURLParsing(t, parser, parent, "wikimediafoundation.org/path", false, "")

	parent, _ = url.Parse("http://en.wikipedia.com:8080/path")
	doTestURLParsing(t, parser, parent, "http://en.wikipedia.com/path2", false, "")
	doTestURLParsing(t, parser, parent, "http://www.wikimediafoundation.org/path", false, "")
	doTestURLParsing(t, parser, parent, "www.wikimediafoundation.org", false, "")
	doTestURLParsing(t, parser, parent, "www.wikimediafoundation.org/path", false, "")
	doTestURLParsing(t, parser, parent, "wikimediafoundation.org", false, "")
	doTestURLParsing(t, parser, parent, "wikimediafoundation.org/path", false, "")

	// now some which do match
	parent, _ = url.Parse("http://en.wikipedia.com/path")
	doTestURLParsing(t, parser, parent, "http://en.wikipedia.com", true, "http://en.wikipedia.com")
	doTestURLParsing(t, parser, parent, "http://en.wikipedia.com/", true, "http://en.wikipedia.com")
	doTestURLParsing(t, parser, parent, "https://en.wikipedia.com", true, "https://en.wikipedia.com")
	doTestURLParsing(t, parser, parent, "https://en.wikipedia.com/", true, "https://en.wikipedia.com")
	doTestURLParsing(t, parser, parent, "https://en.wikipedia.com/newpath", true, "https://en.wikipedia.com/newpath")
	doTestURLParsing(t, parser, parent, "https://en.wikipedia.com/newpath?ABC", true, "https://en.wikipedia.com/newpath?ABC")
	doTestURLParsing(t, parser, parent, "en.wikipedia.com", true, "http://en.wikipedia.com")
	doTestURLParsing(t, parser, parent, "en.wikipedia.com/", true, "http://en.wikipedia.com")
	doTestURLParsing(t, parser, parent, "en.wikipedia.com/path/2", true, "http://en.wikipedia.com/path/2")
	doTestURLParsing(t, parser, parent, "en.wikipedia.com/path/2/", true, "http://en.wikipedia.com/path/2")

	// some more not matching
	parent, _ = url.Parse("http://en.wikipedia.com/path")
	doTestURLParsing(t, parser, parent, "en.wikipedia.com/path", false, "") // resolves to same path
	doTestURLParsing(t, parser, parent, "ftp://en.wikipedia.com/doc", false, "")
}
