package v2_test

import (
	"context"
	"testing"

	v2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestProve_BeaconState_FinalizedRoot(t *testing.T) {
	ctx := context.Background()
	st, _ := util.DeterministicGenesisStateAltair(t, 256)
	htr, err := st.HashTreeRoot(ctx)
	require.NoError(t, err)
	finalizedRoot := st.FinalizedCheckpoint().Root
	proof, err := st.ProveFinalizedRoot()
	require.NoError(t, err)
	gIndex := v2.FinalizedRootGeneralizedIndex()
	valid := trie.VerifyMerkleProof(htr[:], finalizedRoot, gIndex, proof)
	require.Equal(t, true, valid)
}
