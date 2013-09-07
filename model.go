package track

import (
	"github.com/sourcegraph/go-nnz/nnz"
	"time"
)

// ClientInfo represents additional information about a client.
type ClientInfo struct {
	// IPAddress is the client's IP address, in dotted string notation (e.g., "1.2.3.4").
	IPAddress string

	// UserAgent is the client's User-Agent string.
	UserAgent string
}

// Instance represents a single instantiation of the app; e.g., all of the
// activity that occurs in one browser tab, starting with loading the
// initial static base page.
type Instance struct {
	// ID is the primary key.
	ID int

	// ClientID is the unique ID assigned to the client, stored in a cookie.
	ClientID int64

	// User is the login of the current user, or "" if the user is not logged in.
	User nnz.String

	// URL is the original (i.e., first) URL requested by the client in this instance.
	URL string

	// ReferrerURL is the value of the HTTP "Referer" header.
	ReferrerURL string

	ClientInfo

	// Start is when the application was instantiated.
	Start time.Time
}

// ViewID is the primary key for a View.
type ViewID struct {
	// Instance is the ID of the app instance that this view occurred in.
	Instance int

	// Seq is the sequence number of this view in the instance, incremented by the client.
	Seq int
}

// View represents a client's viewing an application state.
type View struct {
	ViewID

	// RequestURI is the request URI of the URL in the client's address bar
	// after the state has loaded. The host and port of the URL are guaranteed
	// to be the same as that of the instance's URL.
	RequestURI string

	// State is the name of the application state that was viewed.
	State string

	// StateParams is a map of the state parameters for this view.
	StateParams Params

	// Date is when the view occurred.
	Date time.Time
}

// Call represents an API call made by a client.
type Call struct {
	// ID is the unique ID of this call.
	ID int64

	// Instance is the ID of the instance that this call occurred in.
	Instance int

	// ViewSeq is current view seq at the time the client initiated this
	// call. It is nonzero only if the client sends the X-Track-View header.
	ViewSeq nnz.Int

	// URL is the full URL of the request.
	URL string

	// Route is the name of the route used to handle this request.
	Route string

	// RouteParams is a map of the route parameters in the request.
	RouteParams Params

	// QueryParams is a map of the querystring parameters in the request.
	QueryParams Params

	// Date is when the request occurred.
	Date time.Time
}

func (c *Call) ViewID() *ViewID {
	return &ViewID{Instance: c.Instance, Seq: int(c.ViewSeq)}
}

// CallStatus represents status information that is collected after a call.
type CallStatus struct {
	// CallID is the call ID that this status information describes.
	CallID int64

	// Duration is the total execution time (in nanoseconds) of the inner HTTP
	// handler, not counting handlers in the track package.
	Duration int64

	// BodyLength is the length, in bytes, of the HTTP response body.
	BodyLength int

	// HTTPStatusCode is the HTTP response status code.
	HTTPStatusCode int

	// Panicked is true iff the inner HTTP handler panicked (without recovering
	// itself).
	Panicked bool
}

// Params is a map of parameters for states and calls.
type Params map[string]interface{}
