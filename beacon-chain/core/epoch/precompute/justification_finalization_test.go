package precompute_test

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProcessJustificationAndFinalizationPreCompute_ConsecutiveEpochs(t *testing.T) {
	e := params.BeaconConfig().FarFutureEpoch
	a := params.BeaconConfig().MaxEffectiveBalance
	blockRoots := make([][]byte, params.BeaconConfig().SlotsPerEpoch*2+1)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i)}
	}
	state := &pb.BeaconState{
		Slot: params.BeaconConfig().SlotsPerEpoch*2 + 1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
		JustificationBits:   bitfield.Bitvector4{0x0F}, // 0b1111
		Validators:          []*ethpb.Validator{{ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}},
		Balances:            []uint64{a, a, a, a}, // validator total balance should be 128000000000
		BlockRoots:          blockRoots,
	}
	attestedBalance := 4 * e * 3 / 2
	b := &precompute.Balance{PrevEpochTargetAttesters: attestedBalance}
	newState, err := precompute.ProcessJustificationAndFinalizationPreCompute(state, b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(newState.CurrentJustifiedCheckpoint.Root, []byte{byte(64)}) {
		t.Errorf("Wanted current justified root: %v, got: %v",
			[]byte{byte(64)}, newState.CurrentJustifiedCheckpoint.Root)
	}
	if newState.CurrentJustifiedCheckpoint.Epoch != 2 {
		t.Errorf("Wanted justified epoch: %d, got: %d",
			2, newState.CurrentJustifiedCheckpoint.Epoch)
	}
	if newState.PreviousJustifiedCheckpoint.Epoch != 0 {
		t.Errorf("Wanted previous justified epoch: %d, got: %d",
			0, newState.PreviousJustifiedCheckpoint.Epoch)
	}
	if !bytes.Equal(newState.FinalizedCheckpoint.Root, params.BeaconConfig().ZeroHash[:]) {
		t.Errorf("Wanted current finalized root: %v, got: %v",
			params.BeaconConfig().ZeroHash, newState.FinalizedCheckpoint.Root)
	}
	if newState.FinalizedCheckpoint.Epoch != 0 {
		t.Errorf("Wanted finalized epoch: 0, got: %d", newState.FinalizedCheckpoint.Epoch)
	}
}

func TestProcessJustificationAndFinalizationPreCompute_JustifyCurrentEpoch(t *testing.T) {
	e := params.BeaconConfig().FarFutureEpoch
	a := params.BeaconConfig().MaxEffectiveBalance
	blockRoots := make([][]byte, params.BeaconConfig().SlotsPerEpoch*2+1)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i)}
	}
	state := &pb.BeaconState{
		Slot: params.BeaconConfig().SlotsPerEpoch*2 + 1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
		JustificationBits:   bitfield.Bitvector4{0x03}, // 0b0011
		Validators:          []*ethpb.Validator{{ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}},
		Balances:            []uint64{a, a, a, a}, // validator total balance should be 128000000000
		BlockRoots:          blockRoots,
	}
	attestedBalance := 4 * e * 3 / 2
	b := &precompute.Balance{PrevEpochTargetAttesters: attestedBalance}
	newState, err := precompute.ProcessJustificationAndFinalizationPreCompute(state, b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(newState.CurrentJustifiedCheckpoint.Root, []byte{byte(64)}) {
		t.Errorf("Wanted current justified root: %v, got: %v",
			[]byte{byte(64)}, newState.CurrentJustifiedCheckpoint.Root)
	}
	if newState.CurrentJustifiedCheckpoint.Epoch != 2 {
		t.Errorf("Wanted justified epoch: %d, got: %d",
			2, newState.CurrentJustifiedCheckpoint.Epoch)
	}
	if newState.PreviousJustifiedCheckpoint.Epoch != 0 {
		t.Errorf("Wanted previous justified epoch: %d, got: %d",
			0, newState.PreviousJustifiedCheckpoint.Epoch)
	}
	if !bytes.Equal(newState.FinalizedCheckpoint.Root, params.BeaconConfig().ZeroHash[:]) {
		t.Errorf("Wanted current finalized root: %v, got: %v",
			params.BeaconConfig().ZeroHash, newState.FinalizedCheckpoint.Root)
	}
	if newState.FinalizedCheckpoint.Epoch != 0 {
		t.Errorf("Wanted finalized epoch: 0, got: %d", newState.FinalizedCheckpoint.Epoch)
	}
}

func TestProcessJustificationAndFinalizationPreCompute_JustifyPrevEpoch(t *testing.T) {
	e := params.BeaconConfig().FarFutureEpoch
	a := params.BeaconConfig().MaxEffectiveBalance
	blockRoots := make([][]byte, params.BeaconConfig().SlotsPerEpoch*2+1)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i)}
	}
	state := &pb.BeaconState{
		Slot: params.BeaconConfig().SlotsPerEpoch*2 + 1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		JustificationBits: bitfield.Bitvector4{0x03}, // 0b0011
		Validators:        []*ethpb.Validator{{ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}},
		Balances:          []uint64{a, a, a, a}, // validator total balance should be 128000000000
		BlockRoots:        blockRoots, FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	attestedBalance := 4 * e * 3 / 2
	b := &precompute.Balance{PrevEpochTargetAttesters: attestedBalance}
	newState, err := precompute.ProcessJustificationAndFinalizationPreCompute(state, b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(newState.CurrentJustifiedCheckpoint.Root, []byte{byte(64)}) {
		t.Errorf("Wanted current justified root: %v, got: %v",
			[]byte{byte(64)}, newState.CurrentJustifiedCheckpoint.Root)
	}
	if newState.PreviousJustifiedCheckpoint.Epoch != 0 {
		t.Errorf("Wanted previous justified epoch: %d, got: %d",
			0, newState.PreviousJustifiedCheckpoint.Epoch)
	}
	if newState.CurrentJustifiedCheckpoint.Epoch != 2 {
		t.Errorf("Wanted justified epoch: %d, got: %d",
			2, newState.CurrentJustifiedCheckpoint.Epoch)
	}
	if !bytes.Equal(newState.FinalizedCheckpoint.Root, params.BeaconConfig().ZeroHash[:]) {
		t.Errorf("Wanted current finalized root: %v, got: %v",
			params.BeaconConfig().ZeroHash, newState.FinalizedCheckpoint.Root)
	}
	if newState.FinalizedCheckpoint.Epoch != 0 {
		t.Errorf("Wanted finalized epoch: 0, got: %d", newState.FinalizedCheckpoint.Epoch)
	}
}
