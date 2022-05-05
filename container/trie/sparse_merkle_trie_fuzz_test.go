package trie_test

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func FuzzSparseMerkleTrie_HashTreeRoot(f *testing.F) {
	h := hash.Hash([]byte("hi"))
	pb := &ethpb.SparseMerkleTrie{
		Layers: []*ethpb.TrieLayer{
			{
				Layer: [][]byte{h[:]},
			},
			{
				Layer: [][]byte{h[:]},
			},
			{
				Layer: [][]byte{},
			},
		},
		Depth: 4,
	}
	b, err := proto.Marshal(pb)
	require.NoError(f, err)
	f.Add(b)

	f.Fuzz(func(t *testing.T, b []byte) {
		pb := &ethpb.SparseMerkleTrie{}
		if err := proto.Unmarshal(b, pb); err != nil {
			return
		}
		trie.CreateTrieFromProto(pb).HashTreeRoot()
	})
}

func FuzzSparseMerkleTrie_MerkleProof(f *testing.F) {
	h := hash.Hash([]byte("hi"))
	pb := &ethpb.SparseMerkleTrie{
		Layers: []*ethpb.TrieLayer{
			{
				Layer: [][]byte{h[:]},
			},
			{
				Layer: [][]byte{h[:]},
			},
			{
				Layer: [][]byte{},
			},
		},
		Depth: 4,
	}
	b, err := proto.Marshal(pb)
	require.NoError(f, err)
	f.Add(b, 0)

	f.Fuzz(func(t *testing.T, b []byte, i int) {
		pb := &ethpb.SparseMerkleTrie{}
		if err := proto.Unmarshal(b, pb); err != nil {
			return
		}
		trie.CreateTrieFromProto(pb).MerkleProof(i)
	})
}
