package state_test

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestFieldTrie_NewTrie(t *testing.T) {
	newState, _ := testutil.DeterministicGenesisState(t, 40)

	// 5 represents the enum value of state roots
	trie, err := state.NewFieldTrie(5, newState.StateRoots(), params.BeaconConfig().SlotsPerHistoricalRoot)
	if err != nil {
		t.Fatal(err)
	}
	root, err := stateutil.RootsArrayHashTreeRoot(newState.StateRoots(), params.BeaconConfig().SlotsPerHistoricalRoot, "StateRoots")
	if err != nil {
		t.Fatal(err)
	}
	newRoot, err := trie.TrieRoot()
	if newRoot != root {
		t.Errorf("Wanted root of %#x but got %#x", root, newRoot)
	}
}

func TestFieldTrie_RecomputeTrie(t *testing.T) {
	newState, _ := testutil.DeterministicGenesisState(t, 32)
	// 10 represents the enum value of validators
	trie, err := state.NewFieldTrie(11, newState.Validators(), params.BeaconConfig().ValidatorRegistryLimit)
	if err != nil {
		t.Fatal(err)
	}

	changedIdx := []uint64{2, 29}
	val1, err := newState.ValidatorAtIndex(10)
	if err != nil {
		t.Fatal(err)
	}
	val2, err := newState.ValidatorAtIndex(11)
	if err != nil {
		t.Fatal(err)
	}
	val1.Slashed = true
	val1.ExitEpoch = 20

	val2.Slashed = true
	val2.ExitEpoch = 40

	changedVals := []*ethpb.Validator{val1, val2}
	if err := newState.UpdateValidatorAtIndex(changedIdx[0], changedVals[0]); err != nil {
		t.Fatal(err)
	}
	if err := newState.UpdateValidatorAtIndex(changedIdx[1], changedVals[1]); err != nil {
		t.Fatal(err)
	}

	expectedRoot, err := stateutil.ValidatorRegistryRoot(newState.Validators())
	if err != nil {
		t.Fatal(err)
	}
	root, err := trie.RecomputeTrie(changedIdx, newState.Validators())
	if err != nil {
		t.Fatal(err)
	}
	if root != expectedRoot {
		t.Errorf("Wanted root of %#x but got %#x", expectedRoot, root)
	}
}

func TestFieldTrie_CopyTrieImmutable(t *testing.T) {
	newState, _ := testutil.DeterministicGenesisState(t, 32)
	// 12 represents the enum value of randao mixes.
	trie, err := state.NewFieldTrie(13, newState.RandaoMixes(), params.BeaconConfig().EpochsPerHistoricalVector)
	if err != nil {
		t.Fatal(err)
	}

	newTrie := trie.CopyTrie()

	changedIdx := []uint64{2, 29}

	changedVals := [][32]byte{{'A', 'B'}, {'C', 'D'}}
	if err := newState.UpdateRandaoMixesAtIndex(changedIdx[0], changedVals[0][:]); err != nil {
		t.Fatal(err)
	}
	if err := newState.UpdateRandaoMixesAtIndex(changedIdx[1], changedVals[1][:]); err != nil {
		t.Fatal(err)
	}

	root, err := trie.RecomputeTrie(changedIdx, newState.RandaoMixes())
	if err != nil {
		t.Fatal(err)
	}
	newRoot, err := newTrie.TrieRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root == newRoot {
		t.Errorf("Wanted roots to be different, but they are the same: %#x", root)
	}
}
