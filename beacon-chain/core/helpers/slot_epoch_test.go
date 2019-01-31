package helpers

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestSlotToEpoch(t *testing.T) {
	tests := []struct {
		slot  uint64
		epoch uint64
	}{
		{slot: 0, epoch: 0 / config.EpochLength},
		{slot: 50, epoch: 0 / config.EpochLength},
		{slot: 64, epoch: 64 / config.EpochLength},
		{slot: 128, epoch: 128 / config.EpochLength},
		{slot: 200, epoch: 200 / config.EpochLength},
	}
	for _, tt := range tests {
		if tt.epoch != SlotToEpoch(tt.slot) {
			t.Errorf("SlotToEpoch(%d) = %d, wanted: %d", tt.slot, SlotToEpoch(tt.slot), tt.epoch)
		}
	}
}

func TestCurrentEpoch(t *testing.T) {
	tests := []struct {
		slot  uint64
		epoch uint64
	}{
		{slot: 0, epoch: 0 / config.EpochLength},
		{slot: 50, epoch: 0 / config.EpochLength},
		{slot: 64, epoch: 64 / config.EpochLength},
		{slot: 128, epoch: 128 / config.EpochLength},
		{slot: 200, epoch: 200 / config.EpochLength},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{Slot: tt.slot}
		if tt.epoch != CurrentEpoch(state) {
			t.Errorf("CurrentEpoch(%d) = %d, wanted: %d", state.Slot, CurrentEpoch(state), tt.epoch)
		}
	}
}

func TestEpochStartSlot(t *testing.T) {
	tests := []struct {
		epoch     uint64
		startSlot uint64
	}{
		{epoch: 0, startSlot: 0 * config.EpochLength},
		{epoch: 1, startSlot: 1 * config.EpochLength},
		{epoch: 10, startSlot: 10 * config.EpochLength},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{Slot: tt.epoch}
		if tt.startSlot != StartSlot(tt.epoch) {
			t.Errorf("StartSlot(%d) = %d, wanted: %d", state.Slot, StartSlot(tt.epoch), tt.startSlot)
		}
	}
}
