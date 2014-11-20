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
}

var marshalTests = []marshalTest{
	{
		in:  "foo",
		out: []byte{MCPACKV2_STRING, 0, 4, 0, 0, 0, 'f', 'o', 'o', 0},
	},
	{
		in: &M{Foo: "bar"},
		out: []byte{MCPACKV2_OBJECT, 0, 0x12, 0, 0, 0, 1, 0, 0, 0,
			MCPACKV2_STRING, 4, 4, 0, 0, 0, 'f', 'o', 'o', 0, 'b', 'a', 'r', 0},
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
