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

type obj struct {
	Foo string `json:"foo"`
}

var unmarshalTests = []unmarshalTest{
	{
		in:  []byte{MCPACKV2_STRING, 0, 4, 0, 0, 0, 'f', 'o', 'o', 0},
		ptr: new(string),
		out: "foo",
	},
	{
		in:  []byte{MCPACKV2_INT32, 0, 4, 0, 0, 0},
		ptr: new(int32),
		out: int32(4),
	},
	{
		in: []byte{MCPACKV2_OBJECT, 0, 0, 0, 0, 0, 1, 0, 0, 0,
			MCPACKV2_STRING, 4, 4, 0, 0, 0, 'f', 'o', 'o', 0, 'b', 'a', 'r', 0},
		ptr: new(obj),
		out: obj{Foo: "bar"},
	},
	{
		in: []byte{MCPACKV2_ARRAY, 0, 0, 0, 0, 0, 1, 0, 0, 0,
			MCPACKV2_STRING, 0, 4, 0, 0, 0, 'f', 'o', 'o', 0},
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
			t.Errorf("mismatch %d, got %#+v", i, tt.ptr)
		}
	}
}
