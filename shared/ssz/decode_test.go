package ssz

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

// TODOs for this PR:
// - Aggregate test cases as a data-driven form
// - Add more tests
// - Do not use nested struct

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

func TestDecodeSlice(t *testing.T) {
	input := []byte{0, 0, 0, 10, 0, 1, 0, 2, 0, 3, 0, 4, 0, 5}
	bytesReader := bytes.NewReader(input)
	output := new([]uint16)
	if err := Decode(bytesReader, output); err != nil {
		t.Errorf("%v", err)
	}
	fmt.Println(*output)
}

func TestDecodeSlice1(t *testing.T) {
	input := []byte{0, 0, 0, 24, 0, 0, 0, 8, 0, 1, 0, 2, 0, 3, 0, 4, 0, 0, 0, 8, 0, 5, 0, 6, 0, 7, 0, 8}
	bytesReader := bytes.NewReader(input)
	output := new([][]uint16)
	if err := Decode(bytesReader, output); err != nil {
		t.Errorf("%v", err)
	}
	fmt.Println(*output)
}

func TestDecodeStruct(t *testing.T) {
	input := []byte{0, 0, 0, 7, 0, 0, 0, 2, 0, 11, 10}
	bytesReader := bytes.NewReader(input)

	type Subelem struct {
		Num uint16
	}
	type Elem struct {
		Num    byte
		Member Subelem
	}
	output := new(Elem)
	if err := Decode(bytesReader, output); err != nil {
		t.Errorf("%v", err)
	}
	fmt.Println(*output)
	if !reflect.DeepEqual(*output, Elem{
		Num:    10,
		Member: Subelem{11},
	}) {
		t.Error("decode result incorrect")
	}
}

func TestDecodeStruct1(t *testing.T) {
	input := []byte{0, 0, 0, 12, 0, 0, 0, 2, 0, 10, 0, 0, 0, 2, 0, 11}
	bytesReader := bytes.NewReader(input)

	type Subelem struct {
		Num uint16
	}
	type Elem struct {
		Member2 Subelem
		Member  Subelem
	}
	output := new(Elem)
	if err := Decode(bytesReader, output); err != nil {
		t.Errorf("%v", err)
	}
	fmt.Println(*output)
	if !reflect.DeepEqual(*output, Elem{
		Member2: Subelem{11},
		Member:  Subelem{10},
	}) {
		t.Error("decode result incorrect")
	}
}
