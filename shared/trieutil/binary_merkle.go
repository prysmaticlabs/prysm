package trieutil

import (
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func MerkleTree(leaves [][]byte) [][]byte {
	paddedLength := mathutil.NextPowerOf2(len(leaves))
	parents := make([][]byte, paddedLength)
	paddedLeaves := make([][]byte, paddedLength - len(leaves))

	for i:=0; i<len(parents); i++ {
		parents[i] = params.BeaconConfig().ZeroHash[:]
	}
	for i:=0; i<len(paddedLeaves); i++ {
		paddedLeaves[i] = params.BeaconConfig().ZeroHash[:]
	}

	merkleTree := make([][]byte, 0, len(parents) + len(leaves) + len(paddedLeaves))
	merkleTree = append(merkleTree, parents...)
	merkleTree = append(merkleTree, leaves...)
	merkleTree = append(merkleTree, paddedLeaves...)

	for i:=len(paddedLeaves)-1; i > 0; i-- {
		a := append(merkleTree[2*i], merkleTree[2*i+1]...)
		b := hashutil.Hash(a)
		merkleTree[i] = b[:]
	}

	return merkleTree
}
