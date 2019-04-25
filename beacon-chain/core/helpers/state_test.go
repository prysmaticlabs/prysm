package helpers

import (
	"bytes"
	"fmt"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestStateRootAtSlot_CorrectStateRoot(t *testing.T) {
	var stateRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		stateRoots = append(stateRoots, []byte{byte(i)})
	}
	s := &pb.BeaconState{
		LatestStateRoots: stateRoots,
	}

	tests := []struct {
		slot         uint64
		stateSlot    uint64
		expectedRoot []byte
	}{
		{
			slot:         1,
			stateSlot:    6,
			expectedRoot: []byte{1},
		},
		{
			slot:         10,
			stateSlot:    20,
			expectedRoot: []byte{10},
		},
		{
			slot:         100,
			stateSlot:    128,
			expectedRoot: []byte{100},
		}, {
			slot:         2999,
			stateSlot:    3005,
			expectedRoot: []byte{183},
		}, {
			slot:         2873,
			stateSlot:    4000,
			expectedRoot: []byte{57},
		},
	}
	for _, tt := range tests {
		s.Slot = tt.stateSlot + params.BeaconConfig().GenesisSlot
		wantedSlot := tt.slot + params.BeaconConfig().GenesisSlot
		result, err := StateRoot(s, wantedSlot)
		if err != nil {
			t.Fatalf("failed to get state root at slot %d: %v",
				wantedSlot-params.BeaconConfig().GenesisSlot, err)
		}
		if !bytes.Equal(result, tt.expectedRoot) {
			t.Errorf(
				"result state root was an unexpected value, wanted %v, got %v",
				tt.expectedRoot,
				result,
			)
		}
	}
}

func TestStateRootAtSlot_OutOfBounds(t *testing.T) {
	var stateRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		stateRoots = append(stateRoots, []byte{byte(i)})
	}
	state := &pb.BeaconState{
		LatestStateRoots: stateRoots,
	}

	tests := []struct {
		slot        uint64
		stateSlot   uint64
		expectedErr string
	}{
		{
			slot:      params.BeaconConfig().GenesisSlot + 2000,
			stateSlot: params.BeaconConfig().GenesisSlot + 500,
			expectedErr: fmt.Sprintf("slot %d is not within range %d to %d",
				2000,
				0,
				500),
		},
		{
			slot:        params.BeaconConfig().GenesisSlot + 120,
			stateSlot:   params.BeaconConfig().GenesisSlot + 401,
			expectedErr: "slot 120 is not within range 272 to 400",
		},
	}
	for _, tt := range tests {
		state.Slot = tt.stateSlot
		_, err := StateRoot(state, tt.slot)
		if err != nil && err.Error() != tt.expectedErr {
			t.Errorf("Expected error \"%s\" got \"%v\"", tt.expectedErr, err)
		}
	}
}
