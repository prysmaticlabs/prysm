package state

import (
	"reflect"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStateReferenceSharing_Finalizer(t *testing.T) {
	// This test showcases the logic on a the RandaoMixes field with the GC finalizer.

	a, err := InitializeFromProtoUnsafe(&p2ppb.BeaconState{RandaoMixes: [][]byte{[]byte("foo")}})
	require.NoError(t, err)
	assert.Equal(t, uint(1), a.sharedFieldReferences[randaoMixes].refs, "Expected a single reference for RANDAO mixes")

	func() {
		// Create object in a different scope for GC
		b := a.Copy()
		assert.Equal(t, uint(2), a.sharedFieldReferences[randaoMixes].refs, "Expected 2 references to RANDAO mixes")
		_ = b
	}()

	runtime.GC() // Should run finalizer on object b
	assert.Equal(t, uint(1), a.sharedFieldReferences[randaoMixes].refs, "Expected 1 shared reference to RANDAO mixes!")

	b := a.Copy()
	assert.Equal(t, uint(2), b.sharedFieldReferences[randaoMixes].refs, "Expected 2 shared references to RANDAO mixes")
	require.NoError(t, b.UpdateRandaoMixesAtIndex(0, []byte("bar")))
	if b.sharedFieldReferences[randaoMixes].refs != 1 || a.sharedFieldReferences[randaoMixes].refs != 1 {
		t.Error("Expected 1 shared reference to RANDAO mix for both a and b")
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
	require.NoError(t, err)
	assertRefCount(t, a, blockRoots, 1)
	assertRefCount(t, a, stateRoots, 1)

	// Copy, increases reference count.
	b := a.Copy()
	assertRefCount(t, a, blockRoots, 2)
	assertRefCount(t, a, stateRoots, 2)
	assertRefCount(t, b, blockRoots, 2)
	assertRefCount(t, b, stateRoots, 2)
	assert.Equal(t, 1, len(b.state.GetBlockRoots()), "No block roots found")
	assert.Equal(t, 1, len(b.state.GetStateRoots()), "No state roots found")

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
	require.NoError(t, a.UpdateBlockRootAtIndex(0, root2))
	require.NoError(t, a.UpdateStateRootAtIndex(0, root2))

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
	assert.DeepEqual(t, root2[:], a.state.GetBlockRoots()[0], "Expected mutation not found")
	assert.DeepEqual(t, root2[:], a.state.GetStateRoots()[0], "Expected mutation not found")
	assert.DeepEqual(t, root1[:], blockRootsB[0], "Unexpected mutation found")
	assert.DeepEqual(t, root1[:], stateRootsB[0], "Unexpected mutation found")

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
	require.NoError(t, err)
	assertRefCount(t, a, randaoMixes, 1)

	// Copy, increases reference count.
	b := a.Copy()
	assertRefCount(t, a, randaoMixes, 2)
	assertRefCount(t, b, randaoMixes, 2)
	assert.Equal(t, 1, len(b.state.GetRandaoMixes()), "No randao mixes found")

	// Assert shared state.
	mixesA := a.state.GetRandaoMixes()
	mixesB := b.state.GetRandaoMixes()
	if len(mixesA) != len(mixesB) || len(mixesA) < 1 {
		t.Errorf("Unexpected number of mix values, want: %v", 1)
	}
	assertValFound(t, mixesA, val1)
	assertValFound(t, mixesB, val1)

	// Mutator should only affect calling state: a.
	require.NoError(t, a.UpdateRandaoMixesAtIndex(0, val2))

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
	assert.DeepEqual(t, val2, a.state.GetRandaoMixes()[0], "Expected mutation not found")
	assert.DeepEqual(t, val1, mixesB[0], "Unexpected mutation found")

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
	require.NoError(t, err)
	assertRefCount(t, a, previousEpochAttestations, 1)
	assertRefCount(t, a, currentEpochAttestations, 1)

	// Update initial state.
	atts := []*p2ppb.PendingAttestation{
		{AggregationBits: bitfield.NewBitlist(1)},
		{AggregationBits: bitfield.NewBitlist(2)},
	}
	require.NoError(t, a.SetPreviousEpochAttestations(atts[:1]))
	require.NoError(t, a.SetCurrentEpochAttestations(atts[:1]))
	assert.Equal(t, 1, len(a.CurrentEpochAttestations()), "Unexpected number of attestations")
	assert.Equal(t, 1, len(a.PreviousEpochAttestations()), "Unexpected number of attestations")

	// Copy, increases reference count.
	b := a.Copy()
	assertRefCount(t, a, previousEpochAttestations, 2)
	assertRefCount(t, a, currentEpochAttestations, 2)
	assertRefCount(t, b, previousEpochAttestations, 2)
	assertRefCount(t, b, currentEpochAttestations, 2)
	assert.Equal(t, 1, len(b.state.GetPreviousEpochAttestations()), "Unexpected number of attestations")
	assert.Equal(t, 1, len(b.state.GetCurrentEpochAttestations()), "Unexpected number of attestations")

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
	require.NoError(t, a.AppendCurrentEpochAttestations(atts[1]))
	require.NoError(t, a.AppendPreviousEpochAttestations(atts[1]))
	assert.Equal(t, 2, len(a.CurrentEpochAttestations()), "Unexpected number of attestations")
	assert.Equal(t, 2, len(a.PreviousEpochAttestations()), "Unexpected number of attestations")
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
