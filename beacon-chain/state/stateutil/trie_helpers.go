package stateutil

import (
	"bytes"

	"github.com/protolambda/zssz/merkle"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func ReturnTrieLayer(elements [][32]byte, length uint64) [][]*[32]byte {
	hasher := hashutil.CustomSHA256Hasher()
	leaves := elements

	if len(leaves) == 1 {
		return [][]*[32]byte{{&leaves[0]}}
	}
	hashLayer := leaves
	layers := make([][][32]byte, merkle.GetDepth(length)+1)
	layers[0] = hashLayer
	layers, _ = merkleizeTrieLeaves(layers, hashLayer, hasher)
	refLayers := make([][]*[32]byte, len(layers))
	for i, val := range layers {
		refLayers[i] = make([]*[32]byte, len(val))
		for j, innerVal := range val {
			newVal := innerVal
			refLayers[i][j] = &newVal
		}
	}
	return refLayers
}

func ReturnTrieLayerVariable(elements [][32]byte, length uint64) [][]*[32]byte {
	hasher := hashutil.CustomSHA256Hasher()
	depth := merkle.GetDepth(length)
	layers := make([][]*[32]byte, depth+1)
	// Return zerohash at depth
	if len(elements) == 0 {
		zerohash := trieutil.ZeroHashes[depth]
		layers[len(layers)-1] = []*[32]byte{&zerohash}
		return layers
	}
	transformedLeaves := make([]*[32]byte, len(elements))
	for i := range elements {
		arr := elements[i]
		transformedLeaves[i] = &arr
	}
	layers[0] = transformedLeaves
	buffer := bytes.NewBuffer([]byte{})
	buffer.Grow(64)
	for i := 0; i < int(depth); i++ {
		oddNodeLength := len(layers[i])%2 == 1
		if oddNodeLength {
			zerohash := trieutil.ZeroHashes[i]
			layers[i] = append(layers[i], &zerohash)
		}
		updatedValues := make([]*[32]byte, 0, len(layers[i])/2)
		for j := 0; j < len(layers[i]); j += 2 {
			buffer.Write(layers[i][j][:])
			buffer.Write(layers[i][j+1][:])
			concat := hasher(buffer.Bytes())
			updatedValues = append(updatedValues, &concat)
			buffer.Reset()
		}
		if oddNodeLength {
			layers[i] = layers[i][:len(layers[i])-1]
		}
		layers[i+1] = updatedValues
	}
	return layers
}

func RecomputeFromLayer(changedLeaves [][32]byte, changedIdx []uint64, layer [][]*[32]byte) ([32]byte, [][]*[32]byte, error) {
	hasher := hashutil.CustomSHA256Hasher()
	for i, idx := range changedIdx {
		layer[0][idx] = &changedLeaves[i]
	}

	if len(changedIdx) == 0 {
		return *layer[0][0], layer, nil
	}

	leaves := layer[0]

	// We need to ensure we recompute indices of the Merkle tree which
	// changed in-between calls to this function. This check adds an offset
	// to the recomputed indices to ensure we do so evenly.
	maxChangedIndex := changedIdx[len(changedIdx)-1]
	if int(maxChangedIndex+2) == len(leaves) && maxChangedIndex%2 != 0 {
		changedIdx = append(changedIdx, maxChangedIndex+1)
	}

	root := *layer[0][0]
	var err error

	for _, idx := range changedIdx {
		root, layer, err = recomputeRootFromLayer(int(idx), layer, leaves, hasher)
		if err != nil {
			return [32]byte{}, nil, err
		}
	}
	return root, layer, nil
}

func RecomputeFromLayerVariable(changedLeaves [][32]byte, changedIdx []uint64, layer [][]*[32]byte) ([32]byte, [][]*[32]byte, error) {
	hasher := hashutil.CustomSHA256Hasher()
	if len(changedIdx) == 0 {
		return *layer[0][0], layer, nil
	}
	root := *layer[len(layer)-1][0]
	var err error

	for i, idx := range changedIdx {
		root, layer, err = recomputeRootFromLayerVariable(int(idx), changedLeaves[i], layer, hasher)
		if err != nil {
			return [32]byte{}, nil, err
		}
	}
	return root, layer, nil
}

func recomputeRootFromLayer(idx int, layers [][]*[32]byte, chunks []*[32]byte,
	hasher func([]byte) [32]byte) ([32]byte, [][]*[32]byte, error) {
	root := *chunks[idx]
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
			neighbor = *layers[i][neighborIdx]
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
		rootVal := root
		if len(layers[i+1]) == 0 {
			layers[i+1] = append(layers[i+1], &rootVal)
		} else {
			layers[i+1][parentIdx] = &rootVal
		}
		currentIndex = parentIdx
	}
	// If there is only a single leaf, we return it (the identity element).
	if len(layers[0]) == 1 {
		return *layers[0][0], layers, nil
	}
	return root, layers, nil
}

func recomputeRootFromLayerVariable(idx int, item [32]byte, layers [][]*[32]byte,
	hasher func([]byte) [32]byte) ([32]byte, [][]*[32]byte, error) {
	for idx >= len(layers[0]) {
		zerohash := trieutil.ZeroHashes[0]
		layers[0] = append(layers[0], &zerohash)
	}
	layers[0][idx] = &item

	currentIndex := idx
	root := item
	for i := 0; i < len(layers)-1; i++ {
		isLeft := currentIndex%2 == 0
		neighborIdx := currentIndex ^ 1

		neighbor := [32]byte{}
		if neighborIdx >= len(layers[i]) {
			neighbor = trieutil.ZeroHashes[i]
		} else {
			neighbor = *layers[i][neighborIdx]
		}
		if isLeft {
			parentHash := hasher(append(root[:], neighbor[:]...))
			root = parentHash
		} else {
			parentHash := hasher(append(neighbor[:], root[:]...))
			root = parentHash
		}
		parentIdx := currentIndex / 2
		if len(layers[i+1]) == 0 || parentIdx >= len(layers[i+1]) {
			newItem := root
			layers[i+1] = append(layers[i+1], &newItem)
		} else {
			newItem := root
			layers[i+1][parentIdx] = &newItem
		}
		currentIndex = parentIdx
	}
	return root, layers, nil
}
