// Package trieutil contains definitions for building a Merkle trie for validator deposits
// as defined in the Ethereum Serenity specification, as well as utilities to generate
// and verify Merkle proofs.
package trieutil

import (
	"encoding/binary"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// DepositTrie represents a Merkle trie tracking deposits on the ETH 1.0
// PoW chain contract created in Vyper.
type DepositTrie struct {
	trie       *pb.DepositTrie
	zeroHashes [32][32]byte
}

// NewDepositTrie creates a new struct instance representing a Merkle trie for deposits
// and tracking an initial deposit count of 0.
func NewDepositTrie() *DepositTrie {
	var zeroHashes [32][32]byte
	trie := &pb.DepositTrie{}

	zeroHashes[0] = params.BeaconConfig().ZeroHash
	trie.Branch = append(trie.Branch, params.BeaconConfig().ZeroHash[:])

	for i := 0; i < 31; i++ {

		zeroHashes[i+1] = hashutil.Hash(append(zeroHashes[i][:], zeroHashes[i][:]...))
		trie.Branch = append(trie.Branch, zeroHashes[i+1][:])
	}
	trie.DepositCount = 0
	return &DepositTrie{
		trie:       trie,
		zeroHashes: zeroHashes,
	}
}

// UpdateDepositTrie updates the Merkle trie representing deposits on
// the ETH 1.0 PoW chain contract.
func (d *DepositTrie) UpdateDepositTrie(depositData []byte) {
	index := d.trie.DepositCount
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
			hashedData = hashutil.Hash(append(d.trie.Branch[k][:], hashedData[:]...))
		}
	}
	d.trie.Branch[i] = hashedData[:]
	d.trie.DepositCount++
}

// Root returns the Merkle root of the calculated deposit trie.
func (d *DepositTrie) Root() [32]byte {
	root := params.BeaconConfig().ZeroHash
	size := d.trie.DepositCount

	for i := 0; i < 32; i++ {

		if size%2 == 1 {
			root = hashutil.Hash(append(d.trie.Branch[i][:], root[:]...))
		} else {
			root = hashutil.Hash(append(root[:], d.zeroHashes[i][:]...))
		}

		size /= 2
	}
	return root
}

// GetTrie returns proto DepositTrie structure of DepositTrie
func (d *DepositTrie) GetTrie() *pb.DepositTrie {
	return d.trie
}

// SetTrie set DepositTrie givven a proto DepositTrie structure
func (d *DepositTrie) SetTrie(dt *pb.DepositTrie) {
	d.trie = dt
}

// Branch returns the merkle branch of the left most leaf of the trie.
func (d *DepositTrie) Branch() [][]byte {
	nBranch := make([][]byte, 32)
	for i := range nBranch {
		nBranch[i] = d.trie.Branch[i][:]
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
