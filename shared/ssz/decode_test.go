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

	// bytes
	{input: "00000000", ptr: new([]byte), value: []byte{}},
	{input: "0000000101", ptr: new([]byte), value: []byte{1}},
	{input: "00000006 010203040506", ptr: new([]byte), value: []byte{1, 2, 3, 4, 5, 6}},

	// slice
	{input: "00000000", ptr: new([]uint16), value: []uint16(nil)},
	{input: "00000000", ptr: new([]uint16), value: []uint16(nil)},
	{input: "00000004 0001 0002", ptr: new([]uint16), value: []uint16{1, 2}},
	{input: "00000018 00000008 0001 0002 0003 0004 00000008 0005 0006 0007 0008", ptr: new([][]uint16),
		value: [][]uint16{
			{1, 2, 3, 4},
			{5, 6, 7, 8},
		},
	},

	// struct
	{input: "00000000", ptr: new(simpleStruct), value: simpleStruct{}},
	{input: "00000003 01 0002", ptr: new(simpleStruct), value: simpleStruct{B: 2, A: 1}},
	{input: "00000007 00000002 0006 03", ptr: new(outerStruct),
		value: outerStruct{
			V:    3,
			SubV: innerStruct{6},
		}},

	// error: nil target
	{input: "00", ptr: nil, value: nil, error: "decode error: cannot decode into nil for output type <nil>"},

	// error: non-pointer target
	{input: "00", ptr: uint8(0), error: "decode error: can only decode into pointer target for output type uint8"},

	// error: unsupported type
	{input: "00", ptr: new(string), error: "decode error: type string is not serializable for output type string"},

	// error: bool: wrong input value
	{input: "02", ptr: new(bool), error: "decode error: expect 0 or 1 for decoding bool for output type bool"},

	// error: uint16: wrong header
	{input: "00", ptr: new(uint16), error: "decode error: can only read 1 bytes while expected to read 2 bytes for output type uint16"},

	// error: bytes: wrong input
	{input: "00000001", ptr: new([]byte), error: "decode error: can only read 0 bytes while expected to read 1 bytes for output type []uint8"},

	// error: slice: wrong header
	{input: "000001", ptr: new([]uint16), error: "decode error: failed to decode header of slice: can only read 3 bytes while expected to read 4 bytes for output type []uint16"},

	// error: slice: wrong input
	{input: "00000001", ptr: new([]uint16), error: "decode error: failed to decode element of slice: can only read 0 bytes while expected to read 2 bytes for output type []uint16"},

	// error: struct: wrong header
	{input: "000001", ptr: new(simpleStruct), error: "decode error: failed to decode header of struct: can only read 3 bytes while expected to read 4 bytes for output type ssz.simpleStruct"},

	// error: struct: wrong input
	{input: "00000003 01 02", ptr: new(simpleStruct), error: "decode error: failed to decode field of slice: can only read 1 bytes while expected to read 2 bytes for output type ssz.simpleStruct"},
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
