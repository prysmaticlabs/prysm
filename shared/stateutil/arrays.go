package stateutil

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/protolambda/zssz/merkle"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

var (
	leavesCache = make(map[string][][]byte)
	layersCache = make(map[string][][][]byte)
	lock        sync.Mutex
	readLock    sync.RWMutex
)

func (h *stateRootHasher) arraysRoot(roots [][]byte, fieldName string) ([32]byte, error) {
	lock.Lock()
	if _, ok := layersCache[fieldName]; !ok && h.rootsCache != nil {
		depth := merkle.GetDepth(uint64(len(roots)))
		layersCache[fieldName] = make([][][]byte, depth+1)
	}
	lock.Unlock()

	hashKeyElements := make([]byte, len(roots)*32)
	leaves := make([][]byte, len(roots))
	emptyKey := hashutil.FastSum256(hashKeyElements)
	bytesProcessed := 0
	changedIndices := make([]int, 0)
	readLock.RLock()
	prevLeaves, ok := leavesCache[fieldName]
	readLock.RUnlock()
	if len(prevLeaves) == 0 || h.rootsCache == nil {
		prevLeaves = leaves
	}
	if h.rootsCache != nil && len(leaves) != len(prevLeaves) {
		// We invalidate the cache completely in this case.
		prevLeaves = leaves
		//depth := merkle.GetDepth(uint64(len(leaves)))
		//lock.Lock()
		//layersCache[fieldName] = make([][][]byte, depth+1)
		//lock.Unlock()
	}
	for i := 0; i < len(roots) && i < len(leaves) && i < len(prevLeaves); i++ {
		padded := bytesutil.ToBytes32(roots[i])
		copy(hashKeyElements[bytesProcessed:bytesProcessed+32], padded[:])
		leaves[i] = padded[:]
		// We check if any items changed since the roots were last recomputed.
		if ok && h.rootsCache != nil && !bytes.Equal(leaves[i], prevLeaves[i]) {
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
		for i := 0; i < len(changedIndices); i++ {
			rt, err = recomputeRoot(changedIndices[i], chunks, fieldName)
			if err != nil {
				return [32]byte{}, err
			}
		}
		if fieldName == "RandaoMixes" {
			fmt.Println("Branch Recompute Merkle With Cache")
			fmt.Printf("Changed indices: %v\n", changedIndices)
			prettyPrintTree(layersCache[fieldName])
		}
		return rt, nil
	}

	hashKey := hashutil.FastSum256(hashKeyElements)
	if hashKey != emptyKey && h.rootsCache != nil {
		if found, ok := h.rootsCache.Get(fieldName + string(hashKey[:])); found != nil && ok {
			return found.([32]byte), nil
		}
	}

	var res [32]byte
	res = h.merkleizeWithCache(leaves, fieldName)
	if h.rootsCache != nil {
		lock.Lock()
		leavesCache[fieldName] = leaves
		lock.Unlock()
	}
	if hashKey != emptyKey && h.rootsCache != nil {
		h.rootsCache.Set(fieldName+string(hashKey[:]), res, 32)
	}
	return res, nil
}

func (h *stateRootHasher) merkleizeWithCache(leaves [][]byte, fieldName string) [32]byte {
	lock.Lock()
	defer lock.Unlock()
	if len(leaves) == 1 {
		var root [32]byte
		copy(root[:], leaves[0])
		return root
	}
	for !mathutil.IsPowerOf2(uint64(len(leaves))) {
		leaves = append(leaves, make([]byte, 32))
	}
	hashLayer := leaves
	layers := make([][][]byte, merkle.GetDepth(uint64(len(leaves)))+1)
	if fieldName == "RandaoMixes" {
		fmt.Println("Setting layers to empty")
	}
	if items, ok := layersCache[fieldName]; ok && h.rootsCache != nil {
		if len(items[0]) == len(leaves) {
			layers = items
			if fieldName == "RandaoMixes" {
				fmt.Printf("Setting layers to items: %v\n", items)
			}
		}
	}
	layers[0] = hashLayer
	if fieldName == "RandaoMixes" {
		fmt.Printf("Updated first layer: %v\n", layers[0])
	}
	// We keep track of the hash layers of a Merkle trie until we reach
	// the top layer of length 1, which contains the single root element.
	//        [Root]      -> Top layer has length 1.
	//    [E]       [F]   -> This layer has length 2.
	// [A]  [B]  [C]  [D] -> The bottom layer has length 4 (needs to be a power of two).
	i := 1
	for len(hashLayer) > 1 && i < len(layers) {
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
	if h.rootsCache != nil {
		layersCache[fieldName] = layers
		if fieldName == "RandaoMixes" {
			fmt.Println("Regular Merkle With Cache")
			prettyPrintTree(layersCache[fieldName])
		}
	} else {
		if fieldName == "RandaoMixes" {
			fmt.Println("Regular Merkle No Cache")
			prettyPrintTree(layers)
		}
	}
	return root
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
	//fmt.Printf("Recomputing index %d\n", idx)
	layers[0][idx] = root
	//fmt.Printf("Set index %d to %#x\n", idx, bytesutil.Trunc(root))
	// The merkle tree structure looks as follows:
	// [[r1, r2, r3, r4], [parent1, parent2], [root]]
	// Using information about the index which changed, idx, we recompute
	// only its branch up the tree.
	currentIndex := idx
	for i := 0; i < len(layers)-1; i++ {
		isLeft := currentIndex%2 == 0
		//fmt.Printf("At layer %d, current index %d is left: %v\n", i, currentIndex, isLeft)
		neighborIdx := currentIndex ^ 1
		// fmt.Printf("Neighbor index: %d\n", neighborIdx)

		neighbor := make([]byte, 32)
		if layers[i] != nil && len(layers[i]) != 0 && neighborIdx < len(layers[i]) {
			neighbor = layers[i][neighborIdx]
		}
		// fmt.Printf("Neighbor: %#x\n", bytesutil.Trunc(neighbor))
		if isLeft {
			parentHash := hashutil.Hash(append(root, neighbor...))
			root = parentHash[:]
		} else {
			parentHash := hashutil.Hash(append(neighbor, root...))
			root = parentHash[:]
		}
		parentIdx := currentIndex / 2
		// Update the cached layers at the parent index.
		fmt.Printf("i+1=%d, numlayers=%d, len(layers+1)=%d, parentidx=%d\n", i+1, len(layers), len(layers[i+1]), parentIdx)
		if len(layers[i+1]) == 0 {
			layers[i+1] = append(layers[i+1], root)
		}
		layers[i+1][parentIdx] = root
		currentIndex = parentIdx
	}
	layersCache[fieldName] = layers
	return bytesutil.ToBytes32(root), nil
}

func prettyPrintTree(layers [][][]byte) {
	fmt.Println("***ROOT")
	if len(layers[len(layers)-1]) != 0 {
		fmt.Printf("%#x\n", bytesutil.Trunc(layers[len(layers)-1][0]))
	}
	for i := len(layers) - 1; i >= 0; i-- {
		fmt.Printf("***LAYER %d\n", i)
		fmt.Print("Nodes: ")
		for j := 0; j < len(layers[i]); j++ {
			if j == len(layers[i])-1 {
				fmt.Printf("%#x", bytesutil.Trunc(layers[i][j]))
			} else {
				fmt.Printf("%#x, ", bytesutil.Trunc(layers[i][j]))
			}
		}
		fmt.Println("")
	}
}
