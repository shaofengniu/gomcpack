package gomcpack

import (
	"reflect"
	"testing"
)

type unmarshalTest struct {
	in  []byte
	ptr interface{}
	out interface{}
}

type object struct {
	Foo string `json:"foo"`
}

var unmarshalTests = []unmarshalTest{
	{
		in:  []byte{MCPACKV2_STRING, 0, 3, 0, 0, 0, 'f', 'o', 'o', 0},
		ptr: new(string),
		out: "foo",
	},
	{
		in: []byte{MCPACKV2_OBJECT, 0, 0, 0, 0, 0, 1, 0, 0, 0,
			MCPACKV2_STRING, 4, 3, 0, 0, 0, 'f', 'o', 'o', 0, 'b', 'a', 'r', 0},
		ptr: new(object),
		out: object{Foo: "bar"},
	},
	{
		in: []byte{MCPACKV2_ARRAY, 0, 0, 0, 0, 0, 1, 0, 0, 0,
			MCPACKV2_STRING, 0, 3, 0, 0, 0, 'f', 'o', 'o', 0},
		ptr: new([]string),
		out: []string{"foo"},
	},
}

func TestUnmarshal(t *testing.T) {
	for i, tt := range unmarshalTests {
		if err := Unmarshal(tt.in, tt.ptr); err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(reflect.ValueOf(tt.ptr).Elem().Interface(), tt.out) {
			t.Errorf("mismatch %d, got %v", i, tt.ptr)
		}
	}
}
