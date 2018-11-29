package main

import (
	"strings"
	"testing"
)
// helper function to validate a parsed page is correct
func validatePage(t *testing.T, err error, page *WebPage, expectedUrl string, expectedTitle string, expectedLinks []string) {
	if err != nil || page == nil {
		t.Fatalf("Failed to parse valid HTML: %v", err)
	}
	if page.Url != expectedUrl {
		t.Fatalf("Page returned with incorrect url: Got %s, expected %s", page.Url, expectedUrl)
	}
	if page.Title != expectedTitle {
		t.Fatalf("Page returned with incorrect title: Got %s, Expected %s", page.Title, expectedTitle)
	}

	if expectedLinks != nil {
		expectedCount := len(expectedLinks)
		for _ , expected := range expectedLinks {
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

	Url := "https://example.com"
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
	expectedLinks := []string { "http://example.com/1",
								"https://example.com/3",
								"https://example.com/2" }
	page, err := parser.ParseDocument(Url, strings.NewReader(html))
	validatePage(t,err, page, Url, "Page Title", expectedLinks)
}

func TestParseDocumentNoLinks(t *testing.T) {

	Url := "http://example2.com"
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
	page, err := parser.ParseDocument(Url, strings.NewReader(html))
	validatePage(t,err, page, Url, "Page Title 2", nil)
}

func TestParseMultiLineTitle(t *testing.T) {

	Url := "http://example2.com"
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
	page, err := parser.ParseDocument(Url, strings.NewReader(html))
	validatePage(t,err, page, Url, "Page Title 2", nil)
}