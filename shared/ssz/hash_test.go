package ssz

import (
	"bytes"
	"fmt"
	"testing"
)

type hashTest struct {
	val           interface{}
	output, error string
}

// Notice: spaces in the output string will be ignored.
var hashTests = []hashTest{
	// boolean
	{val: false, output: "0000000000000000000000000000000000000000000000000000000000000000"},
	{val: true, output: "0100000000000000000000000000000000000000000000000000000000000000"},

	// uint8
	{val: uint8(0), output: "0000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint8(1), output: "0100000000000000000000000000000000000000000000000000000000000000"},
	{val: uint8(16), output: "1000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint8(128), output: "8000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint8(255), output: "FF00000000000000000000000000000000000000000000000000000000000000"},

	// uint16
	{val: uint16(0), output: "0000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint16(1), output: "0001000000000000000000000000000000000000000000000000000000000000"},
	{val: uint16(16), output: "0010000000000000000000000000000000000000000000000000000000000000"},
	{val: uint16(128), output: "0080000000000000000000000000000000000000000000000000000000000000"},
	{val: uint16(255), output: "00FF000000000000000000000000000000000000000000000000000000000000"},
	{val: uint16(65535), output: "FFFF000000000000000000000000000000000000000000000000000000000000"},

	// uint32
	{val: uint32(0), output: "0000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint32(1), output: "0000000100000000000000000000000000000000000000000000000000000000"},
	{val: uint32(16), output: "0000001000000000000000000000000000000000000000000000000000000000"},
	{val: uint32(128), output: "0000008000000000000000000000000000000000000000000000000000000000"},
	{val: uint32(255), output: "000000FF00000000000000000000000000000000000000000000000000000000"},
	{val: uint32(65535), output: "0000FFFF00000000000000000000000000000000000000000000000000000000"},
	{val: uint32(4294967295), output: "FFFFFFFF00000000000000000000000000000000000000000000000000000000"},

	// uint64
	{val: uint64(0), output: "0000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint64(1), output: "0000000000000001000000000000000000000000000000000000000000000000"},
	{val: uint64(16), output: "0000000000000010000000000000000000000000000000000000000000000000"},
	{val: uint64(128), output: "0000000000000080000000000000000000000000000000000000000000000000"},
	{val: uint64(255), output: "00000000000000FF000000000000000000000000000000000000000000000000"},
	{val: uint64(65535), output: "000000000000FFFF000000000000000000000000000000000000000000000000"},
	{val: uint64(4294967295), output: "00000000FFFFFFFF000000000000000000000000000000000000000000000000"},
	{val: uint64(18446744073709551615), output: "FFFFFFFFFFFFFFFF000000000000000000000000000000000000000000000000"},

	//// bytes
	//{val: []byte{}, output: "00000000"},
	//{val: []byte{1}, output: "00000001 01"},
	//{val: []byte{1, 2, 3, 4, 5, 6}, output: "00000006 010203040506"},
	//
	//// slice
	//{val: []uint16{}, output: "00000000"},
	//{val: []uint16{1}, output: "00000002 0001"},
	//{val: []uint16{1, 2}, output: "00000004 0001 0002"},
	//{val: [][]uint16{
	//	{1, 2, 3, 4},
	//	{5, 6, 7, 8},
	//}, output: "00000018 00000008 0001 0002 0003 0004 00000008 0005 0006 0007 0008"},
	//
	//// array
	//{val: [1]byte{1}, output: "00000001 01"},
	//{val: [6]byte{1, 2, 3, 4, 5, 6}, output: "00000006 010203040506"},
	//{val: [1]uint16{1}, output: "00000002 0001"},
	//{val: [2]uint16{1, 2}, output: "00000004 0001 0002"},
	//{val: [2][4]uint16{
	//	{1, 2, 3, 4},
	//	{5, 6, 7, 8},
	//}, output: "00000018 00000008 0001 0002 0003 0004 00000008 0005 0006 0007 0008"},
	//
	//// struct
	//{val: simpleStruct{}, output: "00000003 00 0000"},
	//{val: simpleStruct{B: 2, A: 1}, output: "00000003 01 0002"},
	//{val: outerStruct{
	//	V:    3,
	//	SubV: innerStruct{V: 6},
	//}, output: "00000007 00000002 0006 03"},
	//
	//// slice + struct
	//{val: arrayStruct{
	//	V: []simpleStruct{
	//		{B: 2, A: 1},
	//		{B: 4, A: 3},
	//	},
	//}, output: "00000012 0000000E 00000003 010002 00000003 030004"},
	//{val: []outerStruct{
	//	{V: 3, SubV: innerStruct{V: 6}},
	//	{V: 5, SubV: innerStruct{V: 7}},
	//}, output: "00000016 00000007 00000002 0006 03 00000007 00000002 0007 05"},
	//
	//// pointer
	//{val: &simpleStruct{B: 2, A: 1}, output: "00000003 01 0002"},
	//{val: pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "00000008 00000003 01 0002 03"},
	//{val: &pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "00000008 00000003 01 0002 03"},
	//{val: &[]uint8{1, 2, 3, 4}, output: "00000004 01020304"},
	//{val: &[]uint64{1, 2}, output: "00000010 0000000000000001 0000000000000002"},
	//{val: []*simpleStruct{
	//	{B: 2, A: 1},
	//	{B: 4, A: 3},
	//}, output: "0000000E 00000003 01 0002 00000003 03 0004"},
	//{val: [2]*simpleStruct{
	//	{B: 2, A: 1},
	//	{B: 4, A: 3},
	//}, output: "0000000E 00000003 01 0002 00000003 03 0004"},
	//{val: []*pointerStruct{
	//	{P: &simpleStruct{B: 2, A: 1}, V: 0},
	//	{P: &simpleStruct{B: 4, A: 3}, V: 1},
	//}, output: "00000018 00000008 00000003 01 0002 00 00000008 00000003 03 0004 01"},
	//
	//// error: nil pointer
	//{val: nil, error: "encode error: nil is not supported for input type <nil>"},
	//{val: (*[]uint8)(nil), error: "encode error: nil is not supported for input type *[]uint8"},
	//{val: pointerStruct{P: nil, V: 0}, error: "encode error: failed to encode field of struct: nil is not supported for input type ssz.pointerStruct"},
	//
	//// error: unsupported type
	//{val: string("abc"), error: "encode error: type string is not serializable for input type string"},
}

func runHashTests(t *testing.T, hash func(val interface{}) ([32]byte, error)) {
	for i, test := range hashTests {
		output, err := hash(test.val)
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
		if err == nil && !bytes.Equal(output[:], unhex(test.output)) {
			t.Errorf("test %d: output mismatch:\ngot   %X\nwant  %s\nvalue %#v\ntype  %T",
				i, output, stripSpace(test.output), test.val, test.val)
		}
	}
}

func TestHash(t *testing.T) {
	runHashTests(t, func(val interface{}) ([32]byte, error) {
		return Hash(val)
	})
}
