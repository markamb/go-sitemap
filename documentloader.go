package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// DocumentLoader interface for loading and parsing documents from URLs and returning the WebPage
type DocumentLoader interface {

	// LoadURL method loads a URL supplied as a string and returns a WebPage representing its contents
	// Only HTML documents are processed, with all other types being ignored.
	LoadURL(urlStr string) (*WebPage, error)
}

// DocLoader implements the DocumentLoader interface using HTTP to fetch the document and parses
// it using the supplied DocumentParser interface.
type DocLoader struct {
	parser DocumentParser // store the interface used to parse pages as they are loaded
}

// CreateDocumentLoader creates a document loader using the supplied DocumentParser interface
func CreateDocumentLoader(p DocumentParser) *DocLoader {
	return &DocLoader{parser: p}
}

// LoadURL loads then parses a web document. See DocumentLoader interface for details.
func (loader *DocLoader) LoadURL(urlStr string) (*WebPage, error) {
	start := time.Now()
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		return nil, fmt.Errorf("unsupported content type %v for URL (%v)", contentType, urlStr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code, status code %d (%s) for URL (%v)", resp.StatusCode, resp.Status, urlStr)
	}
	page, err := loader.parser.ParseDocument(urlStr, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse contents for URL %s :%v", urlStr, err)
	}

	loadSecs := time.Since(start).Seconds()
	log.Printf("INFO: Loaded and parsed %s in %f secs", urlStr, loadSecs)
	return page, nil
}
