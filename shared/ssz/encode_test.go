package ssz

import (
	"bytes"
	"fmt"
	"testing"
)

type encTest struct {
	val           interface{}
	output, error string
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

func TestEncode(t *testing.T) {
	runEncTests(t, func(val interface{}) ([]byte, error) {
		b := new(bytes.Buffer)
		err := Encode(b, val)
		return b.Bytes(), err
	})
}
