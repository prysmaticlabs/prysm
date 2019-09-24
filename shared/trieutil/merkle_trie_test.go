package trieutil

import (
	"testing"
)

func TestNextPowerOf2(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{input: 0, want: 0},
		{input: 1, want: 1},
		{input: 2, want: 2},
		{input: 3, want: 4},
		{input: 5, want: 8},
		{input: 9, want: 16},
		{input: 20, want: 32},
	}
	for _, tt := range tests {
		if got := NextPowerOf2(tt.input); got != tt.want {
			t.Errorf("NextPowerOf2() = %d, want %d", got, tt.want)
		}
	}
}

func TestPrevPowerOf2(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{input: 0, want: 0},
		{input: 1, want: 1},
		{input: 2, want: 2},
		{input: 3, want: 2},
		{input: 5, want: 4},
		{input: 9, want: 8},
		{input: 20, want: 16},
	}
	for _, tt := range tests {
		if got := PrevPowerOf2(tt.input); got != tt.want {
			t.Errorf("PrevPowerOf2() = %d, want %d", got, tt.want)
		}
	}
}

func TestMerkleTreeLength(t *testing.T) {
	tests := []struct {
		leaves [][]byte
		length int
	}{
		{[][]byte{{'A'}, {'B'}, {'C'}}, 8},
		{[][]byte{{'A'}, {'B'}, {'C'}, {'D'}}, 8},
		{[][]byte{{'A'}, {'B'}, {'C'}, {'D'}, {'E'}}, 16},
	}
	for _, tt := range tests {
		if got := MerkleTree(tt.leaves); len(got) != tt.length {
			t.Errorf("len(MerkleTree()) = %d, want %d", got, tt.length)
		}
	}
}

func TestConcatGeneralizedIndices(t *testing.T) {
	tests := []struct {
		indices []int
		result int
	}{
		{[]int{1,2}, 2},
		{[]int{1,3,6}, 14},
		{[]int{1,2,4,8}, 64},
		{[]int{1,2,5,10}, 74},
	}
	for _, tt := range tests {
		if got := ConcatGeneralizedIndices(tt.indices); got != tt.result {
			t.Errorf("ConcatGeneralizedIndices() = %d, want %d", got, tt.result)
		}
	}
}

func TestIntegerSquareRoot(t *testing.T) {
	tests := []struct {
		number int
		root   int
	}{
		{
			number: 20,
			root:   4,
		},
		{
			number: 200,
			root:   7,
		},
		{
			number: 1987,
			root:   10,
		},
		{
			number: 34989843,
			root:   25,
		},
		{
			number: 97282,
			root:   16,
		},
	}

	for _, tt := range tests {
		root := GeneralizedIndexLength(tt.number)
		if tt.root != root {
			t.Errorf("GeneralizedIndexLength() = %d, want %d", tt.root, root)
		}
	}
}
