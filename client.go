package track

import (
	"github.com/gorilla/context"
	"github.com/gorilla/securecookie"
	"net/http"
	"strconv"
	"time"
)

// GetClientID returns the client ID, or 0 if none exists.
func GetClientID(r *http.Request) (id int64, err error) {
	// Try to get it from the context, in case we created it during this request
	// (i.e., it isn't in the cookies from the user).
	if v, present := context.GetOk(r, clientID); present {
		id, _ = v.(int64)
		return
	}

	var c *http.Cookie
	c, err = r.Cookie(clientIDCookieName)
	if err == http.ErrNoCookie {
		err = nil
		return
	}
	return (*clientIDCookie)(c).decodeClientID()
}

func setClientID(r *http.Request, id int64) {
	context.Set(r, clientID, id)
}

// SecureCookie is used to sign and encrypt cookie values, if set.
var SecureCookie *securecookie.SecureCookie

const (
	clientIDCookieName      = "track_clientid"
	clientIDCookieValueBase = 36
)

type clientIDCookie http.Cookie

func newClientIDCookie(clientID int64) (c *clientIDCookie, err error) {
	c = &clientIDCookie{
		Name:    clientIDCookieName,
		Value:   strconv.FormatInt(clientID, clientIDCookieValueBase),
		Path:    "/",
		Expires: time.Now().Add(time.Hour * 24 * 365 * 10),
	}
	if SecureCookie != nil {
		c.Value, err = SecureCookie.Encode(clientIDCookieName, c.Value)
	}
	return
}

func (c *clientIDCookie) decodeClientID() (clientID int64, err error) {
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

func deleteClientIDCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   clientIDCookieName,
		Path:   "/",
		MaxAge: -1,
	})
}

// nextClientID returns the next client ID from the PostgreSQL sequence.
func nextClientID() (clientID int64, err error) {
	row := DB.QueryRow(`SELECT nextval('"` + DBSchema + `".client_id_sequence')`)
	err = row.Scan(&clientID)
	return
}
