package ssz

import (
	"reflect"
	"strings"
	"testing"
)

type sigOrder1 struct {
	Signature []byte
	bla       byte
	bloo      bool
	blim      byte
}
type sigOrder2 struct {
	bla       byte
	bloo      bool
	blim      byte
	Signature []byte
}
type sigOrder3 struct {
	bla       byte
	bloo      bool
	Signature []byte
	blim      byte
}
type badStruct struct {
	bla       byte
	bloo      int
	Signature []byte
	blim      byte
}
type missSig struct {
	bla  byte
	bloo bool
	blim byte
}
type fieldTest struct {
	val      interface{}
	index    int
	sigFound bool
	error    string
}

func TestStructFieldMap_OK(t *testing.T) {

	var fmTests = []fieldTest{
		{val: sigOrder1{Signature: []byte{1}, bla: 1, bloo: true, blim: 2}, index: 0, error: "", sigFound: true},
		{val: sigOrder2{Signature: []byte{1}, bla: 1, bloo: true, blim: 2}, index: 3, error: "", sigFound: true},
		{val: sigOrder3{Signature: []byte{1}, bla: 1, bloo: true, blim: 2}, index: 2, error: "", sigFound: true},
		{val: missSig{bla: 1, bloo: false, blim: 2}, index: 0, error: "", sigFound: false},
		{val: badStruct{Signature: []byte{1}, bla: 1, bloo: 1, blim: 2}, index: 0, error: "failed to get ssz utils", sigFound: false},
		{val: 0.00003, index: 0, error: "wrong object type", sigFound: false},
	}
	s := "Signature"
	for i, test := range fmTests {
		typeObj1 := reflect.TypeOf(test.val)
		sfm, err := structFieldMap(typeObj1)
		// Check unexpected error
		if test.error == "" && err != nil {
			t.Errorf("test %d: unexpected error: %v\nvalue %#v\ntype %T",
				i, err, test.val, test.val)
			continue
		}
		// Check expected error
		if test.error != "" && !strings.Contains(err.Error(), test.error) {
			t.Errorf("test %d: error mismatch\ngot   %v\nwant  %v\nvalue %#v\ntype  %T",
				i, err, test.error, test.val, test.val)
			continue
		}
		sig, ok := sfm["Signature"]
		if test.sigFound {
			if !ok {
				t.Errorf("test %d: signature was not found in struct. map is: %v", i, sfm)
				continue
			}
			if sig.name != s {
				t.Errorf("test %d: wrong signature field name from map, want: %v got: %v", i, s, sig.name)
				continue
			}
			if sig.index != test.index {
				t.Errorf("test %d: wrong signature field index, wanted: %v got: %v", i, test.index, sig.index)
				continue
			}
		} else {
			if ok {
				t.Errorf("test %d: signature field found in struct. map is: %v, ", i, sfm)
				continue
			}
		}

	}

}
