package ssz

import (
	"bytes"
	"fmt"
	"testing"
)

// TODOs for this PR:
// - Make the test data-driven (data + runner). Do not write so many test functions

func TestEncodeUint8(t *testing.T) {
	b := new(bytes.Buffer)
	if err := Encode(b, uint8(12)); err != nil {
		t.Errorf("%v", err)
	}
	fmt.Println(b)
}

func TestEncodeUint16(t *testing.T) {
	b := new(bytes.Buffer)
	if err := Encode(b, uint16(256)); err != nil {
		t.Errorf("%v", err)
	}
	fmt.Println(b)
}

func TestEncodeBytes(t *testing.T) {
	b := new(bytes.Buffer)
	data := []byte{1, 2, 3, 4, 5, 6}
	if err := Encode(b, data); err != nil {
		t.Errorf("%v", err)
	}
	fmt.Println(b)
}

func TestEncodeSlice(t *testing.T) {
	data := []uint16{1, 2, 3, 4, 5}
	b := new(bytes.Buffer)
	if err := Encode(b, data); err != nil {
		t.Errorf("%v", err)
	}
	fmt.Println(b)
}

func TestEncodeSlice1(t *testing.T) {
	data := [][]uint16{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}
	b := new(bytes.Buffer)
	if err := Encode(b, data); err != nil {
		t.Errorf("%v", err)
	}
	fmt.Println(b)
}

func TestEncodeStruct(t *testing.T) {
	type subelem struct {
		num uint16
	}
	type elem struct {
		num    byte
		member subelem
	}
	data := elem{
		num: 10,
		member: subelem{
			num: 11,
		},
	}
	b := new(bytes.Buffer)
	if err := Encode(b, data); err != nil {
		t.Errorf("%v", err)
	}
	fmt.Println(b)
}

func TestEncodeStruct1(t *testing.T) {
	type subelem struct {
		num uint16
	}
	type elem struct {
		member2 subelem
		member  subelem
	}
	data := elem{
		member2: subelem{
			num: 11,
		},
		member: subelem{
			10,
		},
	}
	b := new(bytes.Buffer)
	if err := Encode(b, data); err != nil {
		t.Errorf("%v", err)
	}
	fmt.Println(b)
}
