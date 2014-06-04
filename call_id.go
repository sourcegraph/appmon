package appmon

import (
	"github.com/gorilla/context"
	"log"
	"net/http"
	"strconv"
)

// ParentCallIDHeader is the HTTP request header ("X-Appmon-Parent-Call-ID")
// that contains the call ID associated with the API call.
const ParentCallIDHeader = "X-Appmon-Parent-Call-ID"

// GetCallID gets the current call ID (if any) from the request context.
func GetCallID(r *http.Request) (int64, bool) {
	if v, present := context.GetOk(r, callID); present {
		id, ok := v.(int64)
		return id, ok
	}
	return 0, false
}

// GetParentCallID gets the parent call ID (if any) of the current call from the
// current HTTP request's headers.
func GetParentCallID(r *http.Request) (int64, bool) {
	v := r.Header.Get(ParentCallIDHeader)
	if v == "" {
		return 0, false
	}
	callID, err := strconv.ParseInt(v, 10, 64)
	return callID, err == nil
}

func AddParentCallIDHeader(parent *http.Request, h http.Header) {
	parentCallID, present := GetCallID(parent)
	if !present {
		log.Printf("warning: AddParentCallIDHeader: no call ID")
		return
	}
	addParentCallIDHeader(parentCallID, h)
}

func addParentCallIDHeader(parentCallID int64, h http.Header) {
	h.Set(ParentCallIDHeader, strconv.FormatInt(parentCallID, 10))
}

func setCallID(r *http.Request, id int64) {
	context.Set(r, callID, id)
}
