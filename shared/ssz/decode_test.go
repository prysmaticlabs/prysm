package ssz

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
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
	{input: "0100", ptr: new(uint16), value: uint16(1)},
	{input: "1000", ptr: new(uint16), value: uint16(16)},
	{input: "8000", ptr: new(uint16), value: uint16(128)},
	{input: "FF00", ptr: new(uint16), value: uint16(255)},
	{input: "FFFF", ptr: new(uint16), value: uint16(65535)},

	// uint32
	{input: "00000000", ptr: new(uint32), value: uint32(0)},
	{input: "01000000", ptr: new(uint32), value: uint32(1)},
	{input: "10000000", ptr: new(uint32), value: uint32(16)},
	{input: "80000000", ptr: new(uint32), value: uint32(128)},
	{input: "FF000000", ptr: new(uint32), value: uint32(255)},
	{input: "FFFF0000", ptr: new(uint32), value: uint32(65535)},
	{input: "FFFFFFFF", ptr: new(uint32), value: uint32(4294967295)},

	// uint64
	{input: "0000000000000000", ptr: new(uint64), value: uint64(0)},
	{input: "0100000000000000", ptr: new(uint64), value: uint64(1)},
	{input: "1000000000000000", ptr: new(uint64), value: uint64(16)},
	{input: "8000000000000000", ptr: new(uint64), value: uint64(128)},
	{input: "FF00000000000000", ptr: new(uint64), value: uint64(255)},
	{input: "FFFF000000000000", ptr: new(uint64), value: uint64(65535)},
	{input: "FFFFFFFF00000000", ptr: new(uint64), value: uint64(4294967295)},
	{input: "FFFFFFFFFFFFFFFF", ptr: new(uint64), value: uint64(18446744073709551615)},

	// bytes
	{input: "00000000", ptr: new([]byte), value: []byte{}},
	{input: "0100000001", ptr: new([]byte), value: []byte{1}},
	{input: "06000000 010203040506", ptr: new([]byte), value: []byte{1, 2, 3, 4, 5, 6}},

	// slice
	{input: "00000000", ptr: new([]uint16), value: []uint16(nil)},
	{input: "04000000 0100 0200", ptr: new([]uint16), value: []uint16{1, 2}},
	{input: "18000000 08000000 0100 0200 0300 0400 08000000 0500 0600 0700 0800", ptr: new([][]uint16),
		value: [][]uint16{
			{1, 2, 3, 4},
			{5, 6, 7, 8},
		},
	},

	// array
	{input: "01000000 01", ptr: new([1]byte), value: [1]byte{1}},
	{input: "06000000 010203040506", ptr: new([6]byte), value: [6]byte{1, 2, 3, 4, 5, 6}},
	{input: "02000000 0100", ptr: new([1]uint16), value: [1]uint16{1}},
	{input: "04000000 0100 0200", ptr: new([2]uint16), value: [2]uint16{1, 2}},
	{input: "18000000 08000000 0100 0200 0300 0400 08000000 0500 0600 0700 0800", ptr: new([2][4]uint16),
		value: [2][4]uint16{
			{1, 2, 3, 4},
			{5, 6, 7, 8},
		},
	},

	// struct
	{input: "03000000 00 0000", ptr: new(simpleStruct), value: simpleStruct{}},
	{input: "03000000 0200 01", ptr: new(simpleStruct), value: simpleStruct{B: 2, A: 1}},
	{input: "07000000 03 02000000 0600", ptr: new(outerStruct),
		value: outerStruct{
			V:    3,
			SubV: innerStruct{6},
		}},

	// slice + struct
	{input: "12000000 0E000000 03000000 020001 03000000 040003", ptr: new(arrayStruct),
		value: arrayStruct{
			V: []simpleStruct{
				{B: 2, A: 1},
				{B: 4, A: 3},
			},
		}},
	{input: "16000000 07000000 03 02000000 0600 07000000 05 02000000 0700", ptr: new([]outerStruct),
		value: []outerStruct{
			{V: 3, SubV: innerStruct{V: 6}},
			{V: 5, SubV: innerStruct{V: 7}},
		}},

	// pointer
	{input: "03000000 0200 01", ptr: new(*simpleStruct), value: &simpleStruct{B: 2, A: 1}},
	{input: "08000000 03000000 0200 01 03", ptr: new(pointerStruct),
		value: pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}},
	{input: "08000000 03000000 0200 01 03", ptr: new(*pointerStruct),
		value: &pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}},
	{input: "04000000 01020304", ptr: new(*[]uint8), value: &[]uint8{1, 2, 3, 4}},
	{input: "10000000 0100000000000000 0200000000000000", ptr: new(*[]uint64), value: &[]uint64{1, 2}},
	{input: "0E000000 03000000 0200 01 03000000 0400 03", ptr: new([]*simpleStruct),
		value: []*simpleStruct{
			{B: 2, A: 1},
			{B: 4, A: 3},
		},
	},
	{input: "0E000000 03000000 0200 01 03000000 0400 03", ptr: new([2]*simpleStruct),
		value: [2]*simpleStruct{
			{B: 2, A: 1},
			{B: 4, A: 3},
		},
	},
	{input: "18000000 08000000 03000000 0200 01 00 08000000 03000000 0400 03 01", ptr: new([]*pointerStruct),
		value: []*pointerStruct{
			{P: &simpleStruct{B: 2, A: 1}, V: 0},
			{P: &simpleStruct{B: 4, A: 3}, V: 1},
		},
	},

	// nil pointer
	{input: "00000000", ptr: new(*[]uint8), value: (*[]uint8)(nil)},
	{input: "05000000 00000000 00", ptr: new(pointerStruct), value: pointerStruct{}},
	{input: "05000000 00000000 00", ptr: new(*pointerStruct), value: &pointerStruct{}},
	{input: "08000000 00000000 00000000", ptr: new([]*pointerStruct), value: []*pointerStruct{nil, nil}},

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
	{input: "01000000", ptr: new([]byte), error: "decode error: can only read 0 bytes while expected to read 1 bytes for output type []uint8"},

	// error: slice: wrong header
	{input: "010000", ptr: new([]uint16), error: "decode error: failed to decode header of slice: can only read 3 bytes while expected to read 4 bytes for output type []uint16"},

	// error: slice: wrong input
	{input: "01000000", ptr: new([]uint16), error: "decode error: failed to decode element of slice: can only read 0 bytes while expected to read 2 bytes for output type []uint16"},

	// error: byte array: wrong input
	{input: "01000000 01", ptr: new([2]byte), error: "decode error: input byte array size (1) isn't euqal to output array size (2) for output type [2]uint8"},

	// error: array: input too short
	{input: "02000000 0100", ptr: new([2]uint16), error: "decode error: input is too short for output type [2]uint16"},

	// error: array: input too long
	{input: "04000000 0100 0200", ptr: new([1]uint16), error: "decode error: input is too long for output type [1]uint16"},

	// error: struct: wrong header
	{input: "010000", ptr: new(simpleStruct), error: "decode error: failed to decode header of struct: can only read 3 bytes while expected to read 4 bytes for output type ssz.simpleStruct"},

	// error: struct: wrong input
	{input: "03000000 01 02", ptr: new(simpleStruct), error: "decode error: failed to decode field of slice: can only read 0 bytes while expected to read 1 bytes for output type ssz.simpleStruct"},

	// error: struct: input too short
	{input: "02000000 0200", ptr: new(simpleStruct), error: "decode error: input is too short for output type ssz.simpleStruct"},

	// error: struct: input too long
	{input: "04000000 0200 01 01", ptr: new(simpleStruct), error: "decode error: input is too long for output type ssz.simpleStruct"},
}

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		CacheTreeHash: false,
	})
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
