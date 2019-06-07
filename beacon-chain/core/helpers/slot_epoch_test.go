package helpers

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestSlotToEpoch_OK(t *testing.T) {
	tests := []struct {
		slot  uint64
		epoch uint64
	}{
		{slot: 0, epoch: 0 / params.BeaconConfig().SlotsPerEpoch},
		{slot: 50, epoch: 0 / params.BeaconConfig().SlotsPerEpoch},
		{slot: 64, epoch: 64 / params.BeaconConfig().SlotsPerEpoch},
		{slot: 128, epoch: 128 / params.BeaconConfig().SlotsPerEpoch},
		{slot: 200, epoch: 200 / params.BeaconConfig().SlotsPerEpoch},
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
		{slot: 0, epoch: 0 / params.BeaconConfig().SlotsPerEpoch},
		{slot: 50, epoch: 0 / params.BeaconConfig().SlotsPerEpoch},
		{slot: 64, epoch: 64 / params.BeaconConfig().SlotsPerEpoch},
		{slot: 128, epoch: 128 / params.BeaconConfig().SlotsPerEpoch},
		{slot: 200, epoch: 200 / params.BeaconConfig().SlotsPerEpoch},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{Slot: tt.slot}
		if tt.epoch != CurrentEpoch(state) {
			t.Errorf("CurrentEpoch(%d) = %d, wanted: %d", state.Slot, CurrentEpoch(state), tt.epoch)
		}
	}
}

func TestPrevEpoch_OK(t *testing.T) {
	tests := []struct {
		slot  uint64
		epoch uint64
	}{
		{slot: params.BeaconConfig().GenesisSlot, epoch: params.BeaconConfig().GenesisEpoch},
		{slot: params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch + 1, epoch: params.BeaconConfig().GenesisEpoch},
		{slot: params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch, epoch: params.BeaconConfig().GenesisEpoch + 1},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{Slot: tt.slot}
		if tt.epoch != PrevEpoch(state) {
			t.Errorf("PrevEpoch(%d) = %d, wanted: %d", state.Slot, PrevEpoch(state), tt.epoch)
		}
	}
}

func TestNextEpoch_OK(t *testing.T) {
	tests := []struct {
		slot  uint64
		epoch uint64
	}{
		{slot: 0, epoch: 0/params.BeaconConfig().SlotsPerEpoch + 1},
		{slot: 50, epoch: 0/params.BeaconConfig().SlotsPerEpoch + 1},
		{slot: 64, epoch: 64/params.BeaconConfig().SlotsPerEpoch + 1},
		{slot: 128, epoch: 128/params.BeaconConfig().SlotsPerEpoch + 1},
		{slot: 200, epoch: 200/params.BeaconConfig().SlotsPerEpoch + 1},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{Slot: tt.slot}
		if tt.epoch != NextEpoch(state) {
			t.Errorf("NextEpoch(%d) = %d, wanted: %d", state.Slot, NextEpoch(state), tt.epoch)
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
