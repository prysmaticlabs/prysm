package state_test

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestFieldTrie_NewTrie(t *testing.T) {
	newState, _ := testutil.DeterministicGenesisState(t, 40)

	// 5 represents the enum value of state roots
	trie, err := state.NewFieldTrie(5, newState.StateRoots(), params.BeaconConfig().SlotsPerHistoricalRoot)
	require.NoError(t, err)
	root, err := stateutil.RootsArrayHashTreeRoot(newState.StateRoots(), params.BeaconConfig().SlotsPerHistoricalRoot, "StateRoots")
	require.NoError(t, err)
	newRoot, err := trie.TrieRoot()
	assert.Equal(t, root, newRoot)
}

func TestFieldTrie_RecomputeTrie(t *testing.T) {
	newState, _ := testutil.DeterministicGenesisState(t, 32)
	// 10 represents the enum value of validators
	trie, err := state.NewFieldTrie(11, newState.Validators(), params.BeaconConfig().ValidatorRegistryLimit)
	require.NoError(t, err)

	changedIdx := []uint64{2, 29}
	val1, err := newState.ValidatorAtIndex(10)
	require.NoError(t, err)
	val2, err := newState.ValidatorAtIndex(11)
	require.NoError(t, err)
	val1.Slashed = true
	val1.ExitEpoch = 20

	val2.Slashed = true
	val2.ExitEpoch = 40

	changedVals := []*ethpb.Validator{val1, val2}
	require.NoError(t, newState.UpdateValidatorAtIndex(changedIdx[0], changedVals[0]))
	require.NoError(t, newState.UpdateValidatorAtIndex(changedIdx[1], changedVals[1]))

	expectedRoot, err := stateutil.ValidatorRegistryRoot(newState.Validators())
	require.NoError(t, err)
	root, err := trie.RecomputeTrie(changedIdx, newState.Validators())
	require.NoError(t, err)
	assert.Equal(t, expectedRoot, root)
}

func TestFieldTrie_CopyTrieImmutable(t *testing.T) {
	newState, _ := testutil.DeterministicGenesisState(t, 32)
	// 12 represents the enum value of randao mixes.
	trie, err := state.NewFieldTrie(13, newState.RandaoMixes(), params.BeaconConfig().EpochsPerHistoricalVector)
	require.NoError(t, err)

	newTrie := trie.CopyTrie()

	changedIdx := []uint64{2, 29}

	changedVals := [][32]byte{{'A', 'B'}, {'C', 'D'}}
	require.NoError(t, newState.UpdateRandaoMixesAtIndex(changedIdx[0], changedVals[0][:]))
	require.NoError(t, newState.UpdateRandaoMixesAtIndex(changedIdx[1], changedVals[1][:]))

	root, err := trie.RecomputeTrie(changedIdx, newState.RandaoMixes())
	require.NoError(t, err)
	newRoot, err := newTrie.TrieRoot()
	require.NoError(t, err)
	if root == newRoot {
		t.Errorf("Wanted roots to be different, but they are the same: %#x", root)
	}
}
