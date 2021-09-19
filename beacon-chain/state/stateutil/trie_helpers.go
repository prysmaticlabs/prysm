package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	htrutils2 "github.com/prysmaticlabs/prysm/encoding/htrutils"
)

// ReturnTrieLayer returns the representation of a merkle trie when
// provided with the elements of a fixed sized trie and the corresponding depth of
// it.
func ReturnTrieLayer(elements [][32]byte, length uint64) [][]*[32]byte {
	hasher := hash.CustomSHA256Hasher()
	leaves := elements

	if len(leaves) == 1 {
		return [][]*[32]byte{{&leaves[0]}}
	}
	hashLayer := leaves
	layers := make([][][32]byte, htrutils2.Depth(length)+1)
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
		layer := make([][32]byte, len(hashLayer)/2)
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

// ReturnTrieLayerVariable returns the representation of a merkle trie when
// provided with the elements of a variable sized trie and the corresponding depth of
// it.
func ReturnTrieLayerVariable(elements [][32]byte, length uint64) [][]*[32]byte {
	hasher := hash.CustomSHA256Hasher()
	depth := htrutils2.Depth(length)
	layers := make([][]*[32]byte, depth+1)
	// Return zerohash at depth
	if len(elements) == 0 {
		zerohash := trie.ZeroHashes[depth]
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
			zerohash := trie.ZeroHashes[i]
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
		// remove zerohash node from tree
		if oddNodeLength {
			layers[i] = layers[i][:len(layers[i])-1]
		}
		layers[i+1] = updatedValues
	}
	return layers
}

// RecomputeFromLayer recomputes specific branches of a fixed sized trie depending on the provided changed indexes.
func RecomputeFromLayer(changedLeaves [][32]byte, changedIdx []uint64, layer [][]*[32]byte) ([32]byte, [][]*[32]byte, error) {
	hasher := hash.CustomSHA256Hasher()
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

// RecomputeFromLayerVariable recomputes specific branches of a variable sized trie depending on the provided changed indexes.
func RecomputeFromLayerVariable(changedLeaves [][32]byte, changedIdx []uint64, layer [][]*[32]byte) ([32]byte, [][]*[32]byte, error) {
	hasher := hash.CustomSHA256Hasher()
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

// this method assumes that the provided trie already has all its elements included
// in the base depth.
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

// this method assumes that the base branch does not consist of all leaves of the
// trie. Instead missing leaves are assumed to be zerohashes, following the structure
// of a sparse merkle trie.
func recomputeRootFromLayerVariable(idx int, item [32]byte, layers [][]*[32]byte,
	hasher func([]byte) [32]byte) ([32]byte, [][]*[32]byte, error) {
	for idx >= len(layers[0]) {
		zerohash := trie.ZeroHashes[0]
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
			neighbor = trie.ZeroHashes[i]
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

// AddInMixin describes a method from which a lenth mixin is added to the
// provided root.
func AddInMixin(root [32]byte, length uint64) ([32]byte, error) {
	rootBuf := new(bytes.Buffer)
	if err := binary.Write(rootBuf, binary.LittleEndian, length); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal eth1data votes length")
	}
	// We need to mix in the length of the slice.
	rootBufRoot := make([]byte, 32)
	copy(rootBufRoot, rootBuf.Bytes())
	return htrutils2.MixInLength(root, rootBufRoot), nil
}

// Merkleize 32-byte leaves into a Merkle trie for its adequate depth, returning
// the resulting layers of the trie based on the appropriate depth. This function
// pads the leaves to a length of 32.
func Merkleize(leaves [][]byte) [][][]byte {
	hashFunc := hash.CustomSHA256Hasher()
	layers := make([][][]byte, htrutils2.Depth(uint64(len(leaves)))+1)
	for len(leaves) != 32 {
		leaves = append(leaves, make([]byte, 32))
	}
	currentLayer := leaves
	layers[0] = currentLayer

	// We keep track of the hash layers of a Merkle trie until we reach
	// the top layer of length 1, which contains the single root element.
	//        [Root]      -> Top layer has length 1.
	//    [E]       [F]   -> This layer has length 2.
	// [A]  [B]  [C]  [D] -> The bottom layer has length 4 (needs to be a power of two).
	i := 1
	for len(currentLayer) > 1 && i < len(layers) {
		layer := make([][]byte, 0)
		for i := 0; i < len(currentLayer); i += 2 {
			hashedChunk := hashFunc(append(currentLayer[i], currentLayer[i+1]...))
			layer = append(layer, hashedChunk[:])
		}
		currentLayer = layer
		layers[i] = currentLayer
		i++
	}
	return layers
}

// MerkleizeTrieLeaves merkleize the trie leaves.
func MerkleizeTrieLeaves(layers [][][32]byte, hashLayer [][32]byte,
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
		layer := make([][32]byte, len(hashLayer)/2)
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
