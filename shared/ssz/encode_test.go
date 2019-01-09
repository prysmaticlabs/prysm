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

	// uint32
	{val: uint32(0), output: "00000000"},
	{val: uint32(1), output: "00000001"},
	{val: uint32(16), output: "00000010"},
	{val: uint32(128), output: "00000080"},
	{val: uint32(255), output: "000000FF"},
	{val: uint32(65535), output: "0000FFFF"},
	{val: uint32(4294967295), output: "FFFFFFFF"},

	// uint64
	{val: uint64(0), output: "0000000000000000"},
	{val: uint64(1), output: "0000000000000001"},
	{val: uint64(16), output: "0000000000000010"},
	{val: uint64(128), output: "0000000000000080"},
	{val: uint64(255), output: "00000000000000FF"},
	{val: uint64(65535), output: "000000000000FFFF"},
	{val: uint64(4294967295), output: "00000000FFFFFFFF"},
	{val: uint64(18446744073709551615), output: "FFFFFFFFFFFFFFFF"},

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

	// array
	{val: [1]byte{1}, output: "00000001 01"},
	{val: [6]byte{1, 2, 3, 4, 5, 6}, output: "00000006 010203040506"},
	{val: [1]uint16{1}, output: "00000002 0001"},
	{val: [2]uint16{1, 2}, output: "00000004 0001 0002"},
	{val: [2][4]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, output: "00000018 00000008 0001 0002 0003 0004 00000008 0005 0006 0007 0008"},

	// struct
	{val: simpleStruct{}, output: "00000003 00 0000"},
	{val: simpleStruct{B: 2, A: 1}, output: "00000003 0002 01"},
	{val: outerStruct{
		V:    3,
		SubV: innerStruct{V: 6},
	}, output: "00000007 03 00000002 0006"},

	// slice + struct
	{val: arrayStruct{
		V: []simpleStruct{
			{B: 2, A: 1},
			{B: 4, A: 3},
		},
	}, output: "00000012 0000000E 00000003 000201 00000003 000403"},
	{val: []outerStruct{
		{V: 3, SubV: innerStruct{V: 6}},
		{V: 5, SubV: innerStruct{V: 7}},
	}, output: "00000016 00000007 03 00000002 0006 00000007 05 00000002 0007"},

	// pointer
	{val: &simpleStruct{B: 2, A: 1}, output: "00000003 0002 01"},
	{val: pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "00000008 00000003 0002 01 03"},
	{val: &pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "00000008 00000003 0002 01 03"},
	{val: &[]uint8{1, 2, 3, 4}, output: "00000004 01020304"},
	{val: &[]uint64{1, 2}, output: "00000010 0000000000000001 0000000000000002"},
	{val: []*simpleStruct{
		{B: 2, A: 1},
		{B: 4, A: 3},
	}, output: "0000000E 00000003 0002 01 00000003 0004 03"},
	{val: [2]*simpleStruct{
		{B: 2, A: 1},
		{B: 4, A: 3},
	}, output: "0000000E 00000003 0002 01 00000003 0004 03"},
	{val: []*pointerStruct{
		{P: &simpleStruct{B: 2, A: 1}, V: 0},
		{P: &simpleStruct{B: 4, A: 3}, V: 1},
	}, output: "00000018 00000008 00000003 0002 01 00 00000008 00000003 0004 03 01"},

	// error: nil pointer
	{val: nil, error: "encode error: nil is not supported for input type <nil>"},
	{val: (*[]uint8)(nil), error: "encode error: nil is not supported for input type *[]uint8"},
	{val: pointerStruct{P: nil, V: 0}, error: "encode error: failed to encode field of struct: nil is not supported for input type ssz.pointerStruct"},

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

	// error: nil pointer
	{val: nil, error: "encode error: nil is not supported for input type <nil>"},
	{val: (*[]uint8)(nil), error: "encode error: nil is not supported for input type *[]uint8"},
	{val: pointerStruct{P: nil, V: 0}, error: "encode error: failed to get encode size for field of struct: nil is not supported for input type ssz.pointerStruct"},

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
