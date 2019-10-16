package precompute

import (
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestNew(t *testing.T) {
	ffe := params.BeaconConfig().FarFutureEpoch
	s := &pb.BeaconState{
		Slot: params.BeaconConfig().SlotsPerEpoch,
		// Validator 0 is slashed
		// Validator 1 is withdrawable
		// Validator 2 is active prev epoch and current epoch
		// Validator 3 is active prev epoch
		Validators: []*ethpb.Validator{
			{Slashed: true, WithdrawableEpoch: ffe, EffectiveBalance: 100},
			{EffectiveBalance: 100},
			{WithdrawableEpoch: ffe, ExitEpoch: ffe, EffectiveBalance: 100},
			{WithdrawableEpoch: ffe, ExitEpoch: 1, EffectiveBalance: 100},
		},
	}
	v, b := New(s)
	if !reflect.DeepEqual(v[0], &Validator{IsSlashed: true, CurrentEpochEffectiveBalance: 100}) {
		t.Error("Incorrect validator 0 status")
	}
	if !reflect.DeepEqual(v[1], &Validator{IsWithdrawableCurrentEpoch: true, CurrentEpochEffectiveBalance: 100}) {
		t.Error("Incorrect validator 1 status")
	}
	if !reflect.DeepEqual(v[2], &Validator{IsActiveCurrentEpoch: true, IsActivePrevEpoch: true, CurrentEpochEffectiveBalance: 100}) {
		t.Error("Incorrect validator 2 status")
	}
	if !reflect.DeepEqual(v[3], &Validator{IsActivePrevEpoch: true, CurrentEpochEffectiveBalance: 100}) {
		t.Error("Incorrect validator 3 status")
	}

	wantedBalances := &Balance{
		CurrentEpoch: 100,
		PrevEpoch:    200,
	}
	if !reflect.DeepEqual(b, wantedBalances) {
		t.Error("Incorrect wanted balance")
	}
}

func TestUpdateValidator(t *testing.T) {
	vp := []*Validator{{}, {}, {}, {}, {}, {}}
	record := &Validator{IsCurrentEpochAttester: true, IsCurrentEpochTargetAttester: true,
		IsPrevEpochAttester: true, IsPrevEpochTargetAttester: true, IsPrevEpochHeadAttester: true}
	a := &pb.PendingAttestation{InclusionDelay: 1, ProposerIndex: 2}

	// Indices 1 3 and 5 attested
	vp = updateValidator(vp, record, []uint64{1, 3, 5}, a)

	wanted := &Validator{IsCurrentEpochAttester: true, IsCurrentEpochTargetAttester: true,
		IsPrevEpochAttester: true, IsPrevEpochTargetAttester: true, IsPrevEpochHeadAttester: true, ProposerIndex: 2, InclusionDistance: 1}
	wantedVp := []*Validator{{}, wanted, {}, wanted, {}, wanted}
	if !reflect.DeepEqual(vp, wantedVp) {
		t.Error("Incorrect attesting validator calculations")
	}
}

func TestUpdateBalance(t *testing.T) {
	vp := []*Validator{
		{IsCurrentEpochAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsCurrentEpochTargetAttester: true, IsCurrentEpochAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsCurrentEpochTargetAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsPrevEpochAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsPrevEpochAttester: true, IsPrevEpochTargetAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsPrevEpochHeadAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsPrevEpochAttester: true, IsPrevEpochHeadAttester: true, CurrentEpochEffectiveBalance: 100},
		{IsSlashed: true, IsCurrentEpochAttester: true, CurrentEpochEffectiveBalance: 100},
	}
	wantedBp := &Balance{
		CurrentEpochAttesters:       200,
		CurrentEpochTargetAttesters: 200,
		PrevEpochAttesters:          300,
		PrevEpochTargetAttesters:    100,
		PrevEpochHeadAttesters:      200,
	}
	bp := updateBalance(vp, &Balance{})
	if !reflect.DeepEqual(bp, wantedBp) {
		t.Error("Incorrect balance calculations")
	}
}
