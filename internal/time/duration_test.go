package time

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMarshalJSON(t *testing.T) {
	testStruct := struct {
		D Duration `json:"d"`
	}{D: Duration(time.Second)}

	data, err := json.Marshal(&testStruct)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != `{"d":"1s"}` {
		t.Fatalf("exp 1s got %s", data)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	data := []byte(`{"d":"1s"}`)

	testStruct := struct {
		D Duration `json:"d"`
	}{}

	err := json.Unmarshal(data, &testStruct)
	if err != nil {
		t.Fatal(err)
	}

	if testStruct.D != Duration(time.Second) {
		t.Fatalf("exp %d got %d", Duration(time.Second), testStruct.D)
	}
}

func TestStd(t *testing.T) {
	d := Duration(time.Second)
	if d.Std() != time.Second {
		t.Fatal("exp same type")
	}
}

func TestString(t *testing.T) {
	d := Duration(time.Second)
	if d.String() != "1s" {
		t.Fatal("exp 1s")
	}
}
