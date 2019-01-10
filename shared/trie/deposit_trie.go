package trie

import (
	"math"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

type DepositTrie struct {
	depositCount uint64
	hashList     [][32]byte
}

// UpdateDepositTrie updates the Merkle trie representing deposits on
// the ETH 1.0 PoW chain contract.
func (d *DepositTrie) UpdateDepositTrie(depositData []byte) {
	twoToPowerOfTreeDepth := math.Pow(2, float64(params.BeaconConfig().POWContractMerkleTreeDepth))
	index := d.depositCount + uint64(twoToPowerOfTreeDepth)
	d.hashList[index] = hashutil.Hash(depositData)
	for i := uint64(0); i < params.BeaconConfig().POWContractMerkleTreeDepth; i++ {
		index := index / 2
		left := d.hashList[index*2]
		right := d.hashList[index*2+1]
		d.hashList[index] = hashutil.Hash(append(left[:], right[:]...))
	}
	d.depositCount++
}

// Root returns the merkle root of the calculated deposit trie.
func (d *DepositTrie) Root() [32]byte {
	return d.hashList[0]
}
