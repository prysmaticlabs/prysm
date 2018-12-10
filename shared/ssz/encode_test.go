package ssz

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

type encTest struct {
	val           interface{}
	output, error string
}

type encSizeTest struct {
	val   interface{}
	size  uint32
	error string
}

// Notice: spaces in the output string will be ignored.
var encodeTests = []encTest{
	// boolean
	{val: false, output: "00"},
	{val: true, output: "01"},

	// uint8
	{val: uint8(0), output: "00"},
	{val: uint8(1), output: "01"},
	{val: uint8(16), output: "10"},
	{val: uint8(128), output: "80"},
	{val: uint8(255), output: "FF"},

	// uint16
	{val: uint16(0), output: "0000"},
	{val: uint16(1), output: "0001"},
	{val: uint16(16), output: "0010"},
	{val: uint16(128), output: "0080"},
	{val: uint16(255), output: "00FF"},
	{val: uint16(65535), output: "FFFF"},

	// bytes
	{val: []byte{}, output: "00000000"},
	{val: []byte{1}, output: "00000001 01"},
	{val: []byte{1, 2, 3, 4, 5, 6}, output: "00000006 010203040506"},

	// slice
	{val: []uint16{}, output: "00000000"},
	{val: []uint16{1}, output: "00000002 0001"},
	{val: []uint16{1, 2}, output: "00000004 0001 0002"},
	{val: [][]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, output: "00000018 00000008 0001 0002 0003 0004 00000008 0005 0006 0007 0008"},

	// struct
	{val: simpleStruct{}, output: "00000003 00 0000"},
	{val: simpleStruct{B: 2, A: 1}, output: "00000003 01 0002"},
	{val: outerStruct{
		V:    3,
		SubV: innerStruct{V: 6},
	}, output: "00000007 00000002 0006 03"},

	// error: unsupported type
	{val: string("abc"), error: "encode error: type string is not serializable for input type string"},
}

var encodeSizeTests = []encSizeTest{
	// boolean
	{val: false, size: 1},

	// uint8
	{val: uint8(0), size: 1},
	{val: uint8(255), size: 1},

	// uint16
	{val: uint16(0), size: 2},
	{val: uint16(65535), size: 2},

	// bytes
	{val: []byte{}, size: 0},
	{val: []byte{1}, size: 1},
	{val: []byte{1, 2, 3, 4, 5, 6}, size: 6},

	// slice
	{val: []uint16{}, size: 4},
	{val: []uint16{1}, size: 6},
	{val: []uint16{1, 2}, size: 8},
	{val: [][]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, size: 28},

	// struct
	{val: simpleStruct{}, size: 7},
	{val: simpleStruct{B: 2, A: 1}, size: 7},
	{val: outerStruct{
		V:    3,
		SubV: innerStruct{V: 6},
	}, size: 11},

	// error: unsupported type
	{val: string("abc"), error: "encode error: type string is not serializable for input type string"},
}

func runEncTests(t *testing.T, encode func(val interface{}) ([]byte, error)) {
	for i, test := range encodeTests {
		output, err := encode(test.val)
		// Check unexpected error
		if test.error == "" && err != nil {
			t.Errorf("test %d: unexpected error: %v\nvalue %#v\ntype %T",
				i, err, test.val, test.val)
			continue
		}
		// Check expected error
		if test.error != "" && fmt.Sprint(err) != test.error {
			t.Errorf("test %d: error mismatch\ngot   %v\nwant  %v\nvalue %#v\ntype  %T",
				i, err, test.error, test.val, test.val)
			continue
		}
		// Check expected output
		if err == nil && !bytes.Equal(output, unhex(test.output)) {
			t.Errorf("test %d: output mismatch:\ngot   %X\nwant  %s\nvalue %#v\ntype  %T",
				i, output, stripSpace(test.output), test.val, test.val)
		}
	}
}

func runEncSizeTests(t *testing.T, encodeSize func(val interface{}) (uint32, error)) {
	for i, test := range encodeSizeTests {
		size, err := encodeSize(test.val)
		// Check unexpected error
		if test.error == "" && err != nil {
			t.Errorf("test %d: unexpected error: %v\nvalue %#v\ntype %T",
				i, err, test.val, test.val)
			continue
		}
		// Check expected error
		if test.error != "" && fmt.Sprint(err) != test.error {
			t.Errorf("test %d: error mismatch\ngot   %v\nwant  %v\nvalue %#v\ntype  %T",
				i, err, test.error, test.val, test.val)
			continue
		}
		// Check expected output
		if err == nil && size != test.size {
			t.Errorf("test %d: output mismatch:\ngot   %d\nwant  %d\nvalue %#v\ntype  %T",
				i, size, test.size, test.val, test.val)
		}
	}
}

func TestEncode(t *testing.T) {
	runEncTests(t, func(val interface{}) ([]byte, error) {
		b := new(bytes.Buffer)
		err := Encode(b, val)
		return b.Bytes(), err
	})
}

func TestEncodeSize(t *testing.T) {
	runEncSizeTests(t, func(val interface{}) (uint32, error) {
		size, err := EncodeSize(val)
		return size, err
	})
}

// unhex converts a hex string to byte array
func unhex(str string) []byte {
	b, err := hex.DecodeString(stripSpace(str))
	if err != nil {
		panic(fmt.Sprintf("invalid hex string: %q", str))
	}
	return b
}

func stripSpace(str string) string {
	return strings.Replace(str, " ", "", -1)
}

type simpleStruct struct {
	B uint16
	A uint8
}

type innerStruct struct {
	V uint16
}

type outerStruct struct {
	V    uint8
	SubV innerStruct
}
