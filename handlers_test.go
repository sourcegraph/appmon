package track

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sourcegraph/go-nnz/nnz"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestTrackAPICall_NoAssociatedView(t *testing.T) {
	dbSetUp()
	httpSetUp()
	defer dbTearDown()
	defer httpTearDown()

	var called bool
	h := TrackAPICall(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		viewID, err := GetViewID(r)
		if err != nil {
			t.Fatal("GetViewID", err)
		}
		if viewID != nil {
			t.Errorf("want viewID == nil, got %+v", viewID)
		}
	}))

	routeName := "abc"
	rt := mux.NewRouter()
	rt.Path(`/abc/{name:\w+}/{id:[0-9]+}`).Methods("GET").Handler(h).Name(routeName)
	rootMux.Handle("/", rt)

	wantCall := &Call{
		URL:         "/abc/alice/123?foo=bar",
		Route:       routeName,
		RouteParams: map[string]interface{}{"name": "alice", "id": "123"},
		QueryParams: map[string]interface{}{"foo": []interface{}{"bar"}},
	}

	url, err := rt.GetRoute(routeName).URL("name", "alice", "id", "123")
	if err != nil {
		t.Fatal("GetRoute", err)
	}
	url = serverURL.ResolveReference(url)
	url.RawQuery = "foo=bar"
	httpGet(t, url.String(), "", "")

	// Check that call was tracked.
	if !called {
		t.Errorf("!called")
	}
	call := getOnlyOneCall(t)
	// ID and Date vary, so don't bother checking them.
	call.ID = 0
	call.Date = time.Time{}
	if !reflect.DeepEqual(wantCall, call) {
		t.Errorf("want call == %+v, got %+v", wantCall, call)
	}
}

func TestTrackAPICall_WithAssociatedView(t *testing.T) {
	dbSetUp()
	httpSetUp()
	defer dbTearDown()
	defer httpTearDown()

	wantViewID := &ViewID{Instance: 123, Seq: 456}
	var called bool
	h := TrackAPICall(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		viewID, err := GetViewID(r)
		if err != nil {
			t.Fatal("GetViewID", err)
		}
		if !reflect.DeepEqual(wantViewID, viewID) {
			t.Errorf("want viewID == %+v, got %+v", wantViewID, viewID)
		}
	}))

	rt := mux.NewRouter()
	rt.Path(`/`).Methods("GET").Handler(h)
	rootMux.Handle("/", rt)

	wantCall := &Call{Instance: wantViewID.Instance, ViewSeq: nnz.Int(wantViewID.Seq)}

	httpGet(t, serverURL.String(), ViewIDHeader, makeViewIDHeader(*wantViewID))

	// Check that call was tracked.
	if !called {
		t.Errorf("!called")
	}
	call := getOnlyOneCall(t)
	if !reflect.DeepEqual(wantCall.ViewID(), call.ViewID()) {
		t.Errorf("want call.View == %+v, got %+v", wantCall.ViewID(), call.ViewID())
	}
}

func makeViewIDHeader(id ViewID) string {
	return fmt.Sprintf("%d %d", id.Instance, id.Seq)
}

// getOnlyOneCall returns the only Call in the database if there is exactly 1
// Call in the database, and calls t.Fatalf otherwise.
func getOnlyOneCall(t *testing.T) *Call {
	calls, err := QueryCalls("")
	if err != nil {
		t.Fatal("QueryCalls", err)
	}
	if len(calls) != 1 {
		t.Fatalf("want len(calls) == 1, got %d", len(calls))
	}
	return calls[0]
}

func TestParseViewIDHeader(t *testing.T) {
	tests := []struct {
		input string
		want  ViewID
		err   bool
	}{}
	for _, test := range tests {
		got, err := parseViewIDHeader(test.input)
		if test.err && err == nil {
			t.Fatal("%q: want err != nil, got nil", test.input)
		} else if !test.err && err != nil {
			t.Fatal("%q: want err == nil, got %q", test.input, err)
		}
		if test.want != *got {
			t.Errorf("%q: want viewID == %+v, got %+v", test.input, test.want, *got)
		}
	}
}
