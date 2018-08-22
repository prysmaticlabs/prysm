package utils

import (
	"testing"
)

func TestCheckBit(t *testing.T) {
	tests := []struct {
		a []byte
		b int
		c bool
	}{
		{a: []byte{200}, b: 4, c: true},  //11001000
		{a: []byte{148}, b: 5, c: true},  //10010100
		{a: []byte{146}, b: 4, c: false}, //10010010
		{a: []byte{179}, b: 7, c: true},  //10110011
		{a: []byte{49}, b: 6, c: false},  //00110001

	}
	for _, tt := range tests {
		set, err := CheckBit(tt.a, tt.b)
		if err != nil {
			t.Fatalf("Call check bit failed: %v", err)
		}
		if set != tt.c {
			t.Errorf("Test check bit set failed with %v and location %v", tt.a, tt.b)
		}
	}
	if _, err := CheckBit([]byte{1}, 8); err == nil {
		t.Errorf("Call check bit should have failed with invalid index")
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
		if int(BitSetCount(tt.a)) != tt.b {
			t.Errorf("Expected %v, Got %v", tt.b, int(BitSetCount(tt.a)))
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
			t.Errorf("Expected %v, Got %v", tt.b, BitLength(tt.b))
		}
	}
}
