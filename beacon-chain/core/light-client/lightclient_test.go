package light_client_test

import (
	"testing"

	lightClient "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/light-client"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	v2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestLightClient_NewLightClientOptimisticUpdateFromBeaconStateCapella(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestCapella()

	update, err := lightClient.NewLightClientOptimisticUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update)
	l.CheckAttestedHeader(update)

	require.Equal(t, (*v2.LightClientHeaderContainer)(nil), update.FinalizedHeader, "Finalized header is not nil")
	require.DeepSSZEqual(t, ([][]byte)(nil), update.FinalityBranch, "Finality branch is not nil")
}

func TestLightClient_NewLightClientOptimisticUpdateFromBeaconStateAltair(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestAltair()

	update, err := lightClient.NewLightClientOptimisticUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update)
	l.CheckAttestedHeader(update)

	require.Equal(t, (*v2.LightClientHeaderContainer)(nil), update.FinalizedHeader, "Finalized header is not nil")
	require.DeepSSZEqual(t, ([][]byte)(nil), update.FinalityBranch, "Finality branch is not nil")
}

func TestLightClient_NewLightClientOptimisticUpdateFromBeaconStateDeneb(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestDeneb()

	update, err := lightClient.NewLightClientOptimisticUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update)
	l.CheckAttestedHeader(update)

	require.Equal(t, (*v2.LightClientHeaderContainer)(nil), update.FinalizedHeader, "Finalized header is not nil")
	require.DeepSSZEqual(t, ([][]byte)(nil), update.FinalityBranch, "Finality branch is not nil")
}
func TestLightClient_NewLightClientFinalityUpdateFromBeaconStateCapella(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestCapella()
	update, err := lightClient.NewLightClientFinalityUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, nil)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update)
	l.CheckAttestedHeader(update)

	zeroHash := params.BeaconConfig().ZeroHash[:]
	require.NotNil(t, update.FinalizedHeader, "Finalized header is nil")
	updateFinalizedHeaderBeacon, err := update.FinalizedHeader.GetBeacon()
	require.NoError(t, err)
	require.Equal(t, primitives.Slot(0), updateFinalizedHeaderBeacon.Slot, "Finalized header slot is not zero")
	require.Equal(t, primitives.ValidatorIndex(0), updateFinalizedHeaderBeacon.ProposerIndex, "Finalized header proposer index is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.ParentRoot, "Finalized header parent root is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.StateRoot, "Finalized header state root is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.BodyRoot, "Finalized header body root is not zero")
	require.Equal(t, lightClient.FinalityBranchNumOfLeaves, len(update.FinalityBranch), "Invalid finality branch leaves")
	for _, leaf := range update.FinalityBranch {
		require.DeepSSZEqual(t, zeroHash, leaf, "Leaf is not zero")
	}
}

func TestLightClient_NewLightClientFinalityUpdateFromBeaconStateAltair(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestAltair()

	update, err := lightClient.NewLightClientFinalityUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, nil)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update)
	l.CheckAttestedHeader(update)

	zeroHash := params.BeaconConfig().ZeroHash[:]
	require.NotNil(t, update.FinalizedHeader, "Finalized header is nil")
	updateFinalizedHeaderBeacon, err := update.FinalizedHeader.GetBeacon()
	require.NoError(t, err)
	require.Equal(t, primitives.Slot(0), updateFinalizedHeaderBeacon.Slot, "Finalized header slot is not zero")
	require.Equal(t, primitives.ValidatorIndex(0), updateFinalizedHeaderBeacon.ProposerIndex, "Finalized header proposer index is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.ParentRoot, "Finalized header parent root is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.StateRoot, "Finalized header state root is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.BodyRoot, "Finalized header body root is not zero")
	require.Equal(t, lightClient.FinalityBranchNumOfLeaves, len(update.FinalityBranch), "Invalid finality branch leaves")
	for _, leaf := range update.FinalityBranch {
		require.DeepSSZEqual(t, zeroHash, leaf, "Leaf is not zero")
	}
}

func TestLightClient_NewLightClientFinalityUpdateFromBeaconStateDeneb(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestDeneb()

	update, err := lightClient.NewLightClientFinalityUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, nil)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update)
	l.CheckAttestedHeader(update)

	zeroHash := params.BeaconConfig().ZeroHash[:]
	require.NotNil(t, update.FinalizedHeader, "Finalized header is nil")
	updateFinalizedHeaderBeacon, err := update.FinalizedHeader.GetBeacon()
	require.NoError(t, err)
	require.Equal(t, primitives.Slot(0), updateFinalizedHeaderBeacon.Slot, "Finalized header slot is not zero")
	require.Equal(t, primitives.ValidatorIndex(0), updateFinalizedHeaderBeacon.ProposerIndex, "Finalized header proposer index is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.ParentRoot, "Finalized header parent root is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.StateRoot, "Finalized header state root is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.BodyRoot, "Finalized header body root is not zero")
	require.DeepSSZEqual(t, zeroHash, update.FinalizedHeader.GetHeaderDeneb().Execution.BlockHash, "Execution BlockHash is not zero")
	require.Equal(t, lightClient.FinalityBranchNumOfLeaves, len(update.FinalityBranch), "Invalid finality branch leaves")
	for _, leaf := range update.FinalityBranch {
		require.DeepSSZEqual(t, zeroHash, leaf, "Leaf is not zero")
	}
}

// TODO - add finality update tests with non-nil finalized block for different versions
