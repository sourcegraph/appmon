package track

import (
	"github.com/gorilla/context"
	"github.com/sourcegraph/go-nnz/nnz"
	"log"
	"net/http"
	"strings"
	"time"
)

// InstantiateApp wraps HTTP handlers that return the base HTML page for an
// application.
func InstantiateApp(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the ClientID cookie if it doesn't already exist.
		clientID, err := GetClientID(r)
		if err != nil {
			log.Printf("GetClientID failed: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if clientID == 0 {
			clientID, err = nextClientID()
			if err != nil {
				log.Printf("nextClientID failed: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			c, err := newClientIDCookie(clientID)
			if err != nil {
				log.Printf("newClientIDCookie failed: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			setClientID(r, clientID)
			http.SetCookie(w, (*http.Cookie)(c))
		}

		// Create a new instance.
		instance := &Instance{
			ClientID:    clientID,
			URL:         r.URL.String(),
			ReferrerURL: r.Referer(),
			ClientInfo:  ClientInfo{IPAddress: removePort(r.RemoteAddr), UserAgent: r.UserAgent()},
			Start:       time.Now(),
		}

		// Look up the current user, if any.
		if CurrentUser != nil {
			user, err := CurrentUser(r)
			if err != nil {
				log.Printf("CurrentUser failed: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			instance.User = nnz.String(user)
		}

		err = InsertInstance(instance)
		if err != nil {
			log.Printf("InsertInstance failed: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		setInstance(r, instance.ID)

		h.ServeHTTP(w, r)
	})
}

func GetInstance(r *http.Request) (id int) {
	if v, present := context.GetOk(r, instanceID); present {
		id, _ = v.(int)
	} else {
		log.Printf("warn: no instanceID set for request %q (is the app base handler wrapped with InstantiateApp and are clients sending an X-Track-View header?)", r.RequestURI)
	}
	return
}

func setInstance(r *http.Request, id int) {
	context.Set(r, instanceID, id)
}

// removePort removes the ":port" from "host:port" strings; e.g.,
//   removePort("1.2.3.4:8888") == "1.2.3.4"
func removePort(remoteAddr string) string {
	colon := strings.LastIndex(remoteAddr, ":")
	if colon == -1 {
		colon = len(remoteAddr) - 1
	}
	return remoteAddr[:colon]
}
