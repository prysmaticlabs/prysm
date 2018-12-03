package ssz

import (
	"bytes"
	"fmt"
	"testing"
)

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
	data := []uint16{1,2,3,4,5}
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
