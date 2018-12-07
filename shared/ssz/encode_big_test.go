package ssz

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

type encTest struct {
	val           interface{}
	output, error string
}

var encTests = []encTest{
	// boolean
	{val: false, output: "00"},
	{val: true, output: "01"},

	// uint8
	{val: uint8(0), output: "00"},
	{val: uint8(1), output: "01"},
	{val: uint8(16), output: "10"},
	{val: uint8(128), output: "80"},
	{val: uint8(255), output: "FF"},
}

func runEncTests(t *testing.T, f func(val interface{}) ([]byte, error)) {
	for i, test := range encTests {
		output, err := f(test.val)
		if err != nil && test.error == "" {
			t.Errorf("test %d: unexpected error: %v\nvalue %#v\ntype %T",
				i, err, test.val, test.val)
			continue
		}
		if test.error != "" && fmt.Sprint(err) != test.error {
			t.Errorf("test %d: error mismatch\ngot   %v\nwant  %v\nvalue %#v\ntype  %T",
				i, err, test.error, test.val, test.val)
			continue
		}
		if err == nil && !bytes.Equal(output, unhex(test.output)) {
			t.Errorf("test %d: output mismatch:\ngot   %X\nwant  %s\nvalue %#v\ntype  %T",
				i, output, test.output, test.val, test.val)
		}
	}
}

func TestEncode(t *testing.T) {
	runEncTests(t, func(val interface{}) ([]byte, error) {
		b := new(bytes.Buffer)
		err := Encode(b, val)
		return b.Bytes(), err
	})
}

func unhex(str string) []byte {
	b, err := hex.DecodeString(strings.Replace(str, " ", "", -1))
	if err != nil {
		panic(fmt.Sprintf("invalid hex string: %q", str))
	}
	return b
}
