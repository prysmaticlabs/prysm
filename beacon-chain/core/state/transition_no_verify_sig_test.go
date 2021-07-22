package state_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestExecuteStateTransitionNoVerify_FullProcess(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	eth1Data := &ethpb.Eth1Data{
		DepositCount: 100,
		DepositRoot:  bytesutil.PadTo([]byte{2}, 32),
		BlockHash:    make([]byte, 32),
	}
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch-1))
	e := beaconState.Eth1Data()
	e.DepositCount = 100
	require.NoError(t, beaconState.SetEth1Data(e))
	bh := beaconState.LatestBlockHeader()
	bh.Slot = beaconState.Slot()
	require.NoError(t, beaconState.SetLatestBlockHeader(bh))
	require.NoError(t, beaconState.SetEth1DataVotes([]*ethpb.Eth1Data{eth1Data}))

	require.NoError(t, beaconState.SetSlot(beaconState.Slot()+1))
	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := testutil.RandaoReveal(beaconState, epoch, privKeys)
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()-1))

	nextSlotState, err := state.ProcessSlots(context.Background(), beaconState.Copy(), beaconState.Slot()+1)
	require.NoError(t, err)
	parentRoot, err := nextSlotState.LatestBlockHeader().HashTreeRoot()
	require.NoError(t, err)
	proposerIdx, err := helpers.BeaconProposerIndex(nextSlotState)
	require.NoError(t, err)
	block := testutil.NewBeaconBlock()
	block.Block.ProposerIndex = proposerIdx
	block.Block.Slot = beaconState.Slot() + 1
	block.Block.ParentRoot = parentRoot[:]
	block.Block.Body.RandaoReveal = randaoReveal
	block.Block.Body.Eth1Data = eth1Data

	stateRoot, err := state.CalculateStateRoot(context.Background(), beaconState, wrapper.WrappedPhase0SignedBeaconBlock(block))
	require.NoError(t, err)

	block.Block.StateRoot = stateRoot[:]

	sig, err := testutil.BlockSignature(beaconState, block.Block, privKeys)
	require.NoError(t, err)
	block.Signature = sig.Marshal()

	set, _, err := state.ExecuteStateTransitionNoVerifyAnySig(context.Background(), beaconState, wrapper.WrappedPhase0SignedBeaconBlock(block))
	assert.NoError(t, err)
	verified, err := set.Verify()
	assert.NoError(t, err)
	assert.Equal(t, true, verified, "Could not verify signature set")
}

func TestExecuteStateTransitionNoVerifySignature_CouldNotVerifyStateRoot(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	eth1Data := &ethpb.Eth1Data{
		DepositCount: 100,
		DepositRoot:  bytesutil.PadTo([]byte{2}, 32),
		BlockHash:    make([]byte, 32),
	}
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch-1))
	e := beaconState.Eth1Data()
	e.DepositCount = 100
	require.NoError(t, beaconState.SetEth1Data(e))
	bh := beaconState.LatestBlockHeader()
	bh.Slot = beaconState.Slot()
	require.NoError(t, beaconState.SetLatestBlockHeader(bh))
	require.NoError(t, beaconState.SetEth1DataVotes([]*ethpb.Eth1Data{eth1Data}))

	require.NoError(t, beaconState.SetSlot(beaconState.Slot()+1))
	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := testutil.RandaoReveal(beaconState, epoch, privKeys)
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()-1))

	nextSlotState, err := state.ProcessSlots(context.Background(), beaconState.Copy(), beaconState.Slot()+1)
	require.NoError(t, err)
	parentRoot, err := nextSlotState.LatestBlockHeader().HashTreeRoot()
	require.NoError(t, err)
	proposerIdx, err := helpers.BeaconProposerIndex(nextSlotState)
	require.NoError(t, err)
	block := testutil.NewBeaconBlock()
	block.Block.ProposerIndex = proposerIdx
	block.Block.Slot = beaconState.Slot() + 1
	block.Block.ParentRoot = parentRoot[:]
	block.Block.Body.RandaoReveal = randaoReveal
	block.Block.Body.Eth1Data = eth1Data

	stateRoot, err := state.CalculateStateRoot(context.Background(), beaconState, wrapper.WrappedPhase0SignedBeaconBlock(block))
	require.NoError(t, err)

	block.Block.StateRoot = stateRoot[:]

	sig, err := testutil.BlockSignature(beaconState, block.Block, privKeys)
	require.NoError(t, err)
	block.Signature = sig.Marshal()

	block.Block.StateRoot = bytesutil.PadTo([]byte{'a'}, 32)
	_, _, err = state.ExecuteStateTransitionNoVerifyAnySig(context.Background(), beaconState, wrapper.WrappedPhase0SignedBeaconBlock(block))
	require.ErrorContains(t, "could not validate state root", err)
}

func TestProcessBlockNoVerify_PassesProcessingConditions(t *testing.T) {
	beaconState, block, _, _, _ := createFullBlockWithOperations(t)
	set, _, err := state.ProcessBlockNoVerifyAnySig(context.Background(), beaconState, wrapper.WrappedPhase0SignedBeaconBlock(block))
	require.NoError(t, err)
	// Test Signature set verifies.
	verified, err := set.Verify()
	require.NoError(t, err)
	assert.Equal(t, true, verified, "Could not verify signature set.")
}
