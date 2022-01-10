package v3_test

import (
	"context"
	"testing"

	v3 "github.com/prysmaticlabs/prysm/beacon-chain/state-proto/v3"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestBeaconStateMerkleProofs(t *testing.T) {
	ctx := context.Background()
	st, _ := util.DeterministicGenesisStateAltair(t, 256)
	htr, err := st.HashTreeRoot(ctx)
	require.NoError(t, err)
	t.Run("current sync committee", func(t *testing.T) {
		sc, err := st.CurrentSyncCommittee()
		require.NoError(t, err)

		// Verify the Merkle proof.
		scRoot, err := sc.HashTreeRoot()
		require.NoError(t, err)
		proof, err := st.CurrentSyncCommitteeProof()
		require.NoError(t, err)
		valid := trie.VerifyMerkleProof(htr[:], scRoot[:], v3.CurrentSyncCommitteeGeneralizedIndex(), proof)
		require.Equal(t, true, valid)
	})
	t.Run("next sync committee", func(t *testing.T) {
		nextSC, err := st.NextSyncCommittee()
		require.NoError(t, err)
		proof, err := st.NextSyncCommitteeProof()
		require.NoError(t, err)

		// Verify the Merkle proof.
		nextSCRoot, err := nextSC.HashTreeRoot()
		require.NoError(t, err)
		valid := trie.VerifyMerkleProof(htr[:], nextSCRoot[:], v3.NextSyncCommitteeGeneralizedIndex(), proof)
		require.Equal(t, true, valid)

		// Edit the sync committee.
		privKey, err := bls.RandKey()
		require.NoError(t, err)
		nextSC.AggregatePubkey = privKey.PublicKey().Marshal()
		require.NoError(t, st.SetNextSyncCommittee(nextSC))

		// Verifying the old Merkle proof for the new value should fail.
		nextSCRoot, err = nextSC.HashTreeRoot()
		require.NoError(t, err)
		valid = trie.VerifyMerkleProof(htr[:], nextSCRoot[:], v3.NextSyncCommitteeGeneralizedIndex(), proof)
		require.Equal(t, false, valid)

		// Generating a new, valid proof should pass.
		proof, err = st.NextSyncCommitteeProof()
		require.NoError(t, err)
		htr, err = st.HashTreeRoot(ctx)
		require.NoError(t, err)
		valid = trie.VerifyMerkleProof(htr[:], nextSCRoot[:], v3.NextSyncCommitteeGeneralizedIndex(), proof)
		require.Equal(t, true, valid)
	})
	t.Run("finalized root", func(t *testing.T) {
		finalizedRoot := st.FinalizedCheckpoint().Root

		// Verify the Merkle proof.
		htr, err = st.HashTreeRoot(ctx)
		require.NoError(t, err)
		proof, err := st.FinalizedRootProof()
		require.NoError(t, err)
		gIndex := v3.FinalizedRootGeneralizedIndex()
		valid := trie.VerifyMerkleProof(htr[:], finalizedRoot, gIndex, proof)
		require.Equal(t, true, valid)
	})
}
