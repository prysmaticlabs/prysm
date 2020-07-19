package precompute_test

import (
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestUpdateValidator_Works(t *testing.T) {
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

func TestUpdateValidator_InclusionOnlyCountsPrevEpoch(t *testing.T) {
	e := params.BeaconConfig().FarFutureEpoch
	vp := []*precompute.Validator{{InclusionSlot: e}}
	record := &precompute.Validator{IsCurrentEpochAttester: true, IsCurrentEpochTargetAttester: true}
	a := &pb.PendingAttestation{InclusionDelay: 1, ProposerIndex: 2}

	// Verify inclusion info doesnt get updated.
	vp = precompute.UpdateValidator(vp, record, []uint64{0}, a, 100)
	wanted := &precompute.Validator{IsCurrentEpochAttester: true, IsCurrentEpochTargetAttester: true, InclusionSlot: e}
	wantedVp := []*precompute.Validator{wanted}
	if !reflect.DeepEqual(vp, wantedVp) {
		t.Error("Incorrect attesting validator calculations")
	}
}

func TestUpdateBalance(t *testing.T) {
	vp := []*precompute.Validator{
		{IsCurrentEpochAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsCurrentEpochTargetAttester: true, IsCurrentEpochAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsCurrentEpochTargetAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsPrevEpochAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsPrevEpochAttester: true, IsPrevEpochTargetAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsPrevEpochHeadAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsPrevEpochAttester: true, IsPrevEpochHeadAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsSlashed: true, IsCurrentEpochAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
	}
	wantedPBal := &precompute.Balance{
		ActiveCurrentEpoch:         params.BeaconConfig().EffectiveBalanceIncrement,
		ActivePrevEpoch:            params.BeaconConfig().EffectiveBalanceIncrement,
		CurrentEpochAttested:       200 * params.BeaconConfig().EffectiveBalanceIncrement,
		CurrentEpochTargetAttested: 200 * params.BeaconConfig().EffectiveBalanceIncrement,
		PrevEpochAttested:          300 * params.BeaconConfig().EffectiveBalanceIncrement,
		PrevEpochTargetAttested:    100 * params.BeaconConfig().EffectiveBalanceIncrement,
		PrevEpochHeadAttested:      200 * params.BeaconConfig().EffectiveBalanceIncrement,
	}
	pBal := precompute.UpdateBalance(vp, &precompute.Balance{})
	if !reflect.DeepEqual(pBal, wantedPBal) {
		t.Error("Incorrect balance calculations")
	}
}

func TestSameHead(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)
	if err := beaconState.SetSlot(1); err != nil {
		t.Fatal(err)
	}
	att := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 0}}}
	r := [32]byte{'A'}
	br := beaconState.BlockRoots()
	br[0] = r[:]
	if err := beaconState.SetBlockRoots(br); err != nil {
		t.Fatal(err)
	}
	att.Data.BeaconBlockRoot = r[:]
	same, err := precompute.SameHead(beaconState, &pb.PendingAttestation{Data: att.Data})
	if err != nil {
		t.Fatal(err)
	}
	if !same {
		t.Error("head in state does not match head in attestation")
	}
	newRoot := [32]byte{'B'}
	att.Data.BeaconBlockRoot = newRoot[:]
	same, err = precompute.SameHead(beaconState, &pb.PendingAttestation{Data: att.Data})
	if err != nil {
		t.Fatal(err)
	}
	if same {
		t.Error("head in state matches head in attestation")
	}
}

func TestSameTarget(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)
	if err := beaconState.SetSlot(1); err != nil {
		t.Fatal(err)
	}
	att := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 0}}}
	r := [32]byte{'A'}
	br := beaconState.BlockRoots()
	br[0] = r[:]
	if err := beaconState.SetBlockRoots(br); err != nil {
		t.Fatal(err)
	}
	att.Data.Target.Root = r[:]
	same, err := precompute.SameTarget(beaconState, &pb.PendingAttestation{Data: att.Data}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !same {
		t.Error("head in state does not match head in attestation")
	}
	newRoot := [32]byte{'B'}
	att.Data.Target.Root = newRoot[:]
	same, err = precompute.SameTarget(beaconState, &pb.PendingAttestation{Data: att.Data}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if same {
		t.Error("head in state matches head in attestation")
	}
}

func TestAttestedPrevEpoch(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}
	att := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 0}}}
	r := [32]byte{'A'}
	br := beaconState.BlockRoots()
	br[0] = r[:]
	if err := beaconState.SetBlockRoots(br); err != nil {
		t.Fatal(err)
	}
	att.Data.Target.Root = r[:]
	att.Data.BeaconBlockRoot = r[:]
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
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch + 1); err != nil {
		t.Fatal(err)
	}
	att := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 1}}}
	r := [32]byte{'A'}

	br := beaconState.BlockRoots()
	br[params.BeaconConfig().SlotsPerEpoch] = r[:]
	if err := beaconState.SetBlockRoots(br); err != nil {
		t.Fatal(err)
	}
	att.Data.Target.Root = r[:]
	att.Data.BeaconBlockRoot = r[:]
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
	params.UseMinimalConfig()
	defer params.UseMainnetConfig()

	validators := uint64(64)
	beaconState, _ := testutil.DeterministicGenesisState(t, validators)
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}

	bf := []byte{0xff}
	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 0}},
		AggregationBits: bf}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 0}},
		AggregationBits: bf}
	rt := [32]byte{'A'}
	att1.Data.Target.Root = rt[:]
	att1.Data.BeaconBlockRoot = rt[:]
	br := beaconState.BlockRoots()
	newRt := [32]byte{'B'}
	br[0] = newRt[:]
	if err := beaconState.SetBlockRoots(br); err != nil {
		t.Fatal(err)
	}
	att2.Data.Target.Root = newRt[:]
	att2.Data.BeaconBlockRoot = newRt[:]
	err := beaconState.SetPreviousEpochAttestations([]*pb.PendingAttestation{{Data: att1.Data, AggregationBits: bf, InclusionDelay: 1}})
	if err != nil {
		t.Fatal(err)
	}
	err = beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{{Data: att2.Data, AggregationBits: bf, InclusionDelay: 1}})
	if err != nil {
		t.Fatal(err)
	}

	pVals := make([]*precompute.Validator, validators)
	for i := 0; i < len(pVals); i++ {
		pVals[i] = &precompute.Validator{CurrentEpochEffectiveBalance: 100}
	}
	pBal := &precompute.Balance{}
	pVals, pBal, err = precompute.ProcessAttestations(context.Background(), beaconState, pVals, pBal)
	if err != nil {
		t.Fatal(err)
	}

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att1.Data.Slot, att1.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	indices := attestationutil.AttestingIndices(att1.AggregationBits, committee)
	for _, i := range indices {
		if !pVals[i].IsPrevEpochAttester {
			t.Error("Not a prev epoch attester")
		}
	}
	committee, err = helpers.BeaconCommitteeFromState(beaconState, att2.Data.Slot, att2.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	indices = attestationutil.AttestingIndices(att2.AggregationBits, committee)
	for _, i := range indices {
		if !pVals[i].IsPrevEpochAttester {
			t.Error("Not a prev epoch attester")
		}
		if !pVals[i].IsPrevEpochTargetAttester {
			t.Error("Not a prev epoch target attester")
		}
		if !pVals[i].IsPrevEpochHeadAttester {
			t.Error("Not a prev epoch head attester")
		}
	}
}

func TestEnsureBalancesLowerBound(t *testing.T) {
	b := &precompute.Balance{}
	b = precompute.EnsureBalancesLowerBound(b)
	if b.ActiveCurrentEpoch != params.BeaconConfig().EffectiveBalanceIncrement {
		t.Error("Did not get wanted active current balance")
	}
	if b.ActivePrevEpoch != params.BeaconConfig().EffectiveBalanceIncrement {
		t.Error("Did not get wanted active previous balance")
	}
	if b.CurrentEpochAttested != params.BeaconConfig().EffectiveBalanceIncrement {
		t.Error("Did not get wanted current attested balance")
	}
	if b.CurrentEpochTargetAttested != params.BeaconConfig().EffectiveBalanceIncrement {
		t.Error("Did not get wanted target attested balance")
	}
	if b.PrevEpochAttested != params.BeaconConfig().EffectiveBalanceIncrement {
		t.Error("Did not get wanted prev attested balance")
	}
	if b.PrevEpochTargetAttested != params.BeaconConfig().EffectiveBalanceIncrement {
		t.Error("Did not get wanted prev target attested balance")
	}
	if b.PrevEpochHeadAttested != params.BeaconConfig().EffectiveBalanceIncrement {
		t.Error("Did not get wanted prev head attested balance")
	}
}
