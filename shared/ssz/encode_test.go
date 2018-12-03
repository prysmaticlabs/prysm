package ssz

import (
	"bytes"
	"testing"
)

// TODOs for this PR:
// - Make the test data-driven (data + runner). Do not write so many test functions
// - Add more tests (try cross-over types)

func TestEncodeUint8(t *testing.T) {
	b := new(bytes.Buffer)
	if err := Encode(b, uint8(12)); err != nil {
		t.Errorf("%v", err)
	}
	if b.String() != string([]byte{12}) {
		t.Error("wrong encode result")
	}
}

func TestEncodeUint16(t *testing.T) {
	b := new(bytes.Buffer)
	if err := Encode(b, uint16(256)); err != nil {
		t.Errorf("%v", err)
	}
	if b.String() != string([]byte{1, 0}) {
		t.Error("wrong encode result")
	}
}

func TestEncodeBytes(t *testing.T) {
	b := new(bytes.Buffer)
	data := []byte{1, 2, 3, 4, 5, 6}
	if err := Encode(b, data); err != nil {
		t.Errorf("%v", err)
	}
	if b.String() != string([]byte{0, 0, 0, 6, 1, 2, 3, 4, 5, 6}) {
		t.Error("wrong encode result")
	}
}

func TestEncodeSlice(t *testing.T) {
	data := []uint16{1, 2, 3, 4, 5}
	b := new(bytes.Buffer)
	if err := Encode(b, data); err != nil {
		t.Errorf("%v", err)
	}
	if b.String() != string([]byte{0, 0, 0, 10, 0, 1, 0, 2, 0, 3, 0, 4, 0, 5}) {
		t.Error("wrong encode result")
	}
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
	if b.String() != string([]byte{0, 0, 0, 24, 0, 0, 0, 8, 0, 1, 0, 2, 0, 3, 0, 4, 0, 0, 0, 8, 0, 5, 0, 6, 0, 7, 0, 8}) {
		t.Error("wrong encode result")
	}
}

func TestEncodeStruct(t *testing.T) {
	type Subelem struct {
		Num uint16
	}
	type Elem struct {
		Num    byte
		Member Subelem
	}
	data := Elem{
		Num: 10,
		Member: Subelem{
			Num: 11,
		},
	}
	b := new(bytes.Buffer)
	if err := Encode(b, data); err != nil {
		t.Errorf("%v", err)
	}
	if b.String() != string([]byte{0, 0, 0, 7, 0, 0, 0, 2, 0, 11, 10}) {
		t.Error("wrong encode result")
	}
}

func TestEncodeStruct1(t *testing.T) {
	type Subelem struct {
		Num uint16
	}
	type Elem struct {
		Member2 Subelem
		Member  Subelem
	}
	data := Elem{
		Member2: Subelem{
			Num: 11,
		},
		Member: Subelem{
			10,
		},
	}
	b := new(bytes.Buffer)
	if err := Encode(b, data); err != nil {
		t.Errorf("%v", err)
	}
	if b.String() != string([]byte{0, 0, 0, 12, 0, 0, 0, 2, 0, 10, 0, 0, 0, 2, 0, 11}) {
		t.Error("wrong encode result")
	}
}
