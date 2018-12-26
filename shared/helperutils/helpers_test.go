package helperutils

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestMerkleRoot(t *testing.T) {
	valueSet := [][]byte{
		[]byte{'a'},
		[]byte{'b'},
		[]byte{'c'},
		[]byte{'d'},
	}

	leftNode := hashutil.Hash(append([]byte{'a'}, []byte{'b'}...))
	rightNode := hashutil.Hash(append([]byte{'c'}, []byte{'d'}...))
	expectedRoot := hashutil.Hash(append(leftNode[:], rightNode[:]...))

	if !bytes.Equal(expectedRoot[:], MerkleRoot(valueSet)) {
		t.Errorf("Expected Merkle root and computed merkle root are not equal %#x , %#x", expectedRoot, MerkleRoot(valueSet))
	}

}
