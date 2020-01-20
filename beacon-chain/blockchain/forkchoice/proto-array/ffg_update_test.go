package protoarray

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestFFGUpdates_OneBranch(t *testing.T) {
	balances := []uint64{1, 2}

	f := New(0, 0, params.BeaconConfig().ZeroHash)
	r, err := f.Head(0, 0, params.BeaconConfig().ZeroHash, balances)
	if err != nil {
		t.Fatal(err)
	}
	if r != params.BeaconConfig().ZeroHash {
		t.Errorf("Incorrect head with genesis")
	}

	if err := f.ProcessBlock(1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(2, indexToHash(2), indexToHash(1), 1, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(3, indexToHash(3), indexToHash(2), 2, 1); err != nil {
		t.Fatal(err)
	}
	r, err = f.Head(0, 0, params.BeaconConfig().ZeroHash, balances)
	if err != nil {
		t.Fatal(err)
	}
	if r != indexToHash(3) {
		t.Error("Incorrect head for with justified epoch at 0")
	}

	r, err = f.Head(1, 0, indexToHash(2), balances)
	if err != nil {
		t.Fatal(err)
	}
	if r != indexToHash(2) {
		t.Error("Incorrect head with justified epoch at 1")
	}

	r, err = f.Head(2, 1, indexToHash(3), balances)
	if err != nil {
		t.Fatal(err)
	}
	if r != indexToHash(3) {
		t.Error("Incorrect head with justified epoch at 2")
	}
}

func TestFFGUpdates_TwoBranches(t *testing.T) {
	balances := []uint64{1, 2}

	f := New(1, 1, params.BeaconConfig().ZeroHash)
	r, err := f.Head(0, 0, params.BeaconConfig().ZeroHash, balances)
	if err != nil {
		t.Fatal(err)
	}
	if r != params.BeaconConfig().ZeroHash {
		t.Errorf("Incorrect head with genesis")
	}
	// Left branch.
	if err := f.ProcessBlock(1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(2, indexToHash(3), indexToHash(1), 1, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(3, indexToHash(5), indexToHash(3), 1, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(4, indexToHash(7), indexToHash(5), 1, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(4, indexToHash(9), indexToHash(7), 2, 0); err != nil {
		t.Fatal(err)
	}
	// Right branch.
	if err := f.ProcessBlock(1, indexToHash(2), params.BeaconConfig().ZeroHash, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(2, indexToHash(4), indexToHash(2), 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(3, indexToHash(6), indexToHash(4), 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(4, indexToHash(8), indexToHash(6), 1, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.ProcessBlock(4, indexToHash(10), indexToHash(8), 2, 0); err != nil {
		t.Fatal(err)
	}

	r, err = f.Head(0, 0, params.BeaconConfig().ZeroHash, balances)
	if err != nil {
		t.Fatal(err)
	}
	if r != indexToHash(10) {
		t.Error("Incorrect head with justified epoch at 0")
	}

	f.ProcessAttestation([]uint64{0}, indexToHash(1), 0)
	r, err = f.Head(0, 0, params.BeaconConfig().ZeroHash, balances)
	if err != nil {
		t.Fatal(err)
	}
	if r != indexToHash(9) {
		t.Error("Incorrect head with justified epoch at 0")
	}

	f.ProcessAttestation([]uint64{1}, indexToHash(2), 0)
	r, err = f.Head(0, 0, params.BeaconConfig().ZeroHash, balances)
	if err != nil {
		t.Fatal(err)
	}
	if r != indexToHash(10) {
		t.Error("Incorrect head with justified epoch at 0")
	}
}
