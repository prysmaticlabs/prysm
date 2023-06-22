package state_native

import (
	"reflect"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/types"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestStateReferenceSharing_Finalizer_Phase0(t *testing.T) {
	// This test showcases the logic on the slashings field with the GC finalizer.

	s, err := InitializeFromProtoUnsafePhase0(&ethpb.BeaconState{Slashings: []uint64{123}})
	require.NoError(t, err)
	a, ok := s.(*BeaconState)
	require.Equal(t, true, ok)
	assert.Equal(t, uint(1), a.sharedFieldReferences[types.Slashings].Refs(), "Expected a single reference to slashings")

	func() {
		// Create object in a different scope for GC
		b := a.Copy()
		assert.Equal(t, uint(2), a.sharedFieldReferences[types.Slashings].Refs(), "Expected 2 references to slashings")
		_ = b
	}()

	runtime.GC() // Should run finalizer on object b
	assert.Equal(t, uint(1), a.sharedFieldReferences[types.Slashings].Refs(), "Expected 1 shared reference to slashings!")

	copied := a.Copy()
	b, ok := copied.(*BeaconState)
	require.Equal(t, true, ok)
	assert.Equal(t, uint(2), b.sharedFieldReferences[types.Slashings].Refs(), "Expected 2 shared references to slashings")
	require.NoError(t, b.UpdateSlashingsAtIndex(0, 456))
	if b.sharedFieldReferences[types.Slashings].Refs() != 1 || a.sharedFieldReferences[types.Slashings].Refs() != 1 {
		t.Error("Expected 1 shared reference to slashings for both a and b")
	}
}

func TestStateReferenceSharing_Finalizer_Altair(t *testing.T) {
	// This test showcases the logic on the slashings field with the GC finalizer.

	s, err := InitializeFromProtoUnsafeAltair(&ethpb.BeaconStateAltair{Slashings: []uint64{123}})
	require.NoError(t, err)
	a, ok := s.(*BeaconState)
	require.Equal(t, true, ok)
	assert.Equal(t, uint(1), a.sharedFieldReferences[types.Slashings].Refs(), "Expected a single reference to slashings")

	func() {
		// Create object in a different scope for GC
		b := a.Copy()
		assert.Equal(t, uint(2), a.sharedFieldReferences[types.Slashings].Refs(), "Expected 2 references to slashings")
		_ = b
	}()

	runtime.GC() // Should run finalizer on object b
	assert.Equal(t, uint(1), a.sharedFieldReferences[types.Slashings].Refs(), "Expected 1 shared reference to slashings!")

	copied := a.Copy()
	b, ok := copied.(*BeaconState)
	require.Equal(t, true, ok)
	assert.Equal(t, uint(2), b.sharedFieldReferences[types.Slashings].Refs(), "Expected 2 shared references to slashings")
	require.NoError(t, b.UpdateSlashingsAtIndex(0, 456))
	if b.sharedFieldReferences[types.Slashings].Refs() != 1 || a.sharedFieldReferences[types.Slashings].Refs() != 1 {
		t.Error("Expected 1 shared reference to slashings for both a and b")
	}
}

func TestStateReferenceSharing_Finalizer_Bellatrix(t *testing.T) {
	// This test showcases the logic on the slashings field with the GC finalizer.

	s, err := InitializeFromProtoUnsafeBellatrix(&ethpb.BeaconStateBellatrix{Slashings: []uint64{123}})
	require.NoError(t, err)
	a, ok := s.(*BeaconState)
	require.Equal(t, true, ok)
	assert.Equal(t, uint(1), a.sharedFieldReferences[types.Slashings].Refs(), "Expected a single reference to slashings")

	func() {
		// Create object in a different scope for GC
		b := a.Copy()
		assert.Equal(t, uint(2), a.sharedFieldReferences[types.Slashings].Refs(), "Expected 2 references to slashings")
		_ = b
	}()

	runtime.GC() // Should run finalizer on object b
	assert.Equal(t, uint(1), a.sharedFieldReferences[types.Slashings].Refs(), "Expected 1 shared reference to slashings!")

	copied := a.Copy()
	b, ok := copied.(*BeaconState)
	require.Equal(t, true, ok)
	assert.Equal(t, uint(2), b.sharedFieldReferences[types.Slashings].Refs(), "Expected 2 shared references to slashings")
	require.NoError(t, b.UpdateSlashingsAtIndex(0, 456))
	if b.sharedFieldReferences[types.Slashings].Refs() != 1 || a.sharedFieldReferences[types.Slashings].Refs() != 1 {
		t.Error("Expected 1 shared reference to slashings for both a and b")
	}
}

func TestStateReferenceSharing_Finalizer_Capella(t *testing.T) {
	// This test showcases the logic on the slashings field with the GC finalizer.

	s, err := InitializeFromProtoUnsafeCapella(&ethpb.BeaconStateCapella{Slashings: []uint64{123}})
	require.NoError(t, err)
	a, ok := s.(*BeaconState)
	require.Equal(t, true, ok)
	assert.Equal(t, uint(1), a.sharedFieldReferences[types.Slashings].Refs(), "Expected a single reference to slashings")

	func() {
		// Create object in a different scope for GC
		b := a.Copy()
		assert.Equal(t, uint(2), a.sharedFieldReferences[types.Slashings].Refs(), "Expected 2 references to slashings")
		_ = b
	}()

	runtime.GC() // Should run finalizer on object b
	assert.Equal(t, uint(1), a.sharedFieldReferences[types.Slashings].Refs(), "Expected 1 shared reference to slashings!")

	copied := a.Copy()
	b, ok := copied.(*BeaconState)
	require.Equal(t, true, ok)
	assert.Equal(t, uint(2), b.sharedFieldReferences[types.Slashings].Refs(), "Expected 2 shared references to slashings")
	require.NoError(t, b.UpdateSlashingsAtIndex(0, 456))
	if b.sharedFieldReferences[types.Slashings].Refs() != 1 || a.sharedFieldReferences[types.Slashings].Refs() != 1 {
		t.Error("Expected 1 shared reference to slashings for both a and b")
	}
}

func TestStateReferenceCopy_NoUnexpectedAttestationsMutation(t *testing.T) {
	assertAttFound := func(vals []*ethpb.PendingAttestation, val uint64) {
		for i := range vals {
			if reflect.DeepEqual(vals[i].AggregationBits, bitfield.NewBitlist(val)) {
				return
			}
		}
		t.Log(string(debug.Stack()))
		t.Fatalf("Expected attestation not found (%v), want: %v", vals, val)
	}
	assertAttNotFound := func(vals []*ethpb.PendingAttestation, val uint64) {
		for i := range vals {
			if reflect.DeepEqual(vals[i].AggregationBits, bitfield.NewBitlist(val)) {
				t.Log(string(debug.Stack()))
				t.Fatalf("Unexpected attestation found (%v): %v", vals, val)
				return
			}
		}
	}

	s, err := InitializeFromProtoUnsafePhase0(&ethpb.BeaconState{})
	require.NoError(t, err)
	a, ok := s.(*BeaconState)
	require.Equal(t, true, ok)
	assertRefCount(t, a, types.PreviousEpochAttestations, 1)
	assertRefCount(t, a, types.CurrentEpochAttestations, 1)

	// Update initial state.
	atts := []*ethpb.PendingAttestation{
		{AggregationBits: bitfield.NewBitlist(1)},
		{AggregationBits: bitfield.NewBitlist(2)},
	}
	a.setPreviousEpochAttestations(atts[:1])
	a.setCurrentEpochAttestations(atts[:1])
	curAtt, err := a.CurrentEpochAttestations()
	require.NoError(t, err)
	assert.Equal(t, 1, len(curAtt), "Unexpected number of attestations")
	preAtt, err := a.PreviousEpochAttestations()
	require.NoError(t, err)
	assert.Equal(t, 1, len(preAtt), "Unexpected number of attestations")

	// Copy, increases reference count.
	copied := a.Copy()
	b, ok := copied.(*BeaconState)
	require.Equal(t, true, ok)
	assertRefCount(t, a, types.PreviousEpochAttestations, 2)
	assertRefCount(t, a, types.CurrentEpochAttestations, 2)
	assertRefCount(t, b, types.PreviousEpochAttestations, 2)
	assertRefCount(t, b, types.CurrentEpochAttestations, 2)
	prevAtts, err := b.PreviousEpochAttestations()
	require.NoError(t, err)
	assert.Equal(t, 1, len(prevAtts), "Unexpected number of attestations")
	currAtts, err := b.CurrentEpochAttestations()
	require.NoError(t, err)
	assert.Equal(t, 1, len(currAtts), "Unexpected number of attestations")

	// Assert shared state.
	currAttsA, err := a.CurrentEpochAttestations()
	require.NoError(t, err)
	prevAttsA, err := a.PreviousEpochAttestations()
	require.NoError(t, err)
	currAttsB, err := b.CurrentEpochAttestations()
	require.NoError(t, err)
	prevAttsB, err := b.PreviousEpochAttestations()
	require.NoError(t, err)
	if len(currAttsA) != len(currAttsB) || len(currAttsA) < 1 {
		t.Errorf("Unexpected number of attestations, want: %v", 1)
	}
	if len(prevAttsA) != len(prevAttsB) || len(prevAttsA) < 1 {
		t.Errorf("Unexpected number of attestations, want: %v", 1)
	}
	assertAttFound(currAttsA, 1)
	assertAttFound(prevAttsA, 1)
	assertAttFound(currAttsB, 1)
	assertAttFound(prevAttsB, 1)

	// Extend state attestations.
	require.NoError(t, a.AppendCurrentEpochAttestations(atts[1]))
	require.NoError(t, a.AppendPreviousEpochAttestations(atts[1]))
	curAtt, err = a.CurrentEpochAttestations()
	require.NoError(t, err)
	assert.Equal(t, 2, len(curAtt), "Unexpected number of attestations")
	preAtt, err = a.PreviousEpochAttestations()
	require.NoError(t, err)
	assert.Equal(t, 2, len(preAtt), "Unexpected number of attestations")
	currAttsA, err = a.CurrentEpochAttestations()
	require.NoError(t, err)
	prevAttsA, err = a.PreviousEpochAttestations()
	require.NoError(t, err)
	currAttsB, err = b.CurrentEpochAttestations()
	require.NoError(t, err)
	prevAttsB, err = b.PreviousEpochAttestations()
	require.NoError(t, err)
	assertAttFound(currAttsA, 1)
	assertAttFound(prevAttsA, 1)
	assertAttFound(currAttsA, 2)
	assertAttFound(prevAttsA, 2)
	assertAttFound(currAttsB, 1)
	assertAttFound(prevAttsB, 1)
	assertAttNotFound(currAttsB, 2)
	assertAttNotFound(prevAttsB, 2)

	// Mutator should only affect calling state: a.
	applyToEveryAttestation := func(state *BeaconState) {
		// One MUST copy on write.
		currEpochAtts, err := state.CurrentEpochAttestations()
		require.NoError(t, err)
		atts = make([]*ethpb.PendingAttestation, len(currEpochAtts))
		copy(atts, currEpochAtts)
		state.setCurrentEpochAttestations(atts)
		currEpochAtts, err = state.CurrentEpochAttestations()
		require.NoError(t, err)
		for i := range currEpochAtts {
			att := ethpb.CopyPendingAttestation(currEpochAtts[i])
			att.AggregationBits = bitfield.NewBitlist(3)
			currEpochAtts[i] = att
		}
		state.setCurrentEpochAttestations(currEpochAtts)

		prevEpochAtts, err := state.PreviousEpochAttestations()
		require.NoError(t, err)
		atts = make([]*ethpb.PendingAttestation, len(prevEpochAtts))
		copy(atts, prevEpochAtts)
		state.setPreviousEpochAttestations(atts)
		prevEpochAtts, err = state.PreviousEpochAttestations()
		require.NoError(t, err)
		for i := range prevEpochAtts {
			att := ethpb.CopyPendingAttestation(prevEpochAtts[i])
			att.AggregationBits = bitfield.NewBitlist(3)
			prevEpochAtts[i] = att
		}
		state.setPreviousEpochAttestations(prevEpochAtts)
	}
	applyToEveryAttestation(a)

	// Assert no shared state mutation occurred only on state a (copy on write).
	currAttsA, err = a.CurrentEpochAttestations()
	require.NoError(t, err)
	prevAttsA, err = a.PreviousEpochAttestations()
	require.NoError(t, err)
	assertAttFound(currAttsA, 3)
	assertAttFound(prevAttsA, 3)
	assertAttNotFound(currAttsA, 1)
	assertAttNotFound(prevAttsA, 1)
	assertAttNotFound(currAttsA, 2)
	assertAttNotFound(prevAttsA, 2)
	// State b must be unaffected.
	currAttsB, err = b.CurrentEpochAttestations()
	require.NoError(t, err)
	prevAttsB, err = b.PreviousEpochAttestations()
	require.NoError(t, err)
	assertAttNotFound(currAttsB, 3)
	assertAttNotFound(prevAttsB, 3)
	assertAttFound(currAttsB, 1)
	assertAttFound(prevAttsB, 1)
	assertAttNotFound(currAttsB, 2)
	assertAttNotFound(prevAttsB, 2)

	// Copy on write happened, reference counters are reset.
	assertRefCount(t, a, types.CurrentEpochAttestations, 1)
	assertRefCount(t, b, types.CurrentEpochAttestations, 1)
	assertRefCount(t, a, types.PreviousEpochAttestations, 1)
	assertRefCount(t, b, types.PreviousEpochAttestations, 1)
}

// assertRefCount checks whether reference count for a given state
// at a given index is equal to expected amount.
func assertRefCount(t *testing.T, b *BeaconState, idx types.FieldIndex, want uint) {
	if cnt := b.sharedFieldReferences[idx].Refs(); cnt != want {
		t.Errorf("Unexpected count of references for index %d, want: %v, got: %v", idx, want, cnt)
	}
}
