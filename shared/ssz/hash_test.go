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

type merkleHashTest struct {
	val           [][]byte
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
	{val: uint16(1), output: "0100000000000000000000000000000000000000000000000000000000000000"},
	{val: uint16(16), output: "1000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint16(128), output: "8000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint16(255), output: "FF00000000000000000000000000000000000000000000000000000000000000"},
	{val: uint16(65535), output: "FFFF000000000000000000000000000000000000000000000000000000000000"},

	// uint32
	{val: uint32(0), output: "0000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint32(1), output: "0100000000000000000000000000000000000000000000000000000000000000"},
	{val: uint32(16), output: "1000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint32(128), output: "8000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint32(255), output: "FF00000000000000000000000000000000000000000000000000000000000000"},
	{val: uint32(65535), output: "FFFF000000000000000000000000000000000000000000000000000000000000"},
	{val: uint32(4294967295), output: "FFFFFFFF00000000000000000000000000000000000000000000000000000000"},

	// uint64
	{val: uint64(0), output: "0000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint64(1), output: "0100000000000000000000000000000000000000000000000000000000000000"},
	{val: uint64(16), output: "1000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint64(128), output: "8000000000000000000000000000000000000000000000000000000000000000"},
	{val: uint64(255), output: "FF00000000000000000000000000000000000000000000000000000000000000"},
	{val: uint64(65535), output: "FFFF000000000000000000000000000000000000000000000000000000000000"},
	{val: uint64(4294967295), output: "FFFFFFFF00000000000000000000000000000000000000000000000000000000"},
	{val: uint64(18446744073709551615), output: "FFFFFFFFFFFFFFFF000000000000000000000000000000000000000000000000"},

	// bytes
	{val: []byte{}, output: "E8E77626586F73B955364C7B4BBF0BB7F7685EBD40E852B164633A4ACBD3244C"},
	{val: []byte{1}, output: "B2559FED89F0EC17542C216683DC6B75506F3754E0C045742936742CAE6343CA"},
	{val: []byte{1, 2, 3, 4, 5, 6}, output: "1310542D28BE8E0B3FF72E985BC06232B9A30D93AE1AD2E33C5383A54AB5C9A7"},

	//// slice
	{val: []uint16{}, output: "DFDED4ED5AC76BA7379CFE7B3B0F53E768DCA8D45A34854E649CFC3C18CBD9CD"},
	{val: []uint16{1}, output: "BEC34688811267C6AD6DE290D692AB772FE62734661807B991309B4B7CE6A885"},
	{val: []uint16{1, 2}, output: "A4D2A862B7630CF1D1A13105D7E5753879DFF4D5460F86DE3F666435CE23DB17"},
	{val: [][]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, output: "3AF15AD122A4352C64FA83A670052DFD371560FDC6299A45B82B03A021B6E436"},

	// array
	{val: [1]byte{1}, output: "B2559FED89F0EC17542C216683DC6B75506F3754E0C045742936742CAE6343CA"},
	{val: [6]byte{1, 2, 3, 4, 5, 6}, output: "1310542D28BE8E0B3FF72E985BC06232B9A30D93AE1AD2E33C5383A54AB5C9A7"},
	{val: [1]uint16{1}, output: "BEC34688811267C6AD6DE290D692AB772FE62734661807B991309B4B7CE6A885"},
	{val: [2]uint16{1, 2}, output: "A4D2A862B7630CF1D1A13105D7E5753879DFF4D5460F86DE3F666435CE23DB17"},
	{val: [2][4]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, output: "3AF15AD122A4352C64FA83A670052DFD371560FDC6299A45B82B03A021B6E436"},

	// struct
	{val: simpleStruct{}, output: "99FF0D9125E1FC9531A11262E15AEB2C60509A078C4CC4C64CEFDFB06FF68647"},
	{val: simpleStruct{B: 2, A: 1}, output: "D2B49B00C76582823E30B56FE608FF030EF7B6BD7DCC16B8994C9D74860A7E1C"},
	{val: outerStruct{
		V:    3,
		SubV: innerStruct{V: 6},
	}, output: "BB2F30386C55445381EEE7A33C3794227B8C8E4BE4CAA54506901A4DDFE79EE2"},

	// slice + struct
	{val: arrayStruct{
		V: []simpleStruct{
			{B: 2, A: 1},
			{B: 4, A: 3},
		},
	}, output: "649C84878A58ECED5C367948DC3A578BF0AFD078E7E615898131B08F5404D90B"},
	{val: []outerStruct{
		{V: 3, SubV: innerStruct{V: 6}},
		{V: 5, SubV: innerStruct{V: 7}},
	}, output: "1C68D7F36C8B0E36E393F63A5401AE1EAB92E3343574404C97162BC1BAACEECE"},

	// pointer
	{val: &simpleStruct{B: 2, A: 1}, output: "D2B49B00C76582823E30B56FE608FF030EF7B6BD7DCC16B8994C9D74860A7E1C"},
	{val: pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "D365B04884AA7B9160F5E405796F0EB7521FC69BD79D934DA72EDA1FC98B5971"},
	{val: &pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "D365B04884AA7B9160F5E405796F0EB7521FC69BD79D934DA72EDA1FC98B5971"},
	{val: &[]uint8{1, 2, 3, 4}, output: "5C8046AB6A4E32E5C0017620A1844E5851074E4EDA685A920E8C70007E675E5C"},
	{val: &[]uint64{1, 2}, output: "5411B1731A42DF98549CE1319707D5741BBBFEB69E50AD0DFA44E30CFF7F9E99"},
	{val: []*simpleStruct{
		{B: 2, A: 1},
		{B: 4, A: 3},
	}, output: "BCE003DAEA45CE54FDAFC72BE4B6B1A077230CD80EB553F8678E1D0CC9516749"},
	{val: [2]*simpleStruct{
		{B: 2, A: 1},
		{B: 4, A: 3},
	}, output: "BCE003DAEA45CE54FDAFC72BE4B6B1A077230CD80EB553F8678E1D0CC9516749"},
	{val: []*pointerStruct{
		{P: &simpleStruct{B: 2, A: 1}, V: 0},
		{P: &simpleStruct{B: 4, A: 3}, V: 1},
	}, output: "40B328604B555B0CF96678DFB333C2C4D46B238C40D33B6CFE78EFD02E9A55E3"},

	// error: nil pointer
	{val: nil, error: "hash error: nil is not supported for input type <nil>"},
	{val: (*[]uint8)(nil), error: "hash error: nil is not supported for input type *[]uint8"},
	{val: pointerStruct{P: nil, V: 0}, error: "hash error: failed to hash field of struct: nil is not supported for input type ssz.pointerStruct"},

	// error: unsupported type
	{val: string("abc"), error: "hash error: type string is not serializable for input type string"},
}

var merkleHashTests = []merkleHashTest{
	{val: [][]byte{}, output: "DFDED4ED5AC76BA7379CFE7B3B0F53E768DCA8D45A34854E649CFC3C18CBD9CD"},
	{val: [][]byte{{1, 2}, {3, 4}}, output: "D065B8CB25F0C84C86028FE9C3DE7BF08262D5188AA6B0B6AA8781513399262E"},
	{val: [][]byte{
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
		{3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3},
		{4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},
		{5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5},
		{6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6},
		{7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7},
		{8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8},
		{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9},
		{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
	}, output: "359D4DA50D11CDC0BAC57DC4888E60C0ACAA0498F76050E5E50CE0A51466CEEE"},
	{val: [][]byte{
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
		{3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3},
		{4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},
		{5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5},
		{6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6},
		{7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7},
		{8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8},
		{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9},
		{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
	}, output: "21C56180C7CCC7BE1BB5A4A27117A61DCEE85D7009F2241E77587387FE5C5A46"},
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

func runMerkleHashTests(t *testing.T, merkleHash func([][]byte) ([]byte, error)) {
	for i, test := range merkleHashTests {
		output, err := merkleHash(test.val)
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
		return TreeHash(val)
	})
}

func TestMerkleHash(t *testing.T) {
	runMerkleHashTests(t, func(val [][]byte) ([]byte, error) {
		return merkleHash(val)
	})
}
