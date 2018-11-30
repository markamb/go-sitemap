package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

//
// Create mock document parer
//
type MockParser struct {
	calls 			int			// number of calls made
	recievedUrl		string		// URL of first call
	recievedDoc		string		// document supplied in 1st call
	result 			*WebPage	// result to return
	err 			error 		// result to return
}

// Mock Document Parser - Just store the doc being parsed
func (m *MockParser) ParseDocument(urlStr string, reader io.Reader) (*WebPage, error) {
	m.recievedUrl = urlStr
	if b, err := ioutil.ReadAll(reader); err == nil {
		m.recievedDoc = string(b)
	}
	m.calls++
	return m.result, m.err
}

func TestDocumentLoader(t *testing.T) {

	doc := "My Test Document Contents"
	path := "/mypath/mydoc.html"

	// mock server request handler
	mockHandler := func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Content-Type", "text/html more stuff")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(doc))		// return our document
	}

	mockServer := httptest.NewServer(http.HandlerFunc(mockHandler))
	defer mockServer.Close()

	mockParser := &MockParser{
		result: 	&WebPage {Title: "My Web Page Title" },
		err:		nil,
	}
	docLoader := CreateDocumentLoader(mockParser)
	Url := mockServer.URL + path
	page, err := docLoader.LoadUrl(Url)

	// validate
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if mockParser.calls != 1 {
		t.Errorf("Incorrect number of calls to mock server: expected %d, got %d", 1, mockParser.calls)
	}
	if mockParser.recievedUrl !=  Url {
		t.Errorf("Incorrect Url sent to mock parser: expected %s, got %s", Url, mockParser.recievedUrl)
	}
	if mockParser.recievedDoc != doc {
		t.Errorf("Incorrect contents sent to mock parser: expected %s, got %s", doc, mockParser.recievedDoc)
	}
	if page != mockParser.result {
		t.Errorf("Incorrect result from LoadUrl: expected %v, got %v", mockParser.result, page)
	}
}

func TestDocumentLoaderBadContentType(t *testing.T) {
	doc := "My Test Document Contents"

	// mock server request handler
	mockHandler := func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Content-Type", "text/json more stuff")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(doc))		// return our document
	}

	mockServer := httptest.NewServer(http.HandlerFunc(mockHandler))
	defer mockServer.Close()

	mockParser := &MockParser{ }
	docLoader := CreateDocumentLoader(mockParser)
	page, err := docLoader.LoadUrl(mockServer.URL + "/path")

	// validate
	// Unsupported content type - mock should not have been called
	if mockParser.calls != 0 {
		t.Errorf("Incorrect number of calls to mock server: expected %d, got %d", 1, mockParser.calls)
	}
	if page != nil {
		t.Errorf("Incorrect result from LoadUrl: expected %v, got %v", nil, page)
	}
	if err == nil {
		t.Error("Missing expected error from LoadUrl")
	}
}

func TestDocumentLoaderBadResponseCode(t *testing.T) {
	doc := "My Test Document Contents"

	// mock server request handler
	mockHandler := func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
		rw.Header().Add("Content-Type", "text/html more stuff")
		rw.Write([]byte(doc))		// return our document
	}

	mockServer := httptest.NewServer(http.HandlerFunc(mockHandler))
	defer mockServer.Close()

	mockParser := &MockParser{ }
	docLoader := CreateDocumentLoader(mockParser)
	page, err := docLoader.LoadUrl(mockServer.URL + "/path")

	// validate
	// Error status code returned
	if mockParser.calls != 0 {
		t.Errorf("Incorrect number of calls to mock server: expected %d, got %d", 1, mockParser.calls)
	}
	if page != nil {
		t.Errorf("Incorrect result from LoadUrl: expected %v, got %v", nil, page)
	}
	if err == nil {
		t.Error("Missing expected error from LoadUrl")
	}
}

