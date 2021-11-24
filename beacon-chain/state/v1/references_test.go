package v1

import (
	"reflect"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestStateReferenceSharing_Finalizer(t *testing.T) {
	// This test showcases the logic on a the RandaoMixes field with the GC finalizer.

	a, err := InitializeFromProtoUnsafe(&ethpb.BeaconState{RandaoMixes: [][]byte{[]byte("foo")}})
	require.NoError(t, err)
	assert.Equal(t, uint(1), a.sharedFieldReferences[randaoMixes].Refs(), "Expected a single reference for RANDAO mixes")

	func() {
		// Create object in a different scope for GC
		b := a.Copy()
		assert.Equal(t, uint(2), a.sharedFieldReferences[randaoMixes].Refs(), "Expected 2 references to RANDAO mixes")
		_ = b
	}()

	runtime.GC() // Should run finalizer on object b
	assert.Equal(t, uint(1), a.sharedFieldReferences[randaoMixes].Refs(), "Expected 1 shared reference to RANDAO mixes!")

	copied := a.Copy()
	b, ok := copied.(*BeaconState)
	require.Equal(t, true, ok)
	assert.Equal(t, uint(2), b.sharedFieldReferences[randaoMixes].Refs(), "Expected 2 shared references to RANDAO mixes")
	require.NoError(t, b.UpdateRandaoMixesAtIndex(0, bytesutil.ToBytes32([]byte("bar"))))
	if b.sharedFieldReferences[randaoMixes].Refs() != 1 || a.sharedFieldReferences[randaoMixes].Refs() != 1 {
		t.Error("Expected 1 shared reference to RANDAO mix for both a and b")
	}
}

func TestStateReferenceCopy_NoUnexpectedRootsMutation(t *testing.T) {
	root1, root2 := bytesutil.ToBytes32([]byte("foo")), bytesutil.ToBytes32([]byte("bar"))
	a, err := InitializeFromProtoUnsafe(&ethpb.BeaconState{
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
	copied := a.Copy()
	b, ok := copied.(*BeaconState)
	require.Equal(t, true, ok)
	assertRefCount(t, a, blockRoots, 2)
	assertRefCount(t, a, stateRoots, 2)
	assertRefCount(t, b, blockRoots, 2)
	assertRefCount(t, b, stateRoots, 2)

	// Assert shared state.
	bRootsA := make([][]byte, len(a.BlockRoots()))
	for i, r := range a.BlockRoots() {
		tmp := r
		bRootsA[i] = tmp[:]
	}
	sRootsA := make([][]byte, len(a.StateRoots()))
	for i, r := range a.StateRoots() {
		tmp := r
		sRootsA[i] = tmp[:]
	}
	blockRootsB := b.BlockRoots()
	bRootsB := make([][]byte, len(b.BlockRoots()))
	for i, r := range b.BlockRoots() {
		tmp := r
		bRootsB[i] = tmp[:]
	}
	stateRootsB := b.StateRoots()
	sRootsB := make([][]byte, len(b.StateRoots()))
	for i, r := range b.StateRoots() {
		tmp := r
		sRootsB[i] = tmp[:]
	}
	assertValFound(t, bRootsA, root1[:])
	assertValFound(t, bRootsB, root1[:])
	assertValFound(t, sRootsA, root1[:])
	assertValFound(t, sRootsB, root1[:])

	// Mutator should only affect calling state: a.
	require.NoError(t, a.UpdateBlockRootAtIndex(0, root2))
	require.NoError(t, a.UpdateStateRootAtIndex(0, root2))

	// Assert no shared state mutation occurred only on state a (copy on write).
	bRootsA = make([][]byte, len(a.BlockRoots()))
	for i, r := range a.BlockRoots() {
		tmp := r
		bRootsA[i] = tmp[:]
	}
	sRootsA = make([][]byte, len(a.StateRoots()))
	for i, r := range a.StateRoots() {
		tmp := r
		sRootsA[i] = tmp[:]
	}
	blockRootsB = b.BlockRoots()
	bRootsB = make([][]byte, len(b.BlockRoots()))
	for i, r := range b.BlockRoots() {
		tmp := r
		bRootsB[i] = tmp[:]
	}
	stateRootsB = b.StateRoots()
	sRootsB = make([][]byte, len(b.StateRoots()))
	for i, r := range b.StateRoots() {
		tmp := r
		sRootsB[i] = tmp[:]
	}
	assertValNotFound(t, bRootsA, root1[:])
	assertValNotFound(t, sRootsA, root1[:])
	assertValFound(t, bRootsA, root2[:])
	assertValFound(t, sRootsA, root2[:])
	assertValFound(t, bRootsB, root1[:])
	assertValFound(t, sRootsB, root1[:])
	assert.DeepEqual(t, root2, a.BlockRoots()[0], "Expected mutation not found")
	assert.DeepEqual(t, root2, a.StateRoots()[0], "Expected mutation not found")
	assert.DeepEqual(t, root1, blockRootsB[0], "Unexpected mutation found")
	assert.DeepEqual(t, root1, stateRootsB[0], "Unexpected mutation found")

	// Copy on write happened, reference counters are reset.
	assertRefCount(t, a, blockRoots, 1)
	assertRefCount(t, a, stateRoots, 1)
	assertRefCount(t, b, blockRoots, 1)
	assertRefCount(t, b, stateRoots, 1)
}

func TestStateReferenceCopy_NoUnexpectedRandaoMutation(t *testing.T) {

	val1, val2 := bytesutil.ToBytes32([]byte("foo")), bytesutil.ToBytes32([]byte("bar"))
	a, err := InitializeFromProtoUnsafe(&ethpb.BeaconState{
		RandaoMixes: [][]byte{
			val1[:],
		},
	})
	require.NoError(t, err)
	assertRefCount(t, a, randaoMixes, 1)

	// Copy, increases reference count.
	copied := a.Copy()
	b, ok := copied.(*BeaconState)
	require.Equal(t, true, ok)
	assertRefCount(t, a, randaoMixes, 2)
	assertRefCount(t, b, randaoMixes, 2)

	// Assert shared state.
	mixesA := make([][]byte, len(a.RandaoMixes()))
	for i, r := range a.RandaoMixes() {
		tmp := r
		mixesA[i] = tmp[:]
	}
	mixesB := make([][]byte, len(b.RandaoMixes()))
	for i, r := range b.RandaoMixes() {
		tmp := r
		mixesB[i] = tmp[:]
	}
	assertValFound(t, mixesA, val1[:])
	assertValFound(t, mixesB, val1[:])

	// Mutator should only affect calling state: a.
	require.NoError(t, a.UpdateRandaoMixesAtIndex(0, val2))

	// Assert no shared state mutation occurred only on state a (copy on write).
	mixesA = make([][]byte, len(a.RandaoMixes()))
	for i, r := range a.RandaoMixes() {
		tmp := r
		mixesA[i] = tmp[:]
	}
	mixesB = make([][]byte, len(b.RandaoMixes()))
	for i, r := range b.RandaoMixes() {
		tmp := r
		mixesB[i] = tmp[:]
	}
	assertValFound(t, mixesA, val2[:])
	assertValNotFound(t, mixesA, val1[:])
	assertValFound(t, mixesB, val1[:])
	assertValNotFound(t, mixesB, val2[:])
	assertValFound(t, mixesB, val1[:])
	assertValNotFound(t, mixesB, val2[:])
	assert.DeepEqual(t, bytesutil.ToBytes32(val2[:]), a.RandaoMixes()[0], "Expected mutation not found")
	assert.DeepEqual(t, val1[:], mixesB[0], "Unexpected mutation found")

	// Copy on write happened, reference counters are reset.
	assertRefCount(t, a, randaoMixes, 1)
	assertRefCount(t, b, randaoMixes, 1)
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

	a, err := InitializeFromProtoUnsafe(&ethpb.BeaconState{})
	require.NoError(t, err)
	assertRefCount(t, a, previousEpochAttestations, 1)
	assertRefCount(t, a, currentEpochAttestations, 1)

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
	assertRefCount(t, a, previousEpochAttestations, 2)
	assertRefCount(t, a, currentEpochAttestations, 2)
	assertRefCount(t, b, previousEpochAttestations, 2)
	assertRefCount(t, b, currentEpochAttestations, 2)
	bPrevEpochAtts, err := b.PreviousEpochAttestations()
	require.NoError(t, err)
	bCurrEpochAtts, err := b.CurrentEpochAttestations()
	require.NoError(t, err)
	assert.Equal(t, 1, len(bPrevEpochAtts), "Unexpected number of attestations")
	assert.Equal(t, 1, len(bCurrEpochAtts), "Unexpected number of attestations")

	// Assert shared state.
	aCurrEpochAtts, err := a.CurrentEpochAttestations()
	require.NoError(t, err)
	curAttsA := aCurrEpochAtts
	aPrevEpochAtts, err := a.PreviousEpochAttestations()
	require.NoError(t, err)
	prevAttsA := aPrevEpochAtts
	bCurrEpochAtts, err = b.CurrentEpochAttestations()
	require.NoError(t, err)
	curAttsB := bCurrEpochAtts
	bPrevEpochAtts, err = b.PreviousEpochAttestations()
	require.NoError(t, err)
	prevAttsB := bPrevEpochAtts
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
	curAtt, err = a.CurrentEpochAttestations()
	require.NoError(t, err)
	assert.Equal(t, 2, len(curAtt), "Unexpected number of attestations")
	preAtt, err = a.PreviousEpochAttestations()
	require.NoError(t, err)
	assert.Equal(t, 2, len(preAtt), "Unexpected number of attestations")
	aCurrEpochAtts, err = a.CurrentEpochAttestations()
	require.NoError(t, err)
	aPrevEpochAtts, err = a.PreviousEpochAttestations()
	require.NoError(t, err)
	bCurrEpochAtts, err = b.CurrentEpochAttestations()
	require.NoError(t, err)
	bPrevEpochAtts, err = b.PreviousEpochAttestations()
	require.NoError(t, err)
	assertAttFound(aCurrEpochAtts, 1)
	assertAttFound(aPrevEpochAtts, 1)
	assertAttFound(aCurrEpochAtts, 2)
	assertAttFound(aPrevEpochAtts, 2)
	assertAttFound(bCurrEpochAtts, 1)
	assertAttFound(bPrevEpochAtts, 1)
	assertAttNotFound(bCurrEpochAtts, 2)
	assertAttNotFound(bPrevEpochAtts, 2)

	// Mutator should only affect calling state: a.
	applyToEveryAttestation := func(state *BeaconState) {
		// One MUST copy on write.
		atts = make([]*ethpb.PendingAttestation, len(state.currentEpochAttestations))
		copy(atts, state.currentEpochAttestations)
		state.currentEpochAttestations = atts
		currEpochAtts, err := state.CurrentEpochAttestations()
		require.NoError(t, err)
		for i := range currEpochAtts {
			att := ethpb.CopyPendingAttestation(state.currentEpochAttestations[i])
			att.AggregationBits = bitfield.NewBitlist(3)
			state.currentEpochAttestations[i] = att
		}

		atts = make([]*ethpb.PendingAttestation, len(state.previousEpochAttestations))
		copy(atts, state.previousEpochAttestations)
		state.previousEpochAttestations = atts
		prevEpochAtts, err := state.PreviousEpochAttestations()
		require.NoError(t, err)
		for i := range prevEpochAtts {
			att := ethpb.CopyPendingAttestation(state.previousEpochAttestations[i])
			att.AggregationBits = bitfield.NewBitlist(3)
			state.previousEpochAttestations[i] = att
		}
	}
	applyToEveryAttestation(a)

	aCurrEpochAtts, err = a.CurrentEpochAttestations()
	require.NoError(t, err)
	aPrevEpochAtts, err = a.PreviousEpochAttestations()
	require.NoError(t, err)
	bCurrEpochAtts, err = b.CurrentEpochAttestations()
	require.NoError(t, err)
	bPrevEpochAtts, err = b.PreviousEpochAttestations()
	require.NoError(t, err)
	// Assert no shared state mutation occurred only on state a (copy on write).
	assertAttFound(aCurrEpochAtts, 3)
	assertAttFound(aPrevEpochAtts, 3)
	assertAttNotFound(aCurrEpochAtts, 1)
	assertAttNotFound(aPrevEpochAtts, 1)
	assertAttNotFound(aCurrEpochAtts, 2)
	assertAttNotFound(aPrevEpochAtts, 2)
	// State b must be unaffected.
	assertAttNotFound(bCurrEpochAtts, 3)
	assertAttNotFound(bPrevEpochAtts, 3)
	assertAttFound(bCurrEpochAtts, 1)
	assertAttFound(bPrevEpochAtts, 1)
	assertAttNotFound(bCurrEpochAtts, 2)
	assertAttNotFound(bPrevEpochAtts, 2)

	// Copy on write happened, reference counters are reset.
	assertRefCount(t, a, currentEpochAttestations, 1)
	assertRefCount(t, b, currentEpochAttestations, 1)
	assertRefCount(t, a, previousEpochAttestations, 1)
	assertRefCount(t, b, previousEpochAttestations, 1)
}

func TestValidatorReferences_RemainsConsistent(t *testing.T) {
	a, err := InitializeFromProtoUnsafe(&ethpb.BeaconState{
		Validators: []*ethpb.Validator{
			{PublicKey: []byte{'A'}},
			{PublicKey: []byte{'B'}},
			{PublicKey: []byte{'C'}},
			{PublicKey: []byte{'D'}},
			{PublicKey: []byte{'E'}},
		},
	})
	require.NoError(t, err)

	// Create a second state.
	copied := a.Copy()
	b, ok := copied.(*BeaconState)
	require.Equal(t, true, ok)

	// Update First Validator.
	assert.NoError(t, a.UpdateValidatorAtIndex(0, &ethpb.Validator{PublicKey: []byte{'Z'}}))

	assert.DeepNotEqual(t, a.Validators()[0], b.Validators()[0], "validators are equal when they are supposed to be different")
	// Modify all validators from copied state.
	assert.NoError(t, b.ApplyToEveryValidator(func(idx int, val *ethpb.Validator) (bool, *ethpb.Validator, error) {
		return true, &ethpb.Validator{PublicKey: []byte{'V'}}, nil
	}))

	// Ensure reference is properly accounted for.
	assert.NoError(t, a.ReadFromEveryValidator(func(idx int, val state.ReadOnlyValidator) error {
		assert.NotEqual(t, bytesutil.ToBytes48([]byte{'V'}), val.PublicKey())
		return nil
	}))
}

// assertRefCount checks whether reference count for a given state
// at a given index is equal to expected amount.
func assertRefCount(t *testing.T, b *BeaconState, idx types.FieldIndex, want uint) {
	if cnt := b.sharedFieldReferences[idx].Refs(); cnt != want {
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
