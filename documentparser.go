package main

import (
	"fmt"
	"golang.org/x/net/html"
	"io"
	"net/url"
	"strings"
)

// DocumentParser interface is used to parse the contents of a document loaded from
// a URL and create a WebPage from the contents
type DocumentParser interface {

	// ParseDocument takes a URL and the contents of page stored there and parses it into a WebPage structure.
	// The document is assumed to contain HTML
	ParseDocument(urlStr string, reader io.Reader) (*WebPage, error)
}

// DocParser type implements the DocumentParser interface
type DocParser struct {
}

// CreateDocumentParser creates a new DocParser for parsing HTML and returning a WebPage
func CreateDocumentParser() *DocParser {
	return &DocParser{}
}

// ParseDocument parses an HTML document and extracts a WebPage. See DocumentParser interface for details
func (p *DocParser) ParseDocument(urlStr string, reader io.Reader) (*WebPage, error) {

	// first parse the URL to allow relative href links to be correctly calculated
	parentURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	rootNode, err := html.Parse(reader)
	if err != nil {
		return nil, err
	}

	page := CreateWebPage(parentURL, "")
	err = p.parseNode(rootNode, parentURL, page)
	if err != nil {
		return nil, err
	}
	return page, nil
}

// parseNode recursively parses the details of the node into the page structure
func (p *DocParser) parseNode(node *html.Node, parentURL *url.URL, page *WebPage) error {

	// is this a link?
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if strings.EqualFold(attr.Key, "href") {
				internal, absURL, err := p.parseURL(parentURL, attr.Val)
				if err != nil {
					return err
				} else if internal {
					page.InternalLinks[absURL] = true
				}
				break
			}
		}
		return nil
	}

	// is it the title?
	if node.Type == html.ElementNode && strings.EqualFold(node.Data, "title") {
		if node.FirstChild != nil && node.FirstChild.Type == html.TextNode {
			// trim whitespace then take the first line as the title
			title := strings.TrimSpace(node.FirstChild.Data)
			if idx := strings.Index(title, "\n"); idx >= 0 {
				title = strings.Split(title, "\n")[0]
			}
			page.Title = title
		}
		return nil
	}

	// no, recursively process its children
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		err := p.parseNode(child, parentURL, page)
		if err != nil {
			return err
		}
	}

	return nil
}

// parseURL parses the url and tests if it is a valid link to a page on the same domain as the parent.
// Returns 3 fields:
//		bool	is this a valid url on the same domain as the parent
//		string	absolute URL in a nomalised form
//		error	error if invalid inputs supplied (note invalid href string is not considered an error)
//
func (p *DocParser) parseURL(parent *url.URL, href string) (bool, string, error) {

	// first a sanity check - the parent must be an absolute url
	if !parent.IsAbs() {
		return false, "", fmt.Errorf("cannot resolve href as relative URL passed as parent: %v", href)
	}

	strURL := href
	if strings.HasPrefix(href, "/") {
		// relative url - create one based off the parent
		tempURL := *parent
		tempURL.Path = href
		strURL = tempURL.String()
	}
	result, err := url.Parse(strURL)
	if err != nil {
		return false, "", err
	}

	// use same scheme as parent on a relative URL
	if len(result.Scheme) == 0 {
		result.Scheme = parent.Scheme
	}

	// is it a supported scheme
	if len(result.Scheme) != 0 && result.Scheme != "http" && result.Scheme != "https" {
		return false, "", nil
	}

	// we remove any training / to ensure equivilent URLS match and ignore fragments
	result.Path = strings.TrimSuffix(result.Path, "/")
	result.Fragment = ""

	// normalise it
	result, err = url.Parse(result.String())
	if err != nil || len(result.Host) == 0 {
		return false, "", err
	}

	// check the domain
	if !sameHost(result.Host, parent.Host) {
		return false, "", nil // different domain
	}

	if len(result.Port()) != 0 && result.Port() != parent.Port() {
		return false, "", nil // different port
	}

	// If they resolve to the same URL as the parent we ignore it
	// Note we only care about the path (not scheme, fragment or query)
	if result.Path == parent.Path {
		return false, "", nil
	}

	return true, result.String(), nil
}

// sameHost checks if 2 hosts represent the same domain.
// We consider  example.com and www.example.com to be the same domain.
func sameHost(h1 string, h2 string) bool {
	h1 = strings.TrimPrefix(h1, "www.")
	h2 = strings.TrimPrefix(h2, "www.")
	return strings.EqualFold(h1, h2)
}
