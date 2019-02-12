// Package trieutil contains definitions for building a Merkle trie for validator deposits
// as defined in the Ethereum Serenity specification, as well as utilities to generate
// and verify Merkle proofs.
package trieutil

import (
	"encoding/binary"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// DepositTrie represents a Merkle trie tracking deposits on the ETH 1.0
// PoW chain contract created in Vyper.
type DepositTrie struct {
	depositCount uint64
	branch       [32][32]byte
	zeroHashes   [32][32]byte
}

// NewDepositTrie creates a new struct instance representing a Merkle trie for deposits
// and tracking an initial deposit count of 0.
func NewDepositTrie() *DepositTrie {
	var zeroHashes [32][32]byte
	var branch [32][32]byte

	zeroHashes[0] = params.BeaconConfig().ZeroHash
	branch[0] = params.BeaconConfig().ZeroHash

	for i := 0; i < 31; i++ {
		zeroHashes[i+1] = hashutil.Hash(append(zeroHashes[i][:], zeroHashes[i][:]...))
		branch[i+1] = zeroHashes[i+1]
	}
	return &DepositTrie{
		depositCount: 0,
		zeroHashes:   zeroHashes,
		branch:       branch,
	}
}

// UpdateDepositTrie updates the Merkle trie representing deposits on
// the ETH 1.0 PoW chain contract.
func (d *DepositTrie) UpdateDepositTrie(depositData []byte) {
	index := d.depositCount
	i := 0
	powerOf2 := uint64(2)

	for j := 0; j < 32; j++ {
		if (index+1)%powerOf2 != 0 {
			break
		}

		i++
		powerOf2 *= 2
	}
	hashedData := hashutil.Hash(depositData)

	for k := 0; k < 32; k++ {
		if k < i {
			hashedData = hashutil.Hash(append(d.branch[k][:], hashedData[:]...))
		}
	}
	d.branch[i] = hashedData
	d.depositCount++
}

// Root returns the Merkle root of the calculated deposit trie.
func (d *DepositTrie) Root() [32]byte {
	root := params.BeaconConfig().ZeroHash
	size := d.depositCount

	for i := 0; i < 32; i++ {

		if size%2 == 1 {
			root = hashutil.Hash(append(d.branch[i][:], root[:]...))
		} else {
			root = hashutil.Hash(append(root[:], d.zeroHashes[i][:]...))
		}

		size /= 2
	}
	return root
}

// Branch returns the merkle branch of the left most leaf of the trie.
func (d *DepositTrie) Branch() [][]byte {
	nBranch := make([][]byte, 32)
	for i := range nBranch {
		nBranch[i] = d.branch[i][:]
	}
	return nBranch
}

// VerifyMerkleBranch verifies a Merkle path in a trie
// by checking the aggregated hash of contiguous leaves along a path
// eventually equals the root hash of the Merkle trie.
func VerifyMerkleBranch(branch [][]byte, root [32]byte, merkleTreeIndex []byte) bool {
	computedRoot := params.BeaconConfig().ZeroHash
	index := binary.LittleEndian.Uint64(merkleTreeIndex)
	size := index + 1
	zHashes := zeroHashes()

	for i := 0; i < 32; i++ {

		if size%2 == 1 {
			computedRoot = hashutil.Hash(append(branch[i][:], computedRoot[:]...))
		} else {
			computedRoot = hashutil.Hash(append(computedRoot[:], zHashes[i][:]...))
		}

		size /= 2
	}

	return computedRoot == root
}

func zeroHashes() [32][32]byte {
	var zeroHashes [32][32]byte

	zeroHashes[0] = params.BeaconConfig().ZeroHash
	for i := 0; i < 31; i++ {
		zeroHashes[i+1] = hashutil.Hash(append(zeroHashes[i][:], zeroHashes[i][:]...))
	}
	return zeroHashes
}
