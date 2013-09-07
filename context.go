package track

import (
	"github.com/gorilla/context"
	"net/http"
)

type contextKey int

const (
	viewID contextKey = iota
	clientID
)

// GetViewID returns the view ID stored in the HTTP request context, or nil if none
// exists.
func GetViewID(r *http.Request) (id *ViewID) {
	v := context.Get(r, viewID)
	if v != nil {
		id, _ = v.(*ViewID)
	}
	return
}

func setViewID(r *http.Request, id ViewID) {
	context.Set(r, viewID, &id)
}

// GetClientID returns the client ID stored in the HTTP request context, or 0 if none
// exists.
func GetClientID(r *http.Request) (id int64) {
	v := context.Get(r, clientID)
	if v != nil {
		id, _ = v.(int64)
	}
	return
}

func setClientID(r *http.Request, id int64) {
	context.Set(r, clientID, id)
}
