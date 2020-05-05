package helpers

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/roughtime"

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
		if tt.epoch != SlotToEpoch(tt.slot) {
			t.Errorf("SlotToEpoch(%d) = %d, wanted: %d", tt.slot, SlotToEpoch(tt.slot), tt.epoch)
		}
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
		if err != nil {
			t.Fatal(err)
		}
		if tt.epoch != CurrentEpoch(state) {
			t.Errorf("ActiveCurrentEpoch(%d) = %d, wanted: %d", state.Slot(), CurrentEpoch(state), tt.epoch)
		}
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
		if err != nil {
			t.Fatal(err)
		}
		if tt.epoch != PrevEpoch(state) {
			t.Errorf("ActivePrevEpoch(%d) = %d, wanted: %d", state.Slot(), PrevEpoch(state), tt.epoch)
		}
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
		if err != nil {
			t.Fatal(err)
		}
		if tt.epoch != NextEpoch(state) {
			t.Errorf("NextEpoch(%d) = %d, wanted: %d", state.Slot(), NextEpoch(state), tt.epoch)
		}
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
		if tt.startSlot != StartSlot(tt.epoch) {
			t.Errorf("StartSlot(%d) = %d, wanted: %d", state.Slot, StartSlot(tt.epoch), tt.startSlot)
		}
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
		if IsEpochStart(tt.slot) != tt.result {
			t.Errorf("IsEpochStart(%d) = %v, wanted %v", tt.slot, IsEpochStart(tt.slot), tt.result)
		}
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
		if IsEpochEnd(tt.slot) != tt.result {
			t.Errorf("IsEpochEnd(%d) = %v, wanted %v", tt.slot, IsEpochEnd(tt.slot), tt.result)
		}
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
		if got := SlotsSinceEpochStarts(tt.slots); got != tt.wantedSlots {
			t.Errorf("SlotsSinceEpochStarts() = %v, want %v", got, tt.wantedSlots)
		}
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
		if tt.roundedUpSlot != RoundUpToNearestEpoch(tt.startSlot) {
			t.Errorf("RoundUpToNearestEpoch(%d) = %d, wanted: %d", tt.startSlot, RoundUpToNearestEpoch(tt.startSlot), tt.roundedUpSlot)
		}
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
