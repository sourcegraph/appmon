package appmon

import (
	"database/sql"
	"flag"
	"reflect"
	"sync"
	"testing"
	"time"
)

var dropSchema = flag.Bool("test.dropschema", false, "drop the appmon schema before initializing it and running tests")
var initSchema = flag.Bool("test.initschema", false, "initialize the test appmon schema before running tests")

var dropSchemaOnce sync.Once
var initSchemaOnce sync.Once

func dbSetUp() {
	DBSchema = "test_appmon"
	err := OpenDB()
	if err != nil {
		panic("OpenDB: " + err.Error())
	}

	if *dropSchema {
		dropSchemaOnce.Do(func() {
			err = DropDBSchema()
			if err != nil {
				panic("DropDBSchema: " + err.Error())
			}
		})
	}
	if *initSchema {
		initSchemaOnce.Do(func() {
			err = InitDBSchema()
			if err != nil {
				panic("InitDBSchema: " + err.Error())
			}
		})
	}

	DB, err = dbConn.Begin()
	if err != nil {
		panic("dbConn.Begin: " + err.Error())
	}
}

func dbTearDown() {
	tx, ok := DB.(*sql.Tx)
	if !ok {
		return
	}
	if tx != nil {
		err := tx.Rollback()
		if err != nil {
			panic("DB.Rollback: " + err.Error())
		}
	}
}

func makeCall() *Call {
	return &Call{
		ParentCallID: 123,
		App:          "api",
		Host:         "example.com",
		UID:          123,
		URL:          "http://example.com/foo",
		HTTPMethod:   "GET",
		Route:        "my-route",
		RouteParams:  map[string]interface{}{"k1": "v1"},
		QueryParams:  map[string]interface{}{"k2": "v2"},
		Start:        dbNow(),
		CallStatus: CallStatus{
			End:            now(),
			BodyPrefix:     `{ "foo": "bar" }`,
			BodyLength:     123,
			HTTPStatusCode: 200,
			Err:            "my error",
		},
	}
}

func TestInsertCall(t *testing.T) {
	dbSetUp()
	defer dbTearDown()

	c := makeCall()
	err := insertCall(c)
	if err != nil {
		t.Fatal("insertCall", err)
	}
	if c.ID == 0 {
		t.Error("c.ID == 0")
	}

	c2 := getOnlyOneCall(t)
	normalizeCall(c)
	normalizeCall(c2)
	if !reflect.DeepEqual(c, c2) {
		t.Errorf("QueryCalls: want %+v, got %+v", c, c2)
	}
}

func TestSetCallStatus(t *testing.T) {
	dbSetUp()
	defer dbTearDown()

	c := makeCall()
	err := insertCall(c)
	if err != nil {
		t.Fatal(err)
	}

	s := &CallStatus{End: now(), BodyPrefix: `{"foo":"bar"}`, BodyLength: 456, HTTPStatusCode: 200, Err: "my error"}
	err = setCallStatus(c.ID, s)
	if err != nil {
		t.Fatal("insertCallStatus", err)
	}

	c.CallStatus = *s
	c2 := getOnlyOneCall(t)
	normalizeCall(c)
	normalizeCall(c2)
	if !reflect.DeepEqual(c, c2) {
		t.Errorf("want Call == %+v, got %+v", c, c2)
	}
}

// getOnlyOneCall returns the only Call in the database if there is exactly 1
// Call in the database, and calls t.Fatalf otherwise.
func getOnlyOneCall(t *testing.T) *Call {
	cs, err := QueryCalls("")
	if err != nil {
		t.Fatal("QueryCalls", err)
	}
	if len(cs) != 1 {
		t.Fatalf("want len(cs) == 1, got %d", len(cs))
	}
	return cs[0]
}

// dbNow returns a time.Time of approximately now that is rounded and configured
// so that writing it to the DB and reading it back results in an object equal
// to the original.
func dbNow() time.Time {
	return roundTime(time.Now())
}

func normalizeCall(c *Call) {
	c.Start = roundTime(c.Start)
	if c.End.Valid {
		c.End.Time = roundTime(c.End.Time)
	}
	c.RemoteAddr = ""
	c.UserAgent = ""
}
