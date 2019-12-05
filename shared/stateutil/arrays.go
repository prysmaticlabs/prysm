package stateutil

import (
	"bytes"
	"errors"
	"sync"

	"github.com/minio/highwayhash"
	"github.com/protolambda/zssz/merkle"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

var (
	fastSumHashKey = bytesutil.ToBytes32([]byte("hash_fast_sum64_key"))
	leavesCache    = make(map[string][][]byte)
	layersCache    = make(map[string][][][]byte)
	lock           sync.Mutex
)

func arraysRoot(roots [][]byte, fieldName string) ([32]byte, error) {
	lock.Lock()
	if _, ok := layersCache[fieldName]; !ok {
		depth := merkle.GetDepth(uint64(len(roots)))
		layersCache[fieldName] = make([][][]byte, depth+1)
	}
	lock.Unlock()

	hashKeyElements := make([]byte, len(roots)*32)
	leaves := make([][]byte, len(roots))
	emptyKey := highwayhash.Sum(hashKeyElements, fastSumHashKey[:])
	bytesProcessed := 0
	changedIndices := make([]int, 0)
	lock.Lock()
	prevLeaves, ok := leavesCache[fieldName]
	lock.Unlock()
	for i := 0; i < len(roots); i++ {
		copy(hashKeyElements[bytesProcessed:bytesProcessed+32], roots[i])
		leaves[i] = roots[i]
		// We check if any items changed since the roots were last recomputed.
		if ok && prevLeaves != nil && !bytes.Equal(leaves[i], prevLeaves[i]) {
			changedIndices = append(changedIndices, i)
		}
		bytesProcessed += 32
	}

	if len(changedIndices) > 0 {
		var rt [32]byte
		var err error
		// If indices did change since last computation, we only recompute
		// the modified branches in the cached Merkle tree for this state field.
		chunks := leaves
		for i := 0; i < len(changedIndices); i++ {
			rt, err = recomputeRoot(changedIndices[i], chunks, fieldName)
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
	lock.Lock()
	leavesCache[fieldName] = leaves
	lock.Unlock()
	if hashKey != emptyKey {
		rootsCache.Set(fieldName+string(hashKey[:]), res, 32)
	}
	return res, nil
}

func recomputeRoot(idx int, chunks [][]byte, fieldName string) ([32]byte, error) {
	lock.Lock()
	defer lock.Unlock()
	items, ok := layersCache[fieldName]
	if !ok {
		return [32]byte{}, errors.New("could not recompute root as there was no cache found")
	}
	if items == nil {
		return [32]byte{}, errors.New("could not recompute root as there were no items found in the layers cache")
	}
	layers := items
	root := chunks[idx]
	layers[0][idx] = root
	// The merkle tree structure looks as follows:
	// [[r1, r2, r3, r4], [parent1, parent2], [root]]
	// Using information about the index which changed, idx, we recompute
	// only its branch up the tree.
	currentIndex := idx
	for i := 0; i < len(layers)-1; i++ {
		isLeft := currentIndex%2 == 0
		neighborIdx := currentIndex ^ 1
		neighbor := layers[i][neighborIdx]
		if isLeft {
			parentHash := hashutil.Hash(append(root, neighbor...))
			root = parentHash[:]
		} else {
			parentHash := hashutil.Hash(append(neighbor, root...))
			root = parentHash[:]
		}
		parentIdx := currentIndex / 2
		// Update the cached layers at the parent index.
		layers[i+1][parentIdx] = root
		currentIndex = parentIdx
	}
	layersCache[fieldName] = layers
	return bytesutil.ToBytes32(root), nil
}

func merkleizeWithCache(leaves [][]byte, fieldName string) [32]byte {
	lock.Lock()
	defer lock.Unlock()
	if len(leaves) == 1 {
		var root [32]byte
		copy(root[:], leaves[0])
		return root
	}
	for !isPowerOf2(len(leaves)) {
		leaves = append(leaves, make([]byte, 32))
	}
	hashLayer := leaves
	layers := make([][][]byte, merkle.GetDepth(uint64(len(leaves)))+1)
	if items, ok := layersCache[fieldName]; ok {
		layers = items
	}
	layers[0] = hashLayer
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
		layers[i] = hashLayer
		i++
	}
	var root [32]byte
	copy(root[:], hashLayer[0])
	layersCache[fieldName] = layers
	return root
}

func isPowerOf2(n int) bool {
	return n != 0 && (n&(n-1)) == 0
}
