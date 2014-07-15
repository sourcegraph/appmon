package main

import (
	"encoding/json"
	"flag"
	"go/build"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sourcegraph/appmon"
	"github.com/sourcegraph/appmon/panel"
	"github.com/sqs/mux"
)

var bind = flag.String("http", ":8888", "HTTP bind address")
var baseURLStr = flag.String("url", "http://localhost:8888", "base URL of this server")
var dir = flag.String("dir", filepath.Join(defaultBase("github.com/sourcegraph/appmon"), "example"), "path to github.com/sourcegraph/appmon/example dir")
var dropSchema = flag.Bool("dropdb", false, "drop the appmon schema before initializing it")
var initSchema = flag.Bool("initdb", false, "initialize the appmon schema before running")

var authUID = flag.Int("uid", 0, "consider all HTTP requests as authenticated as this UID (if non-zero)")

var rt *mux.Router
var baseURL *url.URL

const (
	queryContactsRoute = "query-contacts"
	getContactRoute    = "get-contact"
)

func main() {
	flag.Parse()

	var err error
	baseURL, err = url.Parse(*baseURLStr)
	if err != nil {
		log.Fatal(err)
	}

	err = appmon.OpenDB()
	if err != nil {
		log.Fatalf("appmon.OpenDB: %s", err)
	}

	if *dropSchema {
		err = appmon.DropDBSchema()
		if err != nil {
			log.Fatalf("DropDBSchema: %s", err)
		}
	}
	if *initSchema {
		err = appmon.InitDBSchema()
		if err != nil {
			log.Fatalf("InitDBSchema: %s", err)
		}
	}

	rt = mux.NewRouter()
	t := rt.PathPrefix("/api/appmon").Subrouter()
	panel.Router(t)
	panel.UIRouter("/admin/", rt.PathPrefix("/admin").Subrouter())
	rt.PathPrefix("/api/contacts/{id:[0-9]+}").Methods("GET").Handler(appmon.TrackAPICall("api", http.HandlerFunc(getContact))).Name(getContactRoute)
	rt.PathPrefix("/api/contacts").Methods("GET").Handler(appmon.TrackAPICall("api", http.HandlerFunc(queryContacts))).Name(queryContactsRoute)
	rt.Path("/contacts/{id:[0-9]+}").Handler(appmon.TrackAPICall("example", http.HandlerFunc(showContact)))
	rt.Path("/").Handler(appmon.TrackAPICall("example", http.HandlerFunc(home)))
	http.Handle("/", rt)

	if *authUID != 0 {
		appmon.CurrentUser = func(r *http.Request) int {
			return *authUID
		}
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

func home(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles(assetPath("home.html")))

	url, err := rt.GetRoute(queryContactsRoute).URL()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req, _ := http.NewRequest("GET", baseURL.ResolveReference(url).String(), nil)
	appmon.AddParentCallIDHeader(r, req.Header)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var contacts []*contact
	err = json.NewDecoder(resp.Body).Decode(&contacts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	callID, _ := appmon.GetCallID(r)
	err = tmpl.Execute(w, struct {
		Contacts []*contact
		CallID   int64
	}{
		Contacts: contacts,
		CallID:   callID,
	})
	if err != nil {
		log.Printf("Template execution failed: %s", err)
	}
}

func showContact(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles(assetPath("contact.html")))

	url, err := rt.GetRoute(getContactRoute).URL("id", mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req, _ := http.NewRequest("GET", baseURL.ResolveReference(url).String(), nil)
	appmon.AddParentCallIDHeader(r, req.Header)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var contact_ *contact
	err = json.NewDecoder(resp.Body).Decode(&contact_)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	callID, _ := appmon.GetCallID(r)
	err = tmpl.Execute(w, struct {
		Contact *contact
		CallID  int64
	}{
		Contact: contact_,
		CallID:  callID,
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
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	json.NewEncoder(w).Encode(contacts)
}

func getContact(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Duration(rand.Intn(900)) * time.Millisecond)

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

func defaultBase(path string) string {
	p, err := build.Default.Import(path, "", build.FindOnly)
	if err != nil {
		return "."
	}
	return p.Dir
}
