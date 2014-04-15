package panel

import (
	"encoding/json"
	"github.com/sqs/mux"
	"github.com/sourcegraph/appmon"
	"log"
	"net/http"
)

const (
	appmonQueryCalls = "appmon:queryCalls"
)

// Router adds panel routes to an existing mux.Router.
func Router(rt *mux.Router) *mux.Router {
	rt.Path("/calls").Methods("GET").HandlerFunc(queryCalls).Name(appmonQueryCalls)
	return rt
}

func queryCalls(w http.ResponseWriter, r *http.Request) {
	calls, err := appmon.QueryCalls("")
	if err != nil {
		log.Printf("QueryCalls: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if calls == nil {
		calls = []*appmon.Call{}
	}
	json.NewEncoder(w).Encode(calls)
}
