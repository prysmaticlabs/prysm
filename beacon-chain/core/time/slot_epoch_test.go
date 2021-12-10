package time

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/config/params"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/time/slots"
)

func TestSlotToEpoch_OK(t *testing.T) {
	tests := []struct {
		slot  types.Slot
		epoch types.Epoch
	}{
		{slot: 0, epoch: 0},
		{slot: 50, epoch: 1},
		{slot: 64, epoch: 2},
		{slot: 128, epoch: 4},
		{slot: 200, epoch: 6},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.epoch, slots.ToEpoch(tt.slot), "ToEpoch(%d)", tt.slot)
	}
}

func TestCurrentEpoch_OK(t *testing.T) {
	tests := []struct {
		slot  types.Slot
		epoch types.Epoch
	}{
		{slot: 0, epoch: 0},
		{slot: 50, epoch: 1},
		{slot: 64, epoch: 2},
		{slot: 128, epoch: 4},
		{slot: 200, epoch: 6},
	}
	for _, tt := range tests {
		state, err := v1.InitializeFromProto(&eth.BeaconState{Slot: tt.slot})
		require.NoError(t, err)
		assert.Equal(t, tt.epoch, CurrentEpoch(state), "ActiveCurrentEpoch(%d)", state.Slot())
	}
}

func TestPrevEpoch_OK(t *testing.T) {
	tests := []struct {
		slot  types.Slot
		epoch types.Epoch
	}{
		{slot: 0, epoch: 0},
		{slot: 0 + params.BeaconConfig().SlotsPerEpoch + 1, epoch: 0},
		{slot: 2 * params.BeaconConfig().SlotsPerEpoch, epoch: 1},
	}
	for _, tt := range tests {
		state, err := v1.InitializeFromProto(&eth.BeaconState{Slot: tt.slot})
		require.NoError(t, err)
		assert.Equal(t, tt.epoch, PrevEpoch(state), "ActivePrevEpoch(%d)", state.Slot())
	}
}

func TestNextEpoch_OK(t *testing.T) {
	tests := []struct {
		slot  types.Slot
		epoch types.Epoch
	}{
		{slot: 0, epoch: types.Epoch(0/params.BeaconConfig().SlotsPerEpoch + 1)},
		{slot: 50, epoch: types.Epoch(0/params.BeaconConfig().SlotsPerEpoch + 2)},
		{slot: 64, epoch: types.Epoch(64/params.BeaconConfig().SlotsPerEpoch + 1)},
		{slot: 128, epoch: types.Epoch(128/params.BeaconConfig().SlotsPerEpoch + 1)},
		{slot: 200, epoch: types.Epoch(200/params.BeaconConfig().SlotsPerEpoch + 1)},
	}
	for _, tt := range tests {
		state, err := v1.InitializeFromProto(&eth.BeaconState{Slot: tt.slot})
		require.NoError(t, err)
		assert.Equal(t, tt.epoch, NextEpoch(state), "NextEpoch(%d)", state.Slot())
	}
}

func TestCanUpgradeToAltair(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	bc := params.BeaconConfig()
	bc.AltairForkEpoch = 5
	params.OverrideBeaconConfig(bc)
	tests := []struct {
		name string
		slot types.Slot
		want bool
	}{
		{
			name: "not epoch start",
			slot: 1,
			want: false,
		},
		{
			name: "not altair epoch",
			slot: params.BeaconConfig().SlotsPerEpoch,
			want: false,
		},
		{
			name: "altair epoch",
			slot: types.Slot(params.BeaconConfig().AltairForkEpoch) * params.BeaconConfig().SlotsPerEpoch,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanUpgradeToAltair(tt.slot); got != tt.want {
				t.Errorf("canUpgradeToAltair() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanUpgradeToMerge(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	bc := params.BeaconConfig()
	bc.MergeForkEpoch = 5
	params.OverrideBeaconConfig(bc)
	tests := []struct {
		name string
		slot types.Slot
		want bool
	}{
		{
			name: "not epoch start",
			slot: 1,
			want: false,
		},
		{
			name: "not merge epoch",
			slot: params.BeaconConfig().SlotsPerEpoch,
			want: false,
		},
		{
			name: "merge epoch",
			slot: types.Slot(params.BeaconConfig().MergeForkEpoch) * params.BeaconConfig().SlotsPerEpoch,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanUpgradeToMerge(tt.slot); got != tt.want {
				t.Errorf("CanUpgradeToMerge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanProcessEpoch_TrueOnEpochsLastSlot(t *testing.T) {
	tests := []struct {
		slot            types.Slot
		canProcessEpoch bool
	}{
		{
			slot:            1,
			canProcessEpoch: false,
		}, {
			slot:            63,
			canProcessEpoch: true,
		},
		{
			slot:            64,
			canProcessEpoch: false,
		}, {
			slot:            127,
			canProcessEpoch: true,
		}, {
			slot:            1000000000,
			canProcessEpoch: false,
		},
	}

	for _, tt := range tests {
		b := &eth.BeaconState{Slot: tt.slot}
		s, err := v1.InitializeFromProto(b)
		require.NoError(t, err)
		assert.Equal(t, tt.canProcessEpoch, CanProcessEpoch(s), "CanProcessEpoch(%d)", tt.slot)
	}
}
