package types

import (
	"bytes"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestGenesisState_HashEquality(t *testing.T) {
	state1, _ := NewGenesisBeaconState(nil)
	state2, _ := NewGenesisBeaconState(nil)

	h1, err1 := state1.Hash()
	h2, err2 := state2.Hash()

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to hash state: %v %v", err1, err2)
	}

	if h1 != h2 {
		t.Fatalf("Hash of two genesis states should be equal: %#x", h1)
	}
}

func TestGenesisState_InitializesLatestBlockHashes(t *testing.T) {
	s, _ := NewGenesisBeaconState(nil)
	want, got := len(s.data.LatestBlockRootHash32S), 2*int(params.BeaconConfig().CycleLength)
	if want != got {
		t.Errorf("Wrong number of recent block hashes. Got: %d Want: %d", got, want)
	}

	want = cap(s.data.LatestBlockRootHash32S)
	if want != got {
		t.Errorf("The slice underlying array capacity is wrong. Got: %d Want: %d", got, want)
	}

	zero := make([]byte, 0, 32)
	for _, h := range s.data.LatestBlockRootHash32S {
		if !bytes.Equal(h, zero) {
			t.Errorf("Unexpected non-zero hash data: %v", h)
		}
	}
}

func TestCopyState(t *testing.T) {
	state1, _ := NewGenesisBeaconState(nil)
	state2 := state1.CopyState()

	newAttestations := []*pb.Attestation{
		{
			Slot:  0,
			Shard: 1,
		},
	}

	state1.data.PendingAttestations = append(state1.data.PendingAttestations, newAttestations...)
	if len(state1.data.PendingAttestations) == len(state2.data.PendingAttestations) {
		t.Fatalf("The PendingAttestations should not equal each other %d, %d",
			len(state1.data.PendingAttestations),
			len(state2.data.PendingAttestations),
		)
	}

	state1.data.LatestBlockRootHash32S = [][]byte{{'A'}}
	if len(state1.LatestBlockRootHashes32()) == len(state2.LatestBlockRootHashes32()) {
		t.Fatalf("The LatestBlockHashes should not equal each other %d, %d",
			len(state1.LatestBlockRootHashes32()),
			len(state2.LatestBlockRootHashes32()),
		)
	}

	state1.data.RandaoMixHash32 = []byte{22, 21}
	state2.data.RandaoMixHash32 = []byte{40, 31}
	if state1.data.RandaoMixHash32[0] == state2.data.RandaoMixHash32[0] {
		t.Fatalf("The RandaoMix should not equal each other %d, %d",
			state1.data.RandaoMixHash32[0],
			state2.data.RandaoMixHash32[0],
		)
	}
}

func TestUpdateAttestations(t *testing.T) {
	state, _ := NewGenesisBeaconState(nil)

	newAttestations := []*pb.Attestation{
		{
			Slot:  0,
			Shard: 0,
		},
		{
			Slot:  0,
			Shard: 1,
		},
	}

	state.SetPendingAttestations(newAttestations)
	attestations := state.data.PendingAttestations
	if len(attestations) != 2 {
		t.Fatalf("Updated attestations should be length 2: %d", len(attestations))
	}
}

func TestUpdateAttestationsAfterRecalc(t *testing.T) {
	state, _ := NewGenesisBeaconState(nil)
	newAttestations := []*pb.Attestation{
		{
			Slot:  10,
			Shard: 2,
		},
		{
			Slot:  9,
			Shard: 3,
		},
	}

	state.SetPendingAttestations(newAttestations)
	state.ClearAttestations(8)
	if len(state.PendingAttestations()) != 2 {
		t.Fatalf("Updated attestations should be length 2: %d", len(state.PendingAttestations()))
	}
}

func TestUpdateLatestBlockHashes(t *testing.T) {
	block := &pb.BeaconBlock{
		Slot:             10,
		ParentRootHash32: []byte{'A'},
	}

	recentBlockHashes := [][]byte{}
	for i := 0; i < 2*int(params.BeaconConfig().CycleLength); i++ {
		recentBlockHashes = append(recentBlockHashes, []byte{0})
	}

	state := NewBeaconState(&pb.BeaconState{
		LatestBlockRootHash32S: recentBlockHashes,
	})

	updated, err := state.CalculateNewBlockHashes(block, 0)
	if err != nil {
		t.Fatalf("failed to update recent blockhashes: %v", err)
	}

	if len(updated) != 2*int(params.BeaconConfig().CycleLength) {
		t.Fatalf("length of updated recent blockhashes should be %d: found %d", params.BeaconConfig().CycleLength, len(updated))
	}

	for i := 0; i < len(updated); i++ {
		if i < len(updated)-10 {
			if !areBytesEqual(updated[i], []byte{0}) {
				t.Fatalf("update failed: expected %#x got %#x", []byte{0}, updated[i])
			}
		} else if !areBytesEqual(updated[i], block.GetParentRootHash32()) {
			t.Fatalf("update failed: expected %#x got %#x", block.GetParentRootHash32(), updated[i])
		}
	}
}

func TestCalculateNewBlockHashes_DoesNotMutateData(t *testing.T) {
	interestingData := [][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte("block"),
		[]byte("hash"),
	}

	s, _ := NewGenesisBeaconState(nil)
	copy(s.data.LatestBlockRootHash32S, interestingData)
	original := make([][]byte, 2*params.BeaconConfig().CycleLength)
	copy(original, s.data.LatestBlockRootHash32S)

	if !reflect.DeepEqual(s.data.LatestBlockRootHash32S, original) {
		t.Fatal("setup data should be equal!")
	}

	block := &pb.BeaconBlock{
		Slot:             2,
		ParentRootHash32: []byte{},
	}

	result, _ := s.CalculateNewBlockHashes(block, 0 /*parentSlot*/)

	if !reflect.DeepEqual(s.data.LatestBlockRootHash32S, original) {
		t.Error("data has mutated from the original")
	}

	if reflect.DeepEqual(result, original) {
		t.Error("the resulting data did not change from the original")
	}
}

func areBytesEqual(s1, s2 []byte) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}
