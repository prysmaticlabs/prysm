package trieutil

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
		d.UpdateDepositTrie(tt.deposits[1])
		if d.depositCount != 2 {
			t.Errorf("Expected deposit count to increase by 1, received %d", d.depositCount)
		}
		hash := hashutil.Hash(tt.deposits[1])
		twoToPowerOfTreeDepth := 1 << params.BeaconConfig().DepositContractTreeDepth
		lastLeaf := d.merkleMap[d.depositCount-1+uint64(twoToPowerOfTreeDepth)]
		if lastLeaf != hash {
			t.Errorf("Expected last leaf to equal %#x, received %#x", lastLeaf, hash)
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
	index := d.depositCount - 1
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
