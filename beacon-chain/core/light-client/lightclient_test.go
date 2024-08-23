package light_client_test

import (
	"testing"

	lightClient "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/light-client"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"

	v1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
)

func TestLightClient_NewLightClientOptimisticUpdateFromBeaconState(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTest()

	update, err := lightClient.NewLightClientOptimisticUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update)
	l.CheckAttestedHeader(update)

	require.Equal(t, (*v1.BeaconBlockHeader)(nil), update.FinalizedHeader, "Finalized header is not nil")
	require.DeepSSZEqual(t, ([][]byte)(nil), update.FinalityBranch, "Finality branch is not nil")
}

func TestLightClient_NewLightClientFinalityUpdateFromBeaconState(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTest()

	update, err := lightClient.NewLightClientFinalityUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, nil)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update)
	l.CheckAttestedHeader(update)

	zeroHash := params.BeaconConfig().ZeroHash[:]
	require.NotNil(t, update.FinalizedHeader, "Finalized header is nil")
	require.Equal(t, primitives.Slot(0), update.FinalizedHeader.Slot, "Finalized header slot is not zero")
	require.Equal(t, primitives.ValidatorIndex(0), update.FinalizedHeader.ProposerIndex, "Finalized header proposer index is not zero")
	require.DeepSSZEqual(t, zeroHash, update.FinalizedHeader.ParentRoot, "Finalized header parent root is not zero")
	require.DeepSSZEqual(t, zeroHash, update.FinalizedHeader.StateRoot, "Finalized header state root is not zero")
	require.DeepSSZEqual(t, zeroHash, update.FinalizedHeader.BodyRoot, "Finalized header body root is not zero")
	require.Equal(t, lightClient.FinalityBranchNumOfLeaves, len(update.FinalityBranch), "Invalid finality branch leaves")
	for _, leaf := range update.FinalityBranch {
		require.DeepSSZEqual(t, zeroHash, leaf, "Leaf is not zero")
	}
}
