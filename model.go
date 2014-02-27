package appmon

import (
	"time"

	"github.com/sourcegraph/go-nnz/nnz"
)

// Call represents an API call made by a client.
type Call struct {
	// ID is the unique ID of this call.
	ID int64

	// ParentCallID is the ID of the call that this call originated from.
	ParentCallID nnz.Int64

	// App is the string identifier of the application (e.g., "web" or "ios").
	App string

	// Host is the physical machine that handled this call.
	Host string

	// RemoteAddr is the client's IP address, in dotted string notation (e.g., "1.2.3.4").
	RemoteAddr string

	// UserAgent is the client's User-Agent string.
	UserAgent string

	// UID is the user ID of the authenticated user, if any.
	UID nnz.Int

	// URL is the full URL of the request.
	URL string

	// HTTPMethod is the HTTP method of the request (GET, POST, etc.).
	HTTPMethod string

	// Route is the name of the route used to handle this request.
	Route string

	// RouteParams is a map of the route parameters in the request.
	RouteParams Params

	// QueryParams is a map of the querystring parameters in the request.
	QueryParams Params

	// Start is when the request began.
	Start time.Time

	CallStatus
}

func (c *Call) Duration() time.Duration {
	return c.End.Time.Sub(c.Start)
}

type CallStatus struct {
	// End is when the request was finished processing.
	End NullTime

	// BodyPrefix is the portion of the request body that has been read (up to BodyPrefixLimit)
	BodyPrefix nnz.String

	// BodyLength is the length, in bytes, of the HTTP response body.
	BodyLength int

	// HTTPStatusCode is the HTTP response status code.
	HTTPStatusCode int

	// Err is the error message, if any.
	Err nnz.String
}

// Params is a map of parameters for states and calls.
type Params map[string]interface{}

const BodyPrefixLimit = 1000
