package track

import (
	"time"
)

// Client identifies the originator of a tracked action.
type Client struct {
	// User is the ID of the user who viewed the application state, or null if
	// the client is not logged in.
	User NullString

	// ClientID is the unique ID associated with the client.
	ClientID int64
}

// View represents an instance of a client viewing an application state.
type View struct {
	ViewID

	Client

	// State is the name of the application state that was viewed.
	State string

	// Params is a map of the state parameters for this view.
	Params Params

	// Date is when the view occurred.
	Date time.Time
}

// ViewID is the primary key for a view.
type ViewID struct {
	// Win is a unique ID assigned to the client at the initialization time
	// of the application.
	Win int

	// Seq is the sequence number of this view in the win, incremented by the client.
	Seq int
}

// Call represents an API call made by a client.
type Call struct {
	// ID is the unique ID of this call.
	ID int64

	// View is the ID of the current view of the client that initiated this request.
	View *ViewID

	// RequestURI is the portion of the requested URL after the host and port.
	RequestURI string

	// Route is the name of the route whose handler received this request.
	Route string

	// RouteParams is a map of the route parameters in the request.
	RouteParams Params

	// QueryParams is a map of the querystring parameters in the request.
	QueryParams Params

	// Date is when the request occurred.
	Date time.Time
}

// Params is a map of parameters for states and calls.
type Params map[string]interface{}
