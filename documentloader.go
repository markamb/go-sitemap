package main

import (
	"log"
	"net/http"
	"strings"
	"time"
)

//
// Interface and implementation for loading documents from URLs.
//
type DocumentLoader interface {

	//
	// LoadUrl method loads a URL supplied as a string and returns a WebPage representing its contents
	// Only HTML documents are processed, with all other types being ignored.
	//
	LoadUrl(urlStr string) *WebPage

}

//
// DocLoader implements the DocumentLoader interface using HTTP to fetch the document and then parsing
// it using the supplied DocumentParser interface.
//
type DocLoader struct {
	parser DocumentParser	// store the interface used to parse pages as they are loaded
}

//
// Create a document loader which uses the supplied DocumentParser interface
// for parsing HTML documents as they are loaded
//
func CreateDocumentLoader(p DocumentParser) *DocLoader {
	return &DocLoader { parser: p}
}


// loadUrl loads then parses a we document. See DocumentLoader interface for details.
func (loader *DocLoader) LoadUrl(urlStr string) *WebPage {
	start := time.Now()
	resp, err := http.Get(urlStr)
	if err != nil {
		log.Printf("WARN: Failed to load Url (%v) : %v\n", urlStr, err)
		return nil
	}
	defer resp.Body.Close()
	if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		log.Printf("TRACE: Ignoring content type %v for Url (%v)\n", contentType, urlStr)
		return nil
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("WARN: Ignoring page, status code %d (%s) for Url (%v)\n", resp.StatusCode, resp.Status, urlStr)
		return nil
	}
	page, err := loader.parser.ParseDocument(urlStr, resp.Body)

	loadSecs := time.Since(start).Seconds()
	log.Printf("INFO: Loaded and parsed %s in %f secs\n", urlStr, loadSecs)
	return page
}
