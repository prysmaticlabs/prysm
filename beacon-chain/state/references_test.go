package state

import (
	"runtime"
	"testing"

	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestStateReferenceSharing_Finalizer(t *testing.T) {
	// This test showcases the logic on a the RandaoMixes field with the GC finalizer.

	a, err := InitializeFromProtoUnsafe(&p2ppb.BeaconState{RandaoMixes: [][]byte{[]byte("foo")}})
	if err != nil {
		t.Fatal(err)
	}
	if a.sharedFieldReferences[randaoMixes].refs != 1 {
		t.Error("Expected a single reference for Randao mixes")
	}

	func() {
		// Create object in a different scope for GC
		b := a.Copy()
		if a.sharedFieldReferences[randaoMixes].refs != 2 {
			t.Error("Expected 2 references to randao mixes")
		}
		_ = b
	}()

	runtime.GC() // Should run finalizer on object b
	if a.sharedFieldReferences[randaoMixes].refs != 1 {
		t.Errorf("Expected 1 shared reference to randao mixes!")
	}

	b := a.Copy()
	if b.sharedFieldReferences[randaoMixes].refs != 2 {
		t.Error("Expected 2 shared references to randao mixes")
	}
	if err := b.UpdateRandaoMixesAtIndex(0, []byte("bar")); err != nil {
		t.Fatal(err)
	}
	if b.sharedFieldReferences[randaoMixes].refs != 1 || a.sharedFieldReferences[randaoMixes].refs != 1 {
		t.Error("Expected 1 shared reference to randao mix for both a and b")
	}
}
