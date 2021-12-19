package v3_test

import (
	"context"
	"testing"

	v3 "github.com/prysmaticlabs/prysm/beacon-chain/state/v3"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestBeaconStateMerkleProofs(t *testing.T) {
	ctx := context.Background()
	st, _ := util.DeterministicGenesisStateMerge(t, 256)
	htr, err := st.HashTreeRoot(ctx)
	require.NoError(t, err)
	t.Run("current sync committee", func(t *testing.T) {
		sc, err := st.CurrentSyncCommittee()
		require.NoError(t, err)
		proof, err := st.CurrentSyncCommitteeProof()
		require.NoError(t, err)
		scRoot, err := sc.HashTreeRoot()
		require.NoError(t, err)
		valid := trie.VerifyMerkleProof(htr[:], scRoot[:], v3.CurrentSyncCommitteeGeneralizedIndex(), proof)
		require.Equal(t, true, valid)
	})
	t.Run("next sync committee", func(t *testing.T) {
		nextSC, err := st.NextSyncCommittee()
		require.NoError(t, err)
		proof, err := st.NextSyncCommitteeProof()
		require.NoError(t, err)
		nextSCRoot, err := nextSC.HashTreeRoot()
		require.NoError(t, err)
		valid := trie.VerifyMerkleProof(htr[:], nextSCRoot[:], v3.NextSyncCommitteeGeneralizedIndex(), proof)
		require.Equal(t, true, valid)
	})
	t.Run("finalized root", func(t *testing.T) {
		finalizedRoot := st.FinalizedCheckpoint().Root
		proof, err := st.FinalizedRootProof()
		require.NoError(t, err)
		gIndex := v3.FinalizedRootGeneralizedIndex()
		valid := trie.VerifyMerkleProof(htr[:], finalizedRoot, gIndex, proof)
		require.Equal(t, true, valid)
	})
}
