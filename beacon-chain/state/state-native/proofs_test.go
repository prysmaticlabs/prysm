package state_native_test

import (
	"context"
	"testing"

	statenative "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestBeaconStateMerkleProofs_phase0_notsupported(t *testing.T) {
	ctx := context.Background()
	st, _ := util.DeterministicGenesisState(t, 256)
	t.Run("current sync committee", func(t *testing.T) {
		_, err := st.CurrentSyncCommitteeProof(ctx)
		require.ErrorContains(t, "not supported", err)
	})
	t.Run("next sync committee", func(t *testing.T) {
		_, err := st.NextSyncCommitteeProof(ctx)
		require.ErrorContains(t, "not supported", err)
	})
	t.Run("finalized root", func(t *testing.T) {
		_, err := st.FinalizedRootProof(ctx)
		require.ErrorContains(t, "not supported", err)
	})
}

func TestBeaconStateMerkleProofs_bellatrix(t *testing.T) {
	ctx := context.Background()
	bellatrix, err := util.NewBeaconStateBellatrix()
	require.NoError(t, err)
	htr, err := bellatrix.HashTreeRoot(ctx)
	require.NoError(t, err)
	t.Run("current sync committee", func(t *testing.T) {
		cscp, err := bellatrix.CurrentSyncCommitteeProof(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, cscp)

	})
	t.Run("next sync committee", func(t *testing.T) {
		nscp, err := bellatrix.NextSyncCommitteeProof(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, nscp)
	})
	t.Run("finalized root", func(t *testing.T) {
		finalizedRoot := bellatrix.FinalizedCheckpoint().Root
		proof, err := bellatrix.FinalizedRootProof(ctx)
		require.NoError(t, err)
		gIndex := statenative.FinalizedRootGeneralizedIndex()
		valid := trie.VerifyMerkleProof(htr[:], finalizedRoot, gIndex, proof)
		require.Equal(t, true, valid)
	})
	//t.Run("recomputes root on dirty fields", func(t *testing.T) {
	//	currentRoot, err := st.HashTreeRoot(ctx)
	//	require.NoError(t, err)
	//	cpt := st.FinalizedCheckpoint()
	//	require.NoError(t, err)
	//
	//	// Edit the checkpoint.
	//	cpt.Epoch = 100
	//	require.NoError(t, st.SetFinalizedCheckpoint(cpt))
	//
	//	// Produce a proof for the finalized root.
	//	proof, err := st.FinalizedRootProof(ctx)
	//	require.NoError(t, err)
	//
	//	// We expect the previous step to have triggered
	//	// a recomputation of dirty fields in the beacon state, resulting
	//	// in a new hash tree root as the finalized checkpoint had previously
	//	// changed and should have been marked as a dirty state field.
	//	// The proof validity should be false for the old root, but true for the new.
	//	finalizedRoot := st.FinalizedCheckpoint().Root
	//	gIndex := statenative.FinalizedRootGeneralizedIndex()
	//	valid := trie.VerifyMerkleProof(currentRoot[:], finalizedRoot, gIndex, proof)
	//	require.Equal(t, false, valid)
	//
	//	newRoot, err := st.HashTreeRoot(ctx)
	//	require.NoError(t, err)
	//
	//	valid = trie.VerifyMerkleProof(newRoot[:], finalizedRoot, gIndex, proof)
	//	require.Equal(t, true, valid)
	//})
}
