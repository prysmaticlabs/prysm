package blocks

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/golang/protobuf/ptypes"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestGenesisBlock(t *testing.T) {
	stateHash := []byte{0}
	b1 := NewGenesisBlock(stateHash)
	b2 := NewGenesisBlock(stateHash)

	// We ensure that initializing a proto timestamp from
	// genesis time will lead to no error.
	if _, err := ptypes.TimestampProto(params.BeaconConfig().GenesisTime); err != nil {
		t.Errorf("could not create proto timestamp, expected no error: %v", err)
	}

	if b1.GetParentRootHash32() == nil {
		t.Error("genesis block missing ParentHash field")
	}

	if b1.GetBody().GetAttestations() != nil {
		t.Errorf("genesis block should have 0 attestations")
	}

	if !bytes.Equal(b1.GetRandaoRevealHash32(), []byte{0}) {
		t.Error("genesis block missing RandaoRevealHash32 field")
	}

	if !bytes.Equal(b1.GetCandidatePowReceiptRootHash32(), []byte{0}) {
		t.Error("genesis block missing CandidatePowReceiptRootHash32 field")
	}

	if !bytes.Equal(b1.GetStateRootHash32(), stateHash) {
		t.Error("genesis block StateRootHash32 isn't initialized correctly")
	}

	rd := []byte{}
	if IsRandaoValid(b1.GetRandaoRevealHash32(), rd) {
		t.Error("RANDAO should be empty")
	}

	gt1 := b1.GetTimestamp()
	gt2 := b2.GetTimestamp()
	t1, _ := ptypes.Timestamp(gt1)
	t2, _ := ptypes.Timestamp(gt2)
	if t1 != t2 {
		t.Error("different timestamp")
	}

	if !reflect.DeepEqual(b1, b2) {
		t.Error("genesis blocks proto should be equal")
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
		stateSlot   uint64
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
