package state

import (
	"reflect"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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

func TestStateReferenceCopy_NoUnexpectedRootsMutation(t *testing.T) {
	root1, root2 := bytesutil.ToBytes32([]byte("foo")), bytesutil.ToBytes32([]byte("bar"))
	a, err := InitializeFromProtoUnsafe(&p2ppb.BeaconState{
		BlockRoots: [][]byte{
			root1[:],
		},
		StateRoots: [][]byte{
			root1[:],
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertRefCount(t, a, blockRoots, 1)
	assertRefCount(t, a, stateRoots, 1)

	// Copy, increases reference count.
	b := a.Copy()
	assertRefCount(t, a, blockRoots, 2)
	assertRefCount(t, a, stateRoots, 2)
	assertRefCount(t, b, blockRoots, 2)
	assertRefCount(t, b, stateRoots, 2)
	if len(b.state.GetBlockRoots()) != 1 {
		t.Error("No block roots found")
	}
	if len(b.state.GetStateRoots()) != 1 {
		t.Error("No state roots found")
	}

	// Assert shared state.
	blockRootsA := a.state.GetBlockRoots()
	stateRootsA := a.state.GetStateRoots()
	blockRootsB := b.state.GetBlockRoots()
	stateRootsB := b.state.GetStateRoots()
	if len(blockRootsA) != len(blockRootsB) || len(blockRootsA) < 1 {
		t.Errorf("Unexpected number of block roots, want: %v", 1)
	}
	if len(stateRootsA) != len(stateRootsB) || len(stateRootsA) < 1 {
		t.Errorf("Unexpected number of state roots, want: %v", 1)
	}
	assertValFound(t, blockRootsA, root1[:])
	assertValFound(t, blockRootsB, root1[:])
	assertValFound(t, stateRootsA, root1[:])
	assertValFound(t, stateRootsB, root1[:])

	// Mutator should only affect calling state: a.
	err = a.UpdateBlockRootAtIndex(0, root2)
	if err != nil {
		t.Fatal(err)
	}
	err = a.UpdateStateRootAtIndex(0, root2)
	if err != nil {
		t.Fatal(err)
	}

	// Assert no shared state mutation occurred only on state a (copy on write).
	assertValNotFound(t, a.state.GetBlockRoots(), root1[:])
	assertValNotFound(t, a.state.GetStateRoots(), root1[:])
	assertValFound(t, a.state.GetBlockRoots(), root2[:])
	assertValFound(t, a.state.GetStateRoots(), root2[:])
	assertValFound(t, b.state.GetBlockRoots(), root1[:])
	assertValFound(t, b.state.GetStateRoots(), root1[:])
	if len(blockRootsA) != len(blockRootsB) || len(blockRootsA) < 1 {
		t.Errorf("Unexpected number of block roots, want: %v", 1)
	}
	if len(stateRootsA) != len(stateRootsB) || len(stateRootsA) < 1 {
		t.Errorf("Unexpected number of state roots, want: %v", 1)
	}
	if !reflect.DeepEqual(a.state.GetBlockRoots()[0], root2[:]) {
		t.Errorf("Expected mutation not found")
	}
	if !reflect.DeepEqual(a.state.GetStateRoots()[0], root2[:]) {
		t.Errorf("Expected mutation not found")
	}
	if !reflect.DeepEqual(blockRootsB[0], root1[:]) {
		t.Errorf("Unexpected mutation found")
	}
	if !reflect.DeepEqual(stateRootsB[0], root1[:]) {
		t.Errorf("Unexpected mutation found")
	}

	// Copy on write happened, reference counters are reset.
	assertRefCount(t, a, blockRoots, 1)
	assertRefCount(t, a, stateRoots, 1)
	assertRefCount(t, b, blockRoots, 1)
	assertRefCount(t, b, stateRoots, 1)
}

func TestStateReferenceCopy_NoUnexpectedRandaoMutation(t *testing.T) {

	val1, val2 := []byte("foo"), []byte("bar")
	a, err := InitializeFromProtoUnsafe(&p2ppb.BeaconState{
		RandaoMixes: [][]byte{
			val1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertRefCount(t, a, randaoMixes, 1)

	// Copy, increases reference count.
	b := a.Copy()
	assertRefCount(t, a, randaoMixes, 2)
	assertRefCount(t, b, randaoMixes, 2)
	if len(b.state.GetRandaoMixes()) != 1 {
		t.Error("No randao mixes found")
	}

	// Assert shared state.
	mixesA := a.state.GetRandaoMixes()
	mixesB := b.state.GetRandaoMixes()
	if len(mixesA) != len(mixesB) || len(mixesA) < 1 {
		t.Errorf("Unexpected number of mix values, want: %v", 1)
	}
	assertValFound(t, mixesA, val1)
	assertValFound(t, mixesB, val1)

	// Mutator should only affect calling state: a.
	err = a.UpdateRandaoMixesAtIndex(0, val2)
	if err != nil {
		t.Fatal(err)
	}

	// Assert no shared state mutation occurred only on state a (copy on write).
	if len(mixesA) != len(mixesB) || len(mixesA) < 1 {
		t.Errorf("Unexpected number of mix values, want: %v", 1)
	}
	assertValFound(t, a.state.GetRandaoMixes(), val2)
	assertValNotFound(t, a.state.GetRandaoMixes(), val1)
	assertValFound(t, b.state.GetRandaoMixes(), val1)
	assertValNotFound(t, b.state.GetRandaoMixes(), val2)
	assertValFound(t, mixesB, val1)
	assertValNotFound(t, mixesB, val2)
	if !reflect.DeepEqual(a.state.GetRandaoMixes()[0], val2) {
		t.Errorf("Expected mutation not found")
	}
	if !reflect.DeepEqual(mixesB[0], val1) {
		t.Errorf("Unexpected mutation found")
	}

	// Copy on write happened, reference counters are reset.
	assertRefCount(t, a, randaoMixes, 1)
	assertRefCount(t, b, randaoMixes, 1)
}

func TestStateReferenceCopy_NoUnexpectedAttestationsMutation(t *testing.T) {

	assertAttFound := func(vals []*p2ppb.PendingAttestation, val uint64) {
		for i := range vals {
			if reflect.DeepEqual(vals[i].AggregationBits, bitfield.NewBitlist(val)) {
				return
			}
		}
		t.Log(string(debug.Stack()))
		t.Fatalf("Expected attestation not found (%v), want: %v", vals, val)
	}
	assertAttNotFound := func(vals []*p2ppb.PendingAttestation, val uint64) {
		for i := range vals {
			if reflect.DeepEqual(vals[i].AggregationBits, bitfield.NewBitlist(val)) {
				t.Log(string(debug.Stack()))
				t.Fatalf("Unexpected attestation found (%v): %v", vals, val)
				return
			}
		}
	}

	a, err := InitializeFromProtoUnsafe(&p2ppb.BeaconState{})
	if err != nil {
		t.Fatal(err)
	}
	assertRefCount(t, a, previousEpochAttestations, 1)
	assertRefCount(t, a, currentEpochAttestations, 1)

	// Update initial state.
	atts := []*p2ppb.PendingAttestation{
		{AggregationBits: bitfield.NewBitlist(1)},
		{AggregationBits: bitfield.NewBitlist(2)},
	}
	if err := a.SetPreviousEpochAttestations(atts[:1]); err != nil {
		t.Fatal(err)
	}
	if err := a.SetCurrentEpochAttestations(atts[:1]); err != nil {
		t.Fatal(err)
	}
	if len(a.CurrentEpochAttestations()) != 1 {
		t.Errorf("Unexpected number of attestations, want: %v", 1)
	}
	if len(a.PreviousEpochAttestations()) != 1 {
		t.Errorf("Unexpected number of attestations, want: %v", 1)
	}

	// Copy, increases reference count.
	b := a.Copy()
	assertRefCount(t, a, previousEpochAttestations, 2)
	assertRefCount(t, a, currentEpochAttestations, 2)
	assertRefCount(t, b, previousEpochAttestations, 2)
	assertRefCount(t, b, currentEpochAttestations, 2)
	if len(b.state.GetPreviousEpochAttestations()) != 1 {
		t.Errorf("Unexpected number of attestations, want: %v", 1)
	}
	if len(b.state.GetCurrentEpochAttestations()) != 1 {
		t.Errorf("Unexpected number of attestations, want: %v", 1)
	}

	// Assert shared state.
	curAttsA := a.state.GetCurrentEpochAttestations()
	prevAttsA := a.state.GetPreviousEpochAttestations()
	curAttsB := b.state.GetCurrentEpochAttestations()
	prevAttsB := b.state.GetPreviousEpochAttestations()
	if len(curAttsA) != len(curAttsB) || len(curAttsA) < 1 {
		t.Errorf("Unexpected number of attestations, want: %v", 1)
	}
	if len(prevAttsA) != len(prevAttsB) || len(prevAttsA) < 1 {
		t.Errorf("Unexpected number of attestations, want: %v", 1)
	}
	assertAttFound(curAttsA, 1)
	assertAttFound(prevAttsA, 1)
	assertAttFound(curAttsB, 1)
	assertAttFound(prevAttsB, 1)

	// Extends state a attestations.
	if err := a.AppendCurrentEpochAttestations(atts[1]); err != nil {
		t.Fatal(err)
	}
	if err := a.AppendPreviousEpochAttestations(atts[1]); err != nil {
		t.Fatal(err)
	}
	if len(a.CurrentEpochAttestations()) != 2 {
		t.Errorf("Unexpected number of attestations, want: %v", 2)
	}
	if len(a.PreviousEpochAttestations()) != 2 {
		t.Errorf("Unexpected number of attestations, want: %v", 2)
	}
	assertAttFound(a.state.GetCurrentEpochAttestations(), 1)
	assertAttFound(a.state.GetPreviousEpochAttestations(), 1)
	assertAttFound(a.state.GetCurrentEpochAttestations(), 2)
	assertAttFound(a.state.GetPreviousEpochAttestations(), 2)
	assertAttFound(b.state.GetCurrentEpochAttestations(), 1)
	assertAttFound(b.state.GetPreviousEpochAttestations(), 1)
	assertAttNotFound(b.state.GetCurrentEpochAttestations(), 2)
	assertAttNotFound(b.state.GetPreviousEpochAttestations(), 2)

	// Mutator should only affect calling state: a.
	applyToEveryAttestation := func(state *p2ppb.BeaconState) {
		// One MUST copy on write.
		atts = make([]*p2ppb.PendingAttestation, len(state.CurrentEpochAttestations))
		copy(atts, state.CurrentEpochAttestations)
		state.CurrentEpochAttestations = atts
		for i := range state.GetCurrentEpochAttestations() {
			att := CopyPendingAttestation(state.CurrentEpochAttestations[i])
			att.AggregationBits = bitfield.NewBitlist(3)
			state.CurrentEpochAttestations[i] = att
		}

		atts = make([]*p2ppb.PendingAttestation, len(state.PreviousEpochAttestations))
		copy(atts, state.PreviousEpochAttestations)
		state.PreviousEpochAttestations = atts
		for i := range state.GetPreviousEpochAttestations() {
			att := CopyPendingAttestation(state.PreviousEpochAttestations[i])
			att.AggregationBits = bitfield.NewBitlist(3)
			state.PreviousEpochAttestations[i] = att
		}
	}
	applyToEveryAttestation(a.state)

	// Assert no shared state mutation occurred only on state a (copy on write).
	assertAttFound(a.state.GetCurrentEpochAttestations(), 3)
	assertAttFound(a.state.GetPreviousEpochAttestations(), 3)
	assertAttNotFound(a.state.GetCurrentEpochAttestations(), 1)
	assertAttNotFound(a.state.GetPreviousEpochAttestations(), 1)
	assertAttNotFound(a.state.GetCurrentEpochAttestations(), 2)
	assertAttNotFound(a.state.GetPreviousEpochAttestations(), 2)
	// State b must be unaffected.
	assertAttNotFound(b.state.GetCurrentEpochAttestations(), 3)
	assertAttNotFound(b.state.GetPreviousEpochAttestations(), 3)
	assertAttFound(b.state.GetCurrentEpochAttestations(), 1)
	assertAttFound(b.state.GetPreviousEpochAttestations(), 1)
	assertAttNotFound(b.state.GetCurrentEpochAttestations(), 2)
	assertAttNotFound(b.state.GetPreviousEpochAttestations(), 2)

	// Copy on write happened, reference counters are reset.
	assertRefCount(t, a, currentEpochAttestations, 1)
	assertRefCount(t, b, currentEpochAttestations, 1)
	assertRefCount(t, a, previousEpochAttestations, 1)
	assertRefCount(t, b, previousEpochAttestations, 1)
}

// assertRefCount checks whether reference count for a given state
// at a given index is equal to expected amount.
func assertRefCount(t *testing.T, b *BeaconState, idx fieldIndex, want uint) {
	if cnt := b.sharedFieldReferences[idx].refs; cnt != want {
		t.Errorf("Unexpected count of references for index %d, want: %v, got: %v", idx, want, cnt)
	}
}

// assertValFound checks whether item with a given value exists in list.
func assertValFound(t *testing.T, vals [][]byte, val []byte) {
	for i := range vals {
		if reflect.DeepEqual(vals[i], val) {
			return
		}
	}
	t.Log(string(debug.Stack()))
	t.Fatalf("Expected value not found (%v), want: %v", vals, val)
}

// assertValNotFound checks whether item with a given value doesn't exist in list.
func assertValNotFound(t *testing.T, vals [][]byte, val []byte) {
	for i := range vals {
		if reflect.DeepEqual(vals[i], val) {
			t.Log(string(debug.Stack()))
			t.Errorf("Unexpected value found (%v),: %v", vals, val)
			return
		}
	}
}
