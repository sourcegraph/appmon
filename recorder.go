package appmon

import (
	"bufio"
	"net"
	"net/http"
)

// responseRecorder is an implementation of http.ResponseWriter that
// records its HTTP status code and body length.
type responseRecorder struct {
	Code       int // the HTTP response code from WriteHeader
	BodyLength int

	underlying http.ResponseWriter
}

// newRecorder returns an initialized ResponseRecorder.
func newRecorder(underlying http.ResponseWriter) *responseRecorder {
	return &responseRecorder{underlying: underlying}
}

// Header returns the header map from the underlying ResponseWriter.
func (rw *responseRecorder) Header() http.Header {
	return rw.underlying.Header()
}

// Write always succeeds and writes to rw.Body, if not nil.
func (rw *responseRecorder) Write(buf []byte) (int, error) {
	rw.BodyLength += len(buf)
	if rw.Code == 0 {
		rw.Code = http.StatusOK
	}
	return rw.underlying.Write(buf)
}

// WriteHeader sets rw.Code.
func (rw *responseRecorder) WriteHeader(code int) {
	rw.Code = code
	rw.underlying.WriteHeader(code)
}

func (rw *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return rw.underlying.(http.Hijacker).Hijack()
}
