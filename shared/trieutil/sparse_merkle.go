// Package trieutil defines utilities for sparse merkle tries for eth2.
package trieutil

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	protodb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// SparseMerkleTrie implements a sparse, general purpose Merkle trie to be used
// across ETH2.0 Phase 0 functionality.
type SparseMerkleTrie struct {
	depth         uint
	branches      [][][]byte
	originalItems [][]byte // list of provided items before hashing them into leaves.
}

// NewTrie returns a new merkle trie filled with zerohashes to use.
func NewTrie(depth int) (*SparseMerkleTrie, error) {
	var zeroBytes [32]byte
	items := [][]byte{zeroBytes[:]}
	return GenerateTrieFromItems(items, depth)
}

// CreateTrieFromProto creates a Sparse Merkle Trie from its corresponding merkle trie.
func CreateTrieFromProto(trieObj *protodb.SparseMerkleTrie) *SparseMerkleTrie {
	trie := &SparseMerkleTrie{
		depth:         uint(trieObj.Depth),
		originalItems: trieObj.OriginalItems,
	}
	branches := make([][][]byte, len(trieObj.Layers))
	for i, layer := range trieObj.Layers {
		branches[i] = layer.Layer
	}
	trie.branches = branches
	return trie
}

// GenerateTrieFromItems constructs a Merkle trie from a sequence of byte slices.
func GenerateTrieFromItems(items [][]byte, depth int) (*SparseMerkleTrie, error) {
	if len(items) == 0 {
		return nil, errors.New("no items provided to generate Merkle trie")
	}
	leaves := items
	layers := make([][][]byte, depth+1)
	transformedLeaves := make([][]byte, len(leaves))
	for i := range leaves {
		arr := bytesutil.ToBytes32(leaves[i])
		transformedLeaves[i] = arr[:]
	}
	layers[0] = transformedLeaves
	for i := 0; i < depth; i++ {
		if len(layers[i])%2 == 1 {
			layers[i] = append(layers[i], ZeroHashes[i][:])
		}
		updatedValues := make([][]byte, 0, 0)
		for j := 0; j < len(layers[i]); j += 2 {
			concat := hashutil.Hash(append(layers[i][j], layers[i][j+1]...))
			updatedValues = append(updatedValues, concat[:])
		}
		layers[i+1] = updatedValues
	}
	return &SparseMerkleTrie{
		branches:      layers,
		originalItems: items,
		depth:         uint(depth),
	}, nil
}

// Items returns the original items passed in when creating the Merkle trie.
func (m *SparseMerkleTrie) Items() [][]byte {
	return m.originalItems
}

// Root returns the top-most, Merkle root of the trie.
func (m *SparseMerkleTrie) Root() [32]byte {
	enc := [32]byte{}
	binary.LittleEndian.PutUint64(enc[:], uint64(len(m.originalItems)))
	return hashutil.Hash(append(m.branches[len(m.branches)-1][0], enc[:]...))
}

// Insert an item into the trie.
func (m *SparseMerkleTrie) Insert(item []byte, index int) {
	for index >= len(m.branches[0]) {
		m.branches[0] = append(m.branches[0], ZeroHashes[0][:])
	}
	someItem := bytesutil.ToBytes32(item)
	m.branches[0][index] = someItem[:]
	if index >= len(m.originalItems) {
		m.originalItems = append(m.originalItems, someItem[:])
	} else {
		m.originalItems[index] = someItem[:]
	}
	currentIndex := index
	root := bytesutil.ToBytes32(item)
	for i := 0; i < int(m.depth); i++ {
		isLeft := currentIndex%2 == 0
		neighborIdx := currentIndex ^ 1
		neighbor := make([]byte, 32)
		if neighborIdx >= len(m.branches[i]) {
			neighbor = ZeroHashes[i][:]
		} else {
			neighbor = m.branches[i][neighborIdx]
		}
		if isLeft {
			parentHash := hashutil.Hash(append(root[:], neighbor...))
			root = parentHash
		} else {
			parentHash := hashutil.Hash(append(neighbor, root[:]...))
			root = parentHash
		}
		parentIdx := currentIndex / 2
		if len(m.branches[i+1]) == 0 || parentIdx >= len(m.branches[i+1]) {
			newItem := root
			m.branches[i+1] = append(m.branches[i+1], newItem[:])
		} else {
			newItem := root
			m.branches[i+1][parentIdx] = newItem[:]
		}
		currentIndex = parentIdx
	}
}

// MerkleProof computes a proof from a trie's branches using a Merkle index.
func (m *SparseMerkleTrie) MerkleProof(index int) ([][]byte, error) {
	merkleIndex := uint(index)
	leaves := m.branches[0]
	if index >= len(leaves) {
		return nil, fmt.Errorf("merkle index out of range in trie, max range: %d, received: %d", len(leaves), index)
	}
	proof := make([][]byte, m.depth+1)
	for i := uint(0); i < m.depth; i++ {
		subIndex := (merkleIndex / (1 << i)) ^ 1
		if subIndex < uint(len(m.branches[i])) {
			item := bytesutil.ToBytes32(m.branches[i][subIndex])
			proof[i] = item[:]
		} else {
			proof[i] = ZeroHashes[i][:]
		}
	}
	enc := [32]byte{}
	binary.LittleEndian.PutUint64(enc[:], uint64(len(m.originalItems)))
	proof[len(proof)-1] = enc[:]
	return proof, nil
}

// HashTreeRoot of the Merkle trie as defined in the deposit contract.
//  Spec Definition:
//   sha256(concat(node, self.to_little_endian_64(self.deposit_count), slice(zero_bytes32, start=0, len=24)))
func (m *SparseMerkleTrie) HashTreeRoot() [32]byte {
	var zeroBytes [32]byte
	depositCount := uint64(len(m.originalItems))
	if len(m.originalItems) == 1 && bytes.Equal(m.originalItems[0], zeroBytes[:]) {
		// Accounting for empty tries
		depositCount = 0
	}
	newNode := append(m.branches[len(m.branches)-1][0], bytesutil.Bytes8(depositCount)...)
	newNode = append(newNode, zeroBytes[:24]...)
	return hashutil.Hash(newNode)
}

// ToProto converts the underlying trie into its corresponding
// proto object
func (m *SparseMerkleTrie) ToProto() *protodb.SparseMerkleTrie {
	trie := &protodb.SparseMerkleTrie{
		Depth:         uint64(m.depth),
		Layers:        make([]*protodb.TrieLayer, len(m.branches)),
		OriginalItems: m.originalItems,
	}
	for i, l := range m.branches {
		trie.Layers[i] = &protodb.TrieLayer{
			Layer: l,
		}
	}
	return trie
}

// VerifyMerkleBranch verifies a Merkle branch against a root of a trie.
func VerifyMerkleBranch(root []byte, item []byte, merkleIndex int, proof [][]byte) bool {
	node := bytesutil.ToBytes32(item)
	currentIndex := merkleIndex
	for i := 0; i < len(proof); i++ {
		if currentIndex%2 != 0 {
			node = hashutil.Hash(append(proof[i], node[:]...))
		} else {
			node = hashutil.Hash(append(node[:], proof[i]...))
		}
		currentIndex = currentIndex / 2
	}
	return bytes.Equal(root, node[:])
}

// Copy performs a deep copy of the trie.
func (m *SparseMerkleTrie) Copy() *SparseMerkleTrie {
	dstBranches := make([][][]byte, len(m.branches))
	for i1, srcB1 := range m.branches {
		dstBranches[i1] = bytesutil.Copy2dBytes(srcB1)
	}

	return &SparseMerkleTrie{
		depth:         m.depth,
		branches:      dstBranches,
		originalItems: bytesutil.Copy2dBytes(m.originalItems),
	}
}
