package types

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestGenesisCrystallizedState(t *testing.T) {
	cState1, err1 := NewGenesisCrystallizedState()
	cState2, err2 := NewGenesisCrystallizedState()

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to initialize crystallized state: %v %v", err1, err2)
	}

	h1, err1 := cState1.Hash()
	h2, err2 := cState2.Hash()

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to hash crystallized state: %v %v", err1, err2)
	}

	if h1 != h2 {
		t.Fatalf("Hash of two genesis crystallized states should be equal: %x", h1)
	}
}

func TestInitialDeriveCrystallizedState(t *testing.T) {
	cState, err := NewGenesisCrystallizedState()
	if err != nil {
		t.Fatalf("Failed to initialize crystallized state: %v", err)
	}

	aState := NewGenesisActiveState()

	newCState, err := cState.CalculateNewCrystallizedState(aState, 0)
	if err != nil {
		t.Fatalf("failed to derive new crystallized state: %v", err)
	}

	if newCState.LastJustifiedSlot() != 0 {
		t.Fatalf("expected justified slot to equal %d: got %d", 0, newCState.LastFinalizedSlot())
	}

	if newCState.JustifiedStreak() != 0 {
		t.Fatalf("expected justified streak to equal %d: got %d", 0, newCState.JustifiedStreak())
	}

	if newCState.LastStateRecalc() != params.CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", params.CycleLength, newCState.LastStateRecalc())
	}

	if newCState.LastFinalizedSlot() != 0 {
		t.Fatalf("xpected finalized slot to equal %d, got %d", 0, newCState.LastFinalizedSlot())
	}
}

func TestNextDeriveCrystallizedSlot(t *testing.T) {
	cState, err := NewGenesisCrystallizedState()
	if err != nil {
		t.Fatalf("Failed to initialized crystallized state: %v", err)
	}

	aState := NewGenesisActiveState()
	cState, err = cState.CalculateNewCrystallizedState(aState, 0)
	if err != nil {
		t.Fatalf("failed to derive next crystallized state: %v", err)
	}

	totalDeposits := cState.TotalDeposits()
	recentBlockHashes := make([][]byte, 2*params.CycleLength)
	voteCache := make(map[[32]byte]*VoteCache)
	for i := 0; i < 2*params.CycleLength; i++ {
		blockHash := [32]byte{}
		counter := []byte(strconv.Itoa(i))
		copy(blockHash[:], counter)
		recentBlockHashes[i] = blockHash[:]
		voteCache[blockHash] = &VoteCache{
			VoteTotalDeposit: totalDeposits * 3 / 4,
		}
	}

	aState = NewActiveState(&pb.ActiveState{
		RecentBlockHashes: recentBlockHashes,
	}, voteCache)

	cState, err = cState.CalculateNewCrystallizedState(aState, 0)
	if err != nil {
		t.Fatalf("failed to derive crystallized state: %v", err)
	}
	if cState.LastStateRecalc() != 2*params.CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", 2*params.CycleLength, cState.LastStateRecalc())
	}
	if cState.LastJustifiedSlot() != params.CycleLength-1 {
		t.Fatalf("expected justified slot to equal %d: got %d", params.CycleLength-1, cState.LastJustifiedSlot())
	}
	if cState.JustifiedStreak() != params.CycleLength {
		t.Fatalf("expected justified streak to equal %d: got %d", params.CycleLength, cState.JustifiedStreak())
	}
	if cState.LastFinalizedSlot() != 0 {
		t.Fatalf("expected finalized slot to equal %d: got %d", 0, cState.LastFinalizedSlot())
	}

	cState, err = cState.CalculateNewCrystallizedState(aState, 0)
	if err != nil {
		t.Fatalf("failed to derive crystallized state: %v", err)
	}
	if cState.LastStateRecalc() != 3*params.CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", 3*params.CycleLength, cState.LastStateRecalc())
	}
	if cState.LastJustifiedSlot() != 2*params.CycleLength-1 {
		t.Fatalf("expected justified slot to equal %d: got %d", 2*params.CycleLength-1, cState.LastJustifiedSlot())
	}
	if cState.JustifiedStreak() != 2*params.CycleLength {
		t.Fatalf("expected justified streak to equal %d: got %d", 2*params.CycleLength, cState.JustifiedStreak())
	}
	if cState.LastFinalizedSlot() != params.CycleLength-1 {
		t.Fatalf("expected finalized slot to equal %d: got %d", params.CycleLength-1, cState.LastFinalizedSlot())
	}
}

func TestProcessCrosslinks(t *testing.T) {
	// Set up crosslink record for every shard.
	var clRecords []*pb.CrosslinkRecord
	for i := 0; i < params.ShardCount; i++ {
		clRecord := &pb.CrosslinkRecord{Dynasty: 1, Blockhash: []byte{'A'}, Slot: 1}
		clRecords = append(clRecords, clRecord)
	}

	// Set up validators.
	validators := []*pb.ValidatorRecord{
		{
			Balance:      10000,
			StartDynasty: 0,
			EndDynasty:   params.DefaultEndDynasty,
		},
	}

	// Set up pending attestations.
	pAttestations := []*pb.AttestationRecord{
		{
			Slot:             0,
			ShardId:          0,
			ShardBlockHash:   []byte{'a'},
			AttesterBitfield: []byte{'z', 'z'},
		},
	}

	// Process crosslinks happened at slot 50 and dynasty 2.
	shardAndCommitteesForSlots, err := initialShardAndCommitteesForSlots(validators)
	if err != nil {
		t.Fatalf("failed to initialize indices for slots: %v", err)
	}

	cState := NewCrystallizedState(&pb.CrystallizedState{
		CrosslinkRecords:           clRecords,
		Validators:                 validators,
		CurrentDynasty:             5,
		ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
	})
	newCrosslinks, err := cState.processCrosslinks(pAttestations, 50)
	if err != nil {
		t.Fatalf("process crosslink failed %v", err)
	}

	if newCrosslinks[0].Dynasty != 5 {
		t.Errorf("Dynasty did not change for new cross link. Wanted: 5. Got: %d", newCrosslinks[0].Dynasty)
	}
	if newCrosslinks[0].Slot != 50 {
		t.Errorf("Slot did not change for new cross link. Wanted: 50. Got: %d", newCrosslinks[0].Slot)
	}
	if !bytes.Equal(newCrosslinks[0].Blockhash, []byte{'a'}) {
		t.Errorf("Blockhash did not change for new cross link. Wanted a. Got: %s", newCrosslinks[0].Blockhash)
	}
}
