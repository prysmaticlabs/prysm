package state

import (
	"bytes"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestGenesisState_HashEquality(t *testing.T) {
	state1, _ := NewGenesisBeaconState(nil)
	state2, _ := NewGenesisBeaconState(nil)

	enc1, err1 := proto.Marshal(state1)
	enc2, err2 := proto.Marshal(state2)

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to marshal state to bytes: %v %v", err1, err2)
	}

	h1 := hashutil.Hash(enc1)
	h2 := hashutil.Hash(enc2)
	if h1 != h2 {
		t.Fatalf("Hash of two genesis states should be equal: %#x", h1)
	}
}

func TestGenesisState_InitializesLatestBlockHashes(t *testing.T) {
	s, _ := NewGenesisBeaconState(nil)
	want, got := len(s.GetLatestBlockRootHash32S()), 2*int(params.BeaconConfig().CycleLength)
	if want != got {
		t.Errorf("Wrong number of recent block hashes. Got: %d Want: %d", got, want)
	}

	want = cap(s.GetLatestBlockRootHash32S())
	if want != got {
		t.Errorf("The slice underlying array capacity is wrong. Got: %d Want: %d", got, want)
	}

	zero := make([]byte, 0, 32)
	for _, h := range s.GetLatestBlockRootHash32S() {
		if !bytes.Equal(h, zero) {
			t.Errorf("Unexpected non-zero hash data: %v", h)
		}
	}
}

func TestUpdateAttestationsAfterRecalc(t *testing.T) {
	state, _ := NewGenesisBeaconState(nil)
	newAttestations := []*pb.PendingAttestationRecord{
		{
			Data: &pb.AttestationData{
				Slot:  10,
				Shard: 2,
			},
		},
		{
			Data: &pb.AttestationData{
				Slot:  9,
				Shard: 3,
			},
		},
	}

	state.LatestAttestations = newAttestations
	newAttestations = ClearAttestations(state, 8)
	if len(newAttestations) != 2 {
		t.Fatalf("Updated attestations should be length 2: %d", len(newAttestations))
	}
}

func TestUpdateLatestBlockHashes(t *testing.T) {
	block := &pb.BeaconBlock{
		Slot:             10,
		ParentRootHash32: []byte{'A'},
	}

	recentBlockHashes := make([][]byte, 2*int(params.BeaconConfig().CycleLength))
	for i := 0; i < 2*int(params.BeaconConfig().CycleLength); i++ {
		recentBlockHashes[i] = []byte{0}
	}

	state := &pb.BeaconState{
		LatestBlockRootHash32S: recentBlockHashes,
	}

	updated, err := CalculateNewBlockHashes(state, block, 0)
	if err != nil {
		t.Fatalf("failed to update recent blockhashes: %v", err)
	}

	if len(updated) != 2*int(params.BeaconConfig().CycleLength) {
		t.Fatalf("length of updated recent blockhashes should be %d: found %d", params.BeaconConfig().CycleLength, len(updated))
	}

	for i := 0; i < len(updated); i++ {
		if i < len(updated)-10 {
			if !bytes.Equal(updated[i], []byte{0}) {
				t.Fatalf("update failed: expected %#x got %#x", []byte{0}, updated[i])
			}
		} else if !bytes.Equal(updated[i], block.GetParentRootHash32()) {
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
	copy(s.LatestBlockRootHash32S, interestingData)
	original := make([][]byte, 2*params.BeaconConfig().CycleLength)
	copy(original, s.LatestBlockRootHash32S)

	if !reflect.DeepEqual(s.GetLatestBlockRootHash32S(), original) {
		t.Fatal("setup data should be equal!")
	}

	block := &pb.BeaconBlock{
		Slot:             2,
		ParentRootHash32: []byte{},
	}

	result, _ := CalculateNewBlockHashes(s, block, 0 /*parentSlot*/)

	if !reflect.DeepEqual(s.GetLatestBlockRootHash32S(), original) {
		t.Error("data has mutated from the original")
	}

	if reflect.DeepEqual(result, original) {
		t.Error("the resulting data did not change from the original")
	}
}
