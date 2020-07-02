package stategen

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEpochBoundaryStateCache_CanSave(t *testing.T) {
	e := newBoundaryStateCache()
	s := testutil.NewBeaconState()
	if err := s.SetSlot(1); err != nil {
		t.Fatal(err)
	}
	r := [32]byte{'a'}
	if err := e.put(r, s); err != nil {
		t.Fatal(err)
	}

	got, exists, err := e.get([32]byte{'b'})
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Should not exist")
	}
	if got != nil {
		t.Error("Should not exist")
	}

	got, exists, err = e.get([32]byte{'a'})
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Should exist")
	}
	if !reflect.DeepEqual(got.InnerStateUnsafe(), s.InnerStateUnsafe()) {
		t.Error("Should have the same state")
	}
}

func TestEpochBoundaryStateCache_CanTrim(t *testing.T) {
	e := newBoundaryStateCache()
	offSet := uint64(10)
	for i := uint64(0); i < maxCacheSize + offSet; i ++  {
		s := testutil.NewBeaconState()
		if err := s.SetSlot(i); err != nil {
			t.Fatal(err)
		}
		r := [32]byte{byte(i)}
		if err := e.put(r, s); err != nil {
			t.Fatal(err)
		}
	}

	if len(e.cache.ListKeys()) != int(maxCacheSize) {
		t.Error("Did not trim to the correct amount")
	}
	for _, l := range e.cache.List() {
		i, ok := l.(*stateInfo)
		if !ok {
			t.Fatal("Bad type assertion")
		}
		if i.state.Slot() < offSet {
			t.Error("Did not trim the correct state")
		}
	}
}

func TestEpochBoundaryStateCache_CanClear(t *testing.T) {
	e := newBoundaryStateCache()
	offSet := uint64(10)
	for i := uint64(0); i < maxCacheSize + offSet; i ++  {
		s := testutil.NewBeaconState()
		if err := s.SetSlot(i); err != nil {
			t.Fatal(err)
		}
		r := [32]byte{byte(i)}
		if err := e.put(r, s); err != nil {
			t.Fatal(err)
		}
	}

	if err := e.clear(); err != nil {
		t.Fatal(err)
	}

	if len(e.cache.ListKeys()) != 0 {
		t.Error("Did not clear to the correct amount")
	}
}
