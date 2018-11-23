package state

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestInitialDeriveCrystallizedState(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize beacon state: %v", err)
	}

	var attesterBitfield []byte
	for uint64(len(attesterBitfield))*8 < params.BeaconConfig().BootstrappedValidatorsCount {
		attesterBitfield = append(attesterBitfield, byte(0))
	}

	block := types.NewBlock(&pb.BeaconBlock{
		AncestorHashes:        [][]byte{},
		Slot:                  0,
		ActiveStateRoot:       []byte{},
		CrystallizedStateRoot: []byte{},
		Attestations: []*pb.AggregatedAttestation{{
			Slot:             0,
			AttesterBitfield: attesterBitfield,
			Shard:            0,
		}},
	})

	var blockVoteCache utils.BlockVoteCache
	newState, err := NewStateTransition(beaconState, block, blockVoteCache)
	if err != nil {
		t.Fatalf("failed to derive new state: %v", err)
	}

	if newState.LastJustifiedSlot() != 0 {
		t.Fatalf("expected justified slot to equal %d: got %d", 0, newState.LastJustifiedSlot())
	}

	if newState.JustifiedStreak() != 0 {
		t.Fatalf("expected justified streak to equal %d: got %d", 0, newState.JustifiedStreak())
	}

	if newState.LastStateRecalculationSlot() != params.BeaconConfig().CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", params.BeaconConfig().CycleLength, newState.LastStateRecalculationSlot())
	}

	if newState.LastFinalizedSlot() != 0 {
		t.Fatalf("xpected finalized slot to equal %d, got %d", 0, newState.LastFinalizedSlot())
	}
}

// func TestNextDeriveSlot(t *testing.T) {
// 	beaconState, err := types.NewGenesisBeaconState(nil)
// 	if err != nil {
// 		t.Fatalf("Failed to initialized state: %v", err)
// 	}

// 	block := types.NewBlock(nil)

// 	blockVoteCache := utils.NewBlockVoteCache()
// 	beaconState, err = NewStateTransition(beaconState, block, blockVoteCache)
// 	if err != nil {
// 		t.Fatalf("failed to derive next crystallized state: %v", err)
// 	}

// 	beaconState.SetValidators([]*pb.ValidatorRecord{
// 		{Balance: uint64(params.BeaconConfig().DepositSize * params.BeaconConfig().Gwei),
// 			Status: uint64(params.Active)},
// 	})

// 	totalDeposits := v.TotalActiveValidatorDeposit(beaconState.Validators())
// 	recentShardBlockHashes := make([][]byte, 3*params.BeaconConfig().CycleLength)
// 	for i := 0; i < 3*int(params.BeaconConfig().CycleLength); i++ {
// 		shardBlockHash := [32]byte{}
// 		counter := []byte(strconv.Itoa(i))
// 		copy(shardBlockHash[:], counter)
// 		recentShardBlockHashes[i] = shardBlockHash[:]
// 		blockVoteCache[shardBlockHash] = &utils.BlockVote{
// 			VoteTotalDeposit: totalDeposits * 3 / 4,
// 		}
// 	}

// 	beaconState.SetRecentBlockHashes(recentShardBlockHashes)

// 	beaconState, err = NewStateTransition(beaconState, block, blockVoteCache)
// 	if err != nil {
// 		t.Fatalf("failed to derive state: %v", err)
// 	}
// 	if beaconState.LastStateRecalculationSlot() != 2*params.BeaconConfig().CycleLength {
// 		t.Fatalf("expected last state recalc to equal %d: got %d", 2*params.BeaconConfig().CycleLength, beaconState.LastStateRecalculationSlot())
// 	}
// 	if beaconState.LastJustifiedSlot() != params.BeaconConfig().CycleLength-1 {
// 		t.Fatalf("expected justified slot to equal %d: got %d", params.BeaconConfig().CycleLength-1, beaconState.LastJustifiedSlot())
// 	}
// 	if beaconState.JustifiedStreak() != params.BeaconConfig().CycleLength {
// 		t.Fatalf("expected justified streak to equal %d: got %d", params.BeaconConfig().CycleLength, beaconState.JustifiedStreak())
// 	}
// 	if beaconState.LastFinalizedSlot() != 0 {
// 		t.Fatalf("expected finalized slot to equal %d: got %d", 0, beaconState.LastFinalizedSlot())
// 	}

// 	beaconState, err = NewStateTransition(beaconState, block, blockVoteCache)
// 	if err != nil {
// 		t.Fatalf("failed to derive state: %v", err)
// 	}
// 	if beaconState.LastStateRecalculationSlot() != 3*params.BeaconConfig().CycleLength {
// 		t.Fatalf("expected last state recalc to equal %d: got %d", 3*params.BeaconConfig().CycleLength, beaconState.LastStateRecalculationSlot())
// 	}
// 	if beaconState.LastJustifiedSlot() != 2*params.BeaconConfig().CycleLength-1 {
// 		t.Fatalf("expected justified slot to equal %d: got %d", 2*params.BeaconConfig().CycleLength-1, beaconState.LastJustifiedSlot())
// 	}
// 	if beaconState.JustifiedStreak() != 2*params.BeaconConfig().CycleLength {
// 		t.Fatalf("expected justified streak to equal %d: got %d", 2*params.BeaconConfig().CycleLength, beaconState.JustifiedStreak())
// 	}
// 	if beaconState.LastFinalizedSlot() != params.BeaconConfig().CycleLength-2 {
// 		t.Fatalf("expected finalized slot to equal %d: got %d", params.BeaconConfig().CycleLength-2, beaconState.LastFinalizedSlot())
// 	}

// 	beaconState, err = NewStateTransition(beaconState, block, blockVoteCache)
// 	if err != nil {
// 		t.Fatalf("failed to derive state: %v", err)
// 	}
// 	if beaconState.LastStateRecalculationSlot() != 4*params.BeaconConfig().CycleLength {
// 		t.Fatalf("expected last state recalc to equal %d: got %d", 3*params.BeaconConfig().CycleLength, beaconState.LastStateRecalculationSlot())
// 	}
// 	if beaconState.LastJustifiedSlot() != 3*params.BeaconConfig().CycleLength-1 {
// 		t.Fatalf("expected justified slot to equal %d: got %d", 3*params.BeaconConfig().CycleLength-1, beaconState.LastJustifiedSlot())
// 	}
// 	if beaconState.JustifiedStreak() != 3*params.BeaconConfig().CycleLength {
// 		t.Fatalf("expected justified streak to equal %d: got %d", 3*params.BeaconConfig().CycleLength, beaconState.JustifiedStreak())
// 	}
// 	if beaconState.LastFinalizedSlot() != 2*params.BeaconConfig().CycleLength-2 {
// 		t.Fatalf("expected finalized slot to equal %d: got %d", 2*params.BeaconConfig().CycleLength-2, beaconState.LastFinalizedSlot())
// 	}
// }

// func TestProcessCrosslinks(t *testing.T) {
// 	// Set up crosslink record for every shard.
// 	var clRecords []*pb.CrosslinkRecord
// 	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
// 		clRecord := &pb.CrosslinkRecord{ShardBlockHash: []byte{'A'}, Slot: 1}
// 		clRecords = append(clRecords, clRecord)
// 	}

// 	// Set up validators.
// 	var validators []*pb.ValidatorRecord

// 	for i := 0; i < 20; i++ {
// 		validators = append(validators, &pb.ValidatorRecord{
// 			Balance: 1e18,
// 			Status:  uint64(params.Active),
// 		})
// 	}

// 	// Set up pending attestations.
// 	pAttestations := []*pb.AggregatedAttestation{
// 		{
// 			Slot:             0,
// 			Shard:            1,
// 			ShardBlockHash:   []byte{'a'},
// 			AttesterBitfield: []byte{224},
// 		},
// 	}

// 	// Process crosslinks happened at slot 50.
// 	shardAndCommitteesForSlots, err := v.InitialShardAndCommitteesForSlots(validators)
// 	if err != nil {
// 		t.Fatalf("failed to initialize indices for slots: %v", err)
// 	}

// 	committee := []uint32{0, 4, 6}

// 	shardAndCommitteesForSlots[0].ArrayShardAndCommittee[0].Committee = committee

// 	beaconState := types.NewBeaconState(&pb.BeaconState{
// 		Crosslinks:                 clRecords,
// 		Validators:                 validators,
// 		ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
// 	})
// 	newCrosslinks, err := crossLinkCalculations(beaconState, pAttestations, 100)
// 	if err != nil {
// 		t.Fatalf("process crosslink failed %v", err)
// 	}

// 	if newCrosslinks[1].Slot != params.BeaconConfig().CycleLength {
// 		t.Errorf("Slot did not change for new cross link. Wanted: %d. Got: %d", params.BeaconConfig().CycleLength, newCrosslinks[0].Slot)
// 	}
// 	if !bytes.Equal(newCrosslinks[1].ShardBlockHash, []byte{'a'}) {
// 		t.Errorf("ShardBlockHash did not change for new cross link. Wanted a. Got: %s", newCrosslinks[0].ShardBlockHash)
// 	}
// 	//TODO(#538) Implement tests on balances of the validators in committee once big.Int is introduced.
// }

// func TestIsNewValidatorSetTransition(t *testing.T) {
// 	cState, err := NewGenesisCrystallizedState(nil)
// 	if err != nil {
// 		t.Fatalf("Failed to initialize crystallized state: %v", err)
// 	}
// 	cState.data.ValidatorSetChangeSlot = 1
// 	if cState.isValidatorSetChange(0) {
// 		t.Errorf("Is new validator set change should be false, last changed slot greater than finalized slot")
// 	}
// 	cState.data.LastFinalizedSlot = 2
// 	if cState.isValidatorSetChange(1) {
// 		t.Errorf("Is new validator set change should be false, MinValidatorSetChangeInterval has not reached")
// 	}
// 	shardCommitteeForSlots := []*pb.ShardAndCommitteeArray{{
// 		ArrayShardAndCommittee: []*pb.ShardAndCommittee{
// 			{Shard: 0},
// 			{Shard: 1},
// 			{Shard: 2},
// 		},
// 	},
// 	}
// 	cState.data.ShardAndCommitteesForSlots = shardCommitteeForSlots

// 	crosslinks := []*pb.CrosslinkRecord{
// 		{Slot: 1},
// 		{Slot: 1},
// 		{Slot: 1},
// 	}
// 	cState.data.Crosslinks = crosslinks

// 	if cState.isValidatorSetChange(params.BeaconConfig().MinValidatorSetChangeInterval + 1) {
// 		t.Errorf("Is new validator set change should be false, crosslink slot record is higher than current slot")
// 	}

// 	crosslinks = []*pb.CrosslinkRecord{
// 		{Slot: 2},
// 		{Slot: 2},
// 		{Slot: 2},
// 	}
// 	cState.data.Crosslinks = crosslinks

// 	if !cState.isValidatorSetChange(params.BeaconConfig().MinValidatorSetChangeInterval + 1) {
// 		t.Errorf("New validator set changen failed should have been true")
// 	}
// }

func TestNewValidatorSetRecalculationsInvalid(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}
	// Negative test case, shuffle validators with more than MaxValidators.
	size := params.BeaconConfig().ModuloBias + 1
	validators := make([]*pb.ValidatorRecord, size)
	validator := &pb.ValidatorRecord{Status: uint64(params.Active)}
	for i := uint64(0); i < size; i++ {
		validators[i] = validator
	}
	beaconState.SetValidators(validators)
	if _, err := validatorSetRecalculations(
		beaconState.ShardAndCommitteesForSlots(),
		beaconState.Validators(),
		[32]byte{'A'},
	); err == nil {
		t.Error("Validator set change calculation should have failed with invalid validator count")
	}
}

func TestNewValidatorSetRecalculations(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	// Create shard committee for every slot.
	var shardCommitteesForSlot []*pb.ShardAndCommitteeArray
	for i := 0; i < int(params.BeaconConfig().CycleLength); i++ {
		// Only 10 shards gets crosslinked by validators this period.
		var shardCommittees []*pb.ShardAndCommittee
		for i := 0; i < 10; i++ {
			shardCommittees = append(shardCommittees, &pb.ShardAndCommittee{Shard: uint64(i)})
		}
		shardCommitteesForSlot = append(shardCommitteesForSlot, &pb.ShardAndCommitteeArray{ArrayShardAndCommittee: shardCommittees})
	}

	beaconState.SetShardAndCommitteesForSlots(shardCommitteesForSlot)
	beaconState.SetLastStateRecalculationSlot(65)

	_, err = validatorSetRecalculations(
		beaconState.ShardAndCommitteesForSlots(),
		beaconState.Validators(),
		[32]byte{'A'},
	)
	if err != nil {
		t.Fatalf("Validator set change failed %v", err)
	}
}

func TestPenalizedETH(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}
	beaconState.SetDepositsPenalizedInPeriod([]uint64{100, 200, 300, 400, 500})
	beaconState.PenalizedETH(2)

	tests := []struct {
		a uint64
		b uint64
	}{
		{a: 0, b: 100},
		{a: 1, b: 300},
		{a: 2, b: 600},
		{a: 3, b: 900},
		{a: 4, b: 1200},
	}
	for _, tt := range tests {
		if beaconState.PenalizedETH(tt.a) != tt.b {
			t.Errorf("PenalizedETH(%d) = %v, want = %d", tt.a, beaconState.PenalizedETH(tt.a), tt.b)
		}
	}
}
