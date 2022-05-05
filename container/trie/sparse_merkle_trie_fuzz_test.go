package trie_test

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
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
	f.Add(proto.MarshalTextString(pb))

	f.Fuzz(func(t *testing.T, s string) {
		pb := &ethpb.SparseMerkleTrie{}
		if err := proto.UnmarshalText(s, pb); err != nil {
			return
		}
		trie.CreateTrieFromProto(pb).HashTreeRoot()
	})
}
