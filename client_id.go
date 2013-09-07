package track

import (
	"github.com/gorilla/securecookie"
	"log"
	"net/http"
	"strconv"
	"time"
)

var (
	// SecureCookie is used to sign and encrypt cookie values.
	SecureCookie *securecookie.SecureCookie
)

const clientIDCookieName = "track_clientid"

// ClientEntryPoint stores the client ID value from the client's cookies in the
// request context. If there is no such cookie, it creates a new client ID value
// and sets it in a cookie. It should wrap handlers that return static pages.
func ClientEntryPoint(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(clientIDCookieName)
		if err == http.ErrNoCookie {
			clientID, err := nextClientID()
			if err != nil {
				log.Printf("nextClientID failed: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			c, err = makeClientIDCookie(clientID)
			if err != nil {
				log.Printf("makeClientIDCookie failed: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, c)
		}

		clientID, err := getClientIDFromCookie(c)
		if err != nil {
			log.Printf("getClientIDFromCookie failed: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		setClientID(r, clientID)

		h.ServeHTTP(w, r)
	})
}

// storeClientID stores the client ID value from the cookies in the request
// context. Unlike ClientEntryPoint, it does not generate a new client ID if
// none is included in the request cookies. It should wrap handlers that do not
// return static pages (e.g., TrackCall).
func storeClientID(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(clientIDCookieName)
		if err == nil {
			clientID, err := getClientIDFromCookie(c)
			if err != nil {
				log.Printf("getClientIDFromCookie failed: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			setClientID(r, clientID)
		}

		h.ServeHTTP(w, r)
	})
}

func makeClientIDCookie(clientID int64) (c *http.Cookie, err error) {
	c = &http.Cookie{
		Name:    clientIDCookieName,
		Value:   strconv.FormatInt(clientID, 36),
		Path:    "/",
		Expires: time.Now().Add(time.Hour * 24 * 365 * 10),
	}
	if SecureCookie != nil {
		c.Value, err = SecureCookie.Encode(clientIDCookieName, c.Value)
	}
	return
}

func getClientIDFromCookie(c *http.Cookie) (clientID int64, err error) {
	var clientIDStr string
	if SecureCookie == nil {
		clientIDStr = c.Value
	} else {
		err = SecureCookie.Decode(clientIDCookieName, c.Value, &clientIDStr)
		if err != nil {
			return
		}
	}
	return strconv.ParseInt(clientIDStr, 36, 64)
}

// nextClientID returns the next client ID from the PostgreSQL sequence.
func nextClientID() (clientID int64, err error) {
	row := DB.QueryRow(`SELECT nextval('"` + DBSchema + `".client_id_sequence')`)
	err = row.Scan(&clientID)
	return
}
