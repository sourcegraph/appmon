package track

import (
	"github.com/gorilla/context"
	"github.com/sourcegraph/go-nnz/nnz"
	"log"
	"net/http"
	"strings"
	"time"
)

// InstantiateApp wraps HTTP handlers that return the base HTML page for the
// application named by app (e.g., "web" or "chrome-extension").
func InstantiateApp(app string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the ClientID cookie if it doesn't already exist.
		clientID, err := GetClientID(r)
		if err != nil {
			log.Printf("GetClientID failed: %s", err)
			deleteClientIDCookie(w)
			goto handle
		}
		if clientID == 0 {
			clientID, err = nextClientID()
			if err != nil {
				log.Printf("nextClientID failed: %s", err)
				goto handle
			}

			c, err := newClientIDCookie(clientID)
			if err != nil {
				log.Printf("newClientIDCookie failed: %s", err)
				goto handle
			}

			setClientID(r, clientID)
			http.SetCookie(w, (*http.Cookie)(c))
		}

		if getInstance(r, false) == 0 {
			// Create a new instance.
			instance := &Instance{
				ClientID:    clientID,
				App:         app,
				URL:         r.URL.String(),
				ReferrerURL: r.Referer(),
				ClientInfo:  ClientInfo{IPAddress: removePort(ClientIPAddress(r)), UserAgent: r.UserAgent()},
				Start:       time.Now(),
			}

			// Look up the current user, if any.
			if CurrentUser != nil {
				user, err := CurrentUser(r)
				if err != nil {
					log.Printf("CurrentUser failed: %s", err)
					goto handle
				}
				instance.User = nnz.String(user)
			}

			err = InsertInstance(instance)
			if err != nil {
				log.Printf("InsertInstance failed: %s", err)
				goto handle
			}
			setInstance(r, instance.ID)
		}

	handle:
		h.ServeHTTP(w, r)
	})
}

func GetInstance(r *http.Request) (id int) {
	return getInstance(r, true)
}

func getInstance(r *http.Request, logIfNotPresent bool) (id int) {
	if v, present := context.GetOk(r, instanceID); present {
		id, _ = v.(int)
	} else if logIfNotPresent {
		log.Printf("warn: no instanceID set for request %q (is the app base handler wrapped with InstantiateApp and are clients sending an X-Track-View header?)", r.RequestURI)
	}
	return
}

func setInstance(r *http.Request, id int) {
	context.Set(r, instanceID, id)
}

// ClientIPAddress returns the client IP address that should be used. By
// default, it returns r.RemoteAddr. Library users can override this behavior by
// assigning a new ClientIPAddress func. The port number, if present, is
// stripped from the return value of ClientIPAddress.
var ClientIPAddress = func(r *http.Request) string {
	return r.RemoteAddr
}

// removePort removes the ":port" from "host:port" strings; e.g.,
//   removePort("1.2.3.4:8888") == "1.2.3.4"
func removePort(remoteAddr string) string {
	colon := strings.LastIndex(remoteAddr, ":")
	if colon == -1 {
		return remoteAddr
	}
	return remoteAddr[:colon]
}
