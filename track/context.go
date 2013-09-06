package track

import (
	"github.com/gorilla/context"
	"net/http"
)

type contextKey int

const (
	viewID contextKey = iota
)

// ViewID returns the view ID stored in the HTTP request context, or 0 if none
// exists.
func ViewID(r *http.Request) (id int64) {
	v := context.Get(r, viewID)
	if v != nil {
		id, _ = v.(int64)
	}
	return
}

func setViewID(r *http.Request, id int64) {
	context.Set(r, viewID, id)
}
