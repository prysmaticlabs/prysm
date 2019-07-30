package helpers

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestTotalBalance_OK(t *testing.T) {
	state := &pb.BeaconState{Validators: []*ethpb.Validator{
		{EffectiveBalance: 27 * 1e9}, {EffectiveBalance: 28 * 1e9},
		{EffectiveBalance: 32 * 1e9}, {EffectiveBalance: 40 * 1e9},
	}}

	balance := TotalBalance(state, []uint64{0, 1, 2, 3})
	wanted := state.Validators[0].EffectiveBalance + state.Validators[1].EffectiveBalance +
		state.Validators[2].EffectiveBalance + state.Validators[3].EffectiveBalance

	if balance != wanted {
		t.Errorf("Incorrect TotalBalance. Wanted: %d, got: %d", wanted, balance)
	}
}

func TestTotalBalance_ReturnsOne(t *testing.T) {
	state := &pb.BeaconState{Validators: []*ethpb.Validator{}}

	balance := TotalBalance(state, []uint64{})
	wanted := uint64(1)

	if balance != wanted {
		t.Errorf("Incorrect TotalBalance. Wanted: %d, got: %d", wanted, balance)
	}
}

func TestTotalActiveBalance_OK(t *testing.T) {
	// Cache toggled by feature flag for now. See https://github.com/prysmaticlabs/prysm/issues/3106.
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableActiveBalanceCache: true,
	})
	defer func() {
		featureconfig.InitFeatureConfig(nil)
	}()

	state := &pb.BeaconState{Validators: []*ethpb.Validator{
		{
			EffectiveBalance: 32 * 1e9,
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
		},
		{
			EffectiveBalance: 30 * 1e9,
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
		},
		{
			EffectiveBalance: 30 * 1e9,
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
		},
		{
			EffectiveBalance: 32 * 1e9,
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
		},
	}}

	balance, err := TotalActiveBalance(state)
	if err != nil {
		t.Error(err)
	}
	wanted := state.Validators[0].EffectiveBalance + state.Validators[1].EffectiveBalance +
		state.Validators[2].EffectiveBalance + state.Validators[3].EffectiveBalance

	if balance != wanted {
		t.Errorf("Incorrect TotalActiveBalance. Wanted: %d, got: %d", wanted, balance)
	}

	state.Validators[1].EffectiveBalance += 5
	cacheHitBalance, err := TotalActiveBalance(state)
	if err != nil {
		t.Error(err)
	}

	if cacheHitBalance != wanted {
		t.Errorf("Incorrect Cached TotalActiveBalance. Wanted: %d, got: %d", wanted, cacheHitBalance)
	}
}

func TestGetBalance_OK(t *testing.T) {
	tests := []struct {
		i uint64
		b []uint64
	}{
		{i: 0, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}},
		{i: 1, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}},
		{i: 2, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}},
		{i: 0, b: []uint64{0, 0, 0}},
		{i: 2, b: []uint64{0, 0, 0}},
	}
	for _, test := range tests {
		state := &pb.BeaconState{Balances: test.b}
		if state.Balances[test.i] != test.b[test.i] {
			t.Errorf("Incorrect Validator balance. Wanted: %d, got: %d", test.b[test.i], state.Balances[test.i])
		}
	}
}

func TestIncreaseBalance_OK(t *testing.T) {
	tests := []struct {
		i  uint64
		b  []uint64
		nb uint64
		eb uint64
	}{
		{i: 0, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}, nb: 1, eb: 27*1e9 + 1},
		{i: 1, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}, nb: 0, eb: 28 * 1e9},
		{i: 2, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}, nb: 33 * 1e9, eb: 65 * 1e9},
	}
	for _, test := range tests {
		state := &pb.BeaconState{
			Validators: []*ethpb.Validator{
				{EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 4}},
			Balances: test.b,
		}
		state = IncreaseBalance(state, test.i, test.nb)
		if state.Balances[test.i] != test.eb {
			t.Errorf("Incorrect Validator balance. Wanted: %d, got: %d", test.eb, state.Balances[test.i])
		}
	}
}

func TestDecreseBalance_OK(t *testing.T) {
	tests := []struct {
		i  uint64
		b  []uint64
		nb uint64
		eb uint64
	}{
		{i: 0, b: []uint64{2, 28 * 1e9, 32 * 1e9}, nb: 1, eb: 1},
		{i: 1, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}, nb: 0, eb: 28 * 1e9},
		{i: 2, b: []uint64{27 * 1e9, 28 * 1e9, 1}, nb: 2, eb: 0},
		{i: 3, b: []uint64{27 * 1e9, 28 * 1e9, 1, 28 * 1e9}, nb: 28 * 1e9, eb: 0},
	}
	for _, test := range tests {
		state := &pb.BeaconState{
			Validators: []*ethpb.Validator{
				{EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 3}},
			Balances: test.b,
		}
		state = DecreaseBalance(state, test.i, test.nb)
		if state.Balances[test.i] != test.eb {
			t.Errorf("Incorrect Validator balance. Wanted: %d, got: %d", test.eb, state.Balances[test.i])
		}
	}
}
