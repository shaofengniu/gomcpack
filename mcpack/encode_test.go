package mcpack

import (
	"bytes"
	"testing"
)

type marshalTest struct {
	in  interface{}
	out []byte
}

type T struct {
	A bool
	X string
	Y int
	Z int `json:"-"`
}

type U struct {
	Alphabet string `json:"alpha"`
}

type V struct {
	F1 interface{}
	F2 int32
	F3 Number
}

type Number int

var marshalTests = []marshalTest{
	{
		in: &T{A: true, X: "x", Y: 1, Z: 2},
		out: []byte{MCPACKV2_OBJECT, 0, 31, 0, 0, 0,
			3, 0, 0, 0,
			MCPACKV2_BOOL, 2, 'A', 0, 1,
			MCPACKV2_STRING, 2, 2, 0, 0, 0, 'X', 0, 'x', 0,
			MCPACKV2_INT64, 2, 'Y', 0, 1, 0, 0, 0, 0, 0, 0, 0},
	},
	{
		in: &U{Alphabet: "a-z"},
		out: []byte{MCPACKV2_OBJECT, 0, 20, 0, 0, 0,
			1, 0, 0, 0,
			MCPACKV2_STRING, 6, 4, 0, 0, 0, 'a', 'l', 'p', 'h', 'a', 0, 'a', '-', 'z', 0},
	},
	{
		in: &V{F1: &U{Alphabet: "a-z"}, F2: 1, F3: Number(1)},
		out: []byte{MCPACKV2_OBJECT, 0, 55, 0, 0, 0,
			3, 0, 0, 0,
			MCPACKV2_OBJECT, 3, 20, 0, 0, 0, 'F', '1', 0, 1, 0, 0, 0, MCPACKV2_STRING, 6, 4, 0, 0, 0, 'a', 'l', 'p', 'h', 'a', 0, 'a', '-', 'z', 0,
			MCPACKV2_INT32, 3, 'F', '2', 0, 1, 0, 0, 0,
			MCPACKV2_INT64, 3, 'F', '3', 0, 1, 0, 0, 0, 0, 0, 0, 0},
	},
}

func TestMarshal(t *testing.T) {
	for _, tt := range marshalTests {
		b, err := Marshal(tt.in)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(tt.out, b) {
			t.Errorf("mismatch %#+v, got %#+v", tt.in, b)
		}

	}
}
