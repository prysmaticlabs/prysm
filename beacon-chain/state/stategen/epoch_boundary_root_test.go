package stategen

import "testing"

func TestEpochBoundaryRoot_CanSetGetDelete(t *testing.T) {
	s := &State{
		epochBoundarySlotToRoot: make(map[uint64][32]byte),
	}
	slot := uint64(100)
	r := [32]byte{'A'}

	_, exists := s.epochBoundaryRoot(slot)
	if exists {
		t.Fatal("should not be cached")
	}

	s.setEpochBoundaryRoot(slot, r)

	rReceived, exists := s.epochBoundaryRoot(slot)
	if !exists {
		t.Fatal("should be cached")
	}
	if rReceived != r {
		t.Error("did not cache right value")
	}

	s.deleteEpochBoundaryRoot(100)
	_, exists = s.epochBoundaryRoot(slot)
	if exists {
		t.Fatal("should not be cached")
	}
}
