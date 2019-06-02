package ssz

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

type hashTest struct {
	val           interface{}
	output, error string
}

type merkleHashTest struct {
	val           [][]byte
	output, error string
}

type signatureRootTest struct {
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
	{val: []byte{}, output: "DF3F619804A92FDB4057192DC43DD748EA778ADC52BC498CE80524C014B81119"},
	{val: []byte{1}, output: "A1CB20470D89874F33383802C72D3C27A0668EBFFD81934705AB0CFCBF1A1E3A"},
	{val: []byte{1, 2, 3, 4, 5, 6}, output: "78092EA62B72E4664DF8E488FE3E16286AA8F0CED46D61CB682538347703D0A6"},

	//// slice
	{val: []uint16{}, output: "B393978842A0FA3D3E1470196F098F473F9678E72463CB65EC4AB5581856C2E4"},
	{val: []uint16{1}, output: "B4D8F2028F995136423A87410EC7069CBC9B05F83DD3B11DD58381E10A1804DD"},
	{val: []uint16{1, 2}, output: "54B0FC9A31787379965D1B3DFC356CE565005FD69FD57FBFA8994E90126ABBB8"},
	{val: [][]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, output: "8BED84415CA181318F866F299402F4FB28DAF96582FBBA40EE1CA394B3A99D98"},

	// array
	{val: [1]byte{1}, output: "A1CB20470D89874F33383802C72D3C27A0668EBFFD81934705AB0CFCBF1A1E3A"},
	{val: [6]byte{1, 2, 3, 4, 5, 6}, output: "78092EA62B72E4664DF8E488FE3E16286AA8F0CED46D61CB682538347703D0A6"},
	{val: [1]uint16{1}, output: "B4D8F2028F995136423A87410EC7069CBC9B05F83DD3B11DD58381E10A1804DD"},
	{val: [2]uint16{1, 2}, output: "54B0FC9A31787379965D1B3DFC356CE565005FD69FD57FBFA8994E90126ABBB8"},
	{val: [2][4]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}, output: "8BED84415CA181318F866F299402F4FB28DAF96582FBBA40EE1CA394B3A99D98"},

	// struct
	{val: simpleStruct{}, output: "709E80C88487A2411E1EE4DFB9F22A861492D20C4765150C0C794ABD70F8147C"},
	{val: simpleStruct{B: 2, A: 1}, output: "9463926D4640A53A0CF1AC00657361F5A6D64FBC554DF30721CDACF503AEBC2B"},
	{val: outerStruct{
		V:    3,
		SubV: innerStruct{V: 6},
	}, output: "F71831A4F4EFFF501CF03D39C220BF6EF2DF43258FA9E0CAFC0B22F465D91A06"},

	// slice + struct
	{val: arrayStruct{
		V: []simpleStruct{
			{B: 2, A: 1},
			{B: 4, A: 3},
		},
	}, output: "FA68E4C209C92488D014999B351A7CD77D4B0E6AB21C7D726489B88D171ABACB"},
	{val: []outerStruct{
		{V: 3, SubV: innerStruct{V: 6}},
		{V: 5, SubV: innerStruct{V: 7}},
	}, output: "DC9D5BF0D6D34BACCF62C4D320893FD66B62B8D0C6D8A5BCE9AED20B8EE799AA"},

	// pointer
	{val: &simpleStruct{B: 2, A: 1}, output: "9463926D4640A53A0CF1AC00657361F5A6D64FBC554DF30721CDACF503AEBC2B"},
	{val: pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "B24B2C7E8BF5A64D7CE0B42D73EDAD05818686554D0D92DAA0EA2CB7458D09E3"},
	{val: &pointerStruct{P: &simpleStruct{B: 2, A: 1}, V: 3}, output: "B24B2C7E8BF5A64D7CE0B42D73EDAD05818686554D0D92DAA0EA2CB7458D09E3"},
	{val: &[]uint8{1, 2, 3, 4}, output: "FB39BA6662126DAEDEFD13A832F1878EF1CC49354FE8C99808F156CA6355BC35"},
	{val: &[]uint64{1, 2}, output: "9B7BEBACDC1EFE094F87D28A665905A513156BF729E9A05F2AE3B74679087FA8"},
	{val: []*simpleStruct{
		{B: 2, A: 1},
		{B: 4, A: 3},
	}, output: "9AFBCE02EB46D3DD4DCBF59572F1DE9070A57FD460F36786AA9AEEA9B0746516"},
	{val: [2]*simpleStruct{
		{B: 2, A: 1},
		{B: 4, A: 3},
	}, output: "9AFBCE02EB46D3DD4DCBF59572F1DE9070A57FD460F36786AA9AEEA9B0746516"},
	{val: []*pointerStruct{
		{P: &simpleStruct{B: 2, A: 1}, V: 0},
		{P: &simpleStruct{B: 4, A: 3}, V: 1},
	}, output: "B90EB45037ADE7B643D906260F2636C638F800BBDE16B4E54E8EBFAD2DCDCF4A"},

	// nil pointer (not defined in spec)
	{val: (*[]uint8)(nil), output: "DF3F619804A92FDB4057192DC43DD748EA778ADC52BC498CE80524C014B81119"},
	{val: pointerStruct{}, output: "C02E8D4C3A3175E0BB3267B843630E5BA02DA1498BBB2CD8B2FD213B0F5D8EF5"},
	{val: &pointerStruct{}, output: "C02E8D4C3A3175E0BB3267B843630E5BA02DA1498BBB2CD8B2FD213B0F5D8EF5"},
	{val: []*pointerStruct{nil, nil}, output: "A76CE233C01DE972947B65EE93321F36976D4F2DCFE3BACCAA97D6D5EE1E7505"},

	// error: untyped nil pointer
	{val: nil, error: "hash error: untyped nil is not supported for input type <nil>"},

	// error: unsupported type
	{val: string("abc"), error: "hash error: type string is not serializable for input type string"},
}

var merkleHashTests = []merkleHashTest{
	{val: [][]byte{}, output: "B393978842A0FA3D3E1470196F098F473F9678E72463CB65EC4AB5581856C2E4"},
	{val: [][]byte{{1, 2}, {3, 4}}, output: "40E89B538D2591DD5C880442E89762C5B539A2FA284A8F33526BCEA48A223FCE"},
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
	}, output: "CB5C0AB1A7587DA614CDE70F338B4743CE6E85C515F6BB94D110B5A537AC91D7"},
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
	}, output: "6215D9776BED2795136E531A324CC29CCEF50837BB0BEB7277BB956B331267D3"},
}

var signatureRootTests = []signatureRootTest{
	{val: &pb.BeaconBlockHeader{Slot: 20, Signature: []byte{'A', 'B'}},
		output: "647BA70F3F595DE2A83EE697FC6AA95C3024410AB301B0D195AB90BD5FA5E4CD"},
	{val: &pb.BeaconBlockHeader{Slot: 10, Signature: []byte("TESTING")},
		output: "2690E27CAB55805158C3DA81611E11B987B6654CAEDF7E4F88213D890BA1DB92"},
	{val: &pb.BeaconBlockHeader{
		Slot:       10,
		ParentRoot: []byte{'a', 'b'},
		Signature:  []byte("TESTING23")},
		output: "76EE0421D2A2EBE326EFC2F9CAD0D4D9C28C69D5CDCEE99708154F8935DB0CBE"},

	{val: &pb.IndexedAttestation{Signature: []byte("SigningAttestation")},
		output: "A7684BBCEB3BC4BBA6B8037C1E076CB7DB3B54EF5E7104846F9D2A3D8CD9E04B"},
	{val: &pb.VoluntaryExit{Signature: []byte("SigningExit")},
		output: "FF29ABF4ACBD832EA166C30C387F1E45FD4BBCD0CFAB1D683BF44C61A60320B8"},
	{val: pb.BeaconBlockHeader{Slot: 20, Signature: []byte{'A', 'B'}},
		output: "647BA70F3F595DE2A83EE697FC6AA95C3024410AB301B0D195AB90BD5FA5E4CD"},
	{val: pb.BeaconBlockHeader{
		Slot:       10,
		ParentRoot: []byte{'a', 'b'},
		Signature:  []byte("TESTING23")},
		output: "76EE0421D2A2EBE326EFC2F9CAD0D4D9C28C69D5CDCEE99708154F8935DB0CBE"},
	{val: pb.BeaconBlockHeader{Slot: 10, Signature: []byte("TESTING")},
		output: "2690E27CAB55805158C3DA81611E11B987B6654CAEDF7E4F88213D890BA1DB92"},
	{val: pb.IndexedAttestation{Signature: []byte("SigningAttestation")},
		output: "A7684BBCEB3BC4BBA6B8037C1E076CB7DB3B54EF5E7104846F9D2A3D8CD9E04B"},
	{val: pb.VoluntaryExit{Signature: []byte("SigningExit")},
		output: "FF29ABF4ACBD832EA166C30C387F1E45FD4BBCD0CFAB1D683BF44C61A60320B8"},
	{val: pb.Deposit{Index: 0}, error: "field name Signature is missing from the given struct"},
	{val: 2, error: "given object is neither a struct or a pointer"},
	{val: []byte{'a'}, error: "given object is neither a struct or a pointer"},
	{val: nil, error: "given object is neither a struct or a pointer"},
	{val: (*[]uint8)(nil), error: "nil pointer given"},
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

func runSignedRootTests(t *testing.T, signedRoot func(val interface{}) ([32]byte, error)) {
	for i, test := range signatureRootTests {
		output, err := signedRoot(test.val)
		// Check unexpected error
		if test.error == "" && err != nil {
			t.Errorf("test %d: unexpected error: %v\nvalue %#v\ntype %T",
				i, err, test.val, test.val)
			continue
		}
		// Check expected error
		if test.error != "" && !strings.Contains(fmt.Sprint(err), test.error) {
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

func TestSigningRoot(t *testing.T) {
	runSignedRootTests(t, func(val interface{}) ([32]byte, error) {
		return SigningRoot(val)
	})
}
