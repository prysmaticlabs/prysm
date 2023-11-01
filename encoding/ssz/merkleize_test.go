package ssz_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/crypto/hash"
	"github.com/prysmaticlabs/prysm/v4/encoding/ssz"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
)

func TestGetDepth(t *testing.T) {
	trieSize := uint64(896745231)
	expected := uint8(30)

	result := ssz.Depth(trieSize)
	assert.Equal(t, expected, result)
}

func TestMerkleizeCountGreaterThanLimit(t *testing.T) {
	hashFn := ssz.NewHasherFunc(hash.CustomSHA256Hasher())
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
	ssz.Merkleize(hashFn, count, limit, leafIndexer)
}

func TestMerkleizeLimitAndCountAreZero(t *testing.T) {
	hashFn := ssz.NewHasherFunc(hash.CustomSHA256Hasher())
	count := uint64(0)
	limit := uint64(0)
	chunks := [][]byte{{}}
	leafIndexer := func(i uint64) []byte {
		return chunks[i]
	}
	var expected [32]byte
	result := ssz.Merkleize(hashFn, count, limit, leafIndexer)
	assert.Equal(t, expected, result)
}

func TestMerkleizeNormalPath(t *testing.T) {
	hashFn := ssz.NewHasherFunc(hash.CustomSHA256Hasher())
	count := uint64(2)
	limit := uint64(3)
	chunks := [][]byte{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}}
	leafIndexer := func(i uint64) []byte {
		return chunks[i]
	}
	expected := [32]byte{95, 27, 253, 237, 215, 58, 147, 198, 175, 194, 180, 231, 154, 130, 205, 68, 146, 112, 225, 86, 6, 103, 186, 82, 7, 142, 33, 189, 174, 56, 199, 173}
	result := ssz.Merkleize(hashFn, count, limit, leafIndexer)
	assert.Equal(t, expected, result)
}

func TestDepthOfOne(t *testing.T) {
	assert.Equal(t, uint8(0), ssz.Depth(1))
}
