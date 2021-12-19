package v3_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestProve_BeaconState_FinalizedRoot(t *testing.T) {
	ctx := context.Background()
	st, _ := util.DeterministicGenesisStateMerge(t, 256)
	htr, err := st.HashTreeRoot(ctx)
	require.NoError(t, err)
	finalizedRoot := st.FinalizedCheckpoint().Root
	proof, err := st.ProveFinalizedRoot()
	require.NoError(t, err)
	gIndex, err := st.FinalizedRootGeneralizedIndex()
	require.NoError(t, err)
	valid := trie.VerifyMerkleProof(htr[:], finalizedRoot, gIndex, proof)
	require.Equal(t, true, valid)
}
