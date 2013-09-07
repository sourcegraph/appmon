package track

import (
	"database/sql"
	"encoding/json"
	"flag"
	"reflect"
	"sync"
	"testing"
	"time"
)

var dropDB = flag.Bool("test.dropdb", false, "drop the database before initializing it and running tests")
var initDB = flag.Bool("test.initdb", false, "initialize the test database before running tests")

var dropDBOnce sync.Once
var initDBOnce sync.Once

func dbSetUp() {
	DBSchema = "test_track"
	err := OpenDB()
	if err != nil {
		panic("OpenDB: " + err.Error())
	}

	if *dropDB {
		dropDBOnce.Do(func() {
			err = DropDBSchema()
			if err != nil {
				panic("DropDB: " + err.Error())
			}
		})
	}
	if *initDB {
		initDBOnce.Do(func() {
			err = InitDB()
			if err != nil {
				panic("InitDB: " + err.Error())
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

func TestInsertView(t *testing.T) {
	dbSetUp()
	defer dbTearDown()

	w, err := NextWin()
	if err != nil {
		t.Fatal("NextWin", err)
	}
	v := &View{
		ViewID: ViewID{Win: w, Seq: 123},
		Client: Client{User: NullString{"alice", true}},
		State:  "my.state",
		Params: map[string]interface{}{"k": "v"},
		Date:   dbNow(),
	}

	err = InsertView(v)
	if err != nil {
		t.Fatal("InsertView", err)
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
	dbSetUp()
	defer dbTearDown()

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

func TestInsertCallStatus(t *testing.T) {
	dbSetUp()
	defer dbTearDown()

	s := &CallStatus{CallID: 3, Duration: 123, BodyLength: 456, HTTPStatusCode: 200, Panicked: true}
	err := InsertCallStatus(s)
	if err != nil {
		t.Fatal("InsertCallStatus", err)
	}

	s2 := getOnlyOneCallStatus(t)
	if !reflect.DeepEqual(s, s2) {
		t.Errorf("want CallStatus == %+v, got %+v", s, s2)
	}
}

// getOnlyOneCallStatus returns the only CallStatus in the database if there is exactly 1
// CallStatus in the database, and calls t.Fatalf otherwise.
func getOnlyOneCallStatus(t *testing.T) *CallStatus {
	statuses, err := QueryCallStatuses("")
	if err != nil {
		t.Fatal("QueryCallStatuses", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("want len(statuses) == 1, got %d", len(statuses))
	}
	return statuses[0]
}

// dbNow returns a time.Time of approximately now that is rounded and configured
// so that writing it to the DB and reading it back results in an object equal
// to the original.
func dbNow() time.Time {
	return time.Now().In(time.UTC).Round(time.Millisecond)
}

func TestNullString(t *testing.T) {
	tests := []struct {
		input NullString
		want  []byte
	}{
		{input: NullString{"abc", true}, want: []byte(`"abc"`)},
		{input: NullString{"", true}, want: []byte(`""`)},
		{input: NullString{"", false}, want: []byte(`null`)},
	}

	for _, test := range tests {
		got, err := json.Marshal(test.input)
		if err != nil {
			t.Errorf("%+v: Marshal: %s", test.input, err)
			continue
		}
		if !reflect.DeepEqual(test.want, got) {
			t.Errorf("%+v: want %q, got %q", test.input, test.want, got)
		}

		var input2 NullString
		err = json.Unmarshal(got, &input2)
		if err != nil {
			t.Errorf("%+v: Unmarshal: %s", test.input, err)
			continue
		}
		if !reflect.DeepEqual(test.input, input2) {
			t.Errorf("%+v: want Marshal-then-Unmarshal to return original, got %+v", test.input, input2)
		}
	}
}
