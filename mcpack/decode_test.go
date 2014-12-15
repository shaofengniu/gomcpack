package mcpack_test

import (
	"bytes"
	"reflect"
	"testing"

	. "gitlab.baidu.com/ksarch/gomcpack/mcpack"
)

type unmarshalTest struct {
	in        []byte
	ptr       interface{}
	out       interface{}
	equalFunc func(interface{}, interface{}) bool
}

type obj struct {
	Foo string `json:"foo"`
}

type UV struct {
	F1 *UU    `json:"F1"`
	F2 int32  `json:"F2"`
	F3 Number `json:"F3"`
}

type UU struct {
	Data []byte
}

func (u *UU) UnmarshalMCPACK(b []byte) error {
	u.Data = b
	return nil
}

var unmarshalTests = []unmarshalTest{
	{
		in: []byte{MCPACKV2_OBJECT, 0, 20, 0, 0, 0, 1, 0, 0, 0, MCPACKV2_STRING, 6, 4, 0, 0, 0, 'a', 'l', 'p', 'h', 'a', 0, 'a', '-', 'z', 0},

		ptr: &UU{},
		out: &UU{Data: []byte{MCPACKV2_OBJECT, 0, 20, 0, 0, 0, 1, 0, 0, 0, MCPACKV2_STRING, 6, 4, 0, 0, 0, 'a', 'l', 'p', 'h', 'a', 0, 'a', '-', 'z', 0}},
		equalFunc: func(l, r interface{}) bool {
			var ll, rr *UU = l.(*UU), r.(*UU)
			return bytes.Equal(ll.Data, rr.Data)
		},
	},

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
		if tt.equalFunc != nil {
			if !tt.equalFunc(tt.ptr, tt.out) {
				t.Errorf("mismatch %d, got %#+v, expect %#+v", i, tt.ptr, tt.out)
			}
		} else {
			if !reflect.DeepEqual(reflect.ValueOf(tt.ptr).Elem().Interface(), tt.out) {
				t.Errorf("mismatch %d, got %#+v, expect %#+v", i, tt.ptr, tt.out)
			}
		}
	}
}
