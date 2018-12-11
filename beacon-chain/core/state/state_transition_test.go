package state

import (
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestInitialDeriveState(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize beacon state: %v", err)
	}

	var participationBitfield []byte
	for uint64(len(participationBitfield))*8 < params.BeaconConfig().BootstrappedValidatorsCount {
		participationBitfield = append(participationBitfield, byte(0))
	}

	block := types.NewBlock(&pb.BeaconBlock{
		AncestorHash32S: [][]byte{{'A'}},
		Slot:            0,
		StateRootHash32: []byte{},
		Body: &pb.BeaconBlockBody{
			Attestations: []*pb.Attestation{{
				ParticipationBitfield: participationBitfield,
				Data: &pb.AttestationData{
					Slot:  0,
					Shard: 0,
				},
			}},
		},
	})

	var blockVoteCache utils.BlockVoteCache
	newState, err := NewStateTransition(beaconState, block, 0, blockVoteCache)
	if err != nil {
		t.Fatalf("failed to derive new state: %v", err)
	}

	if newState.LastJustifiedSlot() != 0 {
		t.Fatalf("expected justified slot to equal %d: got %d", 0, newState.LastJustifiedSlot())
	}

	if newState.JustifiedStreak() != 0 {
		t.Fatalf("expected justified streak to equal %d: got %d", 0, newState.JustifiedStreak())
	}

	if newState.LastStateRecalculationSlot() != 1 {
		t.Fatalf("expected last state recalc to equal %d: got %d", 1, newState.LastStateRecalculationSlot())
	}

	if newState.LastFinalizedSlot() != 0 {
		t.Fatalf("expected finalized slot to equal %d, got %d", 0, newState.LastFinalizedSlot())
	}
}

func TestNextDeriveSlot(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialized state: %v", err)
	}

	block := types.NewBlock(&pb.BeaconBlock{
		AncestorHash32S: [][]byte{{'A'}},
		Slot:            0,
	})

	blockVoteCache := utils.NewBlockVoteCache()
	beaconState, err = NewStateTransition(beaconState, block, 0, blockVoteCache)
	if err != nil {
		t.Fatalf("failed to derive next crystallized state: %v", err)
	}

	beaconState.SetValidatorRegistry([]*pb.ValidatorRecord{
		{Balance: uint64(params.BeaconConfig().DepositSize * params.BeaconConfig().Gwei),
			Status: uint64(params.Active)},
	})

	totalDeposits := v.TotalActiveValidatorDeposit(beaconState.ValidatorRegistry())
	recentShardBlockHashes := make([][]byte, 3*params.BeaconConfig().CycleLength)
	for i := 0; i < int(params.BeaconConfig().CycleLength); i++ {
		shardBlockHash := [32]byte{}
		counter := []byte(strconv.Itoa(i))
		copy(shardBlockHash[:], counter)
		recentShardBlockHashes[i] = shardBlockHash[:]
		blockVoteCache[shardBlockHash] = &utils.BlockVote{
			VoteTotalDeposit: totalDeposits * 3 / 4,
		}
	}
	beaconState.SetLatestBlockHashes(recentShardBlockHashes)
	beaconState.SetLastStateRecalculationSlot(params.BeaconConfig().CycleLength - 1)
	block = types.NewBlock(&pb.BeaconBlock{
		AncestorHash32S: [][]byte{{'A'}},
		Slot:            params.BeaconConfig().CycleLength,
	})
	beaconState, err = NewStateTransition(beaconState, block, params.BeaconConfig().CycleLength, blockVoteCache)
	if err != nil {
		t.Fatalf("failed to derive state: %v", err)
	}
	if beaconState.LastStateRecalculationSlot() != params.BeaconConfig().CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", params.BeaconConfig().CycleLength, beaconState.LastStateRecalculationSlot())
	}
	if beaconState.LastJustifiedSlot() != params.BeaconConfig().CycleLength-1 {
		t.Fatalf("expected justified slot to equal %d: got %d", params.BeaconConfig().CycleLength-1, beaconState.LastJustifiedSlot())
	}
	if beaconState.JustifiedStreak() != params.BeaconConfig().CycleLength {
		t.Fatalf("expected justified streak to equal %d: got %d", params.BeaconConfig().CycleLength, beaconState.JustifiedStreak())
	}
	if beaconState.LastFinalizedSlot() != 0 {
		t.Fatalf("expected finalized slot to equal %d: got %d", 0, beaconState.LastFinalizedSlot())
	}

	beaconState.SetLatestBlockHashes(recentShardBlockHashes)
	beaconState.SetLastStateRecalculationSlot(2*params.BeaconConfig().CycleLength - 1)
	block = types.NewBlock(&pb.BeaconBlock{
		AncestorHash32S: [][]byte{{'A'}},
		Slot:            params.BeaconConfig().CycleLength * 2,
	})
	beaconState, err = NewStateTransition(beaconState, block, params.BeaconConfig().CycleLength*2, blockVoteCache)
	if err != nil {
		t.Fatalf("failed to derive state: %v", err)
	}
	if beaconState.LastStateRecalculationSlot() != 2*params.BeaconConfig().CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", 3, beaconState.LastStateRecalculationSlot())
	}
	if beaconState.LastJustifiedSlot() != 2*(params.BeaconConfig().CycleLength-1) {
		t.Fatalf("expected justified slot to equal %d: got %d", 2*params.BeaconConfig().CycleLength-1, beaconState.LastJustifiedSlot())
	}
	if beaconState.JustifiedStreak() != 2*params.BeaconConfig().CycleLength {
		t.Fatalf("expected justified streak to equal %d: got %d", 2*params.BeaconConfig().CycleLength, beaconState.JustifiedStreak())
	}
	if beaconState.LastFinalizedSlot() != params.BeaconConfig().CycleLength-3 {
		t.Fatalf("expected finalized slot to equal %d: got %d", params.BeaconConfig().CycleLength-2, beaconState.LastFinalizedSlot())
	}
}

func TestProcessLatestCrosslinks(t *testing.T) {
	// Set up crosslink record for every shard.
	var clRecords []*pb.CrosslinkRecord
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		clRecord := &pb.CrosslinkRecord{ShardBlockHash: []byte{'A'}, Slot: 1}
		clRecords = append(clRecords, clRecord)
	}

	// Set up validators.
	var validators []*pb.ValidatorRecord

	for i := 0; i < 20; i++ {
		validators = append(validators, &pb.ValidatorRecord{
			Balance: 1e18,
			Status:  uint64(params.Active),
		})
	}

	// Set up latest attestations.
	pAttestations := []*pb.PendingAttestationRecord{
		{
			Data: &pb.AttestationData{
				Slot:             0,
				Shard:            1,
				ShardBlockHash32: []byte{'a'},
			},
		},
	}

	// Process crosslinks happened at slot 50.
	shardAndCommitteesForSlots, err := v.InitialShardAndCommitteesForSlots(validators)
	if err != nil {
		t.Fatalf("failed to initialize indices for slots: %v", err)
	}

	committee := []uint32{0, 4, 6}

	shardAndCommitteesForSlots[0].ArrayShardAndCommittee[0].Committee = committee

	beaconState := types.NewBeaconState(&pb.BeaconState{
		LatestCrosslinks:           clRecords,
		ValidatorRegistry:          validators,
		ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
	})
	newLatestCrosslinks, err := crossLinkCalculations(beaconState, pAttestations, 100)
	if err != nil {
		t.Fatalf("process crosslink failed %v", err)
	}
	_ = newLatestCrosslinks

	// TODO(#781): Pending refactor from new spec.
	//if newLatestCrosslinks[1].Slot != params.BeaconConfig().CycleLength {
	//t.Errorf("Slot did not change for new cross link. Wanted: %d. Got: %d", params.BeaconConfig().CycleLength, newLatestCrosslinks[0].Slot)
	//}
	//if !bytes.Equal(newLatestCrosslinks[1].ShardBlockHash, []byte{'a'}) {
	//t.Errorf("ShardBlockHash did not change for new cross link. Wanted a. Got: %s", newLatestCrosslinks[0].ShardBlockHash)
	//}
	//TODO(#538) Implement tests on balances of the validators in committee once big.Int is introduced.
}

func TestIsNewValidatorSetTransition(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}
	beaconState.SetValidatorRegistryLastChangeSlot(1)
	if beaconState.IsValidatorSetChange(0) {
		t.Errorf("Is new validator set change should be false, last changed slot greater than finalized slot")
	}
	beaconState.SetLastFinalizedSlot(2)
	if beaconState.IsValidatorSetChange(1) {
		t.Errorf("Is new validator set change should be false, MinValidatorSetChangeInterval has not reached")
	}
	shardCommitteeForSlots := []*pb.ShardAndCommitteeArray{{
		ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 0},
			{Shard: 1},
			{Shard: 2},
		},
	},
	}
	beaconState.SetShardAndCommitteesForSlots(shardCommitteeForSlots)

	crosslinks := []*pb.CrosslinkRecord{
		{Slot: 1},
		{Slot: 1},
		{Slot: 1},
	}
	beaconState.SetCrossLinks(crosslinks)

	if beaconState.IsValidatorSetChange(params.BeaconConfig().MinValidatorSetChangeInterval + 1) {
		t.Errorf("Is new validator set change should be false, crosslink slot record is higher than current slot")
	}

	crosslinks = []*pb.CrosslinkRecord{
		{Slot: 2},
		{Slot: 2},
		{Slot: 2},
	}
	beaconState.SetCrossLinks(crosslinks)

	if !beaconState.IsValidatorSetChange(params.BeaconConfig().MinValidatorSetChangeInterval + 1) {
		t.Errorf("New validator set changen failed should have been true")
	}
}

func TestNewValidatorSetRecalculationsInvalid(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}
	// Negative test case, shuffle validators with more than MaxValidatorRegistry.
	size := 1<<(params.BeaconConfig().RandBytes*8) - 1
	validators := make([]*pb.ValidatorRecord, size)
	validator := &pb.ValidatorRecord{Status: uint64(params.Active)}
	for i := 0; i < size; i++ {
		validators[i] = validator
	}
	beaconState.SetValidatorRegistry(validators)
	if _, err := validatorSetRecalculations(
		beaconState.ShardAndCommitteesForSlots(),
		beaconState.ValidatorRegistry(),
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
		beaconState.ValidatorRegistry(),
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
