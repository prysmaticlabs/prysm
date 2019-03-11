package trieutil

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func TestMerkleTrie_BranchIndices(t *testing.T) {
	indices := BranchIndices(1024, 3 /* depth */)
	expected := []int{1024, 512, 256}
	for i := 0; i < len(indices); i++ {
		if expected[i] != indices[i] {
			t.Errorf("Expected %d, received %d", expected[i], indices[i])
		}
	}
}

func TestMerkleTrie_MerkleProofOutOfRange(t *testing.T) {
	h := hashutil.Hash([]byte("hi"))
	m := &MerkleTrie{
		branches: [][][]byte{
			{
				h[:],
			},
			{
				h[:],
			},
			{
				[]byte{},
			},
		},
	}
	if _, err := m.MerkleProof(-1); err == nil {
		t.Error("Expected out of range failure, received nil", err)
	}
	if _, err := m.MerkleProof(2); err == nil {
		t.Error("Expected out of range failure, received nil", err)
	}
	if _, err := m.MerkleProof(0); err == nil {
		t.Error("Expected out of range failure, received nil", err)
	}
}

func TestGenerateTrieFromItems_NoItemsProvided(t *testing.T) {
	if _, err := GenerateTrieFromItems(nil, 32); err == nil {
		t.Error("Expected error when providing nil items received nil")
	}
}

func TestMerkleTrie_VerifyMerkleProof(t *testing.T) {
	items := [][]byte{
		[]byte("short"),
		[]byte("eos"),
		[]byte("long"),
		[]byte("eth"),
		[]byte("4ever"),
		[]byte("eth2"),
		[]byte("moon"),
	}
	m, err := GenerateTrieFromItems(items, 32)
	if err != nil {
		t.Fatalf("Could not generate Merkle trie from items: %v", err)
	}
	proof, err := m.MerkleProof(2)
	if err != nil {
		t.Fatalf("Could not generate Merkle proof: %v", err)
	}
	root := m.Root()
	if ok := VerifyMerkleProof(root[:], items[2], 2, proof); !ok {
		t.Error("Merkle proof did not verify")
	}
	proof, err = m.MerkleProof(3)
	if err != nil {
		t.Fatalf("Could not generate Merkle proof: %v", err)
	}
	if ok := VerifyMerkleProof(root[:], items[3], 3, proof); !ok {
		t.Error("Merkle proof did not verify")
	}
	if ok := VerifyMerkleProof(root[:], []byte("btc"), 3, proof); ok {
		t.Error("Item not in tree should fail to verify")
	}
}

func BenchmarkGenerateTrieFromItems(b *testing.B) {
	items := [][]byte{
		[]byte("short"),
		[]byte("eos"),
		[]byte("long"),
		[]byte("eth"),
		[]byte("4ever"),
		[]byte("eth2"),
		[]byte("moon"),
	}
	for i := 0; i < b.N; i++ {
		if _, err := GenerateTrieFromItems(items, 32); err != nil {
			b.Fatalf("Could not generate Merkle trie from items: %v", err)
		}
	}
}

func BenchmarkVerifyMerkleBranch(b *testing.B) {
	items := [][]byte{
		[]byte("short"),
		[]byte("eos"),
		[]byte("long"),
		[]byte("eth"),
		[]byte("4ever"),
		[]byte("eth2"),
		[]byte("moon"),
	}
	m, err := GenerateTrieFromItems(items, 32)
	if err != nil {
		b.Fatalf("Could not generate Merkle trie from items: %v", err)
	}
	proof, err := m.MerkleProof(2)
	if err != nil {
		b.Fatalf("Could not generate Merkle proof: %v", err)
	}
	for i := 0; i < b.N; i++ {
		if ok := VerifyMerkleProof(m.branches[0][0], items[2], 2, proof); !ok {
			b.Error("Merkle proof did not verify")
		}
	}
}
