package bitutil

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

func TestCheckBit(t *testing.T) {
	tests := []struct {
		a []byte
		b int
		c bool
	}{
		{a: []byte{200}, b: 4, c: true},   //11001000
		{a: []byte{148}, b: 3, c: true},   //10010100
		{a: []byte{146}, b: 7, c: false},  //10010010
		{a: []byte{179}, b: 0, c: true},   //10110011
		{a: []byte{49}, b: 1, c: false},   //00110001
		{a: []byte{49}, b: 100, c: false}, //00110001

	}
	for _, tt := range tests {
		set, _ := CheckBit(tt.a, tt.b)
		if set != tt.c {
			t.Errorf("Test check bit set failed with %08b and location %v", tt.a, tt.b)
		}
	}

	bitFields := []byte{1}
	if set, err := CheckBit(bitFields, 100); err == nil || set {
		t.Error("Test check bit set should error if out of range index")
	}
}

func TestBitSetCount(t *testing.T) {
	tests := []struct {
		a byte
		b int
	}{
		{a: 200, b: 3}, //11001000
		{a: 148, b: 3}, //10010100
		{a: 146, b: 3}, //10010010
		{a: 179, b: 5}, //10110011
		{a: 49, b: 3},  //00110001
	}
	for _, tt := range tests {
		if int(BitSetCount([]byte{tt.a})) != tt.b {
			t.Errorf("BitSetCount(%d) = %v, want = %d", tt.a, int(BitSetCount([]byte{tt.a})), tt.b)
		}
	}
}

func TestByteLength(t *testing.T) {
	tests := []struct {
		a int
		b int
	}{
		{a: 200, b: 25},     //11001000
		{a: 34324, b: 4291}, //10010100
		{a: 146, b: 19},     //10010010
		{a: 179, b: 23},     //10110011
		{a: 49, b: 7},       //00110001
	}
	for _, tt := range tests {
		if BitLength(tt.a) != tt.b {
			t.Errorf("BitLength(%d) = %d, want = %d", tt.a, BitLength(tt.a), tt.b)
		}
	}
}

func TestBitSet(t *testing.T) {
	tests := []struct {
		a int
		b []byte
	}{
		{a: 0, b: []byte{128}},    //10000000
		{a: 1, b: []byte{64}},     //01000000
		{a: 5, b: []byte{4}},      //00000100
		{a: 10, b: []byte{0, 32}}, //00000000 00100000
		{a: 100, b: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8}},
	}
	for _, tt := range tests {
		if !bytes.Equal(SetBitfield(tt.a, mathutil.CeilDiv8(len(tt.b))), tt.b) {
			t.Errorf("SetBitfield(%v) = %d, want = %v", tt.a, SetBitfield(tt.a, mathutil.CeilDiv8(len(tt.b))), tt.b)
		}
	}
}

func TestSetBitfield_LargerCommitteesThanIndex(t *testing.T) {
	tests := []struct {
		a int
		b []byte
		c int
	}{
		{a: 300, b: []byte{128}, c: 40},    //10000000
		{a: 10000, b: []byte{64}, c: 2000}, //01000000
		{a: 800, b: []byte{4}, c: 120},     //00000100
		{a: 809, b: []byte{0, 32}, c: 130}, //00000000 00100000
		{a: 100, b: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8}, c: 14},
	}
	for _, tt := range tests {
		bfield := SetBitfield(tt.a, tt.c)

		if len(bfield) != tt.c {
			t.Errorf("Length of bitfield doesnt match the inputted committee size, got: %d but expected: %d", len(bfield), tt.c)
		}

	}
}
