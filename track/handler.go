package track

import (
	"database/sql"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"time"
)

// ViewIDHeader is the HTTP header ("X-View-ID") that contains the view ID
// passed from the client.
const ViewIDHeader = "X-View-ID"

// TrackCall wraps an http.Handler, tracking a Call in the database describing
// the HTTP request and response.
func TrackCall(h http.Handler) http.Handler {
	return storeViewID(storeCall(h))
}

func storeCall(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assumes that the storeViewID handler has already run.
		viewID := ViewID(r)
		c := &Call{
			ViewID:      sql.NullInt64{viewID, viewID != 0},
			RequestURI:  r.RequestURI,
			Route:       mux.CurrentRoute(r).GetName(),
			RouteParams: mapStringStringAsParams(mux.Vars(r)),
			QueryParams: mapStringSliceOfStringAsParams(r.URL.Query()),
			Date:        time.Now(),
		}
		err := InsertCall(c)
		if err != nil {
			log.Printf("InsertCall failed: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func storeViewID(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		viewIDStr := r.Header.Get(ViewIDHeader)
		if viewIDStr != "" {
			viewID, err := strconv.ParseInt(viewIDStr, 10, 64)
			if err != nil || viewID == 0 {
				if err != nil {
					log.Printf("ParseInt on %s header failed: %s (header value is %q)", ViewIDHeader, err, viewIDStr)
				}
				if viewID == 0 {
					log.Printf("%s header value is 0", ViewIDHeader)
				}
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			setViewID(r, viewID)
		}

		h.ServeHTTP(w, r)
	})
}

func mapStringStringAsParams(m map[string]string) (p Params) {
	p = make(Params)
	for k, v := range m {
		p[k] = v
	}
	return
}

func mapStringSliceOfStringAsParams(m map[string][]string) (p Params) {
	p = make(Params)
	for k, v := range m {
		p[k] = v
	}
	return
}
