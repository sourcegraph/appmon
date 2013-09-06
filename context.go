package track

import (
	"github.com/gorilla/context"
	"net/http"
)

type contextKey int

const (
	viewID contextKey = iota
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
