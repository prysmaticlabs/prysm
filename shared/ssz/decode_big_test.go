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

// decodeTests includes normal cases and corner cases.
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
}

func runTests(t *testing.T, decode func([]byte, interface{}) error) {
	for i, test := range decodeTests {
		input, err := hex.DecodeString(stripSpace(test.input))
		if err != nil {
			t.Errorf("test %d: invalid hex input %q", i, test.input)
			continue
		}
		// TODO: check these "check" code
		err = decode(input, test.ptr)
		if err != nil && test.error == "" {
			t.Errorf("test %d: unexpected Decode error: %v\ndecoding into %T\ninput %q",
				i, err, test.ptr, test.input)
			continue
		}
		if test.error != "" && fmt.Sprint(err) != test.error {
			t.Errorf("test %d: Decode error mismatch\ngot  %v\nwant %v\ndecoding into %T\ninput %q",
				i, err, test.error, test.ptr, test.input)
			continue
		}
		deref := reflect.ValueOf(test.ptr).Elem().Interface()
		if err == nil && !reflect.DeepEqual(deref, test.value) {
			t.Errorf("test %d: value mismatch\ngot  %#v\nwant %#v\ndecoding into %T\ninput %q",
				i, deref, test.value, test.ptr, test.input)
		}
	}
}

func TestDecodeWithByteReader(t *testing.T) {
	runTests(t, func(input []byte, into interface{}) error {
		return Decode(bytes.NewReader(input), into)
	})
}
