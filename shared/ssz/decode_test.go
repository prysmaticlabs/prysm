package ssz

import (
	"bytes"
	"fmt"
	"testing"
)

// TODOs for this PR:
// - Aggregate test cases as a data-driven form

func TestDecodeUint8(t *testing.T) {
	input := []byte{10}
	bytesReader := bytes.NewReader(input)
	output := new(uint8)
	if err := Decode(bytesReader, output); err != nil {
		t.Errorf("%v", err)
	}
	if *output != 10 {
		t.Error("decode result wrong")
	}
	fmt.Println(*output)
}

func TestDecodeUint16(t *testing.T) {
	input := []byte{1, 0}
	bytesReader := bytes.NewReader(input)
	output := new(uint16)
	if err := Decode(bytesReader, output); err != nil {
		t.Errorf("%v", err)
	}
	if *output != 256 {
		t.Error("decode result wrong")
	}
	fmt.Println(*output)
}

func TestDecodeBytes(t *testing.T) {
	input := []byte{0, 0, 0, 6, 1, 2, 3, 4, 5, 6}
	bytesReader := bytes.NewReader(input)
	output := new([]byte)
	if err := Decode(bytesReader, output); err != nil {
		t.Errorf("%v", err)
	}
	fmt.Println(*output)
}
