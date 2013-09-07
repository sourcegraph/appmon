package track

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"net/http"
	"testing"
)

func TestClientEntryPoint_PresentInCookie(t *testing.T) {
	dbSetUp()
	httpSetUp()
	defer dbTearDown()
	defer httpTearDown()

	wantClientID := int64(123)
	var called bool
	h := ClientEntryPoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if clientID := GetClientID(r); wantClientID != clientID {
			t.Errorf("want clientID == %d, got %d", wantClientID, clientID)
		}
	}))

	rt := mux.NewRouter()
	rt.Path(`/`).Methods("GET").Handler(h)
	rootMux.Handle("/", rt)

	c, err := makeClientIDCookie(wantClientID)
	if err != nil {
		t.Fatal("makeClientIDCookie", err)
	}
	httpGet(t, serverURL.String(), "Cookie", c.Name+"="+c.Value)

	// Check that h was called.
	if !called {
		t.Errorf("!called")
	}
}

func TestMakeClientIDCookie(t *testing.T) {
	// Set back to original SecureCookie value when we're done.
	origSecureCookie := SecureCookie
	defer func() {
		SecureCookie = origSecureCookie
	}()

	tests := []struct {
		sc *securecookie.SecureCookie
	}{
		{sc: securecookie.New(securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32))},
		{sc: nil},
	}
	for _, test := range tests {
		SecureCookie = test.sc

		clientID := int64(123)

		c, err := makeClientIDCookie(clientID)
		if err != nil {
			t.Fatal("makeClientIDCookie", err)
		}

		clientID2, err := getClientIDFromCookie(c)
		if err != nil {
			t.Fatal("getClientIDFromCookie", err)
		}
		if clientID != clientID2 {
			t.Errorf("want clientID to be encoded and decoded to %v (original value), got %v", clientID, clientID2)
		}
	}
}

func TestNextClientID(t *testing.T) {
	dbSetUp()
	defer dbTearDown()

	clientID, err := nextClientID()
	if err != nil {
		t.Fatal("nextClientID", err)
	}
	if clientID <= 0 {
		t.Errorf("want clientID <= 0, got %d", clientID)
	}
}
