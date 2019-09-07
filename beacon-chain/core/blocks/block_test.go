package blocks

import (
	"bytes"
	"testing"
)

func TestGenesisBlock_InitializedCorrectly(t *testing.T) {
	stateHash := []byte{0}
	b1 := NewGenesisBlock(stateHash)

	if b1.ParentRoot == nil {
		t.Error("genesis block missing ParentHash field")
	}

	if !bytes.Equal(b1.StateRoot, stateHash) {
		t.Error("genesis block StateRootHash32 isn't initialized correctly")
	}
}
