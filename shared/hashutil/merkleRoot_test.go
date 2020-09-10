package hashutil_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestMerkleRoot(t *testing.T) {
	valueSet := [][]byte{
		{'a'},
		{'b'},
		{'c'},
		{'d'},
	}

	hashedV1 := []byte{'a'}
	hashedV2 := []byte{'b'}
	hashedV3 := []byte{'c'}
	hashedV4 := []byte{'d'}

	leftNode := hashutil.Hash(append(hashedV1[:], hashedV2[:]...))
	rightNode := hashutil.Hash(append(hashedV3[:], hashedV4[:]...))
	expectedRoot := hashutil.Hash(append(leftNode[:], rightNode[:]...))
	assert.DeepEqual(t, expectedRoot[:], hashutil.MerkleRoot(valueSet))
}
