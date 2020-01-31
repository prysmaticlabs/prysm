package helpers_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestBlockRootAtSlot_CorrectBlockRoot(t *testing.T) {
	var blockRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	s := &pb.BeaconState{
		BlockRoots: blockRoots,
	}

	tests := []struct {
		slot         uint64
		stateSlot    uint64
		expectedRoot [32]byte
	}{
		{
			slot:         0,
			stateSlot:    1,
			expectedRoot: [32]byte{0},
		},
		{
			slot:         2,
			stateSlot:    5,
			expectedRoot: [32]byte{2},
		},
		{
			slot:         64,
			stateSlot:    128,
			expectedRoot: [32]byte{64},
		}, {
			slot:         2999,
			stateSlot:    3000,
			expectedRoot: [32]byte{183},
		}, {
			slot:         2873,
			stateSlot:    3000,
			expectedRoot: [32]byte{57},
		},
		{
			slot:         0,
			stateSlot:    params.BeaconConfig().SlotsPerHistoricalRoot,
			expectedRoot: [32]byte{},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			s.Slot = tt.stateSlot
			state, _ := beaconstate.InitializeFromProto(s)
			wantedSlot := tt.slot
			result, err := helpers.BlockRootAtSlot(state, wantedSlot)
			if err != nil {
				t.Fatalf("failed to get block root at slot %d: %v",
					wantedSlot, err)
			}
			if !bytes.Equal(result, tt.expectedRoot[:]) {
				t.Errorf(
					"result block root was an unexpected value, wanted %v, got %v",
					tt.expectedRoot,
					result,
				)
			}
		})
	}
}

func TestBlockRootAtSlot_OutOfBounds(t *testing.T) {
	var blockRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	state := &pb.BeaconState{
		BlockRoots: blockRoots,
	}

	tests := []struct {
		slot        uint64
		stateSlot   uint64
		expectedErr string
	}{
		{
			slot:        1000,
			stateSlot:   500,
			expectedErr: "slot 1000 out of bounds",
		},
		{
			slot:        3000,
			stateSlot:   3000,
			expectedErr: "slot 3000 out of bounds",
		},
		{
			// Edge case where stateSlot is over slots per historical root and
			// slot is not within (stateSlot - HistoricalRootsLimit, statSlot]
			slot:        1,
			stateSlot:   params.BeaconConfig().SlotsPerHistoricalRoot + 2,
			expectedErr: "slot 1 out of bounds",
		},
	}
	for _, tt := range tests {
		state.Slot = tt.stateSlot
		s, _ := beaconstate.InitializeFromProto(state)
		_, err := helpers.BlockRootAtSlot(s, tt.slot)
		if err == nil {
			t.Errorf("Expected error %s, got nil", tt.expectedErr)
		}
		if err != nil && err.Error() != tt.expectedErr {
			t.Errorf("Expected error \"%s\" got \"%v\"", tt.expectedErr, err)
		}
	}
}
