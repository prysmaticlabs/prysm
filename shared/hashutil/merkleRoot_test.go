package hashutil

import (
	"bytes"
	"testing"
)

func TestMerkleRoot(t *testing.T) {
	valueSet := [][]byte{
		[]byte{'a'},
		[]byte{'b'},
		[]byte{'c'},
		[]byte{'d'},
	}

	leftNode := Hash(append([]byte{'a'}, []byte{'b'}...))
	rightNode := Hash(append([]byte{'c'}, []byte{'d'}...))
	expectedRoot := Hash(append(leftNode[:], rightNode[:]...))

	if !bytes.Equal(expectedRoot[:], MerkleRoot(valueSet)) {
		t.Errorf("Expected Merkle root and computed merkle root are not equal %#x , %#x", expectedRoot, MerkleRoot(valueSet))
	}

}
