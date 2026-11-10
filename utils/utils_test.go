package utils

import (
	"bytes"
	"testing"
)

func TestIntToHexIsBigEndian(t *testing.T) {
	cases := map[int64][]byte{
		0:     {0, 0, 0, 0, 0, 0, 0, 0},
		1:     {0, 0, 0, 0, 0, 0, 0, 1},
		0x102: {0, 0, 0, 0, 0, 0, 1, 2},
		-1:    {0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	}
	for value, want := range cases {
		if got := IntToHex(value); !bytes.Equal(got, want) {
			t.Errorf("IntToHex(%d) = %x, want %x", value, got, want)
		}
	}
}

func TestReverseBytes(t *testing.T) {
	cases := [][2][]byte{
		{nil, nil},
		{{1}, {1}},
		{{1, 2}, {2, 1}},
		{{1, 2, 3, 4, 5}, {5, 4, 3, 2, 1}},
	}
	for _, c := range cases {
		data := append([]byte(nil), c[0]...)
		ReverseBytes(data)
		if !bytes.Equal(data, c[1]) {
			t.Errorf("ReverseBytes(%v) = %v, want %v", c[0], data, c[1])
		}
	}
}
