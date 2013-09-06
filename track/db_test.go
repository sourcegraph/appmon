package track

import (
	"database/sql"
	"reflect"
	"sync"
	"testing"
	"time"
)

var initDBOnce sync.Once

func setUp() {
	DBSchema = "test_track"
	err := OpenDB()
	if err != nil {
		panic("OpenDB: " + err.Error())
	}

	initDBOnce.Do(func() {
		err = InitDB()
		if err != nil {
			panic("InitDB: " + err.Error())
		}
	})

	DB, err = dbConn.Begin()
	if err != nil {
		panic("dbConn.Begin: " + err.Error())
	}
}

func tearDown() {
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

func TestInsertView(t *testing.T) {
	setUp()
	defer tearDown()

	v := &View{
		Client: Client{User: sql.NullString{"alice", true}},
		State:  "my.state",
		Params: map[string]interface{}{"k": "v"},
		Date:   dbNow(),
	}

	err := InsertView(v)
	if err != nil {
		t.Fatal("InsertView", err)
	}
	if v.ID == 0 {
		t.Error("v.ID == 0")
	}

	var vs []*View
	vs, err = QueryViews("")
	if err != nil {
		t.Fatal("QueryViews", err)
	}
	if want := []*View{v}; !reflect.DeepEqual(want, vs) {
		t.Errorf("QueryViews: want %v, got %v", want, vs)
	}
}

func TestInsertCall(t *testing.T) {
	setUp()
	defer tearDown()

	c := &Call{
		RequestURI:  "/abc",
		Route:       "my-route",
		RouteParams: map[string]interface{}{"k1": "v1"},
		QueryParams: map[string]interface{}{"k2": "v2"},
		Date:        dbNow(),
	}

	err := InsertCall(c)
	if err != nil {
		t.Fatal("InsertCall", err)
	}
	if c.ID == 0 {
		t.Error("c.ID == 0")
	}

	var cs []*Call
	cs, err = QueryCalls("")
	if err != nil {
		t.Fatal("QueryCalls", err)
	}
	if want := []*Call{c}; !reflect.DeepEqual(want, cs) {
		t.Errorf("QueryCalls: want %+v, got %+v", want[0], cs[0])
	}
}

// dbNow returns a time.Time of approximately now that is rounded and configured
// so that writing it to the DB and reading it back results in an object equal
// to the original.
func dbNow() time.Time {
	return time.Now().In(time.UTC).Round(time.Millisecond)
}
