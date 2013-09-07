package main

import (
	"encoding/json"
	"flag"
	"github.com/gorilla/mux"
	"github.com/sourcegraph/track"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

var bind = flag.String("http", ":8888", "HTTP bind address")
var dir = flag.String("dir", "example", "path to github.com/sourcegraph/track/example dir")
var dropDB = flag.Bool("dropdb", false, "drop the database before initializing it")
var initDB = flag.Bool("initdb", false, "initialize the database before running")

var authUser = flag.String("user", "alice", "consider all HTTP requests as authenticated as this user")

var clientConfig *track.ClientConfig

func main() {
	flag.Parse()

	err := track.OpenDB()
	if err != nil {
		log.Fatalf("track.OpenDB: %s", err)
	}

	if *dropDB {
		err = track.DropDBSchema()
		if err != nil {
			log.Fatalf("DropDB: %s", err)
		}
	}
	if *initDB {
		err = track.InitDB()
		if err != nil {
			log.Fatalf("InitDB: %s", err)
		}
	}

	rt := mux.NewRouter()
	rt.PathPrefix("/static/angular-track").Handler(http.StripPrefix("/static/angular-track", http.FileServer(http.Dir(assetPath("../angular-track")))))
	rt.PathPrefix("/static").Handler(http.StripPrefix("/static", http.FileServer(http.Dir(assetPath("static")))))
	track.APIRouter(rt.PathPrefix("/api/track").Subrouter())
	rt.PathPrefix("/api/contacts/{id:[0-9]+}").Methods("GET").Handler(track.TrackAPICall(http.HandlerFunc(getContact))).Name("getContact")
	rt.PathPrefix("/api/contacts").Methods("GET").Handler(track.TrackAPICall(http.HandlerFunc(queryContacts))).Name("queryContacts")
	rt.Path("/{path:.*}").Handler(track.InstantiateApp(http.HandlerFunc(app)))
	http.Handle("/", rt)

	clientConfig, err = track.MakeClientConfig(rt)
	if err != nil {
		log.Fatalf("track.MakeClientConfig: %s", err)
	}

	track.CurrentUser = func(r *http.Request) (user string, err error) {
		return *authUser, nil
	}

	log.Printf("Listening on %s", *bind)
	err = http.ListenAndServe(*bind, nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %s", err)
	}
}

func assetPath(path string) string {
	p, _ := filepath.Abs(filepath.Join(*dir, path))
	return p
}

func app(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles(assetPath("app.tpl.html")))
	err := tmpl.Execute(w, struct {
		Config *track.ClientConfig
		Data   *track.ClientData
	}{
		Config: clientConfig,
		Data:   track.NewClientData(r),
	})
	if err != nil {
		log.Printf("Template execution failed: %s", err)
	}
}

type contact struct {
	ID   int
	Name string
}

var contacts = []contact{
	{1, "Alice"},
	{2, "Bob"},
	{3, "Charles"},
	{4, "David"},
	{5, "Ellen"},
	{6, "Frank (FAILS)"},
	{7, "Peter (PANICS)"},
}

func queryContacts(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(contacts)
}

func getContact(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	if id <= 0 || id > len(contacts) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	contact := contacts[id-1]
	if strings.Contains(contact.Name, "FAILS") {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if strings.Contains(contact.Name, "PANICS") {
		panic("contact contains PANICS")
	}
	json.NewEncoder(w).Encode(contact)
}
