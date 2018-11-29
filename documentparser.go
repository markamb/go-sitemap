package main

import (
	"fmt"
	"golang.org/x/net/html"
	"io"
	"net/url"
	"strings"
)

//
// DocumentParser interface is used to parse the contents of a document loaded from
// a URL and create a WebPage from the contents
//
// Known issues / unsupported features:
//		<BASE> tag on a page is not supported
//
//
type DocumentParser interface {
	//
	// ParseDocument takes a URL and the contents of page stored there and parses it into a WebPage structure.
	// The document is assumed to contain HTML
	//
	ParseDocument(urlStr string, reader io.Reader) (*WebPage, error)
}


type DocParser struct {
}

func CreateDocumentParser() *DocParser {
	return &DocParser{}
}

// ParseDocument parses an HTML document and extracts a WebPage. See DocumentParser interface for details
func (*DocParser) ParseDocument(urlStr string, reader io.Reader) (*WebPage, error) {

	// first parse the Url to allow relative href links to be correctly calculated
	parentUrl, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	rootNode, err := html.Parse(reader)
	if err != nil {
		return nil, err
	}

	page := CreateWebPage(parentUrl.String(), "")
	err = parseNode(rootNode, parentUrl, page)
	if err != nil {
		return nil, err
	}
	return page, nil
}

// Recursively parse the details of the node into the page structure
func parseNode(node *html.Node, parentUrl *url.URL, page *WebPage) error {

	// is this a link?
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if strings.EqualFold(attr.Key,"href") {
				internal, absUrl, err := parseUrl(parentUrl, attr.Val)
				if err != nil {
					return err
				} else if internal {
//					log.Printf("TRACE: Found internal href to %v\n", absUrl)
					page.InternalLinks[absUrl] = true
				} else {
//					log.Printf("TRACE: Skipping external or invalid href to %v\n", attr.Val)
				}
				break
			}
		}
	}

	// is it the title
	if node.Type == html.ElementNode && strings.EqualFold(node.Data, "title") {
		if node.FirstChild != nil && node.FirstChild.Type == html.TextNode {
			// trim whitespace then take the first line as the title
			title := strings.TrimSpace(node.FirstChild.Data)
			if idx := strings.Index(title, "\n"); idx >= 0 {
				title = strings.Split(title, "\n")[0]
			}
			page.Title = title
		}
	}

	for child := node.FirstChild; child != nil; child= child.NextSibling {
		err := parseNode(child, parentUrl, page)
		if err != nil {
			return err
		}
	}
	return nil
}

//
// Parse the URL urlStr and test if it is a valid link to a page on the same domain as the parent.
// Returns 3 fields:
//		bool	is this a valid url on the same domain as the parent
//		string	absolute URL
//		error	error if invalid inputs supplied (note invalid href string is not considered an error)
//
func parseUrl(parent *url.URL, href string) (bool, string, error) {

	// first a sanity check - the parent must be an absolute url
	if !parent.IsAbs() {
		return false, "", fmt.Errorf("cannot resolve href as relative URL passed as parent: %v", href)
	}

	// First make sure its a valid url
	url, err := url.Parse(href)
	if err != nil {
		return false, "", nil
	}

	// is it a supported scheme
	if len(url.Scheme) != 0 && url.Scheme != "http" && url.Scheme != "https" {
		return false, "", nil
	}

	// check the domain
	if url.IsAbs() && !sameHost(url.Host, parent.Host) {
		return false, "", nil		// different domain
	}

	// same port?
	if len(url.Port()) != 0 && url.Port() != parent.Port() {
		return false, "", nil		// different port
	}

	// convert to an absolute URL
	result := parent.ResolveReference(url)

	// If they resolve to the same URL as the parent we ignore it
	// Note we only care about the path (not scheme, fragment or query)
	if result.Path == parent.Path {
		return false, "", nil		// link back to itself
	}

	// clean it up a bit
	result.Fragment = ""
	return true, result.String(), nil
}

// sameHost checks if 2 hosts represent the same domain.
// We consider  example.com and www.example.com to be the same domain.
func sameHost(h1 string, h2 string) bool {
	h1 = strings.TrimPrefix(h1, "www.")
	h2 = strings.TrimPrefix(h2, "www.")
	return h1 == h2
}