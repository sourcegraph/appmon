package track

import (
	"github.com/gorilla/mux"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestInstantiateApp(t *testing.T) {
	dbSetUp()
	httpSetUp()
	defer dbTearDown()
	defer httpTearDown()

	wantInstance := &Instance{
		App:        "myapp",
		URL:        "/abc",
		ClientInfo: ClientInfo{IPAddress: "127.0.0.1"},
	}

	var called bool
	h := InstantiateApp("myapp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		var err error
		wantInstance.ClientID, err = GetClientID(r)
		if err != nil {
			t.Fatal("GetClientID", err)
		}
		if wantInstance.ClientID == 0 {
			t.Error("clientID == 0")
		}
	}))

	routeName := "testroute"
	rt := mux.NewRouter()
	rt.Path(`/abc`).Methods("GET").Handler(h).Name(routeName)
	rootMux.Handle("/", rt)

	url, err := rt.GetRoute(routeName).URL()
	if err != nil {
		t.Fatal("GetRoute", err)
	}
	url = serverURL.ResolveReference(url)
	res := httpGet(t, url.String(), "", "")

	// Check that call was tracked.
	if !called {
		t.Errorf("!called")
	}

	// Check for ClientID cookie.
	cookies := res.Cookies()
	foundCookie := false
	for _, c := range cookies {
		if c.Name == clientIDCookieName {
			clientID, err := (*clientIDCookie)(c).decodeClientID()
			if err != nil {
				t.Fatal("(*clientIDCookie).decodeClientID", err)
			}
			if wantInstance.ClientID == clientID {
				foundCookie = true
			} else {
				t.Errorf("want cookie clientID == %q, got %q", wantInstance.ClientID, clientID)
			}
		}
	}
	if !foundCookie {
		t.Error("no clientID cookie set in response")
	}

	instance := getOnlyOneInstance(t)
	// ID, Date, etc. vary, so don't bother checking them.
	instance.ID = 0
	instance.Start = time.Time{}
	instance.UserAgent = ""
	if !reflect.DeepEqual(wantInstance, instance) {
		t.Errorf("want instance == %+v, got %+v", wantInstance, instance)
	}
}
