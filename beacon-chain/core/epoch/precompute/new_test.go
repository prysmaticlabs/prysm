package precompute

import (
	"context"
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
	v, b := New(context.Background(), s)
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
