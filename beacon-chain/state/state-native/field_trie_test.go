package state_native_test

import (
	"context"
	"testing"

	state_native "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	customtypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/custom-types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/v4/config/features"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestFieldTrie_NewTrie(t *testing.T) {
	t.Run("native state", func(t *testing.T) {
		runNewTrie(t)
	})
	t.Run("native state with multivalue slice", func(t *testing.T) {
		cfg := features.Get()
		cfg.EnableExperimentalState = true
		reset := features.InitWithReset(cfg)
		runNewTrie(t)

		reset()
	})
}

func runNewTrie(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 40)
	// Initialize state caches
	_, err := newState.HashTreeRoot(context.Background())
	require.NoError(t, err)
	require.NoError(t, newState.UpdateBlockRootAtIndex(0, [32]byte{}))
	_, err = newState.HashTreeRoot(context.Background())
	require.NoError(t, err)

	roots := newState.BlockRoots()
	blockRoots := make([][32]byte, len(roots))
	for i, r := range roots {
		blockRoots[i] = [32]byte(r)
	}

	trie, err := state_native.NewFieldTrie(types.BlockRoots, types.BasicArray, customtypes.BlockRoots(blockRoots), uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
	require.NoError(t, err)
	root, err := stateutil.RootsArrayHashTreeRoot(newState.BlockRoots(), uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
	require.NoError(t, err)
	newRoot, err := trie.TrieRoot()
	require.NoError(t, err)
	assert.Equal(t, root, newRoot)
}

func TestFieldTrie_NewTrie_NilElements(t *testing.T) {
	trie, err := state_native.NewFieldTrie(types.BlockRoots, types.BasicArray, nil, 8234)
	require.NoError(t, err)
	_, err = trie.TrieRoot()
	require.ErrorIs(t, err, state_native.ErrEmptyFieldTrie)
}

func TestFieldTrie_RecomputeTrie(t *testing.T) {
	t.Run("native state", func(t *testing.T) {
		runRecomputeTrie(t)
	})
	t.Run("native state with multivalue slice", func(t *testing.T) {
		cfg := features.Get()
		cfg.EnableExperimentalState = true
		reset := features.InitWithReset(cfg)
		runRecomputeTrie(t)

		reset()
	})
}

func runRecomputeTrie(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 32)
	// Initialize state caches
	_, err := newState.HashTreeRoot(context.Background())
	require.NoError(t, err)
	require.NoError(t, newState.UpdateValidatorAtIndex(0, &ethpb.Validator{
		PublicKey:                  make([]byte, 48),
		WithdrawalCredentials:      nil,
		EffectiveBalance:           1000,
		Slashed:                    false,
		ActivationEligibilityEpoch: 0,
		ActivationEpoch:            0,
		ExitEpoch:                  1000,
		WithdrawableEpoch:          0,
	}))
	_, err = newState.HashTreeRoot(context.Background())
	require.NoError(t, err)

	trie, err := state_native.NewFieldTrie(types.Validators, types.CompositeArray, newState.Validators(), params.BeaconConfig().ValidatorRegistryLimit)
	require.NoError(t, err)

	oldroot, err := trie.TrieRoot()
	require.NoError(t, err)
	require.NotEmpty(t, oldroot)

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
	require.NoError(t, newState.UpdateValidatorAtIndex(primitives.ValidatorIndex(changedIdx[0]), changedVals[0]))
	require.NoError(t, newState.UpdateValidatorAtIndex(primitives.ValidatorIndex(changedIdx[1]), changedVals[1]))

	expectedRoot, err := stateutil.ValidatorRegistryRoot(newState.Validators())
	require.NoError(t, err)
	root, err := trie.RecomputeTrie(changedIdx, newState.Validators())
	require.NoError(t, err)
	assert.Equal(t, expectedRoot, root)
}

func TestFieldTrie_RecomputeTrie_CompressedArray(t *testing.T) {
	t.Run("native state", func(t *testing.T) {
		runRecomputeTrie_CompressedArray(t)
	})
	t.Run("native state with multivalue slice", func(t *testing.T) {
		cfg := features.Get()
		cfg.EnableExperimentalState = true
		reset := features.InitWithReset(cfg)
		runRecomputeTrie_CompressedArray(t)

		reset()
	})
}

func runRecomputeTrie_CompressedArray(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 32)
	// Initialize state caches
	_, err := newState.HashTreeRoot(context.Background())
	require.NoError(t, err)
	require.NoError(t, newState.UpdateBalancesAtIndex(0, 1000))
	_, err = newState.HashTreeRoot(context.Background())
	require.NoError(t, err)
	require.NoError(t, newState.UpdateBalancesAtIndex(1, 10000))
	_, err = newState.HashTreeRoot(context.Background())
	require.NoError(t, err)

	trie, err := state_native.NewFieldTrie(types.Balances, types.CompressedArray, newState.Balances(), stateutil.ValidatorLimitForBalancesChunks())
	require.NoError(t, err)
	require.Equal(t, trie.Length(), stateutil.ValidatorLimitForBalancesChunks())
	changedIdx := []uint64{4, 8}
	require.NoError(t, newState.UpdateBalancesAtIndex(primitives.ValidatorIndex(changedIdx[0]), uint64(100000000)))
	require.NoError(t, newState.UpdateBalancesAtIndex(primitives.ValidatorIndex(changedIdx[1]), uint64(200000000)))
	expectedRoot, err := stateutil.Uint64ListRootWithRegistryLimit(newState.Balances())
	require.NoError(t, err)
	root, err := trie.RecomputeTrie(changedIdx, newState.Balances())
	require.NoError(t, err)

	// not equal for some reason :(
	assert.Equal(t, expectedRoot, root)
}

func TestNewFieldTrie_UnknownType(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 32)
	_, err := state_native.NewFieldTrie(types.Balances, 4, newState.Balances(), 32)
	require.ErrorContains(t, "unrecognized data type", err)
}

func TestFieldTrie_CopyTrieImmutable(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 32)
	mixes := newState.RandaoMixes()
	randaoMixes := make([][32]byte, len(mixes))
	for i, r := range mixes {
		randaoMixes[i] = [32]byte(r)
	}

	trie, err := state_native.NewFieldTrie(types.RandaoMixes, types.BasicArray, customtypes.RandaoMixes(randaoMixes), uint64(params.BeaconConfig().EpochsPerHistoricalVector))
	require.NoError(t, err)

	newTrie := trie.CopyTrie()

	changedIdx := []uint64{2, 29}

	changedVals := [][32]byte{{'A', 'B'}, {'C', 'D'}}
	require.NoError(t, newState.UpdateRandaoMixesAtIndex(changedIdx[0], changedVals[0]))
	require.NoError(t, newState.UpdateRandaoMixesAtIndex(changedIdx[1], changedVals[1]))

	mixes = newState.RandaoMixes()
	randaoMixes = make([][32]byte, len(mixes))
	for i, r := range mixes {
		randaoMixes[i] = [32]byte(r)
	}
	root, err := trie.RecomputeTrie(changedIdx, customtypes.RandaoMixes(randaoMixes))
	require.NoError(t, err)
	newRoot, err := newTrie.TrieRoot()
	require.NoError(t, err)
	if root == newRoot {
		t.Errorf("Wanted roots to be different, but they are the same: %#x", root)
	}
}

func TestFieldTrie_CopyAndTransferEmpty(t *testing.T) {
	trie, err := state_native.NewFieldTrie(types.RandaoMixes, types.BasicArray, nil, uint64(params.BeaconConfig().EpochsPerHistoricalVector))
	require.NoError(t, err)

	require.DeepEqual(t, trie, trie.CopyTrie())
	require.DeepEqual(t, trie, trie.TransferTrie())
}

func TestFieldTrie_TransferTrie(t *testing.T) {
	newState, _ := util.DeterministicGenesisState(t, 32)
	maxLength := (params.BeaconConfig().ValidatorRegistryLimit*8 + 31) / 32
	trie, err := state_native.NewFieldTrie(types.Balances, types.CompressedArray, newState.Balances(), maxLength)
	require.NoError(t, err)
	oldRoot, err := trie.TrieRoot()
	require.NoError(t, err)

	newTrie := trie.TransferTrie()
	root, err := trie.TrieRoot()
	require.ErrorIs(t, err, state_native.ErrEmptyFieldTrie)
	require.Equal(t, root, [32]byte{})
	require.NotNil(t, newTrie)
	newRoot, err := newTrie.TrieRoot()
	require.NoError(t, err)
	require.DeepEqual(t, oldRoot, newRoot)
}

func FuzzFieldTrie(f *testing.F) {
	newState, _ := util.DeterministicGenesisState(f, 40)
	var data []byte
	for _, root := range newState.StateRoots() {
		data = append(data, root...)
	}
	f.Add(5, int(types.BasicArray), data, uint64(params.BeaconConfig().SlotsPerHistoricalRoot))

	f.Fuzz(func(t *testing.T, idx, typ int, data []byte, slotsPerHistRoot uint64) {
		var roots [][]byte
		for i := 32; i < len(data); i += 32 {
			roots = append(roots, data[i-32:i])
		}
		trie, err := state_native.NewFieldTrie(types.FieldIndex(idx), types.DataType(typ), roots, slotsPerHistRoot)
		if err != nil {
			return // invalid inputs
		}
		_, err = trie.TrieRoot()
		if err != nil {
			return
		}
	})
}
