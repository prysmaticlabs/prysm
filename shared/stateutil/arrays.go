package stateutil

import (
	"bytes"
	"errors"

	"github.com/minio/highwayhash"
	"github.com/protolambda/zssz/merkle"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

var (
	fastSumHashKey = bytesutil.ToBytes32([]byte("hash_fast_sum64_key"))
)

func ArraysRoot(roots [][]byte, fieldName string) ([32]byte, error) {
	if _, ok := layersCache.Get(fieldName); !ok {
		depth := merkle.GetDepth(uint64(len(roots)))
		layersCache.Set(fieldName, make([][][]byte, depth+1), int64(depth+1))
	}

	hashKeyElements := make([]byte, len(roots)*32)
	leaves := make([][]byte, len(roots))
	emptyKey := highwayhash.Sum(hashKeyElements, fastSumHashKey[:])
	bytesProcessed := 0
	changedIndices := make([]int, 0)
	prevLeaves, _ := leavesCache.Get(fieldName)
	for i := 0; i < len(roots); i++ {
		copy(hashKeyElements[bytesProcessed:bytesProcessed+32], roots[i])
		leaves[i] = roots[i]
		// We check if any items changed since the roots were last recomputed.
		if prevLeaves != nil && !bytes.Equal(leaves[i], prevLeaves.([][]byte)[i]) {
			changedIndices = append(changedIndices, i)
		}
		bytesProcessed += 32
	}
	if len(changedIndices) > 0 {
		var rt [32]byte
		var err error
		// If indices did change since last computation, we only recompute
		// the modified branches in the cached Merkle tree for this state field.
		for i := 0; i < len(changedIndices); i++ {
			rt, err = recomputeRoot(changedIndices[i], leaves, fieldName)
			if err != nil {
				return [32]byte{}, err
			}
		}
		return rt, nil
	}

	hashKey := highwayhash.Sum(hashKeyElements, fastSumHashKey[:])
	if hashKey != emptyKey {
		if found, ok := rootsCache.Get(fieldName + string(hashKey[:])); found != nil && ok {
			return found.([32]byte), nil
		}
	}

	res := merkleizeWithCache(leaves, fieldName)
	leavesCache.Set(fieldName, leaves, int64(len(leaves)))
	if hashKey != emptyKey {
		rootsCache.Set(fieldName+string(hashKey[:]), res, 32)
	}
	return res, nil
}

func recomputeRoot(idx int, chunks [][]byte, fieldName string) ([32]byte, error) {
	items, ok := layersCache.Get(fieldName)
	if !ok {
		return [32]byte{}, errors.New("could not recompute root as there was no cache found")
	}
	layers := items.([][][]byte)
	root := chunks[idx]
	for i := 0; i < len(layers)-1; i++ {
		subIndex := (uint64(idx) / (1 << uint64(i))) ^ 1
		isLeft := uint64(idx) / (1 << uint64(i))
		parentIdx := uint64(idx) / (1 << uint64(i+1))
		item := layers[i][subIndex]
		if isLeft%2 != 0 {
			parentHash := hashutil.Hash(append(item, root...))
			root = parentHash[:]
		} else {
			parentHash := hashutil.Hash(append(root, item...))
			root = parentHash[:]
		}
		// Update the cached layers at the parent index.
		layers[i+1][parentIdx] = root
	}
	layersCache.Set(fieldName, layers, int64(len(layers)))
	return bytesutil.ToBytes32(root), nil
}

func merkleizeWithCache(leaves [][]byte, fieldName string) [32]byte {
	if len(leaves) == 1 {
		var root [32]byte
		copy(root[:], leaves[0])
		return root
	}
	for !isPowerOf2(len(leaves)) {
		leaves = append(leaves, make([]byte, 32))
	}
	hashLayer := leaves

	var layers [][][]byte
	if items, ok := layersCache.Get(fieldName); ok && items != nil {
		layers = items.([][][]byte)
	}
	if layers != nil {
		layers[0] = hashLayer
	}
	// We keep track of the hash layers of a Merkle trie until we reach
	// the top layer of length 1, which contains the single root element.
	//        [Root]      -> Top layer has length 1.
	//    [E]       [F]   -> This layer has length 2.
	// [A]  [B]  [C]  [D] -> The bottom layer has length 4 (needs to be a power of two).
	i := 1
	for len(hashLayer) > 1 {
		layer := make([][]byte, 0)
		for i := 0; i < len(hashLayer); i += 2 {
			hashedChunk := hashutil.Hash(append(hashLayer[i], hashLayer[i+1]...))
			layer = append(layer, hashedChunk[:])
		}
		hashLayer = layer
		if layers != nil {
			layers[i] = hashLayer
		}
		i++
	}
	var root [32]byte
	copy(root[:], hashLayer[0])
	if layers != nil {
		layersCache.Set(fieldName, layers, int64(len(layers)))
	}
	return root
}

func isPowerOf2(n int) bool {
	return n != 0 && (n&(n-1)) == 0
}
