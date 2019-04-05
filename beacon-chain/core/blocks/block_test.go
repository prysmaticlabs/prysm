package blocks

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestBlockRootAtSlot_AccurateBlockRoot(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("slotsPerEpoch should be 64 for these tests to pass")
	}
	var blockRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	state := &pb.BeaconState{
		LatestBlockRootHash32S: blockRoots,
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
		state.Slot = tt.stateSlot + params.BeaconConfig().GenesisSlot
		wantedSlot := tt.slot + params.BeaconConfig().GenesisSlot
		result, err := BlockRoot(state, wantedSlot)
		if err != nil {
			t.Errorf("failed to get block root at slot %d: %v", wantedSlot, err)
		}
		if !bytes.Equal(result, tt.expectedRoot) {
			t.Errorf(
				"result block root was an unexpected value. Wanted %d, got %d",
				tt.expectedRoot,
				result,
			)
		}
	}
}

func TestBlockRootAtSlot_OutOfBounds(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("slotsPerEpoch should be 64 for these tests to pass")
	}

	var blockRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	state := &pb.BeaconState{
		LatestBlockRootHash32S: blockRoots,
	}

	tests := []struct {
		slot        uint64
		stateSlot   uint64
		expectedErr string
	}{
		{
			slot:      params.BeaconConfig().GenesisSlot + 1000,
			stateSlot: params.BeaconConfig().GenesisSlot + 500,
			expectedErr: fmt.Sprintf("slot %d is not within expected range of %d to %d",
				1000,
				0,
				500),
		},
		{
			slot:        params.BeaconConfig().GenesisSlot + 129,
			stateSlot:   params.BeaconConfig().GenesisSlot + 400,
			expectedErr: "slot 129 is not within expected range of 272 to 399",
		},
	}
	for _, tt := range tests {
		state.Slot = tt.stateSlot
		_, err := BlockRoot(state, tt.slot)
		if err != nil && err.Error() != tt.expectedErr {
			t.Errorf("Expected error \"%s\" got \"%v\"", tt.expectedErr, err)
		}
	}
}

func TestProcessBlockRoots_AccurateMerkleTree(t *testing.T) {
	state := &pb.BeaconState{}

	state.LatestBlockRootHash32S = make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	state.Slot = params.BeaconConfig().LatestBlockRootsLength + 1

	testRoot := [32]byte{'a'}

	newState := ProcessBlockRoots(context.Background(), state, testRoot)
	if !bytes.Equal(newState.LatestBlockRootHash32S[0], testRoot[:]) {
		t.Fatalf("Latest Block root hash not saved."+
			" Supposed to get %#x , but got %#x", testRoot, newState.LatestBlockRootHash32S[0])
	}

	newState.Slot = newState.Slot - 1

	newState = ProcessBlockRoots(context.Background(), newState, testRoot)
	expectedHashes := make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	expectedHashes[0] = testRoot[:]
	expectedHashes[params.BeaconConfig().LatestBlockRootsLength-1] = testRoot[:]

	expectedRoot := hashutil.MerkleRoot(expectedHashes)

	if !bytes.Equal(newState.BatchedBlockRootHash32S[0], expectedRoot[:]) {
		t.Errorf("saved merkle root is not equal to expected merkle root"+
			"\n expected %#x but got %#x", expectedRoot, newState.BatchedBlockRootHash32S[0])
	}
}
