package utils

import (
	"testing"
)

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
