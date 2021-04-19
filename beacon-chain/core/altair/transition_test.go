package altair_test

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	stateAltair "github.com/prysmaticlabs/prysm/beacon-chain/state/state-altair"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	testutilAltair "github.com/prysmaticlabs/prysm/shared/testutil/altair"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProcessEpoch_CanProcess(t *testing.T) {
	epoch := types.Epoch(1)
	slashing := make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)
	base := &pb.BeaconStateAltair{
		Slot:                       params.BeaconConfig().SlotsPerEpoch.Mul(uint64(epoch)) + 1,
		BlockRoots:                 make([][]byte, 128),
		Slashings:                  slashing,
		RandaoMixes:                make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		FinalizedCheckpoint:        &ethpb.Checkpoint{Root: make([]byte, 32)},
		JustificationBits:          bitfield.Bitvector4{0x00},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, 32)},
		Validators:                 []*ethpb.Validator{},
	}
	s, err := stateAltair.InitializeFromProto(base)
	require.NoError(t, err)
	newState, err := altair.ProcessEpoch(context.Background(), s)
	require.NoError(t, err)
	require.Equal(t, uint64(0), newState.Slashings()[2], "Unexpected slashed balance")
}

func TestProcessSlots_CanProcess(t *testing.T) {
	s, _ := testutilAltair.DeterministicGenesisStateAltair(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	slot := types.Slot(100)
	newState, err := altair.ProcessSlots(context.Background(), s, slot)
	require.NoError(t, err)
	require.Equal(t, slot, newState.Slot())
}

func TestProcessSlots_CanProcessWithCache(t *testing.T) {
	s, _ := testutilAltair.DeterministicGenesisStateAltair(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	slot := types.Slot(100)
	copied := s.Copy()
	newState, err := altair.ProcessSlots(context.Background(), s, slot)
	require.NoError(t, err)
	require.Equal(t, slot, newState.Slot())

	newState, err = altair.ProcessSlots(context.Background(), copied, slot+100)
	require.NoError(t, err)
	require.Equal(t, slot+100, newState.Slot())

	// Cancel context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = altair.ProcessSlots(ctx, copied, slot+200)
	require.ErrorContains(t, "context canceled", err)
}

func TestProcessSlots_SameSlotAsParentState(t *testing.T) {
	slot := types.Slot(2)
	parentState, err := stateAltair.InitializeFromProto(&pb.BeaconStateAltair{Slot: slot})
	require.NoError(t, err)

	_, err = altair.ProcessSlots(context.Background(), parentState, slot)
	require.ErrorContains(t, "expected state.slot 2 < slot 2", err)
}

func TestProcessSlots_LowerSlotAsParentState(t *testing.T) {
	slot := types.Slot(2)
	parentState, err := stateAltair.InitializeFromProto(&pb.BeaconStateAltair{Slot: slot})
	require.NoError(t, err)

	_, err = altair.ProcessSlots(context.Background(), parentState, slot-1)
	require.ErrorContains(t, "expected state.slot 2 < slot 1", err)
}
