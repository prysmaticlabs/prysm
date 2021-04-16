package altair_test

import (
	"context"
	"testing"

	fuzz "github.com/google/gofuzz"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	statealtair "github.com/prysmaticlabs/prysm/beacon-chain/state/state-altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateV0"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	altair2 "github.com/prysmaticlabs/prysm/shared/testutil/altair"
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
	s, err := statealtair.InitializeFromProto(base)
	require.NoError(t, err)
	newState, err := altair.ProcessEpoch(context.Background(), s)
	require.NoError(t, err)
	require.Equal(t, uint64(0), newState.Slashings()[2], "Unexpected slashed balance")
}

func TestFuzzProcessEpoch_1000(t *testing.T) {
	ctx := context.Background()
	state := &stateV0.BeaconState{}
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		s, err := altair.ProcessEpoch(ctx, state)
		if err != nil && s != nil {
			t.Fatalf("state should be nil on err. found: %v on error: %v for state: %v", s, err, state)
		}
	}
}

func TestProcessSlots_CanProcess(t *testing.T) {
	s, _ := altair2.DeterministicGenesisStateAltair(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	slot := types.Slot(100)
	newState, err := altair.ProcessSlots(context.Background(), s, slot)
	require.NoError(t, err)
	require.Equal(t, slot, newState.Slot())
}

func TestProcessSlots_SameSlotAsParentState(t *testing.T) {
	slot := types.Slot(2)
	parentState, err := stateV0.InitializeFromProto(&pb.BeaconState{Slot: slot})
	require.NoError(t, err)

	_, err = altair.ProcessSlots(context.Background(), parentState, slot)
	require.ErrorContains(t, "expected state.slot 2 < slot 2", err)
}

func TestProcessSlots_LowerSlotAsParentState(t *testing.T) {
	slot := types.Slot(2)
	parentState, err := stateV0.InitializeFromProto(&pb.BeaconState{Slot: slot})
	require.NoError(t, err)

	_, err = altair.ProcessSlots(context.Background(), parentState, slot-1)
	require.ErrorContains(t, "expected state.slot 2 < slot 1", err)
}

func TestFuzzProcessSlots_1000(t *testing.T) {
	altair.SkipSlotCache.Disable()
	defer altair.SkipSlotCache.Enable()
	ctx := context.Background()
	state := &statealtair.BeaconState{}
	slot := types.Slot(0)
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(&slot)
		s, err := altair.ProcessSlots(ctx, state, slot)
		if err != nil && s != nil {
			t.Fatalf("state should be nil on err. found: %v on error: %v for state: %v", s, err, state)
		}
	}
}
