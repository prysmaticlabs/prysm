package precompute_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestUpdateValidator(t *testing.T) {
	e := params.BeaconConfig().FarFutureEpoch
	vp := []*precompute.Validator{{}, {InclusionSlot: e}, {}, {InclusionSlot: e}, {}, {InclusionSlot: e}}
	record := &precompute.Validator{IsCurrentEpochAttester: true, IsCurrentEpochTargetAttester: true,
		IsPrevEpochAttester: true, IsPrevEpochTargetAttester: true, IsPrevEpochHeadAttester: true}
	a := &pb.PendingAttestation{InclusionDelay: 1, ProposerIndex: 2}

	// Indices 1 3 and 5 attested
	vp = precompute.UpdateValidator(vp, record, []uint64{1, 3, 5}, a, 100)

	wanted := &precompute.Validator{IsCurrentEpochAttester: true, IsCurrentEpochTargetAttester: true,
		IsPrevEpochAttester: true, IsPrevEpochTargetAttester: true, IsPrevEpochHeadAttester: true,
		ProposerIndex: 2, InclusionDistance: 1, InclusionSlot: 101}
	wantedVp := []*precompute.Validator{{}, wanted, {}, wanted, {}, wanted}
	if !reflect.DeepEqual(vp, wantedVp) {
		t.Error("Incorrect attesting validator calculations")
	}
}

func TestUpdateBalance(t *testing.T) {
	vp := []*precompute.Validator{
		{IsCurrentEpochAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsCurrentEpochTargetAttester: true, IsCurrentEpochAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsCurrentEpochTargetAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsPrevEpochAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsPrevEpochAttester: true, IsPrevEpochTargetAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsPrevEpochHeadAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsPrevEpochAttester: true, IsPrevEpochHeadAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsSlashed: true, IsCurrentEpochAttester: true, CurrentEpochEffectiveBalance: 100},
	}
	wantedBp := &precompute.Balance{
		CurrentEpochAttesters:       200,
		CurrentEpochTargetAttesters: 200,
		PrevEpochAttesters:          300,
		PrevEpochTargetAttesters:    100,
		PrevEpochHeadAttesters:      200,
	}
	bp := precompute.UpdateBalance(vp, &precompute.Balance{})
	if !reflect.DeepEqual(bp, wantedBp) {
		t.Error("Incorrect balance calculations")
	}
}

func TestSameHead(t *testing.T) {
	helpers.ClearAllCaches()
	deposits, _, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot = 1
	att := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target:    &ethpb.Checkpoint{Epoch: 0},
		Crosslink: &ethpb.Crosslink{Shard: 0}}}
	attSlot, err := helpers.AttestationDataSlot(beaconState, att.Data)
	if err != nil {
		t.Fatal(err)
	}
	r := []byte{'A'}
	beaconState.BlockRoots[attSlot] = r
	att.Data.BeaconBlockRoot = r
	same, err := precompute.SameHead(beaconState, &pb.PendingAttestation{Data: att.Data})
	if err != nil {
		t.Fatal(err)
	}
	if !same {
		t.Error("head in state does not match head in attestation")
	}
	att.Data.BeaconBlockRoot = []byte{'B'}
	same, err = precompute.SameHead(beaconState, &pb.PendingAttestation{Data: att.Data})
	if err != nil {
		t.Fatal(err)
	}
	if same {
		t.Error("head in state matches head in attestation")
	}
}

func TestSameTarget(t *testing.T) {
	deposits, _, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot = 1
	att := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target:    &ethpb.Checkpoint{Epoch: 0},
		Crosslink: &ethpb.Crosslink{Shard: 0}}}
	attSlot, err := helpers.AttestationDataSlot(beaconState, att.Data)
	if err != nil {
		t.Fatal(err)
	}
	r := []byte{'A'}
	beaconState.BlockRoots[attSlot] = r
	att.Data.Target.Root = r
	same, err := precompute.SameTarget(beaconState, &pb.PendingAttestation{Data: att.Data}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !same {
		t.Error("head in state does not match head in attestation")
	}
	att.Data.Target.Root = []byte{'B'}
	same, err = precompute.SameTarget(beaconState, &pb.PendingAttestation{Data: att.Data}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if same {
		t.Error("head in state matches head in attestation")
	}
}

func TestAttestedPrevEpoch(t *testing.T) {
	deposits, _, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot = params.BeaconConfig().SlotsPerEpoch
	att := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target:    &ethpb.Checkpoint{Epoch: 0},
		Crosslink: &ethpb.Crosslink{Shard: 960}}}
	attSlot, err := helpers.AttestationDataSlot(beaconState, att.Data)
	if err != nil {
		t.Fatal(err)
	}
	r := []byte{'A'}
	beaconState.BlockRoots[attSlot] = r
	att.Data.Target.Root = r
	att.Data.BeaconBlockRoot = r
	votedEpoch, votedTarget, votedHead, err := precompute.AttestedPrevEpoch(beaconState, &pb.PendingAttestation{Data: att.Data})
	if err != nil {
		t.Fatal(err)
	}
	if !votedEpoch {
		t.Error("did not vote epoch")
	}
	if !votedTarget {
		t.Error("did not vote target")
	}
	if !votedHead {
		t.Error("did not vote head")
	}
}

func TestAttestedCurrentEpoch(t *testing.T) {
	deposits, _, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot = params.BeaconConfig().SlotsPerEpoch + 1
	att := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target:    &ethpb.Checkpoint{Epoch: 1},
		Crosslink: &ethpb.Crosslink{}}}
	attSlot, err := helpers.AttestationDataSlot(beaconState, att.Data)
	if err != nil {
		t.Fatal(err)
	}
	r := []byte{'A'}
	beaconState.BlockRoots[attSlot] = r
	att.Data.Target.Root = r
	att.Data.BeaconBlockRoot = r
	votedEpoch, votedTarget, err := precompute.AttestedCurrentEpoch(beaconState, &pb.PendingAttestation{Data: att.Data})
	if err != nil {
		t.Fatal(err)
	}
	if !votedEpoch {
		t.Error("did not vote epoch")
	}
	if !votedTarget {
		t.Error("did not vote target")
	}
}

func TestProcessAttestations(t *testing.T) {
	helpers.ClearAllCaches()

	params.UseMinimalConfig()
	defer params.UseMainnetConfig()

	validators := uint64(64)
	deposits, _, _ := testutil.SetupInitialDeposits(t, validators)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot = params.BeaconConfig().SlotsPerEpoch

	bf := []byte{0xff}
	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target:    &ethpb.Checkpoint{Epoch: 0},
		Crosslink: &ethpb.Crosslink{Shard: 960}}, AggregationBits: bf}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target:    &ethpb.Checkpoint{Epoch: 0},
		Crosslink: &ethpb.Crosslink{Shard: 961}}, AggregationBits: bf}
	beaconState.BlockRoots[0] = []byte{'A'}
	att1.Data.Target.Root = []byte{'A'}
	att1.Data.BeaconBlockRoot = []byte{'A'}
	beaconState.BlockRoots[0] = []byte{'B'}
	att2.Data.Target.Root = []byte{'A'}
	att2.Data.BeaconBlockRoot = []byte{'B'}
	beaconState.PreviousEpochAttestations = []*pb.PendingAttestation{{Data: att1.Data, AggregationBits: bf}}
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{{Data: att2.Data, AggregationBits: bf}}

	vp := make([]*precompute.Validator, validators)
	for i := 0; i < len(vp); i++ {
		vp[i] = &precompute.Validator{CurrentEpochEffectiveBalance: 100}
	}
	bp := &precompute.Balance{}
	vp, bp, err = precompute.ProcessAttestations(context.Background(), beaconState, vp, bp)
	if err != nil {
		t.Fatal(err)
	}
	indices, _ := helpers.AttestingIndices(beaconState, att1.Data, att1.AggregationBits)
	for _, i := range indices {
		if !vp[i].IsPrevEpochAttester {
			t.Error("Not a prev epoch attester")
		}
	}
	indices, _ = helpers.AttestingIndices(beaconState, att2.Data, att2.AggregationBits)
	for _, i := range indices {
		if !vp[i].IsPrevEpochAttester {
			t.Error("Not a prev epoch attester")
		}
		if !vp[i].IsPrevEpochHeadAttester {
			t.Error("Not a prev epoch head attester")
		}
	}
}
