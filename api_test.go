package track

import (
	"github.com/gorilla/mux"
	"reflect"
	"testing"
)

func TestMakeClientConfig(t *testing.T) {
	rt := mux.NewRouter()
	APIRouter(rt)

	want := &ClientConfig{
		NewViewURL: "/instances/:instance/views",
	}

	got, err := MakeClientConfig(rt)
	if err != nil {
		t.Fatal("MakeClientConfig", err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("want ClientConfig == %+v, got %+v", want, got)
	}
}
