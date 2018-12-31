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

	// bytes
	{val: []byte{}, output: "E8E77626586F73B955364C7B4BBF0BB7F7685EBD40E852B164633A4ACBD3244C"},
	{val: []byte{1}, output: "A01F051BE047843977F523E7944513EBBEDD5568CC9911C955850F3CCCC6979F"},
	{val: []byte{1, 2, 3, 4, 5, 6}, output: "BD72911F3235F1E8BBC9F01C95526AE0C7AFE1E7C0E13429CC3E8EE307516B7E"},

	//// slice
	{val: []uint16{}, output: "DFDED4ED5AC76BA7379CFE7B3B0F53E768DCA8D45A34854E649CFC3C18CBD9CD"},
	{val: []uint16{1}, output: "75848BB7F08D2E009E76FDAD5A1C6129E48DF34D81245405F9C43B53D204DFAF"},
	{val: []uint16{1, 2}, output: "02A9991B320FD848FDFF2E069FF4A6E2B2A593FA13C32201EC89D5272332908D"},
	{val: [][]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, output: "D779C77E9E3FE29311097A0D62AA55077C2ACCC2CE89D6C3024877CA73222BB3"},

	// array
	{val: [1]byte{1}, output: "A01F051BE047843977F523E7944513EBBEDD5568CC9911C955850F3CCCC6979F"},
	{val: [6]byte{1, 2, 3, 4, 5, 6}, output: "BD72911F3235F1E8BBC9F01C95526AE0C7AFE1E7C0E13429CC3E8EE307516B7E"},
	{val: [1]uint16{1}, output: "75848BB7F08D2E009E76FDAD5A1C6129E48DF34D81245405F9C43B53D204DFAF"},
	{val: [2]uint16{1, 2}, output: "02A9991B320FD848FDFF2E069FF4A6E2B2A593FA13C32201EC89D5272332908D"},
	{val: [2][4]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, output: "D779C77E9E3FE29311097A0D62AA55077C2ACCC2CE89D6C3024877CA73222BB3"},

	// struct
	{val: simpleStruct{}, output: "99FF0D9125E1FC9531A11262E15AEB2C60509A078C4CC4C64CEFDFB06FF68647"},
	{val: simpleStruct{B: 2, A: 1}, output: "D841AA2CE38FDA992D162D973EBD62CF0C74522F8FEB294C48A03D4DB0A6B78C"},
	{val: outerStruct{
		V:    3,
		SubV: innerStruct{V: 6},
	}, output: "22E310A4B644FBCF2A8F90AC1A5311ED9070E6C9E68A2FE58082BBF78C0D030E"},

	// slice + struct
	{val: arrayStruct{
		V: []simpleStruct{
			{B: 2, A: 1},
			{B: 4, A: 3},
		},
	}, output: "C338A2C69762D374CF6453DDF9B1D685D75243CF59B1FDD9A4DF4DB623E80512"},
	{val: []outerStruct{
		{V: 3, SubV: innerStruct{V: 6}},
		{V: 5, SubV: innerStruct{V: 7}},
	}, output: "D568DF03934C0EE51F49AABE81F20844D2F44159C389411BBC0C6008D5D85395"},

	// pointer
	{val: &simpleStruct{B: 2, A: 1}, output: "D841AA2CE38FDA992D162D973EBD62CF0C74522F8FEB294C48A03D4DB0A6B78C"},
	{val: pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "4CD458194E1DBAF67726FEBDB4124759481C1BADBF2E2BF4A5CCB898760ADFEF"},
	{val: &pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "4CD458194E1DBAF67726FEBDB4124759481C1BADBF2E2BF4A5CCB898760ADFEF"},
	{val: &[]uint8{1, 2, 3, 4}, output: "B95DACF1ACB820C8D826235E82C9F6D62C30C20B56181FC7463F3C3EB0ABA574"},
	{val: &[]uint64{1, 2}, output: "319034D88B14EB956CB208AF60CE41BA6F80B179EF6216626AEE547586B700B0"},
	{val: []*simpleStruct{
		{B: 2, A: 1},
		{B: 4, A: 3},
	}, output: "6E16C02DCAAB3106B973CF6D6F7CE1DC386E438E5E56CC20C604D13BCDDC4CB4"},
	{val: [2]*simpleStruct{
		{B: 2, A: 1},
		{B: 4, A: 3},
	}, output: "6E16C02DCAAB3106B973CF6D6F7CE1DC386E438E5E56CC20C604D13BCDDC4CB4"},
	{val: []*pointerStruct{
		{P: &simpleStruct{B: 2, A: 1}, V: 0},
		{P: &simpleStruct{B: 4, A: 3}, V: 1},
	}, output: "E6A6A93F6E9EE2CB20043A9528AD9A4A6CF9212E4321427783EC9685429C0D26"},

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
