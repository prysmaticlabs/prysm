package blocks

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestGenesisBlock_InitializedCorrectly(t *testing.T) {
	stateHash := []byte{0}
	b1 := NewGenesisBlock(stateHash)

	if b1.ParentRoot == nil {
		t.Error("genesis block missing ParentHash field")
	}

	if !reflect.DeepEqual(b1.Body.Attestations, []*pb.Attestation{}) {
		t.Errorf("genesis block should have 0 attestations")
	}

	if !bytes.Equal(b1.Body.RandaoReveal, params.BeaconConfig().ZeroHash[:]) {
		t.Error("genesis block missing RandaoReveal field")
	}

	if !bytes.Equal(b1.StateRoot, stateHash) {
		t.Error("genesis block StateRootHash32 isn't initialized correctly")
	}
	expectedEth1 := &pb.Eth1Data{
		DepositRoot: params.BeaconConfig().ZeroHash[:],
		BlockRoot:   params.BeaconConfig().ZeroHash[:],
	}
	if !proto.Equal(b1.Eth1Data, expectedEth1) {
		t.Error("genesis block Eth1Data isn't initialized correctly")
	}
}

func TestBlockRootAtSlot_AccurateBlockRoot(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("slotsPerEpoch should be 64 for these tests to pass")
	}
	var blockRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	state := &pb.BeaconState{
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
		state.Slot = tt.stateSlot
		wantedSlot := tt.slot
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
			expectedErr: fmt.Sprintf("slot %d is not within expected range of %d to %d",
				1000,
				0,
				500),
		},
		{
			slot:        129,
			stateSlot:   400,
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

	state.LatestBlockRoots = make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	state.Slot = params.BeaconConfig().LatestBlockRootsLength + 1

	testRoot := [32]byte{'a'}

	newState := ProcessBlockRoots(state, testRoot)
	if !bytes.Equal(newState.LatestBlockRoots[0], testRoot[:]) {
		t.Fatalf("Latest Block root hash not saved."+
			" Supposed to get %#x , but got %#x", testRoot, newState.LatestBlockRoots[0])
	}
}
