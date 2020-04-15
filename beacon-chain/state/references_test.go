package state

import (
	"reflect"
	"runtime"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
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
	if err := b.UpdateRandaoMixesAtIndex([]byte("bar"), 0); err != nil {
		t.Fatal(err)
	}
	if b.sharedFieldReferences[randaoMixes].refs != 1 || a.sharedFieldReferences[randaoMixes].refs != 1 {
		t.Error("Expected 1 shared reference to randao mix for both a and b")
	}
}

func TestStateReferenceCopy_NoUnexpectedValidatorMutation(t *testing.T) {
	// Assert that feature is enabled.
	if cfg := featureconfig.Get(); !cfg.EnableStateRefCopy {
		cfg.EnableStateRefCopy = true
		featureconfig.Init(cfg)
		defer func() {
			cfg := featureconfig.Get()
			cfg.EnableStateRefCopy = false
			featureconfig.Init(cfg)
		}()
	}

	a, err := InitializeFromProtoUnsafe(&p2ppb.BeaconState{})
	if err != nil {
		t.Fatal(err)
	}

	assertRefCount(t, a, validators, 1)

	// Add validator before copying state (so that a and b have shared data).
	pubKey1, pubKey2 := [48]byte{29}, [48]byte{31}
	err = a.AppendValidator(&eth.Validator{
		PublicKey: pubKey1[:],
	})
	if len(a.state.GetValidators()) != 1 {
		t.Error("No validators found")
	}

	// Copy, increases reference count.
	b := a.Copy()
	assertRefCount(t, a, validators, 2)
	assertRefCount(t, b, validators, 2)
	if len(b.state.GetValidators()) != 1 {
		t.Error("No validators found")
	}

	hasValidatorWithPubKey := func(state *p2ppb.BeaconState, key [48]byte) bool {
		for _, val := range state.GetValidators() {
			if reflect.DeepEqual(val.PublicKey, key[:]) {
				return true
			}
		}
		return false
	}

	err = a.AppendValidator(&eth.Validator{
		PublicKey: pubKey2[:],
	})
	if err != nil {
		t.Fatal(err)
	}

	// Copy on write happened, reference counters are reset.
	assertRefCount(t, a, validators, 1)
	assertRefCount(t, b, validators, 1)

	valsA := a.state.GetValidators()
	valsB := b.state.GetValidators()
	if len(valsA) != 2 {
		t.Errorf("Unexpected number of validators, want: %v, got: %v", 2, len(valsA))
	}
	// Both validators are known to a.
	if !hasValidatorWithPubKey(a.state, pubKey1) {
		t.Errorf("Expected validator not found, want: %v", pubKey1)
	}
	if !hasValidatorWithPubKey(a.state, pubKey2) {
		t.Errorf("Expected validator not found, want: %v", pubKey2)
	}
	// Only one validator is known to b.
	if !hasValidatorWithPubKey(b.state, pubKey1) {
		t.Errorf("Expected validator not found, want: %v", pubKey1)
	}
	if hasValidatorWithPubKey(b.state, pubKey2) {
		t.Errorf("Unexpected validator found: %v", pubKey2)
	}
	if len(valsA) == len(valsB) {
		t.Error("Unexpected state mutation")
	}

	// Make sure that function applied to all validators in one state, doesn't affect another.
	changedBalance := uint64(1)
	for i, val := range valsA {
		if val.EffectiveBalance == changedBalance {
			t.Errorf("Unexpected effective balance, want: %v, got: %v", 0, valsA[i].EffectiveBalance)
		}
	}
	for i, val := range valsB {
		if val.EffectiveBalance == changedBalance {
			t.Errorf("Unexpected effective balance, want: %v, got: %v", 0, valsB[i].EffectiveBalance)
		}
	}
	// Applied to a, a and b share reference to the first validator, which shouldn't cause issues.
	err = a.ApplyToEveryValidator(func(idx int, val *eth.Validator) (b bool, err error) {
		return true, nil
	}, func(idx int, val *eth.Validator) error {
		val.EffectiveBalance = 1
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, val := range valsA {
		if val.EffectiveBalance != changedBalance {
			t.Errorf("Unexpected effective balance, want: %v, got: %v", changedBalance, valsA[i].EffectiveBalance)
		}
	}
	for i, val := range valsB {
		if val.EffectiveBalance == changedBalance {
			t.Errorf("Unexpected mutation of effective balance, want: %v, got: %v", 0, valsB[i].EffectiveBalance)
		}
	}
}

func TestStateReferenceCopy_NoUnexpectedRootsMutation(t *testing.T) {
	// Assert that feature is enabled.
	if cfg := featureconfig.Get(); !cfg.EnableStateRefCopy {
		cfg.EnableStateRefCopy = true
		featureconfig.Init(cfg)
		defer func() {
			cfg := featureconfig.Get()
			cfg.EnableStateRefCopy = false
			featureconfig.Init(cfg)
		}()
	}

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

	hasBlockRootWithKey := func(roots [][]byte, key [32]byte) bool {
		for _, root := range roots {
			if reflect.DeepEqual(root, key[:]) {
				return true
			}
		}
		return false
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
	if !hasBlockRootWithKey(a.state.GetStateRoots(), root1) {
		t.Errorf("Expected state root not found, want: %v", root1)
	}
	if !hasBlockRootWithKey(b.state.GetStateRoots(), root1) {
		t.Errorf("Expected state root not found, want: %v", root1)
	}

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
	if hasBlockRootWithKey(a.state.GetBlockRoots(), root1) {
		t.Errorf("Unexpected block root found: %v", root1)
	}
	if hasBlockRootWithKey(a.state.GetStateRoots(), root1) {
		t.Errorf("Unexpected state root found: %v", root1)
	}
	if !hasBlockRootWithKey(a.state.GetBlockRoots(), root2) {
		t.Errorf("Expected block root not found, want: %v", root2)
	}
	if !hasBlockRootWithKey(a.state.GetStateRoots(), root2) {
		t.Errorf("Expected state root not found, want: %v", root2)
	}
	if !hasBlockRootWithKey(b.state.GetBlockRoots(), root1) {
		t.Errorf("Expected block root not found, want: %v", root1)
	}
	if !hasBlockRootWithKey(b.state.GetStateRoots(), root1) {
		t.Errorf("Expected state root not found, want: %v", root1)
	}
	// Get updated pointers to data.
	blockRootsA = a.state.GetBlockRoots()
	stateRootsA = a.state.GetStateRoots()
	blockRootsB = b.state.GetBlockRoots()
	stateRootsB = b.state.GetStateRoots()
	if len(blockRootsA) != len(blockRootsB) || len(blockRootsA) < 1 {
		t.Errorf("Unexpected number of block roots, want: %v", 1)
	}
	if len(stateRootsA) != len(stateRootsB) || len(stateRootsA) < 1 {
		t.Errorf("Unexpected number of state roots, want: %v", 1)
	}
	if !reflect.DeepEqual(blockRootsA[0], root2[:]) {
		t.Errorf("Expected mutation not found")
	}
	if !reflect.DeepEqual(stateRootsA[0], root2[:]) {
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
	// Assert that feature is enabled.
	if cfg := featureconfig.Get(); !cfg.EnableStateRefCopy {
		cfg.EnableStateRefCopy = true
		featureconfig.Init(cfg)
		defer func() {
			cfg := featureconfig.Get()
			cfg.EnableStateRefCopy = false
			featureconfig.Init(cfg)
		}()
	}

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

	assertValFound := func(key []byte, vals [][]byte) {
		for _, val := range vals {
			if reflect.DeepEqual(val, key) {
				return
			}
		}
		t.Errorf("Expected key not found (%v), want: %v", vals, key)
	}
	assertValNotFound := func(key []byte, vals [][]byte) {
		for _, val := range vals {
			if reflect.DeepEqual(val, key) {
				t.Errorf("Unexpected key found (%v), key: %v", vals, key)
				return
			}
		}
	}
	// Assert shared state.
	mixesA := a.state.GetRandaoMixes()
	mixesB := b.state.GetRandaoMixes()
	if len(mixesA) != len(mixesB) || len(mixesA) < 1 {
		t.Errorf("Unexpected number of mix values, want: %v", 1)
	}
	assertValFound(val1, mixesA)
	assertValFound(val1, mixesB)

	// Mutator should only affect calling state: a.
	err = a.UpdateRandaoMixesAtIndex(val2, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Assert no shared state mutation occurred only on state a (copy on write).
	if len(mixesA) != len(mixesB) || len(mixesA) < 1 {
		t.Errorf("Unexpected number of mix values, want: %v", 1)
	}
	assertValFound(val2, a.state.GetRandaoMixes())
	assertValNotFound(val1, a.state.GetRandaoMixes())
	assertValFound(val1, b.state.GetRandaoMixes())
	assertValNotFound(val2, b.state.GetRandaoMixes())
	assertValFound(val1, mixesB)
	assertValNotFound(val2, mixesB)
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

// assertRefCount checks whether reference count for a given state
// at a given index is equal to expected amount.
func assertRefCount(t *testing.T, b *BeaconState, idx fieldIndex, want uint) {
	if cnt := b.sharedFieldReferences[idx].refs; cnt != want {
		t.Errorf("Unexpected count of references for index %d, want: %v, got: %v", idx, want, cnt)
	}
}
