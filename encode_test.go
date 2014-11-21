package gomcpack

import (
	"bytes"
	"testing"
)

type marshalTest struct {
	in  interface{}
	out []byte
}

type M struct {
	Foo string `json:"foo"`
	N   int    `json:"n"`
}

var marshalTests = []marshalTest{
	{
		in:  "foo",
		out: []byte{MCPACKV2_STRING, 0, 4, 0, 0, 0, 'f', 'o', 'o', 0},
	},
	{
		in:  4,
		out: []byte{MCPACKV2_INT64, 0, 4, 0, 0, 0, 0, 0, 0, 0},
	},
	{
		in:  true,
		out: []byte{MCPACKV2_BOOL, 0, 1},
	},
	{
		in: []string{"foo", "bar"},
		out: []byte{MCPACKV2_ARRAY, 0, 0x18, 0, 0, 0, 2, 0, 0, 0,
			MCPACKV2_STRING, 0, 4, 0, 0, 0, 'f', 'o', 'o', 0,
			MCPACKV2_STRING, 0, 4, 0, 0, 0, 'b', 'a', 'r', 0,
		},
	},
	{
		in: &M{Foo: "bar", N: 9},
		out: []byte{MCPACKV2_OBJECT, 0, 0x1e, 0, 0, 0, 2, 0, 0, 0,
			MCPACKV2_STRING, 4, 4, 0, 0, 0, 'f', 'o', 'o', 0, 'b', 'a', 'r', 0,
			MCPACKV2_INT64, 2, 'n', 0, 9, 0, 0, 0, 0, 0, 0, 0},
	},
}

func TestMarshal(t *testing.T) {
	for i, tt := range marshalTests {
		b, err := Marshal(tt.in)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(tt.out, b) {
			t.Errorf("mismatch %d, got %#+v", i, b)
		}

	}
}
