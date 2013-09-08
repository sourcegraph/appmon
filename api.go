package track

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"time"
)

// ClientConfig contains configuration settings that the client requires to send
// tracking information to the server.
type ClientConfig struct {
	// NewViewURL is the URL that the client POSTs new views to.
	NewViewURL string
}

func MakeClientConfig(rt *mux.Router) (config *ClientConfig, err error) {
	newViewURL, err := rt.GetRoute(trackCreateView).URL("instance", ":instance")
	if err != nil {
		return
	}
	return &ClientConfig{
		NewViewURL: newViewURL.RequestURI(),
	}, nil
}

// ClientData contains client-specific information that the client requires to
// send tracking information to the server.
type ClientData struct {
	// Instance is the value that should be associated with all views and
	// calls originating from this instance.
	Instance int
}

func NewClientData(r *http.Request) (data *ClientData) {
	return &ClientData{Instance: GetInstance(r)}
}

// CurrentUser, if set, is called to determine the currently authenticated user
// for the current request. The returned user ID is stored in the View record if
// nonempty. If err != nil, an HTTP 500 is returned.
var CurrentUser func(r *http.Request) (user string, err error)

const (
	trackCreateView = "track:createView"
)

func APIRouter(rt *mux.Router) *mux.Router {
	rt.Path("/instances/{instance}/views").Methods("POST").HandlerFunc(createView).Name(trackCreateView)
	return rt
}

func createView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instance, err := strconv.Atoi(vars["instance"])
	if err != nil || instance <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var v *View
	err = json.NewDecoder(r.Body).Decode(&v)
	if err != nil {
		log.Printf("Decode: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if v.Instance != instance {
		log.Printf("View.Instance (%d) != instance route parameter (%d)", v.Instance, instance)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	v.Date = time.Now()
	err = InsertView(v)
	if err != nil {
		log.Printf("InsertView: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
