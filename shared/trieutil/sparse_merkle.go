package trieutil

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// MerkleTrie implements a sparse, general purpose Merkle trie to be used
// across ETH2.0 Phase 0 functionality.
type MerkleTrie struct {
	depth         uint
	branches      [][][]byte
	originalItems [][]byte // list of provided items before hashing them into leaves.
}

// NewTrie returns a new merkle trie filled with zerohashes to use.
func NewTrie(depth int) (*MerkleTrie, error) {
	var zeroBytes [32]byte
	items := [][]byte{zeroBytes[:]}
	return GenerateTrieFromItems(items, depth)
}

// InsertIntoTrie inserts an item(deposit hash) into the trie.
func (m *MerkleTrie) InsertIntoTrie(item []byte, index int) error {
	if index >= len(m.originalItems) {
		return errors.New("invalid index to be inserting")
	}
	m.insertIntoTrie(item, index)
	return nil
}

// debugging
func (m *MerkleTrie) ReturnLayers() [][][]byte {
	return m.branches
}

func (m *MerkleTrie) insertIntoTrie(item []byte, index int) {
	someItem := bytesutil.ToBytes32(item)
	m.branches[0][index] = someItem[:]
	m.originalItems[index] = someItem[:]
	currentIndex := index
	root := bytesutil.ToBytes32(item)
	for i := 0; i < int(m.depth); i++ {
		isLeft := currentIndex%2 == 0
		neighborIdx := currentIndex ^ 1

		neighbor := make([]byte, 32)
		if m.branches[i] != nil && len(m.branches[i]) != 0 && neighborIdx < len(m.branches[i]) {
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
		// Update the cached layers at the parent index.
		if len(m.branches[i+1]) == 0 {
			newItem := root
			m.branches[i+1] = append(m.branches[i+1], newItem[:])
		} else {
			newItem := root
			m.branches[i+1][parentIdx] = newItem[:]
		}
		currentIndex = parentIdx
	}
}

// GenerateTrieFromItems constructs a Merkle trie from a sequence of byte slices.
func GenerateTrieFromItems(items [][]byte, depth int) (*MerkleTrie, error) {
	if len(items) == 0 {
		return nil, errors.New("no items provided to generate Merkle trie")
	}
	layers := calcTreeFromLeaves(items, depth)
	return &MerkleTrie{
		branches:      layers,
		originalItems: items,
		depth:         uint(depth),
	}, nil
}

// Items returns the original items passed in when creating the Merkle trie.
func (m *MerkleTrie) Items() [][]byte {
	return m.originalItems
}

// Root returns the top-most, Merkle root of the trie.
func (m *MerkleTrie) Root() [32]byte {
	enc := [32]byte{}
	binary.LittleEndian.PutUint64(enc[:], uint64(len(m.originalItems)))
	return hashutil.Hash(append(m.branches[len(m.branches)-1][0], enc[:]...))
}

// MerkleProof computes a proof from a trie's branches using a Merkle index.
func (m *MerkleTrie) MerkleProof(index int) ([][]byte, error) {
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
			proof[i] = zeroHashes[i]
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
func (m *MerkleTrie) HashTreeRoot() [32]byte {
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

// VerifyMerkleProof verifies a Merkle branch against a root of a trie.
func VerifyMerkleProof(root []byte, item []byte, merkleIndex int, proof [][]byte) bool {
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

func calcTreeFromLeaves(leaves [][]byte, depth int) [][][]byte {
	layers := make([][][]byte, depth+1)
	transformedLeaves := make([][]byte, len(leaves))
	for i := range leaves {
		arr := bytesutil.ToBytes32(leaves[i])
		transformedLeaves[i] = arr[:]
	}
	layers[0] = transformedLeaves
	for i := 0; i < depth; i++ {
		if len(layers[i])%2 == 1 {
			layers[i] = append(layers[i], zeroHashes[i])
		}
		updatedValues := make([][]byte, 0, 0)
		for j := 0; j < len(layers[i]); j += 2 {
			concat := hashutil.Hash(append(layers[i][j], layers[i][j+1]...))
			updatedValues = append(updatedValues, concat[:])
		}
		layers[i+1] = updatedValues
	}
	return layers
}

func (m *MerkleTrie) updateTrie() error {
	trie, err := GenerateTrieFromItems(m.originalItems, int(m.depth))
	if err != nil {
		return err
	}
	m.branches = trie.branches
	return nil
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

func prettyPrintProof(proof [][]byte) {
	for i := 0; i < len(proof); i++ {
		fmt.Printf("%#x\n", bytesutil.Trunc(proof[i]))
	}
}
