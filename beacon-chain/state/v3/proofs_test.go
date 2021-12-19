package v3_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

const (
	// Precomputed generalized index in the Merkle trie for the Merge.
	finalizedRootIndex = 105
)

func TestProve_BeaconState_FinalizedRoot(t *testing.T) {
	ctx := context.Background()
	st, _ := util.DeterministicGenesisStateMerge(t, 256)
	htr, err := st.HashTreeRoot(ctx)
	require.NoError(t, err)
	finalizedRoot := st.FinalizedCheckpoint().Root
	proof, err := st.ProveFinalizedRoot()
	require.NoError(t, err)
	valid := trie.VerifyMerkleProof(htr[:], finalizedRoot, finalizedRootIndex, proof)
	require.Equal(t, true, valid)
}
