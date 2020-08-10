package helpers

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestTotalBalance_OK(t *testing.T) {
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{Validators: []*ethpb.Validator{
		{EffectiveBalance: 27 * 1e9}, {EffectiveBalance: 28 * 1e9},
		{EffectiveBalance: 32 * 1e9}, {EffectiveBalance: 40 * 1e9},
	}})
	require.NoError(t, err)

	balance := TotalBalance(state, []uint64{0, 1, 2, 3})
	wanted := state.Validators()[0].EffectiveBalance + state.Validators()[1].EffectiveBalance +
		state.Validators()[2].EffectiveBalance + state.Validators()[3].EffectiveBalance
	assert.Equal(t, wanted, balance, "Incorrect TotalBalance")
}

func TestTotalBalance_ReturnsEffectiveBalanceIncrement(t *testing.T) {
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{Validators: []*ethpb.Validator{}})
	require.NoError(t, err)

	balance := TotalBalance(state, []uint64{})
	wanted := params.BeaconConfig().EffectiveBalanceIncrement
	assert.Equal(t, wanted, balance, "Incorrect TotalBalance")
}

func TestTotalActiveBalance_OK(t *testing.T) {
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{Validators: []*ethpb.Validator{
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
	}})
	require.NoError(t, err)

	balance, err := TotalActiveBalance(state)
	assert.NoError(t, err)
	wanted := state.Validators()[0].EffectiveBalance + state.Validators()[1].EffectiveBalance +
		state.Validators()[2].EffectiveBalance + state.Validators()[3].EffectiveBalance
	assert.Equal(t, wanted, balance, "Incorrect TotalActiveBalance")
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
		state, err := beaconstate.InitializeFromProto(&pb.BeaconState{Balances: test.b})
		require.NoError(t, err)
		assert.Equal(t, test.b[test.i], state.Balances()[test.i], "Incorrect Validator balance")
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
		state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
			Validators: []*ethpb.Validator{
				{EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 4}},
			Balances: test.b,
		})
		require.NoError(t, err)
		require.NoError(t, IncreaseBalance(state, test.i, test.nb))
		assert.Equal(t, test.eb, state.Balances()[test.i], "Incorrect Validator balance")
	}
}

func TestDecreaseBalance_OK(t *testing.T) {
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
		state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
			Validators: []*ethpb.Validator{
				{EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 3}},
			Balances: test.b,
		})
		require.NoError(t, err)
		require.NoError(t, DecreaseBalance(state, test.i, test.nb))
		assert.Equal(t, test.eb, state.Balances()[test.i], "Incorrect Validator balance")
	}
}
