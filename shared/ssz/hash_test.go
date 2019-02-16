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
	{val: []uint16{1}, output: "E3F121F639DAE19B7E2FD6F5002F321B83F17288A7CA7560F81C2ACE832CC5D5"},
	{val: []uint16{1, 2}, output: "A9B7D66D80F70C6DA7060C3DEDB01E6ED6CEA251A3247093CBF27A439ECB0BEA"},
	{val: [][]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, output: "1A400EB17C755E4445C2C57DD2D3A0200A290C56CD68957906DD7BFE04493B10"},

	// array
	{val: [1]byte{1}, output: "B2559FED89F0EC17542C216683DC6B75506F3754E0C045742936742CAE6343CA"},
	{val: [6]byte{1, 2, 3, 4, 5, 6}, output: "1310542D28BE8E0B3FF72E985BC06232B9A30D93AE1AD2E33C5383A54AB5C9A7"},
	{val: [1]uint16{1}, output: "E3F121F639DAE19B7E2FD6F5002F321B83F17288A7CA7560F81C2ACE832CC5D5"},
	{val: [2]uint16{1, 2}, output: "A9B7D66D80F70C6DA7060C3DEDB01E6ED6CEA251A3247093CBF27A439ECB0BEA"},
	{val: [2][4]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, output: "1A400EB17C755E4445C2C57DD2D3A0200A290C56CD68957906DD7BFE04493B10"},

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
	}, output: "F3032DCE4B4218187E34AA8B6EF87A3FABE1F8D734CE92796642DC6B2911277C"},
	{val: []outerStruct{
		{V: 3, SubV: innerStruct{V: 6}},
		{V: 5, SubV: innerStruct{V: 7}},
	}, output: "DE43BC05AA6B011121F9590C10DE1734291A595798C84A0E3EDD1CC1E6710908"},

	// pointer
	{val: &simpleStruct{B: 2, A: 1}, output: "D2B49B00C76582823E30B56FE608FF030EF7B6BD7DCC16B8994C9D74860A7E1C"},
	{val: pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "D365B04884AA7B9160F5E405796F0EB7521FC69BD79D934DA72EDA1FC98B5971"},
	{val: &pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "D365B04884AA7B9160F5E405796F0EB7521FC69BD79D934DA72EDA1FC98B5971"},
	{val: &[]uint8{1, 2, 3, 4}, output: "5C8046AB6A4E32E5C0017620A1844E5851074E4EDA685A920E8C70007E675E5C"},
	{val: &[]uint64{1, 2}, output: "2F3E7F86CF5B91C6FC45FDF54254DE256F4FFFE775F0217C876961C4211E5DC2"},
	{val: []*simpleStruct{
		{B: 2, A: 1},
		{B: 4, A: 3},
	}, output: "1D5CDF2C53DD8AC743E17E1A7A8B1CB6E615FA63EC915347B3E9ACFB58F89158"},
	{val: [2]*simpleStruct{
		{B: 2, A: 1},
		{B: 4, A: 3},
	}, output: "1D5CDF2C53DD8AC743E17E1A7A8B1CB6E615FA63EC915347B3E9ACFB58F89158"},
	{val: []*pointerStruct{
		{P: &simpleStruct{B: 2, A: 1}, V: 0},
		{P: &simpleStruct{B: 4, A: 3}, V: 1},
	}, output: "4AC9B9E64A067F6C007C3FE8116519D86397BDA1D9FBEDEEDF39E50D132669C7"},

	// nil pointer (not defined in spec)
	{val: (*[]uint8)(nil), output: "E8E77626586F73B955364C7B4BBF0BB7F7685EBD40E852B164633A4ACBD3244C"},
	{val: pointerStruct{}, output: "721B2869FA1238991B24C369E9ADB23142AFCD7C0B8454EF79C0EA82B7DEE977"},
	{val: &pointerStruct{}, output: "721B2869FA1238991B24C369E9ADB23142AFCD7C0B8454EF79C0EA82B7DEE977"},
	{val: []*pointerStruct{nil, nil}, output: "83CB52B40904E607A8E0AEF8A018A5A7489229CBCD591E7C6FB7E597BD4F76C3"},

	// error: untyped nil pointer
	{val: nil, error: "hash error: untyped nil is not supported for input type <nil>"},

	// error: unsupported type
	{val: string("abc"), error: "hash error: type string is not serializable for input type string"},
}

var merkleHashTests = []merkleHashTest{
	{val: [][]byte{}, output: "DFDED4ED5AC76BA7379CFE7B3B0F53E768DCA8D45A34854E649CFC3C18CBD9CD"},
	{val: [][]byte{{1, 2}, {3, 4}}, output: "64F741B8BAB62525A01F9084582C148FF56C82F96DC12E270D3E7B5103CF7B48"},
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
	}, output: "839D98509E2EFC53BD1DEA17403921A89856E275BBF4D56C600CC3F6730AAFFA"},
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
	}, output: "55DC6699E7B5713DD9102224C302996F931836C6DAE9A4EC6AB49C966F394685"},
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
