package ssz

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"
)

type decodeTest struct {
	input string
	ptr   interface{}
	value interface{}
	error string
}

// Notice: spaces in the input string will be ignored.
var decodeTests = []decodeTest{
	// bool
	{input: "00", ptr: new(bool), value: false},
	{input: "01", ptr: new(bool), value: true},

	// uint8
	{input: "00", ptr: new(uint8), value: uint8(0)},
	{input: "01", ptr: new(uint8), value: uint8(1)},
	{input: "10", ptr: new(uint8), value: uint8(16)},
	{input: "80", ptr: new(uint8), value: uint8(128)},
	{input: "FF", ptr: new(uint8), value: uint8(255)},

	// uint16
	{input: "0000", ptr: new(uint16), value: uint16(0)},
	{input: "0001", ptr: new(uint16), value: uint16(1)},
	{input: "0010", ptr: new(uint16), value: uint16(16)},
	{input: "0080", ptr: new(uint16), value: uint16(128)},
	{input: "00FF", ptr: new(uint16), value: uint16(255)},
	{input: "FFFF", ptr: new(uint16), value: uint16(65535)},

	// uint32
	{input: "00000000", ptr: new(uint32), value: uint32(0)},
	{input: "00000001", ptr: new(uint32), value: uint32(1)},
	{input: "00000010", ptr: new(uint32), value: uint32(16)},
	{input: "00000080", ptr: new(uint32), value: uint32(128)},
	{input: "000000FF", ptr: new(uint32), value: uint32(255)},
	{input: "0000FFFF", ptr: new(uint32), value: uint32(65535)},
	{input: "FFFFFFFF", ptr: new(uint32), value: uint32(4294967295)},

	// uint64
	{input: "0000000000000000", ptr: new(uint64), value: uint64(0)},
	{input: "0000000000000001", ptr: new(uint64), value: uint64(1)},
	{input: "0000000000000010", ptr: new(uint64), value: uint64(16)},
	{input: "0000000000000080", ptr: new(uint64), value: uint64(128)},
	{input: "00000000000000FF", ptr: new(uint64), value: uint64(255)},
	{input: "000000000000FFFF", ptr: new(uint64), value: uint64(65535)},
	{input: "00000000FFFFFFFF", ptr: new(uint64), value: uint64(4294967295)},
	{input: "FFFFFFFFFFFFFFFF", ptr: new(uint64), value: uint64(18446744073709551615)},

	// bytes
	{input: "00000000", ptr: new([]byte), value: []byte{}},
	{input: "0000000101", ptr: new([]byte), value: []byte{1}},
	{input: "00000006 010203040506", ptr: new([]byte), value: []byte{1, 2, 3, 4, 5, 6}},

	// slice
	{input: "00000000", ptr: new([]uint16), value: []uint16(nil)},
	{input: "00000004 0001 0002", ptr: new([]uint16), value: []uint16{1, 2}},
	{input: "00000018 00000008 0001 0002 0003 0004 00000008 0005 0006 0007 0008", ptr: new([][]uint16),
		value: [][]uint16{
			{1, 2, 3, 4},
			{5, 6, 7, 8},
		},
	},

	// array
	{input: "00000001 01", ptr: new([1]byte), value: [1]byte{1}},
	{input: "00000006 010203040506", ptr: new([6]byte), value: [6]byte{1, 2, 3, 4, 5, 6}},
	{input: "00000002 0001", ptr: new([1]uint16), value: [1]uint16{1}},
	{input: "00000004 0001 0002", ptr: new([2]uint16), value: [2]uint16{1, 2}},
	{input: "00000018 00000008 0001 0002 0003 0004 00000008 0005 0006 0007 0008", ptr: new([2][4]uint16),
		value: [2][4]uint16{
			{1, 2, 3, 4},
			{5, 6, 7, 8},
		},
	},

	// struct
	{input: "00000003 00 0000", ptr: new(simpleStruct), value: simpleStruct{}},
	{input: "00000003 0002 01", ptr: new(simpleStruct), value: simpleStruct{B: 2, A: 1}},
	{input: "00000007 03 00000002 0006", ptr: new(outerStruct),
		value: outerStruct{
			V:    3,
			SubV: innerStruct{6},
		}},

	// slice + struct
	{input: "00000012 0000000E 00000003 000201 00000003 000403", ptr: new(arrayStruct),
		value: arrayStruct{
			V: []simpleStruct{
				{B: 2, A: 1},
				{B: 4, A: 3},
			},
		}},
	{input: "00000016 00000007 03 00000002 0006 00000007 05 00000002 0007", ptr: new([]outerStruct),
		value: []outerStruct{
			{V: 3, SubV: innerStruct{V: 6}},
			{V: 5, SubV: innerStruct{V: 7}},
		}},

	// pointer
	{input: "00000003 0002 01", ptr: new(*simpleStruct), value: &simpleStruct{B: 2, A: 1}},
	{input: "00000008 00000003 0002 01 03", ptr: new(pointerStruct),
		value: pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}},
	{input: "00000008 00000003 0002 01 03", ptr: new(*pointerStruct),
		value: &pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}},
	{input: "00000004 01020304", ptr: new(*[]uint8), value: &[]uint8{1, 2, 3, 4}},
	{input: "00000010 0000000000000001 0000000000000002", ptr: new(*[]uint64), value: &[]uint64{1, 2}},
	{input: "0000000E 00000003 0002 01 00000003 0004 03", ptr: new([]*simpleStruct),
		value: []*simpleStruct{
			{B: 2, A: 1},
			{B: 4, A: 3},
		},
	},
	{input: "0000000E 00000003 0002 01 00000003 0004 03", ptr: new([2]*simpleStruct),
		value: [2]*simpleStruct{
			{B: 2, A: 1},
			{B: 4, A: 3},
		},
	},
	{input: "00000018 00000008 00000003 0002 01 00 00000008 00000003 0004 03 01", ptr: new([]*pointerStruct),
		value: []*pointerStruct{
			{P: &simpleStruct{B: 2, A: 1}, V: 0},
			{P: &simpleStruct{B: 4, A: 3}, V: 1},
		},
	},

	// error: nil target
	{input: "00", ptr: nil, value: nil, error: "decode error: cannot decode into nil for output type <nil>"},

	// error: non-pointer target
	{input: "00", ptr: uint8(0), error: "decode error: can only decode into pointer target for output type uint8"},

	// error: unsupported type
	{input: "00", ptr: new(string), error: "decode error: type string is not serializable for output type string"},

	// error: bool: wrong input value
	{input: "02", ptr: new(bool), error: "decode error: expect 0 or 1 for decoding bool but got 2 for output type bool"},

	// error: uint16: wrong header
	{input: "00", ptr: new(uint16), error: "decode error: can only read 1 bytes while expected to read 2 bytes for output type uint16"},

	// error: bytes: wrong input
	{input: "00000001", ptr: new([]byte), error: "decode error: can only read 0 bytes while expected to read 1 bytes for output type []uint8"},

	// error: slice: wrong header
	{input: "000001", ptr: new([]uint16), error: "decode error: failed to decode header of slice: can only read 3 bytes while expected to read 4 bytes for output type []uint16"},

	// error: slice: wrong input
	{input: "00000001", ptr: new([]uint16), error: "decode error: failed to decode element of slice: can only read 0 bytes while expected to read 2 bytes for output type []uint16"},

	// error: byte array: wrong input
	{input: "00000001 01", ptr: new([2]byte), error: "decode error: input byte array size (1) isn't euqal to output array size (2) for output type [2]uint8"},

	// error: array: input too short
	{input: "00000002 0001", ptr: new([2]uint16), error: "decode error: input is too short for output type [2]uint16"},

	// error: array: input too long
	{input: "00000004 0001 0002", ptr: new([1]uint16), error: "decode error: input is too long for output type [1]uint16"},

	// error: struct: wrong header
	{input: "000001", ptr: new(simpleStruct), error: "decode error: failed to decode header of struct: can only read 3 bytes while expected to read 4 bytes for output type ssz.simpleStruct"},

	// error: struct: wrong input
	{input: "00000003 01 02", ptr: new(simpleStruct), error: "decode error: failed to decode field of slice: can only read 0 bytes while expected to read 1 bytes for output type ssz.simpleStruct"},
}

func runTests(t *testing.T, decode func([]byte, interface{}) error) {
	for i, test := range decodeTests {
		input, err := hex.DecodeString(stripSpace(test.input))
		if err != nil {
			t.Errorf("test %d: invalid hex input %q", i, test.input)
			continue
		}
		err = decode(input, test.ptr)
		// Check unexpected error
		if test.error == "" && err != nil {
			t.Errorf("test %d: unexpected decode error: %v\ndecoding into %T\ninput %q",
				i, err, test.ptr, test.input)
			continue
		}
		// Check expected error
		if test.error != "" && fmt.Sprint(err) != test.error {
			t.Errorf("test %d: decode error mismatch\ngot  %v\nwant %v\ndecoding into %T\ninput %q",
				i, err, test.error, test.ptr, test.input)
			continue
		}
		// Check expected output
		if err == nil {
			output := reflect.ValueOf(test.ptr).Elem().Interface()
			if !reflect.DeepEqual(output, test.value) {
				t.Errorf("test %d: value mismatch\ngot  %#v\nwant %#v\ndecoding into %T\ninput %q",
					i, output, test.value, test.ptr, test.input)
			}
		}
	}
}

func TestDecodeWithByteReader(t *testing.T) {
	runTests(t, func(input []byte, into interface{}) error {
		return Decode(bytes.NewReader(input), into)
	})
}
