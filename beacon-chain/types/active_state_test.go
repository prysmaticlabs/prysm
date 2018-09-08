package types

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestGenesisActiveState(t *testing.T) {
	aState1 := NewGenesisActiveState()
	aState2 := NewGenesisActiveState()

	h1, err1 := aState1.Hash()
	h2, err2 := aState2.Hash()

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to hash crystallized state: %v %v", err1, err2)
	}

	if h1 != h2 {
		t.Fatalf("Hash of two genesis crystallized states should be equal: %x", h1)
	}
}

func TestUpdateAttestations(t *testing.T) {
	aState := NewGenesisActiveState()

	newAttestations := []*pb.AttestationRecord{
		{
			Slot:    0,
			ShardId: 0,
		},
		{
			Slot:    0,
			ShardId: 1,
		},
	}

	updatedAttestations := aState.calculateNewAttestations(newAttestations, 0)
	if len(updatedAttestations) != 2 {
		t.Fatalf("Updated attestations should be length 2: %d", len(updatedAttestations))
	}
}

func TestUpdateAttestationsAfterRecalc(t *testing.T) {
	aState := NewActiveState(&pb.ActiveState{
		PendingAttestations: []*pb.AttestationRecord{
			{
				Slot:    0,
				ShardId: 0,
			},
			{
				Slot:    0,
				ShardId: 1,
			},
		},
	}, nil)

	newAttestations := []*pb.AttestationRecord{
		{
			Slot:    10,
			ShardId: 2,
		},
		{
			Slot:    9,
			ShardId: 3,
		},
	}

	updatedAttestations := aState.calculateNewAttestations(newAttestations, 7)
	if len(updatedAttestations) != 2 {
		t.Fatalf("Updated attestations should be length 2: %d", len(updatedAttestations))
	}
}

func TestUpdateRecentBlockHashes(t *testing.T) {
	block := NewBlock(&pb.BeaconBlock{
		SlotNumber: 10,
	})

	recentBlockHashes := [][]byte{}
	for i := 0; i < 2*params.CycleLength; i++ {
		recentBlockHashes = append(recentBlockHashes, []byte{0})
	}

	aState := NewActiveState(&pb.ActiveState{
		RecentBlockHashes: recentBlockHashes,
	}, nil)

	updated, err := aState.calculateNewBlockHashes(block, 0)
	if err != nil {
		t.Fatalf("failed to update recent blockhashes: %v", err)
	}

	if len(updated) != 2*params.CycleLength {
		t.Fatalf("length of updated recent blockhashes should be %d: found %d", params.CycleLength, len(updated))
	}

	hash, err := block.Hash()
	if err != nil {
		t.Fatalf("failed to hash block: %v", err)
	}
	for i := 0; i < len(updated); i++ {
		if i < len(updated)-10 {
			if !areBytesEqual(updated[i], []byte{0}) {
				t.Fatalf("update failed: expected %x got %x", []byte{0}, updated[i])
			}
		} else if !areBytesEqual(updated[i], hash[:]) {
			t.Fatalf("update failed: expected %x got %x", hash[:], updated[i])
		}
	}
}

func TestBlockVoteCacheNoAttestations(t *testing.T) {
	aState := NewGenesisActiveState()
	cState, err := NewGenesisCrystallizedState()
	if err != nil {
		t.Fatalf("failed to initialize crystallized state: %v", err)
	}
	block := NewBlock(nil)

	newVoteCache, err := aState.calculateNewVoteCache(block, cState)
	if err != nil {
		t.Fatalf("failed to update the block vote cache: %v", err)
	}

	if len(newVoteCache) != 0 {
		t.Fatalf("expected no new votes in cache: found %d", len(newVoteCache))
	}
}

func TestBlockVoteCache(t *testing.T) {
	aState := NewGenesisActiveState()
	cState, err := NewGenesisCrystallizedState()
	if err != nil {
		t.Fatalf("failed to initialize crystallized state: %v", err)
	}
	block := NewBlock(&pb.BeaconBlock{
		SlotNumber: 1,
		Attestations: []*pb.AttestationRecord{
			{
				Slot:             0,
				ShardId:          0,
				AttesterBitfield: []byte{'F', 'F'},
			},
		},
	})

	newVoteCache, err := aState.calculateNewVoteCache(block, cState)
	if err != nil {
		t.Fatalf("failed to update the block vote cache: %v", err)
	}

	if len(newVoteCache) != 1 {
		t.Fatalf("expected one new votes in cache: found %d", len(newVoteCache))
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

func TestCalculateNewActiveState(t *testing.T) {
	block := NewBlock(&pb.BeaconBlock{
		SlotNumber: 10,
	})

	cState, err := NewGenesisCrystallizedState()
	if err != nil {
		t.Fatalf("failed to initialize genesis crystallized state: %v", err)
	}

	recentBlockHashes := [][]byte{}
	for i := 0; i < 2*params.CycleLength; i++ {
		recentBlockHashes = append(recentBlockHashes, []byte{0})
	}

	aState := NewActiveState(&pb.ActiveState{
		PendingAttestations: []*pb.AttestationRecord{
			{
				Slot:    0,
				ShardId: 0,
			},
			{
				Slot:    0,
				ShardId: 1,
			},
		},
		RecentBlockHashes: recentBlockHashes,
	}, nil)

	aState, err = aState.CalculateNewActiveState(block, cState, 0)
	if err != nil {
		t.Fatalf("failed to calculate new active state: %v", err)
	}

	if len(aState.PendingAttestations()) != 2 {
		t.Fatalf("expected 2 pending attestations, got %d", len(aState.PendingAttestations()))
	}

	if len(aState.RecentBlockHashes()) != 2*params.CycleLength {
		t.Fatalf("incorrect number of items in RecentBlockHashes: %d", len(aState.RecentBlockHashes()))
	}
}
