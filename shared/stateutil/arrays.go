package stateutil

import (
	"bytes"

	"github.com/minio/highwayhash"
	"github.com/protolambda/zssz/merkle"

	//"github.com/protolambda/zssz/merkle"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

var (
	cachedLeaves   = make(map[string][][]byte)
	cachedLayers   = make(map[string][][][]byte)
	fastSumHashKey = bytesutil.ToBytes32([]byte("hash_fast_sum64_key"))
)

func arraysRoot(roots [][]byte, fieldName string) ([32]byte, error) {
	if _, ok := cachedLayers[fieldName]; !ok {
		depth := merkle.GetDepth(uint64(len(roots)))
		cachedLayers[fieldName] = make([][][]byte, depth+1)
	}

	hashKeyElements := make([]byte, len(roots)*32)
	emptyKey := highwayhash.Sum(hashKeyElements, fastSumHashKey[:])
	bytesProcessed := 0
	changedIndices := make([]int, 0)
	for i := 0; i < len(roots); i++ {
		copy(hashKeyElements[bytesProcessed:bytesProcessed+32], roots[i])
		// We check if any items changed since the roots were last recomputed.
		if !bytes.Equal(roots[i], cachedLeaves[fieldName][i]) {
			changedIndices = append(changedIndices, i)
		}
	}

	if len(changedIndices) > 0 {
		var rt [32]byte
		// If indices did change since last computation, we only recompute
		// the modified branches in the cached Merkle tree for this state field.
		for i := 0; i < len(changedIndices); i++ {
			rt = recomputeRoot(changedIndices[i], roots, fieldName)
		}
		return rt, nil
	}

	hashKey := highwayhash.Sum(hashKeyElements, fastSumHashKey[:])
	if hashKey != emptyKey {
		if found, ok := cache.Get(hashKey); found != nil && ok {
			return found.([32]byte), nil
		}
	}

	// TODO: Update cached leaves and use custom Merkleize.
	res, err := bitwiseMerkleize(roots, uint64(len(roots)), uint64(len(roots)))
	if err != nil {
		return [32]byte{}, err
	}
	if hashKey != emptyKey {
		cache.Set(hashKey, res, 32)
	}
	return res, nil
}

func blockRoots(roots [][]byte) ([32]byte, error) {
	key := make([]byte, len(roots)*32)
	bytesProcessed := 0
	for i := 0; i < len(roots); i++ {
		copy(key[bytesProcessed:bytesProcessed+32], roots[i])
	}
	found, ok := cache.Get(key)
	if found != nil && ok {
		return found.([32]byte), nil
	}
	res, err := bitwiseMerkleize(roots, uint64(len(roots)), uint64(len(roots)))
	if err != nil {
		return [32]byte{}, err
	}
	cache.Set(key, res, 32)
	return res, nil
}

func stateRoots(roots [][]byte) ([32]byte, error) {
	key := make([]byte, len(roots)*32)
	bytesProcessed := 0
	for i := 0; i < len(roots); i++ {
		copy(key[bytesProcessed:bytesProcessed+32], roots[i])
	}
	found, ok := cache.Get(key)
	if found != nil && ok {
		return found.([32]byte), nil
	}
	res, err := bitwiseMerkleize(roots, uint64(len(roots)), uint64(len(roots)))
	if err != nil {
		return [32]byte{}, err
	}
	cache.Set(key, res, 32)
	return res, nil
}

func randaoRoots(roots [][]byte) ([32]byte, error) {
	key := make([]byte, len(roots)*32)
	bytesProcessed := 0
	for i := 0; i < len(roots); i++ {
		copy(key[bytesProcessed:bytesProcessed+32], roots[i])
	}
	found, ok := cache.Get(key)
	if found != nil && ok {
		return found.([32]byte), nil
	}
	res, err := bitwiseMerkleize(roots, uint64(len(roots)), uint64(len(roots)))
	if err != nil {
		return [32]byte{}, err
	}
	cache.Set(key, res, 32)
	return res, nil
}

func recomputeRoot(idx int, chunks [][]byte, fieldName string) [32]byte {
	root := chunks[idx]
	for i := 0; i < len(cachedLayers[fieldName])-1; i++ {
		subIndex := (uint64(idx) / (1 << uint64(i))) ^ 1
		isLeft := uint64(idx) / (1 << uint64(i))
		parentIdx := uint64(idx) / (1 << uint64(i+1))
		item := cachedLayers[fieldName][i][subIndex]
		if isLeft%2 != 0 {
			parentHash := hashutil.Hash(append(item, root...))
			root = parentHash[:]
		} else {
			parentHash := hashutil.Hash(append(root, item...))
			root = parentHash[:]
		}
		// Update the cached layers at the parent index.
		cachedLayers[fieldName][i+1][parentIdx] = root
	}
	return bytesutil.ToBytes32(root)
}
