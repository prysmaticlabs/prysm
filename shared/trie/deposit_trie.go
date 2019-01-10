package trie

import (
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// DepositTrie represents a Merkle trie tracking deposits on the ETH 1.0
// PoW chain contract created in Vyper.
type DepositTrie struct {
	depositCount uint64
	merkleHashes map[uint64][32]byte
}

// NewDepositTrie creates a new struct instance with a hash list of initial
// length equal to 2 to the power of the deposit contract's tree depth.
func NewDepositTrie() *DepositTrie {
	return &DepositTrie{
		depositCount: 0,
		merkleHashes: make(map[uint64][32]byte),
	}
}

// UpdateDepositTrie updates the Merkle trie representing deposits on
// the ETH 1.0 PoW chain contract.
func (d *DepositTrie) UpdateDepositTrie(depositData []byte) {
	twoToPowerOfTreeDepth := 1 << params.BeaconConfig().DepositContractTreeDepth
	index := d.depositCount + uint64(twoToPowerOfTreeDepth)
	d.merkleHashes[index] = hashutil.Hash(depositData)
	for i := uint64(0); i < params.BeaconConfig().DepositContractTreeDepth; i++ {
		index = index / 2
		left := d.merkleHashes[index*2]
		right := d.merkleHashes[index*2+1]
		if right == [32]byte{} {
			d.merkleHashes[index] = hashutil.Hash(left[:])
		} else {
			d.merkleHashes[index] = hashutil.Hash(append(left[:], right[:]...))
		}
	}
	d.depositCount++
}

// GenerateMerkleBranch for a value up to the root from a leaf in the trie.
func (d *DepositTrie) GenerateMerkleBranch(depositData []byte) [][]byte {
	twoToPowerOfTreeDepth := 1 << params.BeaconConfig().DepositContractTreeDepth
	index := d.depositCount + uint64(twoToPowerOfTreeDepth)
	root := hashutil.Hash(depositData)
	branch := [][]byte{root[:]}
	for i := uint64(0); i < params.BeaconConfig().DepositContractTreeDepth; i++ {
		index = index / 2
		left := d.merkleHashes[index*2]
		right := d.merkleHashes[index*2+1]
		if i%2 == 0 {
			root = hashutil.Hash(append(left[:], root[:]...))
		} else {
			root = hashutil.Hash(append(root[:], right[:]...))
		}
		branch = append(branch, root[:])
	}
	return branch
}

// Root returns the Merkle root of the calculated deposit trie.
func (d *DepositTrie) Root() [32]byte {
	return d.merkleHashes[1]
}

// VerifyMerkleBranch verifies a merkle path in a trie
// by checking the aggregated hash of contiguous leaves along a path
// eventually equals the root hash of the merkle trie.
func VerifyMerkleBranch(leaf [32]byte, branch [][]byte, depth uint64, root [32]byte) bool {
	value := leaf
	for i := uint64(0); i < depth; i++ {
		if i%2 == 0 {
			value = hashutil.Hash(append(branch[i], value[:]...))
		} else {
			value = hashutil.Hash(append(value[:], branch[i]...))
		}
	}
	return value == root
}
