package track

import (
	"github.com/gorilla/mux"
	"github.com/sourcegraph/go-nnz/nnz"
	"log"
	"net/http"
	"time"
)

// ViewIDHeader is the HTTP request header ("X-Track-View") that contains the view
// instance and sequence number.
const ViewIDHeader = "X-Track-View"

// TrackAPICall wraps an API endpoint handler and records incoming API calls.
func TrackAPICall(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := &Call{
			URL:         r.URL.String(),
			Route:       mux.CurrentRoute(r).GetName(),
			RouteParams: mapStringStringAsParams(mux.Vars(r)),
			QueryParams: mapStringSliceOfStringAsParams(r.URL.Query()),
			Date:        time.Now(),
		}

		// Try to get the current view info from the X-Track-View header.
		viewID, err := GetViewID(r)
		if err != nil {
			log.Printf("GetViewID failed: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if viewID != nil {
			c.Instance = viewID.Instance
			c.ViewSeq = nnz.Int(viewID.Seq)
		}

		// Otherwise, try to get the instance from the request context.
		if c.Instance == 0 {
			if i := GetInstance(r); i != 0 {
				c.Instance = i
			}
		}

		err = InsertCall(c)
		if err != nil {
			log.Printf("InsertCall failed: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		status := CallStatus{
			CallID:   c.ID,
			Panicked: true, // assume the worst, set false if no panic
		}
		start := time.Now()
		rw := newRecorder(w)
		defer func() {
			status.Duration = time.Since(start).Nanoseconds()
			status.BodyLength = rw.BodyLength
			status.HTTPStatusCode = rw.Code
			err := InsertCallStatus(&status)
			if err != nil {
				log.Printf("warn: UpdateCallStatus failed (ID=%d): %s", c.ID, err)
			}
		}()

		h.ServeHTTP(rw, r)

		status.Panicked = false
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
