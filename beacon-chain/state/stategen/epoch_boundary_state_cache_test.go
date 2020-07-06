package stategen

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEpochBoundaryStateCache_BadSlotKey(t *testing.T) {
	if _, err := slotKeyFn("sushi"); err == nil || err.Error() != errNotSlotRootInfo.Error() {
		t.Error("Did not get wanted error")
	}
}

func TestEpochBoundaryStateCache_BadRootKey(t *testing.T) {
	if _, err := rootKeyFn("noodle"); err == nil || err.Error() != errNotRootStateInfo.Error() {
		t.Error("Did not get wanted error")
	}
}

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

	got, exists, err := e.getByRoot([32]byte{'b'})
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Should not exist")
	}
	if got != nil {
		t.Error("Should not exist")
	}

	got, exists, err = e.getByRoot([32]byte{'a'})
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Should exist")
	}
	if !reflect.DeepEqual(got.state.InnerStateUnsafe(), s.InnerStateUnsafe()) {
		t.Error("Should have the same state")
	}

	got, exists, err = e.getBySlot(2)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Should not exist")
	}
	if got != nil {
		t.Error("Should not exist")
	}

	got, exists, err = e.getBySlot(1)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Should exist")
	}
	if !reflect.DeepEqual(got.state.InnerStateUnsafe(), s.InnerStateUnsafe()) {
		t.Error("Should have the same state")
	}
}

func TestEpochBoundaryStateCache_CanTrim(t *testing.T) {
	e := newBoundaryStateCache()
	offSet := uint64(10)
	for i := uint64(0); i < maxCacheSize+offSet; i++ {
		s := testutil.NewBeaconState()
		if err := s.SetSlot(i); err != nil {
			t.Fatal(err)
		}
		r := [32]byte{byte(i)}
		if err := e.put(r, s); err != nil {
			t.Fatal(err)
		}
	}

	if len(e.rootStateCache.ListKeys()) != int(maxCacheSize) {
		t.Error("Did not trim to the correct amount")
	}
	if len(e.slotRootCache.ListKeys()) != int(maxCacheSize) {
		t.Error("Did not trim to the correct amount")
	}
	for _, l := range e.rootStateCache.List() {
		i, ok := l.(*rootStateInfo)
		if !ok {
			t.Fatal("Bad type assertion")
		}
		if i.state.Slot() < offSet {
			t.Error("Did not trim the correct state")
		}
	}
}
