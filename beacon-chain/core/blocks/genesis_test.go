package blocks_test

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
)

func TestGenesisBlock_InitializedCorrectly(t *testing.T) {
	stateHash := []byte{0}
	b1 := blocks.NewGenesisBlock(stateHash)

	if b1.Block.ParentRoot == nil {
		t.Error("genesis block missing ParentHash field")
	}

	if !bytes.Equal(b1.Block.StateRoot, stateHash) {
		t.Error("genesis block StateRootHash32 isn't initialized correctly")
	}
}
