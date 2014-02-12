package appmon

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sourcegraph/go-nnz/nnz"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestTrackView(t *testing.T) {
	dbSetUp()
	httpSetUp()
	defer dbTearDown()
	defer httpTearDown()

	apiRouteName, viewRouteName := "api-handler", "view-handler"
	rt := mux.NewRouter()

	wantAPICall := &Call{
		App:         "my-api",
		Host:        hostname,
		URL:         "/api/123?qux=baz",
		HTTPMethod:  "GET",
		Route:       apiRouteName,
		RouteParams: map[string]interface{}{"id": "123"},
		QueryParams: map[string]interface{}{"qux": []interface{}{"baz"}},
	}
	wantViewCall := &Call{
		App:         "my-app",
		Host:        hostname,
		URL:         "/view/alice?foo=bar",
		HTTPMethod:  "GET",
		Route:       viewRouteName,
		RouteParams: map[string]interface{}{"name": "alice"},
		QueryParams: map[string]interface{}{"foo": []interface{}{"bar"}},
	}
	wantCalls := []*Call{wantAPICall, wantViewCall}

	var calledAPIHandler, calledViewHandler bool
	apiHandler := TrackAPICall("my-api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledAPIHandler = true
		if _, ok := GetParentCallID(r); !ok {
			t.Error("no ParentCallID")
		}
		if _, ok := GetCallID(r); !ok {
			t.Error("no CallID")
		}
	}))
	viewHandler := TrackAPICall("my-app", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledViewHandler = true

		if _, ok := GetParentCallID(r); ok {
			t.Error("has ParentCallID")
		}
		if callID, ok := GetCallID(r); !ok {
			t.Error("no CallID")
		} else {
			// update test expectation
			wantAPICall.ParentCallID = nnz.Int64(callID)
		}

		// Make an API request to apiHandler
		apiURI, err := rt.GetRoute(apiRouteName).URL("id", "123")
		if err != nil {
			t.Fatal(err)
		}
		apiURL := serverURL.ResolveReference(apiURI)
		apiURL.RawQuery = "qux=baz"
		req2, err := http.NewRequest("GET", apiURL.String(), nil)
		if err != nil {
			t.Fatal(err)
		}
		AddParentCallIDHeader(r, req2.Header)
		resp2, err := http.DefaultClient.Do(req2)
		if err != nil {
			t.Fatal(err)
		}
		defer resp2.Body.Close()
	}))

	rt.Path(`/api/{id:[0-9]+}`).Methods("GET").Handler(apiHandler).Name(apiRouteName)
	rt.Path(`/view/{name:[a-z]+}`).Methods("GET").Handler(viewHandler).Name(viewRouteName)
	rootMux.Handle("/", rt)

	url, err := rt.GetRoute(viewRouteName).URL("name", "alice")
	if err != nil {
		t.Fatal("GetRoute", err)
	}
	url = serverURL.ResolveReference(url)
	url.RawQuery = "foo=bar"
	httpGet(t, url.String(), 0)

	// Check that the view and calls were tracked.
	if !calledAPIHandler {
		t.Errorf("!calledAPIHandler")
	}
	if !calledViewHandler {
		t.Errorf("!calledViewHandler")
	}

	calls, err := QueryCalls("")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range calls {
		// ID and times vary, so don't bother checking them.
		c.ID, c.Start, c.End = 0, time.Time{}, NullTime{}
		normalizeCall(c)
	}

	if want, got := toJSON(t, wantCalls), toJSON(t, calls); want != got {
		t.Errorf("want calls == \n%s\ngot calls ==\n%s", want, got)
	}
}

func toJSON(t *testing.T, v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestTrackAPICall_NoParentCall(t *testing.T) {
	dbSetUp()
	httpSetUp()
	defer dbTearDown()
	defer httpTearDown()

	var called bool
	h := TrackAPICall("my-api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := GetParentCallID(r); ok {
			t.Error("has ParentCallID")
		}
		if _, ok := GetCallID(r); !ok {
			t.Error("no CallID")
		}
	}))

	routeName := "abc"
	rt := mux.NewRouter()
	rt.Path(`/abc/{name:\w+}/{id:[0-9]+}`).Methods("GET").Handler(h).Name(routeName)
	rootMux.Handle("/", rt)

	wantCall := &Call{
		App:         "my-api",
		Host:        hostname,
		URL:         "/abc/alice/123?foo=bar",
		HTTPMethod:  "GET",
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
	httpGet(t, url.String(), 0)

	// Check that call was tracked.
	if !called {
		t.Errorf("!called")
	}
	call := getOnlyOneCall(t)
	// ID and times vary, so don't bother checking them.
	call.ID, call.Start, call.End = 0, time.Time{}, NullTime{}
	normalizeCall(wantCall)
	normalizeCall(call)
	if !reflect.DeepEqual(wantCall, call) {
		t.Errorf("want call == \n%+v\ngot call ==\n%+v", wantCall, call)
	}
}

func TestTrackAPICall_WithParentCallIDHeader(t *testing.T) {
	dbSetUp()
	httpSetUp()
	defer dbTearDown()
	defer httpTearDown()

	wantParentCallID := int64(123)
	var called bool
	h := TrackAPICall("my-api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := GetParentCallID(r); !ok {
			t.Error("no ParentCallID")
		}
		if _, ok := GetCallID(r); !ok {
			t.Error("no CallID")
		}
	}))

	rt := mux.NewRouter()
	rt.Path(`/`).Methods("GET").Handler(h)
	rootMux.Handle("/", rt)

	wantCall := &Call{
		ParentCallID: nnz.Int64(wantParentCallID),
		App:          "my-api",
		Host:         hostname,
		URL:          "/",
		HTTPMethod:   "GET",
		RouteParams:  map[string]interface{}{},
		QueryParams:  map[string]interface{}{},
	}

	httpGet(t, serverURL.String(), wantParentCallID)

	// Check that call was tracked.
	if !called {
		t.Errorf("!called")
	}
	call := getOnlyOneCall(t)
	// ID and times vary, so don't bother checking them.
	call.ID, call.Start, call.End = 0, time.Time{}, NullTime{}
	normalizeCall(wantCall)
	normalizeCall(call)
	if !reflect.DeepEqual(wantCall, call) {
		t.Errorf("want call ==\n%+v\ngot call ==\n%+v", wantCall, call)
	}
}

func makeParentCallIDHeader(id int64) string {
	return fmt.Sprintf("%d", id)
}
