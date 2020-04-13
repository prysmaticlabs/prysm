package state

import (
	"reflect"
	"runtime"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

func TestStateReferenceSharing_Finalizer(t *testing.T) {
	// This test showcases the logic on a the RandaoMixes field with the GC finalizer.

	a, _ := InitializeFromProtoUnsafe(&p2ppb.BeaconState{RandaoMixes: [][]byte{[]byte("foo")}})
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
	b.UpdateRandaoMixesAtIndex([]byte("bar"), 0)
	if b.sharedFieldReferences[randaoMixes].refs != 1 || a.sharedFieldReferences[randaoMixes].refs != 1 {
		t.Error("Expected 1 shared reference to randao mix for both a and b")
	}
}

func TestStateReferenceCopy_NoUnexpectedMutation(t *testing.T) {
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
		t.Error(err)
	}

	if refsCount := a.sharedFieldReferences[validators].refs; refsCount != 1 {
		t.Errorf("Unexpected count of references for validators field, want: %v, got: %v", 1, refsCount)
	}

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
	if refsCount := a.sharedFieldReferences[validators].refs; refsCount != 2 {
		t.Errorf("Unexpected count of references for validators field, want: %v, got: %v", 2, refsCount)
	}
	if refsCount := b.sharedFieldReferences[validators].refs; refsCount != 2 {
		t.Errorf("Unexpected count of references for validators field, want: %v, got: %v", 2, refsCount)
	}
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

	t.Run("validators", func(t *testing.T) {
		err = a.AppendValidator(&eth.Validator{
			PublicKey: pubKey2[:],
		})

		// Copy on write happened, reference counters are reset.
		if refsCount := a.sharedFieldReferences[validators].refs; refsCount != 1 {
			t.Errorf("Unexpected count of references for validators field, want: %v, got: %v", 1, refsCount)
		}
		if refsCount := b.sharedFieldReferences[validators].refs; refsCount != 1 {
			t.Errorf("Unexpected count of references for validators field, want: %v, got: %v", 1, refsCount)
		}

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
			t.Errorf("Unexpected validator found, want: %v", pubKey2)
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
		err := a.ApplyToEveryValidator(func(idx int, val *eth.Validator) (b bool, err error) {
			return true, nil
		}, func(idx int, val *eth.Validator) error {
			val.EffectiveBalance = 1
			return nil
		})
		if err != nil {
			t.Error(err)
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
	})
}
