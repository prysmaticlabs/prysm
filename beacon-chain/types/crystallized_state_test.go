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

	var attesterBitfield []byte
	for len(attesterBitfield)*8 < params.BootstrappedValidatorsCount {
		attesterBitfield = append(attesterBitfield, byte(0))
	}

	aState := NewGenesisActiveState()
	block := NewBlock(&pb.BeaconBlock{
		ParentHash:            []byte{},
		SlotNumber:            0,
		ActiveStateHash:       []byte{},
		CrystallizedStateHash: []byte{},
		Attestations: []*pb.AggregatedAttestation{{
			Slot:             0,
			AttesterBitfield: attesterBitfield,
			ShardId:          0,
		}},
	})

	newCState, _, err := cState.NewStateRecalculations(aState, block)
	if err != nil {
		t.Fatalf("failed to derive new crystallized state: %v", err)
	}

	if newCState.LastJustifiedSlot() != 63 {
		t.Fatalf("expected justified slot to equal %d: got %d", 0, newCState.LastJustifiedSlot())
	}

	if newCState.JustifiedStreak() != 64 {
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
	block := NewBlock(nil)

	cState, _, err = cState.NewStateRecalculations(aState, block)
	if err != nil {
		t.Fatalf("failed to derive next crystallized state: %v", err)
	}

	cState.data.Validators = []*pb.ValidatorRecord{
		{Balance: 2e18,
			StartDynasty: 0,
			EndDynasty:   2},
	}

	totalDeposits := cState.TotalDeposits()
	recentBlockHashes := make([][]byte, 3*params.CycleLength)
	voteCache := make(map[[32]byte]*VoteCache)
	for i := 0; i < 3*int(params.CycleLength); i++ {
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

	cState, _, err = cState.NewStateRecalculations(aState, block)
	if err != nil {
		t.Fatalf("failed to derive crystallized state: %v", err)
	}
	if cState.LastStateRecalc() != 2*params.CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", 2*params.CycleLength, cState.LastStateRecalc())
	}
	if cState.LastJustifiedSlot() != params.CycleLength-1 {
		t.Fatalf("expected justified slot to equal %d: got %d", params.CycleLength-1, cState.LastJustifiedSlot())
	}
	if cState.JustifiedStreak() != 2*params.CycleLength {
		t.Fatalf("expected justified streak to equal %d: got %d", params.CycleLength, cState.JustifiedStreak())
	}
	if cState.LastFinalizedSlot() != 0 {
		t.Fatalf("expected finalized slot to equal %d: got %d", 0, cState.LastFinalizedSlot())
	}

	cState, _, err = cState.NewStateRecalculations(aState, block)
	if err != nil {
		t.Fatalf("failed to derive crystallized state: %v", err)
	}
	if cState.LastStateRecalc() != 3*params.CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", 3*params.CycleLength, cState.LastStateRecalc())
	}
	if cState.LastJustifiedSlot() != 2*params.CycleLength-1 {
		t.Fatalf("expected justified slot to equal %d: got %d", 2*params.CycleLength-1, cState.LastJustifiedSlot())
	}
	if cState.JustifiedStreak() != 3*params.CycleLength {
		t.Fatalf("expected justified streak to equal %d: got %d", 2*params.CycleLength, cState.JustifiedStreak())
	}
	if cState.LastFinalizedSlot() != params.CycleLength-2 {
		t.Fatalf("expected finalized slot to equal %d: got %d", params.CycleLength-2, cState.LastFinalizedSlot())
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
	var validators []*pb.ValidatorRecord

	for i := 0; i < 20; i++ {
		validators = append(validators, &pb.ValidatorRecord{
			Balance:      1e18,
			StartDynasty: 0,
			EndDynasty:   params.DefaultEndDynasty,
		})
	}

	// Set up pending attestations.
	pAttestations := []*pb.AggregatedAttestation{
		{
			Slot:             0,
			ShardId:          0,
			ShardBlockHash:   []byte{'a'},
			AttesterBitfield: []byte{10},
		},
	}

	// Process crosslinks happened at slot 50 and dynasty 2.
	shardAndCommitteesForSlots, err := initialShardAndCommitteesForSlots(validators)
	if err != nil {
		t.Fatalf("failed to initialize indices for slots: %v", err)
	}

	committee := []uint32{0, 4, 6}

	shardAndCommitteesForSlots[0].ArrayShardAndCommittee[0].Committee = committee

	cState := NewCrystallizedState(&pb.CrystallizedState{
		CrosslinkRecords:           clRecords,
		Validators:                 validators,
		CurrentDynasty:             5,
		ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
	})
	newCrosslinks, err := cState.processCrosslinks(pAttestations, 50, 100)
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

	for _, index := range committee {
		if cState.Validators()[index].Balance == 1e18 {
			t.Errorf("validator with index %d did not have balance changed.", index)
		}
	}
}

func TestIsDynastyTransition(t *testing.T) {
	cState, err := NewGenesisCrystallizedState()
	if err != nil {
		t.Fatalf("Failed to initialize crystallized state: %v", err)
	}
	cState.data.DynastyStart = 1
	if cState.isDynastyTransition(0) {
		t.Errorf("Is Dynasty transtion should be false, dynasty start greater than finalized slot")
	}
	cState.data.LastFinalizedSlot = 2
	if cState.isDynastyTransition(1) {
		t.Errorf("Is Dynasty transtion should be false, MinDynastyLength has not reached")
	}
	shardCommitteeForSlots := []*pb.ShardAndCommitteeArray{{
		ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{ShardId: 0},
			{ShardId: 1},
			{ShardId: 2},
		},
	},
	}
	cState.data.ShardAndCommitteesForSlots = shardCommitteeForSlots

	crosslinkRecords := []*pb.CrosslinkRecord{
		{Slot: 1},
		{Slot: 1},
		{Slot: 1},
	}
	cState.data.CrosslinkRecords = crosslinkRecords

	if cState.isDynastyTransition(params.MinDynastyLength + 1) {
		t.Errorf("Is Dynasty transtion should be false, crosslink records dynasty is higher than current slot")
	}

	crosslinkRecords = []*pb.CrosslinkRecord{
		{Slot: 2},
		{Slot: 2},
		{Slot: 2},
	}
	cState.data.CrosslinkRecords = crosslinkRecords

	if !cState.isDynastyTransition(params.MinDynastyLength + 1) {
		t.Errorf("Dynasty transition failed should have been true")
	}
}

func TestNewDynastyRecalculationsInvalid(t *testing.T) {
	cState, err := NewGenesisCrystallizedState()
	if err != nil {
		t.Fatalf("Failed to initialize crystallized state: %v", err)
	}

	// Negative test case, shuffle validators with more than MaxValidators.
	var validators []*pb.ValidatorRecord
	for i := 0; i < params.MaxValidators+1; i++ {
		validators = append(validators, &pb.ValidatorRecord{StartDynasty: 0, EndDynasty: params.DefaultEndDynasty})
	}
	cState.data.Validators = validators
	if _, _, err := cState.newDynastyRecalculations([32]byte{'A'}); err == nil {
		t.Errorf("Dynasty calculation should have failed with invalid validator count")
	}
}

func TestNewDynastyRecalculations(t *testing.T) {
	cState, err := NewGenesisCrystallizedState()
	if err != nil {
		t.Fatalf("Failed to initialize crystallized state: %v", err)
	}

	// Create shard committee for every slot.
	var shardCommitteesForSlot []*pb.ShardAndCommitteeArray
	for i := 0; i < int(params.CycleLength); i++ {
		// Only 10 shards gets crosslinked by validators this dynasty.
		var shardCommittees []*pb.ShardAndCommittee
		for i := 0; i < 10; i++ {
			shardCommittees = append(shardCommittees, &pb.ShardAndCommittee{ShardId: uint64(i)})
		}
		shardCommitteesForSlot = append(shardCommitteesForSlot, &pb.ShardAndCommitteeArray{ArrayShardAndCommittee: shardCommittees})
	}

	cState.data.ShardAndCommitteesForSlots = shardCommitteesForSlot
	cState.data.CurrentDynasty = 1
	cState.data.LastStateRecalc = 65

	shardCount = 10
	currentDynasty, _, err := cState.newDynastyRecalculations([32]byte{'A'})
	if err != nil {
		t.Fatalf("Dynasty calculation failed %v", err)
	}

	if currentDynasty != 2 {
		t.Errorf("Incorrect dynasty number, wanted 2, got: %d", currentDynasty)
	}
}
