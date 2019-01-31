// Package trieutil contains definitions for building a Merkle trie for validator deposits
// as defined in the Ethereum Serenity specification, as well as utilities to generate
// and verify Merkle proofs.
package trieutil

import (
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// DepositTrie represents a Merkle trie tracking deposits on the ETH 1.0
// PoW chain contract created in Vyper.
type DepositTrie struct {
	depositCount uint64
	merkleMap    map[uint64][32]byte
}

// NewDepositTrie creates a new struct instance representing a Merkle trie for deposits
// and tracking an initial deposit count of 0.
func NewDepositTrie() *DepositTrie {
	return &DepositTrie{
		depositCount: 0,
		merkleMap:    make(map[uint64][32]byte),
	}
}

// UpdateDepositTrie updates the Merkle trie representing deposits on
// the ETH 1.0 PoW chain contract.
func (d *DepositTrie) UpdateDepositTrie(depositData []byte) {
	twoToPowerOfTreeDepth := 1 << params.BeaconConfig().DepositContractTreeDepth
	index := d.depositCount + uint64(twoToPowerOfTreeDepth)
	d.merkleMap[index] = hashutil.Hash(depositData)
	for i := uint64(0); i < params.BeaconConfig().DepositContractTreeDepth; i++ {
		index = index / 2
		left := d.merkleMap[index*2]
		right := d.merkleMap[index*2+1]
		d.merkleMap[index] = hashutil.Hash(append(left[:], right[:]...))
	}
	d.depositCount++
}

// GenerateMerkleBranch for a value up to the root from a leaf in the trie.
func (d *DepositTrie) GenerateMerkleBranch(index uint64) [][]byte {
	twoToPowerOfTreeDepth := 1 << params.BeaconConfig().DepositContractTreeDepth
	idx := index + uint64(twoToPowerOfTreeDepth)
	branch := make([][]byte, params.BeaconConfig().DepositContractTreeDepth)
	for i := uint64(0); i < params.BeaconConfig().DepositContractTreeDepth; i++ {
		if idx%2 == 1 {
			value := d.merkleMap[idx-1]
			branch[i] = value[:]
		} else {
			value := d.merkleMap[idx+1]
			branch[i] = value[:]
		}
		idx = idx / 2
	}
	return branch
}

// Root returns the Merkle root of the calculated deposit trie.
func (d *DepositTrie) Root() [32]byte {
	return d.merkleMap[1]
}

// VerifyMerkleBranch verifies a Merkle path in a trie
// by checking the aggregated hash of contiguous leaves along a path
// eventually equals the root hash of the Merkle trie.
func VerifyMerkleBranch(leaf [32]byte, branch [][]byte, depth uint64, index uint64, root [32]byte) bool {
	twoToPowerOfTreeDepth := 1 << params.BeaconConfig().DepositContractTreeDepth
	idx := index + uint64(twoToPowerOfTreeDepth)
	value := leaf
	for i := uint64(0); i < depth; i++ {
		if idx%2 == 1 {
			value = hashutil.Hash(append(branch[i], value[:]...))
		} else {
			value = hashutil.Hash(append(value[:], branch[i]...))
		}
		idx = idx / 2
	}
	return value == root
}
