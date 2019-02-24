package hashutil

import (
	"bytes"
	"testing"
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

	leftNode := Hash(append(hashedV1[:], hashedV2[:]...))
	rightNode := Hash(append(hashedV3[:], hashedV4[:]...))
	expectedRoot := Hash(append(leftNode[:], rightNode[:]...))

	if !bytes.Equal(expectedRoot[:], MerkleRoot(valueSet)) {
		t.Errorf("Expected Merkle root and computed merkle root are not equal %#x , %#x", expectedRoot, MerkleRoot(valueSet))
	}

}
