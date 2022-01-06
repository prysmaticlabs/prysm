package v1_test

import (
	"context"
	"testing"

	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state-proto/v1"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestBeaconStateMerkleProofs(t *testing.T) {
	ctx := context.Background()
	st, _ := util.DeterministicGenesisState(t, 256)
	htr, err := st.HashTreeRoot(ctx)
	require.NoError(t, err)
	t.Run("current sync committee", func(t *testing.T) {
		_, err := st.CurrentSyncCommitteeProof()
		require.ErrorContains(t, "unsupported", err)
	})
	t.Run("next sync committee", func(t *testing.T) {
		_, err := st.NextSyncCommitteeProof()
		require.ErrorContains(t, "unsupported", err)
	})
	t.Run("finalized root", func(t *testing.T) {
		finalizedRoot := st.FinalizedCheckpoint().Root
		proof, err := st.FinalizedRootProof()
		require.NoError(t, err)
		gIndex := v1.FinalizedRootGeneralizedIndex()
		valid := trie.VerifyMerkleProof(htr[:], finalizedRoot, gIndex, proof)
		require.Equal(t, true, valid)
	})
}
