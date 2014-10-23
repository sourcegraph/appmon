package appmon

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
var DBSchema = "appmon"

// OpenDB connects to the database using connection parameters from the PG*
// environment variables. The database connection is stored in the global
// variable DB.
func OpenDB() (err error) {
	dbConn, err = sql.Open("postgres", "")
	DB = dbConn
	return
}

// InitDBSchema creates the database schema and tables.
func InitDBSchema() (err error) {
	_, err = DB.Exec(`
CREATE SCHEMA "` + DBSchema + `";
CREATE UNLOGGED TABLE "` + DBSchema + `".call (
  id bigserial NOT NULL,
  parent_call_id bigint,

  app text NOT NULL,
  host text NOT NULL,

  remote_addr text NOT NULL,
  user_agent text NOT NULL,
  uid int NULL,

  url text NOT NULL,
  http_method text NOT NULL,
  route text NULL,
  route_params text NOT NULL,
  query_params text NOT NULL,

  start timestamp(3) NOT NULL,

  -- call status fields (filled in post-request)
  "end" timestamp(3),
  body_length int,
  http_status_code int,
  err text,

  CONSTRAINT call_pkey PRIMARY KEY (id)
);
CREATE INDEX call_parent_call_id ON "` + DBSchema + `".call(parent_call_id);
`)
	return
}

// DropDBSchema drops the database schema and tables.
func DropDBSchema() (err error) {
	_, err = DB.Exec(`DROP SCHEMA IF EXISTS "` + DBSchema + `" CASCADE`)
	return
}

// InsertCall adds a Call to the database and writes its serial ID to c.ID.
func InsertCall(c *Call) (err error) {
	return DB.QueryRow(`
INSERT INTO "`+DBSchema+`".call(parent_call_id, app, host, remote_addr, user_agent, uid, url, http_method, route, route_params, query_params, "start", "end", body_length, http_status_code, err)
VALUES($1, $2, $3, left($4, 24), left($5, 500), $6, left($7, 1000), left($8, 12), left($9, 64), left($10, 1000), left($11, 1000), $12, $13, $14, $15, left($16, 1000)) RETURNING id
`, c.ParentCallID, c.App, c.Host, c.RemoteAddr, c.UserAgent, c.UID, c.URL, c.HTTPMethod, c.Route, c.RouteParams, c.QueryParams, c.Start, c.End, c.BodyLength, c.HTTPStatusCode, c.Err).Scan(&c.ID)
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
			&c.ID, &c.ParentCallID, &c.App, &c.Host, &c.RemoteAddr, &c.UserAgent, &c.UID, &c.URL, &c.HTTPMethod,
			&c.Route, &c.RouteParams, &c.QueryParams, &c.Start, &c.End, &c.BodyLength, &c.HTTPStatusCode, &c.Err,
		)
		if err != nil {
			return
		}
		calls = append(calls, c)
	}
	return
}

func setCallStatus(callID int64, s *CallStatus) (err error) {
	_, err = DB.Exec(`
UPDATE "`+DBSchema+`".call SET "end" = $1, body_length = $2, http_status_code = $3, err = $4
WHERE id = $5
`, s.End, s.BodyLength, s.HTTPStatusCode, s.Err, callID)
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

type NullTime struct {
	Time  time.Time
	Valid bool // Valid is true if Time is not NULL
}

// Scan implements the Scanner interface.
func (nt *NullTime) Scan(value interface{}) error {
	nt.Time, nt.Valid = value.(time.Time)
	return nil
}

// Value implements the driver Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}

// MarshalJSON implements the json.Marshaler interface.
func (nt NullTime) MarshalJSON() ([]byte, error) {
	if nt.Valid {
		return json.Marshal(nt.Time)
	}
	return []byte("null"), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (nt *NullTime) UnmarshalJSON(data []byte) (err error) {
	if nt == nil {
		return errors.New("UnmarshalJSON on nil *NullTime pointer")
	}
	if bytes.Compare(data, []byte("null")) == 0 {
		nt.Valid = false
	} else {
		nt.Valid = true
		err = json.Unmarshal(data, &nt.Time)
	}
	return
}

func now() NullTime {
	return NullTime{Time: time.Now().In(time.UTC), Valid: true}
}

func roundTime(t time.Time) time.Time {
	return t.In(time.UTC).Round(time.Millisecond)
}
