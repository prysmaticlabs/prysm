package types

import (
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestCrystallizedState(t *testing.T) {
	if !reflect.DeepEqual(NewCrystallizedState(nil), &CrystallizedState{}) {
		t.Errorf("Crystallized state mismatch, want %v, received %v", NewCrystallizedState(nil), &CrystallizedState{})
	}
	_, crystallized, err := NewGenesisStates()
	if err != nil {
		t.Fatalf("Can't get genesis state: %v", err)
	}
	emptyCrystallized := &CrystallizedState{}
	if _, err := emptyCrystallized.Marshal(); err == nil {
		t.Error("marshal with empty data should fail")
	}
	if _, err := emptyCrystallized.Hash(); err == nil {
		t.Error("hash with empty data should fail")
	}
	if _, err := crystallized.Hash(); err != nil {
		t.Errorf("hashing with data should not fail, received %v", err)
	}
	if !reflect.DeepEqual(crystallized.data, crystallized.Proto()) {
		t.Errorf("inner crystallized state data did not match proto: received %v, wanted %v", crystallized.Proto(), crystallized.data)
	}
}

// newTestBlock is a helper method to create blocks with valid defaults.
// For a generic block, use NewBlock(t, nil).
func newTestBlock(t *testing.T, b *pb.BeaconBlock) *Block {
	if b == nil {
		b = &pb.BeaconBlock{}
	}
	if b.ActiveStateHash == nil {
		b.ActiveStateHash = make([]byte, 32)
	}
	if b.CrystallizedStateHash == nil {
		b.CrystallizedStateHash = make([]byte, 32)
	}
	if b.ParentHash == nil {
		b.ParentHash = make([]byte, 32)
	}

	return NewBlock(b)
}
