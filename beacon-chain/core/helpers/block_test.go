package helpers

import (
	"bytes"
	"fmt"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestBlockRootAtSlot_CorrectBlockRoot(t *testing.T) {
	var blockRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	s := &pb.BeaconState{
		LatestBlockRoots: blockRoots,
	}

	tests := []struct {
		slot         uint64
		stateSlot    uint64
		expectedRoot []byte
	}{
		{
			slot:         0,
			stateSlot:    1,
			expectedRoot: []byte{0},
		},
		{
			slot:         2,
			stateSlot:    5,
			expectedRoot: []byte{2},
		},
		{
			slot:         64,
			stateSlot:    128,
			expectedRoot: []byte{64},
		}, {
			slot:         2999,
			stateSlot:    3000,
			expectedRoot: []byte{183},
		}, {
			slot:         2873,
			stateSlot:    3000,
			expectedRoot: []byte{57},
		},
	}
	for _, tt := range tests {
		s.Slot = tt.stateSlot 
		wantedSlot := tt.slot 
		result, err := BlockRootAtSlot(s, wantedSlot)
		if err != nil {
			t.Fatalf("failed to get block root at slot %d: %v",
				wantedSlot, err)
		}
		if !bytes.Equal(result, tt.expectedRoot) {
			t.Errorf(
				"result block root was an unexpected value, wanted %v, got %v",
				tt.expectedRoot,
				result,
			)
		}
	}
}

func TestBlockRootAtSlot_OutOfBounds(t *testing.T) {
	var blockRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	state := &pb.BeaconState{
		LatestBlockRoots: blockRoots,
	}

	tests := []struct {
		slot        uint64
		stateSlot   uint64
		expectedErr string
	}{
		{
			slot:      1000,
			stateSlot: 500,
			expectedErr: fmt.Sprintf("slot %d is not within range %d to %d",
				1000,
				0,
				500),
		},
		{
			slot:        129,
			stateSlot:   400,
			expectedErr: "slot 129 is not within range 272 to 399",
		},
	}
	for _, tt := range tests {
		state.Slot = tt.stateSlot
		_, err := BlockRootAtSlot(state, tt.slot)
		if err != nil && err.Error() != tt.expectedErr {
			t.Errorf("Expected error \"%s\" got \"%v\"", tt.expectedErr, err)
		}
	}
}
