package panel

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/sourcegraph/track"
	"log"
	"net/http"
	"strconv"
)

const (
	trackQueryCalls = "track:queryCalls"

	trackGetInstance = "track:getInstance"

	trackQueryViews = "track:queryViews"
)

// Router adds panel routes to an existing mux.Router.
func Router(rt *mux.Router) *mux.Router {
	instances := rt.PathPrefix("/instances").Subrouter()

	instances.Path("/{instance}").Methods("GET").HandlerFunc(getInstance).Name(trackGetInstance)

	instance := instances.PathPrefix("/{instance}").Subrouter()
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

	instances, err := track.QueryInstances("WHERE id = $1", id)
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

func queryViews(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instance, err := strconv.Atoi(vars["instance"])
	if err != nil || instance <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	views, err := track.QueryViews("WHERE instance = $1", instance)
	if err != nil {
		log.Printf("QueryViews: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if views == nil {
		views = []*track.View{}
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

	calls, err := track.QueryCalls("WHERE instance = $1 AND view_seq = $2", instance, seq)
	if err != nil {
		log.Printf("QueryCalls: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if calls == nil {
		calls = []*track.Call{}
	}
	json.NewEncoder(w).Encode(calls)
}
