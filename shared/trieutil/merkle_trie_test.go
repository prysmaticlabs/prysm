package trieutil_test

import (
	"math/rand"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func TestNextPowerOf2(t *testing.T) {
	tests := []struct {
		input  int
		result int
	}{
		{input: 0, result: 0},
		{input: 1, result: 1},
		{input: 2, result: 2},
		{input: 3, result: 4},
		{input: 5, result: 8},
		{input: 9, result: 16},
		{input: 20, result: 32},
	}
	for _, tt := range tests {
		if got := trieutil.NextPowerOf2(tt.input); got != tt.result {
			t.Errorf("NextPowerOf2() = %d, result %d", got, tt.result)
		}
	}
}

func TestPrevPowerOf2(t *testing.T) {
	tests := []struct {
		input  int
		result int
	}{
		{input: 0, result: 0},
		{input: 1, result: 1},
		{input: 2, result: 2},
		{input: 3, result: 2},
		{input: 5, result: 4},
		{input: 9, result: 8},
		{input: 20, result: 16},
	}
	for _, tt := range tests {
		if got := trieutil.PrevPowerOf2(tt.input); got != tt.result {
			t.Errorf("PrevPowerOf2() = %d, result %d", got, tt.result)
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
		if got := trieutil.MerkleTree(tt.leaves); len(got) != tt.length {
			t.Errorf("len(MerkleTree()) = %d, result %d", got, tt.length)
		}
	}
}

func TestConcatGeneralizedIndices(t *testing.T) {
	tests := []struct {
		indices []int
		result  int
	}{
		{[]int{1, 2}, 2},
		{[]int{1, 3, 6}, 14},
		{[]int{1, 2, 4, 8}, 64},
		{[]int{1, 2, 5, 10}, 74},
	}
	for _, tt := range tests {
		if got := trieutil.ConcatGeneralizedIndices(tt.indices); got != tt.result {
			t.Errorf("ConcatGeneralizedIndices() = %d, result %d", got, tt.result)
		}
	}
}

func TestGeneralizedIndexLength(t *testing.T) {
	tests := []struct {
		index  int
		result int
	}{
		{index: 20, result: 4},
		{index: 200, result: 7},
		{index: 1987, result: 10},
		{index: 34989843, result: 25},
		{index: 97282, result: 16},
	}
	for _, tt := range tests {
		result := trieutil.GeneralizedIndexLength(tt.index)
		if tt.result != result {
			t.Errorf("GeneralizedIndexLength() = %d, result %d", tt.result, result)
		}
	}
}

func TestGeneralizedIndexBit(t *testing.T) {
	tests := []struct {
		index  uint64
		pos    uint64
		result bool
	}{
		{index: 7, pos: 2, result: true},
		{index: 7, pos: 3, result: false},
		{index: 10, pos: 2, result: false},
		{index: 10, pos: 3, result: true},
	}
	for _, tt := range tests {
		result := trieutil.GeneralizedIndexBit(tt.index, tt.pos)
		if result != tt.result {
			t.Errorf("GeneralizedIndexBit() = %v, result %v", tt.result, result)
		}
	}
}

func TestGeneralizedIndexChild(t *testing.T) {
	tests := []struct {
		index  int
		right  bool
		result int
	}{
		{index: 5, right: true, result: 11},
		{index: 10, right: false, result: 20},
		{index: 1000, right: true, result: 2001},
		{index: 9999, right: false, result: 19998},
	}
	for _, tt := range tests {
		result := trieutil.GeneralizedIndexChild(tt.index, tt.right)
		if result != tt.result {
			t.Errorf("GeneralizedIndexChild() = %v, result %v", tt.result, result)
		}
	}
}

func TestGeneralizedIndexSibling(t *testing.T) {
	tests := []struct {
		index  int
		result int
	}{
		{index: 5, result: 4},
		{index: 10, result: 11},
		{index: 1000, result: 1001},
		{index: 9999, result: 9998},
	}
	for _, tt := range tests {
		result := trieutil.GeneralizedIndexSibling(tt.index)
		if result != tt.result {
			t.Errorf("GeneralizedIndexSibling() = %v, result %v", tt.result, result)
		}
	}
}

func TestGeneralizedIndexParent(t *testing.T) {
	tests := []struct {
		index  int
		result int
	}{
		{index: 5, result: 2},
		{index: 10, result: 5},
		{index: 1000, result: 500},
		{index: 9999, result: 4999},
	}
	for _, tt := range tests {
		result := trieutil.GeneralizedIndexParent(tt.index)
		if result != tt.result {
			t.Errorf("GeneralizedIndexParent() = %v, result %v", tt.result, result)
		}
	}
}

func BenchmarkMerkleTree_Generate(b *testing.B) {
	leaves := make([][]byte, 1<<20)
	for i := 0; i < len(leaves); i++ {
		b := make([]byte, 32)
		rand.Read(b)
		leaves[i] = b
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trieutil.MerkleTree(leaves)
	}
}
