package stateutil_test

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestBlockRoot(t *testing.T) {
	genState, keys := testutil.DeterministicGenesisState(t, 100)
	blk, err := testutil.GenerateFullBlock(genState, keys, testutil.DefaultBlockGenConfig(), 10)
	if err != nil {
		t.Fatal(err)
	}
	expectedRoot, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	receivedRoot, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if receivedRoot != expectedRoot {
		t.Fatalf("Wanted %#x but got %#x", expectedRoot, receivedRoot)
	}
	blk, err = testutil.GenerateFullBlock(genState, keys, testutil.DefaultBlockGenConfig(), 100)
	if err != nil {
		t.Fatal(err)
	}
	expectedRoot, err = stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	receivedRoot, err = stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if receivedRoot != expectedRoot {
		t.Fatalf("Wanted %#x but got %#x", expectedRoot, receivedRoot)
	}
}

func TestBlockBodyRoot_NilIsSameAsEmpty(t *testing.T) {
	a, err := stateutil.BlockBodyRoot(&ethpb.BeaconBlockBody{})
	if err != nil {
		t.Error(err)
	}
	b, err := stateutil.BlockBodyRoot(nil)
	if err != nil {
		t.Error(err)
	}
	if a != b {
		t.Log(a)
		t.Log(b)
		t.Error("A nil and empty block body do not generate the same root")
	}
}
