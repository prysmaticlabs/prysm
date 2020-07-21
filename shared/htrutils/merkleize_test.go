package htrutils_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestGetDepth(t *testing.T) {
	trieSize := uint64(896745231)
	expected := uint8(30)

	result := htrutils.GetDepth(trieSize)
	assert.Equal(t, expected, result)
}

func TestMerkleizeCountGreaterThanLimit(t *testing.T) {
	hashFn := htrutils.NewHasherFunc(hashutil.CustomSHA256Hasher())
	count := uint64(2)
	limit := uint64(1)
	chunks := [][]byte{{}}
	leafIndexer := func(i uint64) []byte {
		return chunks[i]
	}
	// Error if no panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic.")
		}
	}()
	htrutils.Merkleize(hashFn, count, limit, leafIndexer)
}

func TestMerkleizeLimitAndCountAreZero(t *testing.T) {
	hashFn := htrutils.NewHasherFunc(hashutil.CustomSHA256Hasher())
	count := uint64(0)
	limit := uint64(0)
	chunks := [][]byte{{}}
	leafIndexer := func(i uint64) []byte {
		return chunks[i]
	}
	expected := [32]byte{}
	result := htrutils.Merkleize(hashFn, count, limit, leafIndexer)
	assert.Equal(t, expected, result)
}

func TestMerkleizeNormalPath(t *testing.T) {
	hashFn := htrutils.NewHasherFunc(hashutil.CustomSHA256Hasher())
	count := uint64(2)
	limit := uint64(3)
	chunks := [][]byte{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}}
	leafIndexer := func(i uint64) []byte {
		return chunks[i]
	}
	expected := [32]byte{95, 27, 253, 237, 215, 58, 147, 198, 175, 194, 180, 231, 154, 130, 205, 68, 146, 112, 225, 86, 6, 103, 186, 82, 7, 142, 33, 189, 174, 56, 199, 173}
	result := htrutils.Merkleize(hashFn, count, limit, leafIndexer)
	assert.Equal(t, expected, result)
}

func TestConstructProofCountGreaterThanLimit(t *testing.T) {
	hashFn := htrutils.NewHasherFunc(hashutil.CustomSHA256Hasher())
	count := uint64(2)
	limit := uint64(1)
	chunks := [][]byte{{}}
	leafIndexer := func(i uint64) []byte {
		return chunks[i]
	}
	index := uint64(0)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic.")
		}
	}()
	htrutils.ConstructProof(hashFn, count, limit, leafIndexer, index)
}

func TestConstructProofIndexGreaterThanEqualToLimit(t *testing.T) {
	hashFn := htrutils.NewHasherFunc(hashutil.CustomSHA256Hasher())
	count := uint64(1)
	limit := uint64(1)
	chunks := [][]byte{{}}
	leafIndexer := func(i uint64) []byte {
		return chunks[i]
	}
	index := uint64(1)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic.")
		}
	}()
	htrutils.ConstructProof(hashFn, count, limit, leafIndexer, index)
}

func TestConstructProofNormalPath(t *testing.T) {
	hashFn := htrutils.NewHasherFunc(hashutil.CustomSHA256Hasher())
	count := uint64(2)
	limit := uint64(3)
	chunks := [][]byte{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}}
	leafIndexer := func(i uint64) []byte {
		return chunks[i]
	}
	index := uint64(1)
	expected := [][32]byte{
		{1, 2, 3, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{245, 165, 253, 66, 209, 106, 32, 48, 39, 152, 239, 110, 211, 9, 151, 155, 67, 0, 61, 35, 32, 217, 240, 232, 234, 152, 49, 169, 39, 89, 251, 75},
	}
	result := htrutils.ConstructProof(hashFn, count, limit, leafIndexer, index)
	assert.Equal(t, len(expected), len(result))
	for i, v := range expected {
		assert.DeepEqual(t, result[i], v)
	}
}
