package blocks

import (
	"bytes"
	"math"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestGenesisBlock(t *testing.T) {
	stateHash := []byte{0}
	b1 := NewGenesisBlock(stateHash)

	if b1.GetParentRootHash32() == nil {
		t.Error("genesis block missing ParentHash field")
	}

	if !reflect.DeepEqual(b1.GetBody().GetAttestations(), []*pb.Attestation{}) {
		t.Errorf("genesis block should have 0 attestations")
	}

	if !bytes.Equal(b1.GetRandaoRevealHash32(), params.BeaconConfig().ZeroHash[:]) {
		t.Error("genesis block missing RandaoRevealHash32 field")
	}

	if !bytes.Equal(b1.GetStateRootHash32(), stateHash) {
		t.Error("genesis block StateRootHash32 isn't initialized correctly")
	}

	rd := []byte{}
	if IsRandaoValid(b1.GetRandaoRevealHash32(), rd) {
		t.Error("RANDAO should be empty")
	}

}

func TestBlockRootAtSlot_OK(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	var blockRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
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
			expectedRoot: []byte{127},
		}, {
			slot:         2873,
			stateSlot:    3000,
			expectedRoot: []byte{1},
		},
	}
	for _, tt := range tests {
		state.Slot = tt.stateSlot
		result, err := BlockRoot(state, tt.slot)
		if err != nil {
			t.Errorf("Failed to get block root at slot %d: %v", tt.slot, err)
		}
		if !bytes.Equal(result, tt.expectedRoot) {
			t.Errorf(
				"Result block root was an unexpected value. Wanted %d, got %d",
				tt.expectedRoot,
				result,
			)
		}
	}
}

func TestBlockRootAtSlot_OutOfBounds(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	state := &pb.BeaconState{}

	tests := []struct {
		slot        uint64
		expectedErr string
	}{
		{
			slot:        1000,
			expectedErr: "slot 1000 out of bounds: 0 <= slot < 0",
		},
		{
			slot:        129,
			expectedErr: "slot 129 out of bounds: 0 <= slot < 0",
		},
	}
	for _, tt := range tests {
		_, err := BlockRoot(state, tt.slot)
		if err != nil && err.Error() != tt.expectedErr {
			t.Errorf("Expected error \"%s\" got \"%v\"", tt.expectedErr, err)
		}
	}
}

func TestProcessBlockRoots(t *testing.T) {
	state := &pb.BeaconState{}

	state.LatestBlockRootHash32S = make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	state.Slot = params.BeaconConfig().LatestBlockRootsLength + 1

	testRoot := [32]byte{'a'}

	newState := ProcessBlockRoots(state, testRoot)
	if !bytes.Equal(newState.LatestBlockRootHash32S[0], testRoot[:]) {
		t.Fatalf("Latest Block root hash not saved."+
			" Supposed to get %#x , but got %#x", testRoot, newState.LatestBlockRootHash32S[0])
	}

	newState.Slot = newState.Slot - 1

	newState = ProcessBlockRoots(newState, testRoot)
	expectedHashes := make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	expectedHashes[0] = testRoot[:]
	expectedHashes[params.BeaconConfig().LatestBlockRootsLength-1] = testRoot[:]

	expectedRoot := hashutil.MerkleRoot(expectedHashes)

	if !bytes.Equal(newState.BatchedBlockRootHash32S[0], expectedRoot[:]) {
		t.Errorf("Saved merkle root is not equal to expected merkle root"+
			"\n Expected %#x but got %#x", expectedRoot, newState.BatchedBlockRootHash32S[0])
	}
}

func TestForkVersion(t *testing.T) {
	forkData := &pb.ForkData{
		ForkSlot:        10,
		PreForkVersion:  2,
		PostForkVersion: 3,
	}

	if ForkVersion(forkData, 9) != 2 {
		t.Errorf("Fork Version not equal to 2 %d", ForkVersion(forkData, 9))
	}

	if ForkVersion(forkData, 11) != 3 {
		t.Errorf("Fork Version not equal to 3 %d", ForkVersion(forkData, 11))
	}
}

func TestDomainVersion(t *testing.T) {
	forkData := &pb.ForkData{
		ForkSlot:        10,
		PreForkVersion:  2,
		PostForkVersion: 3,
	}

	constant := uint64(math.Pow(2, 32))

	if DomainVersion(forkData, 9, 2) != 2*constant+2 {
		t.Errorf("Incorrect domain version %d", DomainVersion(forkData, 9, 2))
	}

	if DomainVersion(forkData, 11, 3) != 3*constant+3 {
		t.Errorf("Incorrect domain version %d", DomainVersion(forkData, 11, 3))
	}
}
