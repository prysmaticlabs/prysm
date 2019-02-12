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
	{val: uint16(1), output: "0100"},
	{val: uint16(16), output: "1000"},
	{val: uint16(128), output: "8000"},
	{val: uint16(255), output: "FF00"},
	{val: uint16(65535), output: "FFFF"},

	// uint32
	{val: uint32(0), output: "00000000"},
	{val: uint32(1), output: "01000000"},
	{val: uint32(16), output: "10000000"},
	{val: uint32(128), output: "80000000"},
	{val: uint32(255), output: "FF000000"},
	{val: uint32(65535), output: "FFFF0000"},
	{val: uint32(4294967295), output: "FFFFFFFF"},

	// uint64
	{val: uint64(0), output: "0000000000000000"},
	{val: uint64(1), output: "0100000000000000"},
	{val: uint64(16), output: "1000000000000000"},
	{val: uint64(128), output: "8000000000000000"},
	{val: uint64(255), output: "FF00000000000000"},
	{val: uint64(65535), output: "FFFF000000000000"},
	{val: uint64(4294967295), output: "FFFFFFFF00000000"},
	{val: uint64(18446744073709551615), output: "FFFFFFFFFFFFFFFF"},

	// bytes
	{val: []byte{}, output: "00000000"},
	{val: []byte{1}, output: "01000000 01"},
	{val: []byte{1, 2, 3, 4, 5, 6}, output: "06000000 010203040506"},

	// slice
	{val: []uint16{}, output: "00000000"},
	{val: []uint16{1}, output: "02000000 0100"},
	{val: []uint16{1, 2}, output: "04000000 0100 0200"},
	{val: [][]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, output: "18000000 08000000 0100 0200 0300 0400 08000000 0500 0600 0700 0800"},

	// array
	{val: [1]byte{1}, output: "01000000 01"},
	{val: [6]byte{1, 2, 3, 4, 5, 6}, output: "06000000 010203040506"},
	{val: [1]uint16{1}, output: "02000000 0100"},
	{val: [2]uint16{1, 2}, output: "04000000 0100 0200"},
	{val: [2][4]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, output: "18000000 08000000 0100 0200 0300 0400 08000000 0500 0600 0700 0800"},

	// struct
	{val: simpleStruct{}, output: "03000000 00 0000"},
	{val: simpleStruct{B: 2, A: 1}, output: "03000000 0200 01"},
	{val: outerStruct{
		V:    3,
		SubV: innerStruct{V: 6},
	}, output: "07000000 03 02000000 0600"},

	// slice + struct
	{val: arrayStruct{
		V: []simpleStruct{
			{B: 2, A: 1},
			{B: 4, A: 3},
		},
	}, output: "12000000 0E000000 03000000 020001 03000000 040003"},
	{val: []outerStruct{
		{V: 3, SubV: innerStruct{V: 6}},
		{V: 5, SubV: innerStruct{V: 7}},
	}, output: "16000000 07000000 03 02000000 0600 07000000 05 02000000 0700"},

	// pointer
	{val: &simpleStruct{B: 2, A: 1}, output: "03000000 0200 01"},
	{val: pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "08000000 03000000 0200 01 03"},
	{val: &pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "08000000 03000000 0200 01 03"},
	{val: &[]uint8{1, 2, 3, 4}, output: "04000000 01020304"},
	{val: &[]uint64{1, 2}, output: "10000000 0100000000000000 0200000000000000"},
	{val: []*simpleStruct{
		{B: 2, A: 1},
		{B: 4, A: 3},
	}, output: "0E000000 03000000 0200 01 03000000 0400 03"},
	{val: [2]*simpleStruct{
		{B: 2, A: 1},
		{B: 4, A: 3},
	}, output: "0E000000 03000000 0200 01 03000000 0400 03"},
	{val: []*pointerStruct{
		{P: &simpleStruct{B: 2, A: 1}, V: 0},
		{P: &simpleStruct{B: 4, A: 3}, V: 1},
	}, output: "18000000 08000000 03000000 0200 01 00 08000000 03000000 0400 03 01"},

	// nil pointer (not defined in spec)
	{val: (*[]uint8)(nil), output: "00000000"},
	{val: pointerStruct{}, output: "05000000 00000000 00"},
	{val: &pointerStruct{}, output: "05000000 00000000 00"},
	{val: []*pointerStruct{nil, nil}, output: "08000000 00000000 00000000"},

	// error: untyped nil pointer
	{val: nil, error: "encode error: untyped nil is not supported for input type <nil>"},

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

	// uint32
	{val: uint32(0), size: 4},
	{val: uint32(65535), size: 4},

	// uint64
	{val: uint64(0), size: 8},
	{val: uint64(65535), size: 8},

	// bytes
	{val: []byte{}, size: 4},
	{val: []byte{1}, size: 5},
	{val: []byte{1, 2, 3, 4, 5, 6}, size: 10},

	// slice
	{val: []uint16{}, size: 4},
	{val: []uint16{1}, size: 6},
	{val: []uint16{1, 2}, size: 8},
	{val: [][]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, size: 28},

	// array
	{val: [1]byte{1}, size: 5},
	{val: [6]byte{1, 2, 3, 4, 5, 6}, size: 10},
	{val: [1]uint16{1}, size: 6},
	{val: [2]uint16{1, 2}, size: 8},
	{val: [2][4]uint16{
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

	// slice + struct
	{val: arrayStruct{
		V: []simpleStruct{
			{B: 2, A: 1},
			{B: 4, A: 3},
		},
	}, size: 22},
	{val: []outerStruct{
		{V: 3, SubV: innerStruct{V: 6}},
		{V: 5, SubV: innerStruct{V: 7}},
	}, size: 26},

	// pointer
	{val: &simpleStruct{B: 2, A: 1}, size: 7},
	{val: pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, size: 12},
	{val: &pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, size: 12},
	{val: &[]uint8{1, 2, 3, 4}, size: 8},
	{val: &[]uint64{1, 2}, size: 20},
	{val: []*simpleStruct{
		{B: 2, A: 1},
		{B: 4, A: 3},
	}, size: 18},
	{val: []*pointerStruct{
		{P: &simpleStruct{B: 2, A: 1}, V: 0},
		{P: &simpleStruct{B: 4, A: 3}, V: 1},
	}, size: 28},

	// nil pointer (not defined in spec)
	{val: (*[]uint8)(nil), size: 4},
	{val: pointerStruct{}, size: 9},
	{val: &pointerStruct{}, size: 9},
	{val: []*pointerStruct{nil, nil}, size: 12},

	// error: untyped nil pointer
	{val: nil, error: "encode error: untyped nil is not supported for input type <nil>"},

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

// unhex converts a hex string to byte array.
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

type arrayStruct struct {
	V []simpleStruct
}

type pointerStruct struct {
	P *simpleStruct
	V uint8
}
