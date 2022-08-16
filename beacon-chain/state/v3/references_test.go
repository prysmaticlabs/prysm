package v3

import (
	"reflect"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestStateReferenceSharing_Finalizer(t *testing.T) {
	// This test showcases the logic on a the RandaoMixes field with the GC finalizer.

	s, err := InitializeFromProtoUnsafe(&ethpb.BeaconStateBellatrix{RandaoMixes: [][]byte{[]byte("foo")}})
	require.NoError(t, err)
	a, ok := s.(*BeaconState)
	require.Equal(t, true, ok)
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
	require.NoError(t, b.UpdateRandaoMixesAtIndex(0, []byte("bar")))
	if b.sharedFieldReferences[randaoMixes].Refs() != 1 || a.sharedFieldReferences[randaoMixes].Refs() != 1 {
		t.Error("Expected 1 shared reference to RANDAO mix for both a and b")
	}
}

func TestStateReferenceCopy_NoUnexpectedRootsMutation(t *testing.T) {
	root1, root2 := bytesutil.ToBytes32([]byte("foo")), bytesutil.ToBytes32([]byte("bar"))
	s, err := InitializeFromProtoUnsafe(&ethpb.BeaconStateBellatrix{
		BlockRoots: [][]byte{
			root1[:],
		},
		StateRoots: [][]byte{
			root1[:],
		},
	})
	require.NoError(t, err)
	a, ok := s.(*BeaconState)
	require.Equal(t, true, ok)
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
	s, err := InitializeFromProtoUnsafe(&ethpb.BeaconStateBellatrix{
		RandaoMixes: [][]byte{
			val1,
		},
	})
	require.NoError(t, err)
	a, ok := s.(*BeaconState)
	require.Equal(t, true, ok)
	require.NoError(t, err)
	assertRefCount(t, a, randaoMixes, 1)

	// Copy, increases reference count.
	copied := a.Copy()
	b, ok := copied.(*BeaconState)
	require.Equal(t, true, ok)
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

func TestValidatorReferences_RemainsConsistent(t *testing.T) {
	s, err := InitializeFromProtoUnsafe(&ethpb.BeaconStateBellatrix{
		Validators: []*ethpb.Validator{
			{PublicKey: []byte{'A'}},
			{PublicKey: []byte{'B'}},
			{PublicKey: []byte{'C'}},
			{PublicKey: []byte{'D'}},
			{PublicKey: []byte{'E'}},
		},
	})
	require.NoError(t, err)
	a, ok := s.(*BeaconState)
	require.Equal(t, true, ok)

	// Create a second state.
	copied := a.Copy()
	b, ok := copied.(*BeaconState)
	require.Equal(t, true, ok)

	// Update First Validator.
	assert.NoError(t, a.UpdateValidatorAtIndex(0, &ethpb.Validator{PublicKey: []byte{'Z'}}))

	assert.DeepNotEqual(t, a.state.Validators[0], b.state.Validators[0], "validators are equal when they are supposed to be different")
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
