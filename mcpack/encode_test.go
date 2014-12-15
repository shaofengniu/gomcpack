package mcpack

import (
	"bytes"
	"testing"

	. "gitlab.baidu.com/yanyu/gomcpack/mcpack"
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

type W struct {
	S string
	V int16
}

type X struct {
	Beta map[string]string
	Deta [2]int16
}

var longVItem = [299]byte{250: '1', 251: '8', 252: '2', 253: '2', 297: 'S', 298: 'V'}

var marshalTests = []marshalTest{
	{
		in: &T{A: true, X: "x", Y: 1, Z: 2},
		out: []byte{MCPACKV2_OBJECT, 0, 28, 0, 0, 0,
			3, 0, 0, 0,
			MCPACKV2_BOOL, 2, 'A', 0, 1,
			MCPACKV2_SHORT_STRING, 2, 2, 'X', 0, 'x', 0,
			MCPACKV2_INT64, 2, 'Y', 0, 1, 0, 0, 0, 0, 0, 0, 0},
	},
	{
		in: &U{Alphabet: "a-z"},
		out: []byte{MCPACKV2_OBJECT, 0, 17, 0, 0, 0,
			1, 0, 0, 0,
			MCPACKV2_SHORT_STRING, 6, 4, 'a', 'l', 'p', 'h', 'a', 0, 'a', '-', 'z', 0},
	},
	{
		in: &V{F1: &U{Alphabet: "a-z"}, F2: 1, F3: Number(1)},
		out: []byte{MCPACKV2_OBJECT, 0, 52, 0, 0, 0,
			3, 0, 0, 0,
			MCPACKV2_OBJECT, 3, 17, 0, 0, 0, 'F', '1', 0, 1, 0, 0, 0, MCPACKV2_SHORT_STRING, 6, 4, 'a', 'l', 'p', 'h', 'a', 0, 'a', '-', 'z', 0,
			MCPACKV2_INT32, 3, 'F', '2', 0, 1, 0, 0, 0,
			MCPACKV2_INT64, 3, 'F', '3', 0, 1, 0, 0, 0, 0, 0, 0, 0},
	},
	getTestslongVItemW(),
	getTestsKeyTooLongX(),
}

func getTestslongVItemW() marshalTest {
	out := []byte{MCPACKV2_OBJECT, 0, 0x3e, 0x1, 0, 0,
		2, 0, 0, 0,
		MCPACKV2_STRING, 2, 0x2c, 0x1, 0, 0, 'S', 0,
	}
	out = append(out, longVItem[:]...)
	out = append(out, 0, MCPACKV2_INT16, 2, 'V', 0, 1, 0)
	return marshalTest{
		in:  &W{S: string(longVItem[:]), V: 1},
		out: out,
	}
}

func getTestsKeyTooLongX() marshalTest {
	beta := make(map[string]string)
	key := string(longVItem[:])
	beta[key] = "SV"

	deta := [2]int16{-18, -22}
	in := &X{Beta: beta, Deta: deta}

	out := []byte{MCPACKV2_OBJECT, 0, 0x2f, 0x1, 0, 0, //header
		2, 0, 0, 0,
		MCPACKV2_OBJECT, 5, 0x9, 0x1, 0, 0, //map
		'B', 'e', 't', 'a', 0, //key: Beta | 0x0
		1, 0, 0, 0, //count: 1
		MCPACKV2_SHORT_STRING, 0xff, 3}
	out = append(out, longVItem[:254]...)
	out = append(out, 0, 'S', 'V', 0)
	//Deta
	out = append(out, MCPACKV2_ARRAY, 5, 0xc, 0, 0, 0,
		'D', 'e', 't', 'a', 0,
		2, 0, 0, 0,
		MCPACKV2_INT16, 0, 0xee, 0xff, MCPACKV2_INT16, 0, 0xea, 0xff)

	return marshalTest{
		in:  in,
		out: out,
	}
}

func TestMarshal(t *testing.T) {
	for _, tt := range marshalTests {
		b, err := Marshal(tt.in)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(tt.out, b) {
			t.Errorf("mismatch %#+v, got %#+v, expect: %#+v", tt.in, b, tt.out)
		}

	}
}
