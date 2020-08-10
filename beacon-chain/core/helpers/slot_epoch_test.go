package helpers

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"

	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestSlotToEpoch_OK(t *testing.T) {
	tests := []struct {
		slot  uint64
		epoch uint64
	}{
		{slot: 0, epoch: 0},
		{slot: 50, epoch: 1},
		{slot: 64, epoch: 2},
		{slot: 128, epoch: 4},
		{slot: 200, epoch: 6},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.epoch, SlotToEpoch(tt.slot), "SlotToEpoch(%d)", tt.slot)
	}
}

func TestCurrentEpoch_OK(t *testing.T) {
	tests := []struct {
		slot  uint64
		epoch uint64
	}{
		{slot: 0, epoch: 0},
		{slot: 50, epoch: 1},
		{slot: 64, epoch: 2},
		{slot: 128, epoch: 4},
		{slot: 200, epoch: 6},
	}
	for _, tt := range tests {
		state, err := beaconstate.InitializeFromProto(&pb.BeaconState{Slot: tt.slot})
		require.NoError(t, err)
		assert.Equal(t, tt.epoch, CurrentEpoch(state), "ActiveCurrentEpoch(%d)", state.Slot())
	}
}

func TestPrevEpoch_OK(t *testing.T) {
	tests := []struct {
		slot  uint64
		epoch uint64
	}{
		{slot: 0, epoch: 0},
		{slot: 0 + params.BeaconConfig().SlotsPerEpoch + 1, epoch: 0},
		{slot: 2 * params.BeaconConfig().SlotsPerEpoch, epoch: 1},
	}
	for _, tt := range tests {
		state, err := beaconstate.InitializeFromProto(&pb.BeaconState{Slot: tt.slot})
		require.NoError(t, err)
		assert.Equal(t, tt.epoch, PrevEpoch(state), "ActivePrevEpoch(%d)", state.Slot())
	}
}

func TestNextEpoch_OK(t *testing.T) {
	tests := []struct {
		slot  uint64
		epoch uint64
	}{
		{slot: 0, epoch: 0/params.BeaconConfig().SlotsPerEpoch + 1},
		{slot: 50, epoch: 0/params.BeaconConfig().SlotsPerEpoch + 2},
		{slot: 64, epoch: 64/params.BeaconConfig().SlotsPerEpoch + 1},
		{slot: 128, epoch: 128/params.BeaconConfig().SlotsPerEpoch + 1},
		{slot: 200, epoch: 200/params.BeaconConfig().SlotsPerEpoch + 1},
	}
	for _, tt := range tests {
		state, err := beaconstate.InitializeFromProto(&pb.BeaconState{Slot: tt.slot})
		require.NoError(t, err)
		assert.Equal(t, tt.epoch, NextEpoch(state), "NextEpoch(%d)", state.Slot())
	}
}

func TestEpochStartSlot_OK(t *testing.T) {
	tests := []struct {
		epoch     uint64
		startSlot uint64
	}{
		{epoch: 0, startSlot: 0 * params.BeaconConfig().SlotsPerEpoch},
		{epoch: 1, startSlot: 1 * params.BeaconConfig().SlotsPerEpoch},
		{epoch: 10, startSlot: 10 * params.BeaconConfig().SlotsPerEpoch},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{Slot: tt.epoch}
		assert.Equal(t, tt.startSlot, StartSlot(tt.epoch), "StartSlot(%d)", state.Slot)
	}
}

func TestIsEpochStart(t *testing.T) {
	epochLength := params.BeaconConfig().SlotsPerEpoch

	tests := []struct {
		slot   uint64
		result bool
	}{
		{
			slot:   epochLength + 1,
			result: false,
		},
		{
			slot:   epochLength - 1,
			result: false,
		},
		{
			slot:   epochLength,
			result: true,
		},
		{
			slot:   epochLength * 2,
			result: true,
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.result, IsEpochStart(tt.slot), "IsEpochStart(%d)", tt.slot)
	}
}

func TestIsEpochEnd(t *testing.T) {
	epochLength := params.BeaconConfig().SlotsPerEpoch

	tests := []struct {
		slot   uint64
		result bool
	}{
		{
			slot:   epochLength + 1,
			result: false,
		},
		{
			slot:   epochLength,
			result: false,
		},
		{
			slot:   epochLength - 1,
			result: true,
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.result, IsEpochEnd(tt.slot), "IsEpochEnd(%d)", tt.slot)
	}
}

func TestSlotsSinceEpochStarts(t *testing.T) {
	tests := []struct {
		slots       uint64
		wantedSlots uint64
	}{
		{slots: 0, wantedSlots: 0},
		{slots: 1, wantedSlots: 1},
		{slots: params.BeaconConfig().SlotsPerEpoch - 1, wantedSlots: params.BeaconConfig().SlotsPerEpoch - 1},
		{slots: params.BeaconConfig().SlotsPerEpoch + 1, wantedSlots: 1},
		{slots: 10*params.BeaconConfig().SlotsPerEpoch + 2, wantedSlots: 2},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.wantedSlots, SlotsSinceEpochStarts(tt.slots))
	}
}

func TestRoundUpToNearestEpoch_OK(t *testing.T) {
	tests := []struct {
		startSlot     uint64
		roundedUpSlot uint64
	}{
		{startSlot: 0 * params.BeaconConfig().SlotsPerEpoch, roundedUpSlot: 0},
		{startSlot: 1*params.BeaconConfig().SlotsPerEpoch - 10, roundedUpSlot: 1 * params.BeaconConfig().SlotsPerEpoch},
		{startSlot: 10*params.BeaconConfig().SlotsPerEpoch - (params.BeaconConfig().SlotsPerEpoch - 1), roundedUpSlot: 10 * params.BeaconConfig().SlotsPerEpoch},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.roundedUpSlot, RoundUpToNearestEpoch(tt.startSlot), "RoundUpToNearestEpoch(%d)", tt.startSlot)
	}
}

func TestSlotToTime(t *testing.T) {
	type args struct {
		genesisTimeSec uint64
		slot           uint64
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{
			name: "slot_0",
			args: args{
				genesisTimeSec: 0,
				slot:           0,
			},
			want:    time.Unix(0, 0),
			wantErr: false,
		},
		{
			name: "slot_1",
			args: args{
				genesisTimeSec: 0,
				slot:           1,
			},
			want:    time.Unix(int64(1*params.BeaconConfig().SecondsPerSlot), 0),
			wantErr: false,
		},
		{
			name: "slot_12",
			args: args{
				genesisTimeSec: 500,
				slot:           12,
			},
			want:    time.Unix(500+int64(12*params.BeaconConfig().SecondsPerSlot), 0),
			wantErr: false,
		},
		{
			name: "overflow",
			args: args{
				genesisTimeSec: 500,
				slot:           math.MaxUint64,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := SlotToTime(tt.args.genesisTimeSec, tt.args.slot); (err != nil) != tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SlotToTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifySlotTime(t *testing.T) {
	type args struct {
		genesisTime   int64
		slot          uint64
		timeTolerance time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Past slot",
			args: args{
				genesisTime: roughtime.Now().Add(-1 * 5 * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second).Unix(),
				slot:        3,
			},
			wantErr: false,
		},
		{
			name: "within tolerance",
			args: args{
				genesisTime: roughtime.Now().Add(-1 * 5 * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second).Add(20 * time.Millisecond).Unix(),
				slot:        5,
			},
			wantErr: false,
		},
		{
			name: "future slot",
			args: args{
				genesisTime: roughtime.Now().Add(-1 * 5 * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second).Unix(),
				slot:        6,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := VerifySlotTime(uint64(tt.args.genesisTime), tt.args.slot, tt.args.timeTolerance); (err != nil) != tt.wantErr {
				t.Errorf("VerifySlotTime() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
