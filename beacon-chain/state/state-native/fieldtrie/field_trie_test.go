package fieldtrie_test

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/custom-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/fieldtrie"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	stateTypes "github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestFieldTrie_NewTrie(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 40)

	var bRoots customtypes.BlockRoots
	for i, r := range newState.BlockRoots() {
		bRoots[i] = bytesutil.ToBytes32(r)
	}
	// 5 represents the enum value of block roots
	trie, err := fieldtrie.NewFieldTrie(5, stateTypes.BasicArray, &bRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
	require.NoError(t, err)
	root, err := stateutil.RootsArrayHashTreeRoot(newState.StateRoots(), uint64(params.BeaconConfig().SlotsPerHistoricalRoot), "StateRoots")
	require.NoError(t, err)
	newRoot, err := trie.TrieRoot()
	require.NoError(t, err)
	assert.Equal(t, root, newRoot)
}

func TestFieldTrie_RecomputeTrie(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 32)
	// 10 represents the enum value of validators
	trie, err := fieldtrie.NewFieldTrie(11, stateTypes.CompositeArray, newState.Validators(), params.BeaconConfig().ValidatorRegistryLimit)
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
	require.NoError(t, newState.UpdateValidatorAtIndex(types.ValidatorIndex(changedIdx[0]), changedVals[0]))
	require.NoError(t, newState.UpdateValidatorAtIndex(types.ValidatorIndex(changedIdx[1]), changedVals[1]))

	expectedRoot, err := stateutil.ValidatorRegistryRoot(newState.Validators())
	require.NoError(t, err)
	root, err := trie.RecomputeTrie(changedIdx, newState.Validators())
	require.NoError(t, err)
	assert.Equal(t, expectedRoot, root)
}

func TestFieldTrie_CopyTrieImmutable(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 32)
	var mixes customtypes.RandaoMixes
	for i, r := range newState.RandaoMixes() {
		mixes[i] = bytesutil.ToBytes32(r)
	}
	// 13 represents the enum value of randao mixes.
	trie, err := fieldtrie.NewFieldTrie(13, stateTypes.BasicArray, &mixes, uint64(params.BeaconConfig().EpochsPerHistoricalVector))
	require.NoError(t, err)

	newTrie := trie.CopyTrie()

	changedIdx := []uint64{2, 29}

	changedVals := [][32]byte{{'A', 'B'}, {'C', 'D'}}
	require.NoError(t, newState.UpdateRandaoMixesAtIndex(changedIdx[0], changedVals[0][:]))
	require.NoError(t, newState.UpdateRandaoMixesAtIndex(changedIdx[1], changedVals[1][:]))

	var mixes2 customtypes.RandaoMixes
	for i, r := range newState.RandaoMixes() {
		mixes2[i] = bytesutil.ToBytes32(r)
	}
	root, err := trie.RecomputeTrie(changedIdx, &mixes2)
	require.NoError(t, err)
	newRoot, err := newTrie.TrieRoot()
	require.NoError(t, err)
	if root == newRoot {
		t.Errorf("Wanted roots to be different, but they are the same: %#x", root)
	}
}
