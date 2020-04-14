package stateutil_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestReturnTrieLayer_OK(t *testing.T) {
	newState, _ := testutil.DeterministicGenesisState(t, 32)
	root, err := stateutil.RootsArrayHashTreeRoot(newState.BlockRoots(), params.BeaconConfig().SlotsPerHistoricalRoot, "BlockRoots")
	if err != nil {
		t.Fatal(err)
	}
	blockRts := newState.BlockRoots()
	roots := make([][32]byte, 0, len(blockRts))
	for _, rt := range blockRts {
		roots = append(roots, bytesutil.ToBytes32(rt))
	}
	layers := stateutil.ReturnTrieLayer(roots, uint64(len(roots)))
	newRoot := *layers[len(layers)-1][0]
	if newRoot != root {
		t.Errorf("Wanted root of %#x but got %#x", root, newRoot)
	}
}

func TestReturnTrieLayerVariable_OK(t *testing.T) {
	newState, _ := testutil.DeterministicGenesisState(t, 32)
	root, err := stateutil.ValidatorRegistryRoot(newState.Validators())
	if err != nil {
		t.Fatal(err)
	}
	hasher := hashutil.CustomSHA256Hasher()
	validators := newState.Validators()
	roots := make([][32]byte, 0, len(validators))
	for _, val := range validators {
		rt, err := stateutil.ValidatorRoot(hasher, val)
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, rt)
	}
	layers := stateutil.ReturnTrieLayerVariable(roots, params.BeaconConfig().ValidatorRegistryLimit)
	newRoot := *layers[len(layers)-1][0]
	newRoot, err = stateutil.AddInMixin(newRoot, uint64(len(validators)))
	if err != nil {
		t.Fatal(err)
	}
	if newRoot != root {
		t.Errorf("Wanted root of %#x but got %#x", root, newRoot)
	}
}

func TestRecomputeFromLayer_FixedSizedArray(t *testing.T) {
	newState, _ := testutil.DeterministicGenesisState(t, 32)
	blockRts := newState.BlockRoots()
	roots := make([][32]byte, 0, len(blockRts))
	for _, rt := range blockRts {
		roots = append(roots, bytesutil.ToBytes32(rt))
	}
	layers := stateutil.ReturnTrieLayer(roots, uint64(len(roots)))

	changedIdx := []uint64{24, 41}
	changedRoots := [][32]byte{{'A', 'B', 'C'}, {'D', 'E', 'F'}}
	if err := newState.UpdateBlockRootAtIndex(changedIdx[0], changedRoots[0]); err != nil {
		t.Fatal(err)
	}
	if err := newState.UpdateBlockRootAtIndex(changedIdx[1], changedRoots[1]); err != nil {
		t.Fatal(err)
	}

	expectedRoot, err := stateutil.RootsArrayHashTreeRoot(newState.BlockRoots(), params.BeaconConfig().SlotsPerHistoricalRoot, "BlockRoots")
	if err != nil {
		t.Fatal(err)
	}
	root, _, err := stateutil.RecomputeFromLayer(changedRoots, changedIdx, layers)
	if err != nil {
		t.Fatal(err)
	}
	if root != expectedRoot {
		t.Errorf("Wanted root of %#x but got %#x", expectedRoot, root)
	}
}

func TestRecomputeFromLayer_VariableSizedArray(t *testing.T) {
	newState, _ := testutil.DeterministicGenesisState(t, 32)
	validators := newState.Validators()
	hasher := hashutil.CustomSHA256Hasher()
	roots := make([][32]byte, 0, len(validators))
	for _, val := range validators {
		rt, err := stateutil.ValidatorRoot(hasher, val)
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, rt)
	}
	layers := stateutil.ReturnTrieLayerVariable(roots, params.BeaconConfig().ValidatorRegistryLimit)

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
	roots = make([][32]byte, 0, len(changedVals))
	for _, val := range changedVals {
		rt, err := stateutil.ValidatorRoot(hasher, val)
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, rt)
	}
	root, _, err := stateutil.RecomputeFromLayerVariable(roots, changedIdx, layers)
	if err != nil {
		t.Fatal(err)
	}
	root, err = stateutil.AddInMixin(root, uint64(len(validators)))
	if err != nil {
		t.Fatal(err)
	}
	if root != expectedRoot {
		t.Errorf("Wanted root of %#x but got %#x", expectedRoot, root)
	}
}
