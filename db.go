package track

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
)

// DBH is a database handle that is agnostic to whether it is a database
// connection or transaction.
type DBH interface {
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
}

// dbConn is the global database connection.
var dbConn *sql.DB

// DB is the global database handle used by all functions in this package that
// interact with the database.
var DB DBH

// DBSchema is the name of the PostgreSQL database schema used for all SQL
// operations. It is double-quoted in SQL statements sent to the database but
// not escaped.
var DBSchema = "track"

// OpenDB connects to the database using connection parameters from the PG*
// environment variables. The database connection is stored in the global
// variable DB.
func OpenDB() (err error) {
	dbConn, err = sql.Open("postgres", "")
	DB = dbConn
	return
}

// InitDB creates the database schema and tables.
func InitDB() (err error) {
	_, err = DB.Exec(`
CREATE SCHEMA "` + DBSchema + `";
CREATE SEQUENCE "` + DBSchema + `".client_id_sequence;
CREATE SEQUENCE "` + DBSchema + `".instance_sequence;
CREATE TABLE "` + DBSchema + `".instance (
  id serial NOT NULL,
  "user" varchar(32) NULL,
  client_id bigint NOT NULL,
  app varchar(32) NOT NULL,
  url varchar(255) NOT NULL,
  referrer_url varchar(255) NOT NULL,
  ip_addr varchar(15) NOT NULL,
  user_agent varchar(255) NOT NULL,
  start timestamp(3) NOT NULL,
  CONSTRAINT instance_pkey PRIMARY KEY (id)
);
CREATE TABLE "` + DBSchema + `".view (
  instance int NOT NULL,
  seq int NOT NULL,
  request_uri varchar(255) NOT NULL,
  state varchar(32) NOT NULL,
  state_params bytea NOT NULL,
  date timestamp(3) NOT NULL,
  CONSTRAINT view_pkey PRIMARY KEY (instance, seq)
);
CREATE TABLE "` + DBSchema + `".call (
  id serial NOT NULL,
  instance int NOT NULL,
  view_seq int NULL,
  url varchar(255) NOT NULL,
  route varchar(32) NOT NULL,
  route_params bytea NOT NULL,
  query_params bytea NOT NULL,
  date timestamp(3) NOT NULL,
  CONSTRAINT call_pkey PRIMARY KEY (id)
);
CREATE TABLE "` + DBSchema + `".call_status (
  call_id bigint NOT NULL,
  duration bigint NOT NULL,
  body_length int NOT NULL,
  http_status_code int NOT NULL,
  panicked boolean NOT NULL,
  CONSTRAINT call_status_pkey PRIMARY KEY (call_id)
);
`)
	return
}

// DropDBSchema drops the database schema and tables.
func DropDBSchema() (err error) {
	_, err = DB.Exec(`DROP SCHEMA IF EXISTS "` + DBSchema + `" CASCADE`)
	return
}

// InsertInstance adds an instance to the database.
func InsertInstance(o *Instance) (err error) {
	return DB.QueryRow(`
INSERT INTO "`+DBSchema+`".instance("user", client_id, app, url, referrer_url, ip_addr, user_agent, start)
VALUES($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id
`, o.User, o.ClientID, o.App, o.URL, o.ReferrerURL, o.IPAddress, o.UserAgent, o.Start).Scan(&o.ID)
}

// QueryInstances returns all instances matching the SQL query conditions.
func QueryInstances(query string, args ...interface{}) (instances []*Instance, err error) {
	var rows *sql.Rows
	rows, err = DB.Query(`SELECT instance.* FROM "`+DBSchema+`".instance `+query, args...)
	if err != nil {
		return
	}
	for rows.Next() {
		o := new(Instance)
		err = rows.Scan(&o.ID, &o.User, &o.ClientID, &o.App, &o.URL, &o.ReferrerURL, &o.IPAddress, &o.UserAgent, &o.Start)
		if err != nil {
			return
		}
		instances = append(instances, o)
	}
	return
}

// InsertView adds a view to the database.
func InsertView(v *View) (err error) {
	_, err = DB.Exec(`
INSERT INTO "`+DBSchema+`".view(instance, seq, request_uri, state, state_params, date)
VALUES($1, $2, $3, $4, $5, $6)
`, v.Instance, v.Seq, v.RequestURI, v.State, v.StateParams, v.Date)
	return
}

// QueryViews returns all views matching the SQL query conditions.
func QueryViews(query string, args ...interface{}) (views []*View, err error) {
	var rows *sql.Rows
	rows, err = DB.Query(`SELECT view.* FROM "`+DBSchema+`".view `+query, args...)
	if err != nil {
		return
	}
	for rows.Next() {
		v := new(View)
		err = rows.Scan(&v.Instance, &v.Seq, &v.RequestURI, &v.State, &v.StateParams, &v.Date)
		if err != nil {
			return
		}
		views = append(views, v)
	}
	return
}

// InsertCall adds a Call to the database.
func InsertCall(c *Call) (err error) {
	return DB.QueryRow(`
INSERT INTO "`+DBSchema+`".call(instance, view_seq, url, route, route_params, query_params, date)
VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id
`, c.Instance, c.ViewSeq, c.URL, c.Route, c.RouteParams, c.QueryParams, c.Date).Scan(&c.ID)
}

// QueryCalls returns all calls matching the SQL query conditions.
func QueryCalls(query string, args ...interface{}) (calls []*Call, err error) {
	var rows *sql.Rows
	rows, err = DB.Query(`SELECT call.* FROM "`+DBSchema+`".call `+query, args...)
	if err != nil {
		return
	}
	for rows.Next() {
		c := new(Call)
		err = rows.Scan(
			&c.ID, &c.Instance, &c.ViewSeq, &c.URL, &c.Route,
			&c.RouteParams, &c.QueryParams, &c.Date,
		)
		if err != nil {
			return
		}
		calls = append(calls, c)
	}
	return
}

func InsertCallStatus(s *CallStatus) (err error) {
	_, err = DB.Exec(`
INSERT INTO "`+DBSchema+`".call_status(call_id, duration, body_length, http_status_code, panicked)
VALUES($1, $2, $3, $4, $5)
`, s.CallID, s.Duration, s.BodyLength, s.HTTPStatusCode, s.Panicked)
	return
}

// QueryCallStatuses returns all call statuses matching the SQL query conditions.
func QueryCallStatuses(query string, args ...interface{}) (statuses []*CallStatus, err error) {
	var rows *sql.Rows
	rows, err = DB.Query(`SELECT call_status.* FROM "`+DBSchema+`".call_status `+query, args...)
	if err != nil {
		return
	}
	for rows.Next() {
		s := new(CallStatus)
		err = rows.Scan(&s.CallID, &s.Duration, &s.BodyLength, &s.HTTPStatusCode, &s.Panicked)
		if err != nil {
			return
		}
		statuses = append(statuses, s)
	}
	return
}

// Value implements the database/sql/driver.Valuer interface.
func (x Params) Value() (driver.Value, error) {
	if x == nil {
		return nil, nil
	}
	return json.Marshal(x)
}

// Scan implements the database/sql/driver.Scanner interface.
func (x *Params) Scan(v interface{}) error {
	if data, ok := v.([]byte); ok {
		return json.Unmarshal(data, x)
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}
