package track

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ViewIDHeader is the HTTP header ("X-Track-View") that contains the view win and sequence
// passed from the client.
const ViewIDHeader = "X-Track-View"

// TrackCall wraps an http.Handler, tracking a Call in the database describing
// the HTTP request and response.
func TrackCall(h http.Handler) http.Handler {
	return storeClientID(storeViewID(storeCall(h)))
}

func storeCall(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assumes that the storeViewID handler has already run.
		viewID := GetViewID(r)
		c := &Call{
			View:        viewID,
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
		if viewIDStr := r.Header.Get(ViewIDHeader); viewIDStr != "" {
			viewID, err := parseViewIDHeader(viewIDStr)
			if err != nil {
				log.Printf("ParseInt on %s header failed: %s (header value is %q)", ViewIDHeader, err, viewIDStr)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			setViewID(r, viewID)
		}

		h.ServeHTTP(w, r)
	})
}

func parseViewIDHeader(value string) (id ViewID, err error) {
	values := strings.Split(strings.TrimSpace(value), " ")
	if len(values) != 2 {
		err = fmt.Errorf("ViewID header has %d values; must have exactly 2", len(values))
		return
	}
	id.Win, err = strconv.Atoi(values[0])
	if err != nil {
		return
	}
	id.Seq, err = strconv.Atoi(values[1])
	return
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
