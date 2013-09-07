package track

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// GetViewID returns the current view ID (from the X-Track-View request header),
// or nil if none exists.
func GetViewID(r *http.Request) (*ViewID, error) {
	viewIDStr := r.Header.Get(ViewIDHeader)
	if viewIDStr == "" {
		return nil, nil
	}
	return parseViewIDHeader(viewIDStr)
}

func parseViewIDHeader(value string) (id *ViewID, err error) {
	values := strings.Split(strings.TrimSpace(value), " ")
	if len(values) != 2 {
		err = fmt.Errorf("ViewID header has %d values; must have exactly 2", len(values))
		return
	}
	id = new(ViewID)
	id.Instance, err = strconv.Atoi(values[0])
	if err != nil {
		return
	}
	id.Seq, err = strconv.Atoi(values[1])
	return
}
