package appmon

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

var (
	// mux is the HTTP request multiplexer used with the test server.
	rootMux *http.ServeMux

	// server is a test HTTP server.
	server *httptest.Server

	// serverURL is the base URL of the test HTTP server.
	serverURL *url.URL
)

func httpSetUp() {
	var err error
	rootMux = http.NewServeMux()
	server = httptest.NewServer(rootMux)
	serverURL, err = url.Parse(server.URL)
	if err != nil {
		panic("url.Parse: " + err.Error())
	}
}

func httpTearDown() {
	server.Close()
}

func httpGet(t *testing.T, url string, callID int64) (res *http.Response) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal("http.NewRequest", err)
	}

	if callID != 0 {
		addParentCallIDHeader(callID, req.Header)
	}

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal("http.DefaultClient.Do", err)
	}
	return
}
