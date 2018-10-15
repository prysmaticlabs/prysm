package types

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestGenesisActiveState_HashEquality(t *testing.T) {
	aState1 := NewGenesisActiveState()
	aState2 := NewGenesisActiveState()

	h1, err1 := aState1.Hash()
	h2, err2 := aState2.Hash()

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to hash crystallized state: %v %v", err1, err2)
	}

	if h1 != h2 {
		t.Fatalf("Hash of two genesis crystallized states should be equal: %#x", h1)
	}
}

func TestGenesisActiveState_InitializesRecentBlockHashes(t *testing.T) {
	as := NewGenesisActiveState()
	want, got := len(as.data.RecentBlockHashes), 2*int(params.GetConfig().CycleLength)
	if want != got {
		t.Errorf("Wrong number of recent block hashes. Got: %d Want: %d", got, want)
	}

	want = cap(as.data.RecentBlockHashes)
	if want != got {
		t.Errorf("The slice underlying array capacity is wrong. Got: %d Want: %d", got, want)
	}

	zero := make([]byte, 0, 32)
	for _, h := range as.data.RecentBlockHashes {
		if !bytes.Equal(h, zero) {
			t.Errorf("Unexpected non-zero hash data: %v", h)
		}
	}
}

func TestUpdateAttestations(t *testing.T) {
	aState := NewGenesisActiveState()

	newAttestations := []*pb.AggregatedAttestation{
		{
			Slot:  0,
			Shard: 0,
		},
		{
			Slot:  0,
			Shard: 1,
		},
	}

	updatedAttestations := aState.appendNewAttestations(newAttestations)
	if len(updatedAttestations) != 2 {
		t.Fatalf("Updated attestations should be length 2: %d", len(updatedAttestations))
	}
}

func TestUpdateAttestationsAfterRecalc(t *testing.T) {
	aState := NewActiveState(&pb.ActiveState{
		PendingAttestations: []*pb.AggregatedAttestation{
			{
				Slot:  0,
				Shard: 0,
			},
			{
				Slot:  0,
				Shard: 1,
			},
		},
	}, nil)

	newAttestations := []*pb.AggregatedAttestation{
		{
			Slot:  10,
			Shard: 2,
		},
		{
			Slot:  9,
			Shard: 3,
		},
	}

	updatedAttestations := aState.appendNewAttestations(newAttestations)
	aState.data.PendingAttestations = updatedAttestations
	updatedAttestations = aState.cleanUpAttestations(8)
	if len(updatedAttestations) != 2 {
		t.Fatalf("Updated attestations should be length 2: %d", len(updatedAttestations))
	}
}

func TestUpdateRecentBlockHashes(t *testing.T) {
	block := NewBlock(&pb.BeaconBlock{
		Slot:           10,
		AncestorHashes: [][]byte{{'A'}},
	})

	recentBlockHashes := [][]byte{}
	for i := 0; i < 2*int(params.GetConfig().CycleLength); i++ {
		recentBlockHashes = append(recentBlockHashes, []byte{0})
	}

	aState := NewActiveState(&pb.ActiveState{
		RecentBlockHashes: recentBlockHashes,
	}, nil)

	updated, err := aState.calculateNewBlockHashes(block, 0)
	if err != nil {
		t.Fatalf("failed to update recent blockhashes: %v", err)
	}

	if len(updated) != 2*int(params.GetConfig().CycleLength) {
		t.Fatalf("length of updated recent blockhashes should be %d: found %d", params.GetConfig().CycleLength, len(updated))
	}

	for i := 0; i < len(updated); i++ {
		if i < len(updated)-10 {
			if !areBytesEqual(updated[i], []byte{0}) {
				t.Fatalf("update failed: expected %#x got %#x", []byte{0}, updated[i])
			}
		} else if !areBytesEqual(updated[i], block.data.AncestorHashes[0]) {
			t.Fatalf("update failed: expected %#x got %#x", block.data.AncestorHashes[:], updated[i])
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

	s := NewGenesisActiveState()
	copy(s.data.RecentBlockHashes, interestingData)
	original := make([][]byte, 2*params.GetConfig().CycleLength)
	copy(original, s.data.RecentBlockHashes)

	if !reflect.DeepEqual(s.data.RecentBlockHashes, original) {
		t.Fatal("setup data should be equal!")
	}

	block := &Block{
		data: &pb.BeaconBlock{
			Slot:           2,
			AncestorHashes: [][]byte{{}},
		},
	}

	result, _ := s.calculateNewBlockHashes(block, 0 /*parentSlot*/)

	if !reflect.DeepEqual(s.data.RecentBlockHashes, original) {
		t.Error("data has mutated from the original")
	}

	if reflect.DeepEqual(result, original) {
		t.Error("the resulting data did not change from the original")
	}
}

func TestBlockVoteCacheNoAttestations(t *testing.T) {
	aState := NewGenesisActiveState()
	cState, err := NewGenesisCrystallizedState("")
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
	cState, err := NewGenesisCrystallizedState("")
	if err != nil {
		t.Fatalf("failed to initialize crystallized state: %v", err)
	}
	block := NewBlock(&pb.BeaconBlock{
		Slot:           1,
		AncestorHashes: [][]byte{},
		Attestations: []*pb.AggregatedAttestation{
			{
				Slot:             0,
				Shard:            1,
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
		Slot:           10,
		AncestorHashes: [][]byte{{}},
	})

	cState, err := NewGenesisCrystallizedState("")
	if err != nil {
		t.Fatalf("failed to initialize genesis crystallized state: %v", err)
	}

	recentBlockHashes := [][]byte{}
	for i := 0; i < 2*int(params.GetConfig().CycleLength); i++ {
		recentBlockHashes = append(recentBlockHashes, []byte{0})
	}

	aState := NewActiveState(&pb.ActiveState{
		PendingAttestations: []*pb.AggregatedAttestation{
			{
				Slot:  0,
				Shard: 0,
			},
			{
				Slot:  0,
				Shard: 1,
			},
		},
		RecentBlockHashes: recentBlockHashes,
	}, nil)

	aState, err = aState.CalculateNewActiveState(block, cState, 0, false)
	if err != nil {
		t.Fatalf("failed to calculate new active state: %v", err)
	}

	if len(aState.PendingAttestations()) != 2 {
		t.Fatalf("expected 2 pending attestations, got %d", len(aState.PendingAttestations()))
	}

	if len(aState.RecentBlockHashes()) != 2*int(params.GetConfig().CycleLength) {
		t.Fatalf("incorrect number of items in RecentBlockHashes: %d", len(aState.RecentBlockHashes()))
	}

	aState, err = aState.CalculateNewActiveState(block, cState, 0, true)
	if err != nil {
		t.Fatalf("failed to calculate new active state: %v", err)
	}

	if len(aState.PendingAttestations()) != 2 {
		t.Fatalf("expected 2 pending attestations, got %d", len(aState.PendingAttestations()))
	}

	if len(aState.RecentBlockHashes()) != 2*int(params.GetConfig().CycleLength) {
		t.Fatalf("incorrect number of items in RecentBlockHashes: %d", len(aState.RecentBlockHashes()))
	}
}

func createHashFromByte(repeatedByte byte) []byte {
	hash := make([]byte, 32)
	for i := 0; i < 32; i++ {
		hash[i] = repeatedByte
	}

	return hash
}

func TestGetSignedParentHashes(t *testing.T) {
	// Test the scenario described in the spec:
	// https://github.com/ethereum/eth2.0-specs/blob/d7458bf201c8fcb93503272c8844381962488cb7/specs/beacon-chain.md#per-block-processing
	cfg := params.GetConfig()
	oldCycleLength := cfg.CycleLength
	cycleLength := uint64(4)
	cfg.CycleLength = cycleLength
	defer func() {
		cfg.CycleLength = oldCycleLength
	}()

	recentBlockHashes := make([][]byte, 11)
	recentBlockHashes[0] = createHashFromByte('Z')
	recentBlockHashes[1] = createHashFromByte('A')
	recentBlockHashes[2] = createHashFromByte('B')
	recentBlockHashes[3] = createHashFromByte('C')
	recentBlockHashes[4] = createHashFromByte('D')
	recentBlockHashes[5] = createHashFromByte('E')
	recentBlockHashes[6] = createHashFromByte('F')
	recentBlockHashes[7] = createHashFromByte('G')
	recentBlockHashes[8] = createHashFromByte('H')
	recentBlockHashes[9] = createHashFromByte('I')
	recentBlockHashes[10] = createHashFromByte('J')

	aState := NewActiveState(&pb.ActiveState{RecentBlockHashes: recentBlockHashes}, nil)

	b := NewBlock(&pb.BeaconBlock{Slot: 11})

	obliqueParentHashes := make([][]byte, 2)
	obliqueParentHashes[0] = createHashFromByte(0)
	obliqueParentHashes[1] = createHashFromByte(1)
	a := &pb.AggregatedAttestation{
		ObliqueParentHashes: obliqueParentHashes,
		Slot:                5,
	}

	hashes, err := aState.getSignedParentHashes(b, a)
	if err != nil {
		t.Fatalf("failed to getSignedParentHashes: %v", err)
	}
	if hashes[0][0] != 'B' || hashes[1][0] != 'C' {
		t.Fatalf("getSignedParentHashes did not return expected value: %#x and %#x", hashes[0], hashes[1])
	}
	if hashes[2][0] != 0 || hashes[3][0] != 1 {
		t.Fatalf("getSignedParentHashes did not return expected value: %#x and %#x", hashes[0], hashes[1])
	}
}

func TestGetSignedParentHashesIndexFail(t *testing.T) {
	cfg := params.GetConfig()
	oldCycleLength := cfg.CycleLength
	cycleLength := uint64(4)
	cfg.CycleLength = cycleLength
	defer func() {
		cfg.CycleLength = oldCycleLength
	}()

	recentBlockHashes := make([][]byte, 8)
	recentBlockHashes[0] = createHashFromByte('Z')
	recentBlockHashes[1] = createHashFromByte('A')
	recentBlockHashes[2] = createHashFromByte('B')
	recentBlockHashes[3] = createHashFromByte('C')
	recentBlockHashes[4] = createHashFromByte('D')
	recentBlockHashes[5] = createHashFromByte('E')
	recentBlockHashes[6] = createHashFromByte('F')
	recentBlockHashes[7] = createHashFromByte('G')

	aState := NewActiveState(&pb.ActiveState{RecentBlockHashes: recentBlockHashes}, nil)

	b := NewBlock(&pb.BeaconBlock{Slot: 8})
	a := &pb.AggregatedAttestation{
		ObliqueParentHashes: [][]byte{},
		Slot:                2,
	}

	_, err := aState.getSignedParentHashes(b, a)
	if err == nil {
		t.Error("expected getSignedParentHashes to fail")
	}

	a2 := &pb.AggregatedAttestation{
		ObliqueParentHashes: [][]byte{},
		Slot:                8,
	}
	_, err = aState.getSignedParentHashes(b, a2)
	if err == nil {
		t.Error("expected getSignedParentHashes to fail")
	}
}
