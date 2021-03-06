package appmon

import (
	"github.com/gorilla/mux"
	"github.com/sourcegraph/go-nnz/nnz"
	"log"
	"net/http"
	"time"
)

// CurrentUser, if set, is called to determine the currently authenticated user
// for the current request. The returned user ID is stored in the Call record if
// nonzero.
var CurrentUser func(r *http.Request) int

func BeforeAPICall(app string, r *http.Request) {
	c := &Call{
		App:         app,
		Host:        hostname,
		RemoteAddr:  r.RemoteAddr,
		UserAgent:   r.UserAgent(),
		URL:         r.URL.String(),
		HTTPMethod:  r.Method,
		Route:       mux.CurrentRoute(r).GetName(),
		RouteParams: mapStringStringAsParams(mux.Vars(r)),
		QueryParams: mapStringSliceOfStringAsParams(r.URL.Query()),
		Start:       time.Now().In(time.UTC),
	}
	if parentCallID, ok := GetParentCallID(r); ok {
		c.ParentCallID = nnz.Int64(parentCallID)
	}
	if CurrentUser != nil {
		c.UID = nnz.Int(CurrentUser(r))
	}

	err := insertCall(c)
	if err != nil {
		log.Printf("insertCall failed: %s", err)
	}
	setCallID(r, c.ID)
}

func AfterAPICall(r *http.Request, bodyLength, code int, errStr string) {
	callID, ok := GetCallID(r)
	if !ok {
		log.Printf("AfterAPICall: no CallID")
		return
	}

	err := setCallStatus(callID, &CallStatus{
		End:            now(),
		BodyLength:     bodyLength,
		HTTPStatusCode: code,
		Err:            nnz.String(errStr),
	})
	if err != nil {
		log.Printf("setCallStatus failed for call ID %d: %s", callID, err)
	}
}

type Handler struct {
	App     string
	Handler http.Handler
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	BeforeAPICall(h.App, r)

	rw := newRecorder(w)
	h.Handler.ServeHTTP(rw, r)

	AfterAPICall(r, rw.BodyLength, rw.Code, "")
}

// TrackAPICall wraps an API endpoint handler and records incoming API calls.
func TrackAPICall(app string, h http.Handler) http.Handler {
	return Handler{app, h}
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
