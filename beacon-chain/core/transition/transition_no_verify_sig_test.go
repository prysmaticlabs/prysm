package transition_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestExecuteStateTransitionNoVerify_FullProcess(t *testing.T) {
	beaconState, privKeys := util.DeterministicGenesisState(t, 100)

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
	epoch := time.CurrentEpoch(beaconState)
	randaoReveal, err := util.RandaoReveal(beaconState, epoch, privKeys)
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()-1))

	nextSlotState, err := transition.ProcessSlots(context.Background(), beaconState.Copy(), beaconState.Slot()+1)
	require.NoError(t, err)
	parentRoot, err := nextSlotState.LatestBlockHeader().HashTreeRoot()
	require.NoError(t, err)
	proposerIdx, err := helpers.BeaconProposerIndex(context.Background(), nextSlotState)
	require.NoError(t, err)
	block := util.NewBeaconBlock()
	block.Block.ProposerIndex = proposerIdx
	block.Block.Slot = beaconState.Slot() + 1
	block.Block.ParentRoot = parentRoot[:]
	block.Block.Body.RandaoReveal = randaoReveal
	block.Block.Body.Eth1Data = eth1Data

	stateRoot, err := transition.CalculateStateRoot(context.Background(), beaconState, wrapper.WrappedPhase0SignedBeaconBlock(block))
	require.NoError(t, err)

	block.Block.StateRoot = stateRoot[:]

	sig, err := util.BlockSignature(beaconState, block.Block, privKeys)
	require.NoError(t, err)
	block.Signature = sig.Marshal()

	set, _, err := transition.ExecuteStateTransitionNoVerifyAnySig(context.Background(), beaconState, wrapper.WrappedPhase0SignedBeaconBlock(block))
	assert.NoError(t, err)
	verified, err := set.Verify()
	assert.NoError(t, err)
	assert.Equal(t, true, verified, "Could not verify signature set")
}

func TestExecuteStateTransitionNoVerifySignature_CouldNotVerifyStateRoot(t *testing.T) {
	beaconState, privKeys := util.DeterministicGenesisState(t, 100)

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
	epoch := time.CurrentEpoch(beaconState)
	randaoReveal, err := util.RandaoReveal(beaconState, epoch, privKeys)
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()-1))

	nextSlotState, err := transition.ProcessSlots(context.Background(), beaconState.Copy(), beaconState.Slot()+1)
	require.NoError(t, err)
	parentRoot, err := nextSlotState.LatestBlockHeader().HashTreeRoot()
	require.NoError(t, err)
	proposerIdx, err := helpers.BeaconProposerIndex(context.Background(), nextSlotState)
	require.NoError(t, err)
	block := util.NewBeaconBlock()
	block.Block.ProposerIndex = proposerIdx
	block.Block.Slot = beaconState.Slot() + 1
	block.Block.ParentRoot = parentRoot[:]
	block.Block.Body.RandaoReveal = randaoReveal
	block.Block.Body.Eth1Data = eth1Data

	stateRoot, err := transition.CalculateStateRoot(context.Background(), beaconState, wrapper.WrappedPhase0SignedBeaconBlock(block))
	require.NoError(t, err)

	block.Block.StateRoot = stateRoot[:]

	sig, err := util.BlockSignature(beaconState, block.Block, privKeys)
	require.NoError(t, err)
	block.Signature = sig.Marshal()

	block.Block.StateRoot = bytesutil.PadTo([]byte{'a'}, 32)
	_, _, err = transition.ExecuteStateTransitionNoVerifyAnySig(context.Background(), beaconState, wrapper.WrappedPhase0SignedBeaconBlock(block))
	require.ErrorContains(t, "could not validate state root", err)
}

func TestProcessBlockNoVerify_PassesProcessingConditions(t *testing.T) {
	beaconState, block, _, _, _ := createFullBlockWithOperations(t)
	set, _, err := transition.ProcessBlockNoVerifyAnySig(context.Background(), beaconState, wrapper.WrappedPhase0SignedBeaconBlock(block))
	require.NoError(t, err)
	// Test Signature set verifies.
	verified, err := set.Verify()
	require.NoError(t, err)
	assert.Equal(t, true, verified, "Could not verify signature set.")
}

func TestProcessBlockNoVerifyAnySigAltair_OK(t *testing.T) {
	beaconState, block := createFullAltairBlockWithOperations(t)
	wsb, err := wrapper.WrappedAltairSignedBeaconBlock(block)
	require.NoError(t, err)
	beaconState, err = transition.ProcessSlots(context.Background(), beaconState, wsb.Block().Slot())
	require.NoError(t, err)
	set, _, err := transition.ProcessBlockNoVerifyAnySig(context.Background(), beaconState, wsb)
	require.NoError(t, err)
	verified, err := set.Verify()
	require.NoError(t, err)
	require.Equal(t, true, verified, "Could not verify signature set")
}

func TestProcessOperationsNoVerifyAttsSigs_OK(t *testing.T) {
	beaconState, block := createFullAltairBlockWithOperations(t)
	wsb, err := wrapper.WrappedAltairSignedBeaconBlock(block)
	require.NoError(t, err)
	beaconState, err = transition.ProcessSlots(context.Background(), beaconState, wsb.Block().Slot())
	require.NoError(t, err)
	_, err = transition.ProcessOperationsNoVerifyAttsSigs(context.Background(), beaconState, wsb)
	require.NoError(t, err)
}

func TestCalculateStateRootAltair_OK(t *testing.T) {
	beaconState, block := createFullAltairBlockWithOperations(t)
	wsb, err := wrapper.WrappedAltairSignedBeaconBlock(block)
	require.NoError(t, err)
	r, err := transition.CalculateStateRoot(context.Background(), beaconState, wsb)
	require.NoError(t, err)
	require.DeepNotEqual(t, params.BeaconConfig().ZeroHash, r)
}

func TestProcessBlockDifferentVersion(t *testing.T) {
	beaconState, _ := util.DeterministicGenesisState(t, 64) // Phase 0 state
	_, block := createFullAltairBlockWithOperations(t)
	wsb, err := wrapper.WrappedAltairSignedBeaconBlock(block) // Altair block
	require.NoError(t, err)
	_, _, err = transition.ProcessBlockNoVerifyAnySig(context.Background(), beaconState, wsb)
	require.ErrorContains(t, "state and block are different version. 0 != 1", err)
}
