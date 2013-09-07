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
		NewViewURL: newViewURL.String(),
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
	trackQueryCalls = "track:queryCalls"

	trackGetInstance = "track:getInstance"

	trackCreateView = "track:createView"
	trackQueryViews = "track:queryViews"
)

func APIRouter(rt *mux.Router) *mux.Router {
	instances := rt.PathPrefix("/instances").Subrouter()

	instances.Path("/{instance}").Methods("GET").HandlerFunc(getInstance).Name(trackGetInstance)
	instance := instances.PathPrefix("/{instance}").Subrouter()

	instance.Path("/views").Methods("POST").HandlerFunc(createView).Name(trackCreateView)
	instance.Path("/views").Methods("GET").HandlerFunc(queryViews).Name(trackQueryViews)

	view := instance.PathPrefix("/views/{seq}").Subrouter()

	view.Path("/calls").Methods("GET").HandlerFunc(queryCalls).Name(trackQueryCalls)
	return rt
}

func getInstance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["instance"])
	if err != nil || id <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	instances, err := QueryInstances("WHERE id = $1", id)
	if err != nil {
		log.Printf("QueryInstances failed: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(instances) == 1 {
		json.NewEncoder(w).Encode(instances[0])
	} else {
		http.NotFound(w, r)
	}
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

func queryViews(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instance, err := strconv.Atoi(vars["instance"])
	if err != nil || instance <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	views, err := QueryViews("WHERE instance = $1", instance)
	if err != nil {
		log.Printf("QueryViews: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if views == nil {
		views = []*View{}
	}
	json.NewEncoder(w).Encode(views)
}

func queryCalls(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instance, err := strconv.Atoi(vars["instance"])
	if err != nil || instance <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	seq, err := strconv.Atoi(vars["seq"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	calls, err := QueryCalls("WHERE instance = $1 AND view_seq = $2", instance, seq)
	if err != nil {
		log.Printf("QueryCalls: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if calls == nil {
		calls = []*Call{}
	}
	json.NewEncoder(w).Encode(calls)
}
