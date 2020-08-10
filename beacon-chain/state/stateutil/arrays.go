// Package stateutil defines utility functions to compute state roots
// using advanced merkle branch caching techniques.package stateutil
package stateutil

import (
	"bytes"
	"errors"
	"sync"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
)

var (
	leavesCache = make(map[string][][32]byte, fieldCount)
	layersCache = make(map[string][][][32]byte, fieldCount)
	lock        sync.RWMutex
)

// RootsArrayHashTreeRoot computes the Merkle root of arrays of 32-byte hashes, such as [64][32]byte
// according to the Simple Serialize specification of eth2.
func RootsArrayHashTreeRoot(vals [][]byte, length uint64, fieldName string) ([32]byte, error) {
	if featureconfig.Get().EnableSSZCache {
		return cachedHasher.arraysRoot(vals, length, fieldName)
	}
	return nocachedHasher.arraysRoot(vals, length, fieldName)
}

func (h *stateRootHasher) arraysRoot(input [][]byte, length uint64, fieldName string) ([32]byte, error) {
	lock.Lock()
	defer lock.Unlock()
	hashFunc := hashutil.CustomSHA256Hasher()
	if _, ok := layersCache[fieldName]; !ok && h.rootsCache != nil {
		depth := htrutils.GetDepth(length)
		layersCache[fieldName] = make([][][32]byte, depth+1)
	}

	leaves := make([][32]byte, length)
	for i, chunk := range input {
		copy(leaves[i][:], chunk)
	}
	bytesProcessed := 0
	changedIndices := make([]int, 0)
	prevLeaves, ok := leavesCache[fieldName]
	if len(prevLeaves) == 0 || h.rootsCache == nil {
		prevLeaves = leaves
	}

	for i := 0; i < len(leaves); i++ {
		// We check if any items changed since the roots were last recomputed.
		notEqual := leaves[i] != prevLeaves[i]
		if ok && h.rootsCache != nil && notEqual {
			changedIndices = append(changedIndices, i)
		}
		bytesProcessed += 32
	}
	if len(changedIndices) > 0 && h.rootsCache != nil {
		var rt [32]byte
		var err error
		// If indices did change since last computation, we only recompute
		// the modified branches in the cached Merkle tree for this state field.
		chunks := leaves

		// We need to ensure we recompute indices of the Merkle tree which
		// changed in-between calls to this function. This check adds an offset
		// to the recomputed indices to ensure we do so evenly.
		maxChangedIndex := changedIndices[len(changedIndices)-1]
		if maxChangedIndex+2 == len(chunks) && maxChangedIndex%2 != 0 {
			changedIndices = append(changedIndices, maxChangedIndex+1)
		}
		for i := 0; i < len(changedIndices); i++ {
			rt, err = recomputeRoot(changedIndices[i], chunks, length, fieldName, hashFunc)
			if err != nil {
				return [32]byte{}, err
			}
		}
		leavesCache[fieldName] = chunks
		return rt, nil
	}

	res := h.merkleizeWithCache(leaves, length, fieldName, hashFunc)
	if h.rootsCache != nil {
		leavesCache[fieldName] = leaves
	}
	return res, nil
}

func (h *stateRootHasher) merkleizeWithCache(leaves [][32]byte, length uint64,
	fieldName string, hasher func([]byte) [32]byte) [32]byte {
	if len(leaves) == 1 {
		return leaves[0]
	}
	hashLayer := leaves
	layers := make([][][32]byte, htrutils.GetDepth(length)+1)
	if items, ok := layersCache[fieldName]; ok && h.rootsCache != nil {
		if len(items[0]) == len(leaves) {
			layers = items
		}
	}
	layers[0] = hashLayer
	layers, hashLayer = merkleizeTrieLeaves(layers, hashLayer, hasher)
	root := hashLayer[0]
	if h.rootsCache != nil {
		layersCache[fieldName] = layers
	}
	return root
}

func merkleizeTrieLeaves(layers [][][32]byte, hashLayer [][32]byte,
	hasher func([]byte) [32]byte) ([][][32]byte, [][32]byte) {
	// We keep track of the hash layers of a Merkle trie until we reach
	// the top layer of length 1, which contains the single root element.
	//        [Root]      -> Top layer has length 1.
	//    [E]       [F]   -> This layer has length 2.
	// [A]  [B]  [C]  [D] -> The bottom layer has length 4 (needs to be a power of two).
	i := 1
	chunkBuffer := bytes.NewBuffer([]byte{})
	chunkBuffer.Grow(64)
	for len(hashLayer) > 1 && i < len(layers) {
		layer := make([][32]byte, len(hashLayer)/2, len(hashLayer)/2)
		for j := 0; j < len(hashLayer); j += 2 {
			chunkBuffer.Write(hashLayer[j][:])
			chunkBuffer.Write(hashLayer[j+1][:])
			hashedChunk := hasher(chunkBuffer.Bytes())
			layer[j/2] = hashedChunk
			chunkBuffer.Reset()
		}
		hashLayer = layer
		layers[i] = hashLayer
		i++
	}
	return layers, hashLayer
}

func recomputeRoot(idx int, chunks [][32]byte, length uint64,
	fieldName string, hasher func([]byte) [32]byte) ([32]byte, error) {
	items, ok := layersCache[fieldName]
	if !ok {
		return [32]byte{}, errors.New("could not recompute root as there was no cache found")
	}
	if items == nil {
		return [32]byte{}, errors.New("could not recompute root as there were no items found in the layers cache")
	}
	layers := items
	root := chunks[idx]
	layers[0] = chunks
	// The merkle tree structure looks as follows:
	// [[r1, r2, r3, r4], [parent1, parent2], [root]]
	// Using information about the index which changed, idx, we recompute
	// only its branch up the tree.
	currentIndex := idx
	for i := 0; i < len(layers)-1; i++ {
		isLeft := currentIndex%2 == 0
		neighborIdx := currentIndex ^ 1

		neighbor := [32]byte{}
		if layers[i] != nil && len(layers[i]) != 0 && neighborIdx < len(layers[i]) {
			neighbor = layers[i][neighborIdx]
		}
		if isLeft {
			parentHash := hasher(append(root[:], neighbor[:]...))
			root = parentHash
		} else {
			parentHash := hasher(append(neighbor[:], root[:]...))
			root = parentHash
		}
		parentIdx := currentIndex / 2
		// Update the cached layers at the parent index.
		if len(layers[i+1]) == 0 {
			layers[i+1] = append(layers[i+1], root)
		} else {
			layers[i+1][parentIdx] = root
		}
		currentIndex = parentIdx
	}
	layersCache[fieldName] = layers
	// If there is only a single leaf, we return it (the identity element).
	if len(layers[0]) == 1 {
		return layers[0][0], nil
	}
	return root, nil
}
