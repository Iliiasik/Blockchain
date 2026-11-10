package core

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestBase58KnownVectors(t *testing.T) {
	cases := []struct {
		raw     []byte
		encoded string
	}{
		{[]byte("hello world"), "StV1DL6CwTryKyV"},
		{[]byte{0x00}, "1"},
		{[]byte{0x00, 0x00, 0x01}, "112"},
		{[]byte{}, ""},
		{[]byte{0xff}, "5Q"},
	}
	for _, c := range cases {
		if got := string(Base58Encode(c.raw)); got != c.encoded {
			t.Errorf("Base58Encode(%x) = %q, want %q", c.raw, got, c.encoded)
		}
		if got := Base58Decode([]byte(c.encoded)); !bytes.Equal(got, c.raw) {
			t.Errorf("Base58Decode(%q) = %x, want %x", c.encoded, got, c.raw)
		}
	}
}

func TestBase58RoundTrip(t *testing.T) {
	for size := 0; size < 64; size++ {
		data := make([]byte, size)
		if _, err := rand.Read(data); err != nil {
			t.Fatal(err)
		}
		if size > 2 {
			data[0], data[1] = 0, 0
		}
		if got := Base58Decode(Base58Encode(data)); !bytes.Equal(got, data) {
			t.Fatalf("round trip of %x gave %x", data, got)
		}
	}
}

func TestBase58RejectsInvalidCharacters(t *testing.T) {
	for _, input := range []string{"0abc", "Ol", "abc!", "I"} {
		if got := Base58Decode([]byte(input)); got != nil {
			t.Errorf("Base58Decode(%q) = %x, want nil", input, got)
		}
	}
}

func TestWriteHelpersUseLittleEndianLengthPrefix(t *testing.T) {
	buf := writeBytes(nil, []byte{0xaa, 0xbb})
	want := []byte{2, 0, 0, 0, 0, 0, 0, 0, 0xaa, 0xbb}
	if !bytes.Equal(buf, want) {
		t.Fatalf("writeBytes = %x, want %x", buf, want)
	}

	reader := bytes.NewReader(append(writeInt(nil, -5), buf...))
	if got := readInt(reader); got != -5 {
		t.Fatalf("readInt = %d, want -5", got)
	}
	if got := readBytes(reader); !bytes.Equal(got, []byte{0xaa, 0xbb}) {
		t.Fatalf("readBytes = %x", got)
	}
}

func TestReadBytesRejectsBrokenLengths(t *testing.T) {
	cases := map[string][]byte{
		"longer than data": append(writeInt(nil, 100), 1, 2, 3),
		"negative":         writeInt(nil, -1),
		"zero":             writeInt(nil, 0),
		"truncated prefix": {1, 2, 3},
		"empty":            {},
	}
	for name, data := range cases {
		if got := readBytes(bytes.NewReader(data)); len(got) != 0 {
			t.Errorf("%s: readBytes = %x, want empty", name, got)
		}
	}
}
