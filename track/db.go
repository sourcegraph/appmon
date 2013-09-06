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
DO $$
BEGIN
  -- Mimics CREATE SCHEMA IF NOT EXISTS behavior.
  IF NOT EXISTS(
      SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name = '` + DBSchema + `'
    )
  THEN
    EXECUTE 'CREATE SCHEMA "` + DBSchema + `"';
  END IF;
END
$$;
CREATE TABLE IF NOT EXISTS "` + DBSchema + `".view (
  id serial NOT NULL,
  "user" varchar(32) NULL,
  client_id bigint NOT NULL,
  state varchar(32) NOT NULL,
  params json NOT NULL,
  date timestamp(3) NOT NULL,
  CONSTRAINT view_pkey PRIMARY KEY (id)
);
CREATE TABLE IF NOT EXISTS "` + DBSchema + `".call (
  id serial NOT NULL,
  view_id bigint NULL,
  request_uri varchar(128) NOT NULL,
  route varchar(32) NOT NULL,
  route_params json NOT NULL,
  query_params json NOT NULL,
  date timestamp(3) NOT NULL,
  CONSTRAINT call_pkey PRIMARY KEY (id)
);
`)
	return
}

// InsertView adds a view to the database.
func InsertView(v *View) (err error) {
	return DB.QueryRow(`
INSERT INTO "`+DBSchema+`".view("user", client_id, state, params, date)
VALUES($1, $2, $3, $4, $5) RETURNING id
`, v.User, v.ClientID, v.State, v.Params, v.Date).Scan(&v.ID)
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
		err = rows.Scan(&v.ID, &v.User, &v.ClientID, &v.State, &v.Params, &v.Date)
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
INSERT INTO "`+DBSchema+`".call(view_id, request_uri, route, route_params, query_params, date)
VALUES($1, $2, $3, $4, $5, $6) RETURNING id
`, c.ViewID, c.RequestURI, c.Route, c.RouteParams, c.QueryParams, c.Date).Scan(&c.ID)
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
		err = rows.Scan(&c.ID, &c.ViewID, &c.RequestURI, &c.Route, &c.RouteParams, &c.QueryParams, &c.Date)
		if err != nil {
			return
		}
		calls = append(calls, c)
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
