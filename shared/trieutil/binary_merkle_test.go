package trieutil

import (
	"testing"
)

func TestMerkleTreeLength(t *testing.T) {
	tests := []struct {
		leaves [][]byte
		length int
	}{
		{[][]byte{{'A'},{'B'},{'C'}}, 8},
		{[][]byte{{'A'},{'B'},{'C'},{'D'}}, 8},
		{[][]byte{{'A'},{'B'},{'C'},{'D'},{'E'}}, 16},

	}
	for _, tt := range tests {
			if got := MerkleTree(tt.leaves); len(got) != tt.length {
				t.Errorf("len(MerkleTree()) = %v, want %v", got, tt.length)
			}
	}
}
