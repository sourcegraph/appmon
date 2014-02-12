package appmon

import (
	"net/http"
)

// TracingTransport is an http.RoundTripper that adds an HTTP header to allow
// appmon-enabled handlers to read their parent call ID.
type TracingTransport struct {
	ParentCallID int64
	Transport    http.RoundTripper
}

func (t TracingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	tr := t.Transport
	if tr == nil {
		tr = http.DefaultTransport
	}

	if t.ParentCallID != 0 {
		r = cloneRequest(r)
		addParentCallIDHeader(t.ParentCallID, r.Header)
	}

	return tr.RoundTrip(r)
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
// (This function copyright goauth2 authors: https://code.google.com/p/goauth2)
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header)
	for k, s := range r.Header {
		r2.Header[k] = s
	}
	return r2
}
