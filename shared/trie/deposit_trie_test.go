package trie

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestDepositTrie_UpdateDepositTrie(t *testing.T) {
	tests := []struct {
		deposits [][]byte
	}{
		{
			[][]byte{
				{1, 2, 3, 4},
				{5, 6, 7, 8},
			},
		},
		{
			[][]byte{
				{0, 0, 0, 0},
				{0, 0, 0, 0},
			},
		},
	}
	for _, tt := range tests {
		d := NewDepositTrie()
		d.UpdateDepositTrie(tt.deposits[0])
		if d.depositCount != 1 {
			t.Errorf("Expected deposit count to increase by 1, received %d", d.depositCount)
		}
		root := hashutil.Hash(tt.deposits[0])
		for i := uint64(0); i < params.BeaconConfig().DepositContractTreeDepth; i++ {
			root = hashutil.Hash(root[:])
		}
		if d.Root() != root {
			t.Errorf("Expected root to equal %#x, received %#x", d.Root(), root)
		}
		d.UpdateDepositTrie(tt.deposits[1])
		if d.depositCount != 2 {
			t.Errorf("Expected deposit count to increase to 2, received %d", d.depositCount)
		}
		left := d.merkleHashes[2]
		right := d.merkleHashes[3]
		if right == [32]byte{} {
			root = hashutil.Hash(left[:])
		} else {
			root = hashutil.Hash(append(left[:], right[:]...))
		}
		if d.Root() != root {
			t.Errorf("Expected root to equal %#x, received %#x", d.Root(), root)
		}
	}
}

func TestDepositTrie_GenerateMerkleBranch(t *testing.T) {
	d := NewDepositTrie()
	deposit1 := []byte{1, 2, 3}
	d.UpdateDepositTrie(deposit1)
	deposit2 := []byte{5, 6, 7}
	d.UpdateDepositTrie(deposit2)
	deposit3 := []byte{8, 9, 10}
	d.UpdateDepositTrie(deposit3)
	index := d.depositCount-1
	branch := d.GenerateMerkleBranch(index)
	if ok := VerifyMerkleBranch(
		hashutil.Hash(deposit3),
		branch,
		params.BeaconConfig().DepositContractTreeDepth,
		index,
		d.Root(),
	); !ok {
		t.Error("Expected Merkle branch to verify, received false")
	}
}
