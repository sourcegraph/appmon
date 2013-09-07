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
CREATE SEQUENCE "` + DBSchema + `".win_sequence;
CREATE TABLE "` + DBSchema + `".view (
  win int NOT NULL,
  seq int NOT NULL,
  "user" varchar(32) NULL,
  client_id bigint NOT NULL,
  state varchar(32) NOT NULL,
  params bytea NOT NULL,
  date timestamp(3) NOT NULL,
  CONSTRAINT view_pkey PRIMARY KEY (win, seq)
);
CREATE TABLE "` + DBSchema + `".call (
  id serial NOT NULL,
  view_win int NULL,
  view_seq int NULL,
  request_uri varchar(128) NOT NULL,
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
	_, err = DB.Exec(`DROP SCHEMA IF EXISTS "` + DBSchema + `" CASCADE;`)
	return
}

// NextWin returns the next win ID from the PostgreSQL sequence.
func NextWin() (win int, err error) {
	row := DB.QueryRow(`SELECT nextval('"` + DBSchema + `".win_sequence')`)
	err = row.Scan(&win)
	return
}

// InsertView adds a view to the database.
func InsertView(v *View) (err error) {
	_, err = DB.Exec(`
INSERT INTO "`+DBSchema+`".view(win, seq, "user", client_id, state, params, date)
VALUES($1, $2, $3, $4, $5, $6, $7)
`, v.Win, v.Seq, v.User, v.ClientID, v.State, v.Params, v.Date)
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
		err = rows.Scan(&v.Win, &v.Seq, &v.User, &v.ClientID, &v.State, &v.Params, &v.Date)
		if err != nil {
			return
		}
		views = append(views, v)
	}
	return
}

// InsertCall adds a Call to the database.
func InsertCall(c *Call) (err error) {
	var win, seq *int
	if c.View != nil {
		win, seq = &c.View.Win, &c.View.Seq
	}
	return DB.QueryRow(`
INSERT INTO "`+DBSchema+`".call(view_win, view_seq, request_uri, route, route_params, query_params, date)
VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id
`, win, seq, c.RequestURI, c.Route, c.RouteParams, c.QueryParams, c.Date).Scan(&c.ID)
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
		var win, seq NullInt
		err = rows.Scan(
			&c.ID, &win, &seq, &c.RequestURI, &c.Route, &c.RouteParams, &c.QueryParams, &c.Date,
		)
		if err != nil {
			return
		}
		if win.Valid && seq.Valid {
			c.View = &ViewID{Win: win.Int, Seq: seq.Int}
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

// NullInt represents an int that may be null.
type NullInt struct {
	Int   int
	Valid bool // Valid is true if Int is not NULL
}

// Value implements the driver database/sql/driver.Valuer interface.
func (x NullInt) Value() (driver.Value, error) {
	if !x.Valid {
		return nil, nil
	}
	return x.Int, nil
}

// Scan implements the database/sql/driver.Scanner interface.
func (x *NullInt) Scan(v interface{}) error {
	if v == nil {
		x.Int, x.Valid = 0, false
		return nil
	}
	i, ok := v.(int64)
	x.Int = int(i)
	x.Valid = true
	if !ok {
		return fmt.Errorf("%T.Scan failed: %v", x, v)
	}
	return nil
}

// NullString is sql.NullString with an implementation of the
// encoding/json.Marshaler and encoding/json.Unmarshaler interfaces.
type NullString sql.NullString

// Scan implements the database/sql/driver.Scanner interface.
func (ns *NullString) Scan(v interface{}) error {
	if v == nil {
		ns.String, ns.Valid = "", false
		return nil
	}
	b, ok := v.([]byte)
	if !ok {
		return fmt.Errorf("%T.Scan failed: %v", ns, v)
	}
	ns.String = string(b)
	ns.Valid = true
	return nil
}

// Value implements the database/sql/driver.Valuer interface.
func (ns NullString) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.String, nil
}

// MarshalJSON implements the encoding/json.Marshaler interface.
func (s NullString) MarshalJSON() ([]byte, error) {
	if s.Valid {
		return json.Marshal(s.String)
	} else {
		return json.Marshal(nil)
	}
}

// UnmarshalJSON implements the encoding/json.Unmarshaler interface.
func (s *NullString) UnmarshalJSON(data []byte) error {
	var v interface{}
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	if v, ok := v.(string); ok {
		s.String = v
		s.Valid = true
	} else {
		s.String = ""
		s.Valid = false
	}
	return nil
}
