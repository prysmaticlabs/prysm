package types

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	b "github.com/prysmaticlabs/prysm/shared/bytes"
)

func TestGenesisCrystallizedState(t *testing.T) {
	cState1, err1 := NewGenesisCrystallizedState(nil)
	cState2, err2 := NewGenesisCrystallizedState(nil)

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to initialize crystallized state: %v %v", err1, err2)
	}

	h1, err1 := cState1.Hash()
	h2, err2 := cState2.Hash()

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to hash crystallized state: %v %v", err1, err2)
	}

	if h1 != h2 {
		t.Fatalf("Hash of two genesis crystallized states should be equal: %#x", h1)
	}
}

func TestCopyCrystallizedState(t *testing.T) {
	cState1, err1 := NewGenesisCrystallizedState(nil)
	cState2 := cState1.CopyState()

	if err1 != nil {
		t.Fatalf("Failed to initialize crystallized state: %v", err1)
	}

	cState1.data.LastStateRecalculationSlot = 40
	if cState1.LastStateRecalculationSlot() == cState2.LastStateRecalculationSlot() {
		t.Fatalf("The Last State Recalculation Slot should not be equal: %d %d",
			cState1.LastStateRecalculationSlot(),
			cState2.LastStateRecalculationSlot(),
		)
	}

	cState1.data.JustifiedStreak = 40
	if cState1.JustifiedStreak() == cState2.JustifiedStreak() {
		t.Fatalf("The Justified Streak should not be equal: %d %d",
			cState1.JustifiedStreak(),
			cState2.JustifiedStreak(),
		)
	}

	cState1.data.LastJustifiedSlot = 40
	if cState1.LastJustifiedSlot() == cState2.LastJustifiedSlot() {
		t.Fatalf("The Last Justified Slot should not be equal: %d %d",
			cState1.LastJustifiedSlot(),
			cState2.LastJustifiedSlot(),
		)
	}

	cState1.data.LastFinalizedSlot = 40
	if cState1.LastFinalizedSlot() == cState2.LastFinalizedSlot() {
		t.Fatalf("The Last Finalized Slot should not be equal: %d %d",
			cState1.LastFinalizedSlot(),
			cState2.LastFinalizedSlot(),
		)
	}

	cState1.data.ValidatorSetChangeSlot = 40
	if cState1.ValidatorSetChangeSlot() == cState2.ValidatorSetChangeSlot() {
		t.Fatalf("The Last Validator Set Change Slot should not be equal: %d %d",
			cState1.ValidatorSetChangeSlot(),
			cState2.ValidatorSetChangeSlot(),
		)
	}

	var crosslinks []*pb.CrosslinkRecord
	for i := uint64(0); i < shardCount; i++ {
		crosslinks = append(crosslinks, &pb.CrosslinkRecord{
			RecentlyChanged: false,
			ShardBlockHash:  make([]byte, 2, 34),
			Slot:            2,
		})
	}
	cState1.data.Crosslinks = crosslinks
	if cState1.Crosslinks()[0].Slot == cState2.Crosslinks()[0].Slot {
		t.Fatalf("The Crosslinks should not be equal: %d %d",
			cState1.Crosslinks()[0].Slot,
			cState2.Crosslinks()[0].Slot,
		)
	}

	cState1.data.Validators = append(cState1.Validators(), &pb.ValidatorRecord{Balance: 32 * 1e9, Status: uint64(params.Active)})
	if len(cState1.Validators()) == len(cState2.Validators()) {
		t.Fatalf("The Validators should be equal: %d %d",
			len(cState1.Validators()),
			len(cState2.Validators()),
		)
	}

	newArray := &pb.ShardAndCommitteeArray{
		ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4}},
			{Shard: 2, Committee: []uint32{5, 6, 7, 8, 9}},
		},
	}
	cState1.data.ShardAndCommitteesForSlots = append(cState1.ShardAndCommitteesForSlots(), newArray)
	if len(cState1.ShardAndCommitteesForSlots()) == len(cState2.ShardAndCommitteesForSlots()) {
		t.Fatalf("The ShardAndCommitteesForSlots shouldnt be equal: %d %d",
			cState1.ShardAndCommitteesForSlots(),
			cState2.ShardAndCommitteesForSlots(),
		)
	}
}

func TestInitialDeriveCrystallizedState(t *testing.T) {
	cState, err := NewGenesisCrystallizedState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize crystallized state: %v", err)
	}

	var attesterBitfield []byte
	for uint64(len(attesterBitfield))*8 < params.GetBeaconConfig().BootstrappedValidatorsCount {
		attesterBitfield = append(attesterBitfield, byte(0))
	}

	aState := NewGenesisActiveState()
	block := NewBlock(&pb.BeaconBlock{
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

	// Set validator index 9's RANDAO reveal to be A
	validator9Index := b.Bytes8(9)
	aState.data.PendingSpecials = []*pb.SpecialRecord{{Kind: uint32(params.RandaoChange), Data: [][]byte{validator9Index, {byte('A')}}}}

	newCState, err := cState.NewStateRecalculations(aState, block)
	if err != nil {
		t.Fatalf("failed to derive new crystallized state: %v", err)
	}

	if newCState.LastJustifiedSlot() != 0 {
		t.Fatalf("expected justified slot to equal %d: got %d", 0, newCState.LastJustifiedSlot())
	}

	if newCState.JustifiedStreak() != 0 {
		t.Fatalf("expected justified streak to equal %d: got %d", 0, newCState.JustifiedStreak())
	}

	if newCState.LastStateRecalculationSlot() != params.GetBeaconConfig().CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", params.GetBeaconConfig().CycleLength, newCState.LastStateRecalculationSlot())
	}

	if newCState.LastFinalizedSlot() != 0 {
		t.Fatalf("xpected finalized slot to equal %d, got %d", 0, newCState.LastFinalizedSlot())
	}

	if !(bytes.Equal(newCState.Validators()[9].RandaoCommitment, []byte{'A'})) {
		t.Fatal("failed to set validator 9's randao reveal")
	}
}

func TestNextDeriveCrystallizedSlot(t *testing.T) {
	cState, err := NewGenesisCrystallizedState(nil)
	if err != nil {
		t.Fatalf("Failed to initialized crystallized state: %v", err)
	}

	aState := NewGenesisActiveState()
	block := NewBlock(nil)

	cState, err = cState.NewStateRecalculations(aState, block)
	if err != nil {
		t.Fatalf("failed to derive next crystallized state: %v", err)
	}

	cState.data.Validators = []*pb.ValidatorRecord{
		{Balance: uint64(params.GetBeaconConfig().DepositSize * params.GetBeaconConfig().Gwei),
			Status: uint64(params.Active)},
	}

	totalDeposits := cState.TotalDeposits()
	recentShardBlockHashes := make([][]byte, 3*params.GetBeaconConfig().CycleLength)
	voteCache := make(map[[32]byte]*utils.VoteCache)
	for i := 0; i < 3*int(params.GetBeaconConfig().CycleLength); i++ {
		shardBlockHash := [32]byte{}
		counter := []byte(strconv.Itoa(i))
		copy(shardBlockHash[:], counter)
		recentShardBlockHashes[i] = shardBlockHash[:]
		voteCache[shardBlockHash] = &utils.VoteCache{
			VoteTotalDeposit: totalDeposits * 3 / 4,
		}
	}

	aState = NewActiveState(&pb.ActiveState{
		RecentBlockHashes: recentShardBlockHashes,
	}, voteCache)

	cState, err = cState.NewStateRecalculations(aState, block)
	if err != nil {
		t.Fatalf("failed to derive crystallized state: %v", err)
	}
	if cState.LastStateRecalculationSlot() != 2*params.GetBeaconConfig().CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", 2*params.GetBeaconConfig().CycleLength, cState.LastStateRecalculationSlot())
	}
	if cState.LastJustifiedSlot() != params.GetBeaconConfig().CycleLength-1 {
		t.Fatalf("expected justified slot to equal %d: got %d", params.GetBeaconConfig().CycleLength-1, cState.LastJustifiedSlot())
	}
	if cState.JustifiedStreak() != params.GetBeaconConfig().CycleLength {
		t.Fatalf("expected justified streak to equal %d: got %d", params.GetBeaconConfig().CycleLength, cState.JustifiedStreak())
	}
	if cState.LastFinalizedSlot() != 0 {
		t.Fatalf("expected finalized slot to equal %d: got %d", 0, cState.LastFinalizedSlot())
	}

	cState, err = cState.NewStateRecalculations(aState, block)
	if err != nil {
		t.Fatalf("failed to derive crystallized state: %v", err)
	}
	if cState.LastStateRecalculationSlot() != 3*params.GetBeaconConfig().CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", 3*params.GetBeaconConfig().CycleLength, cState.LastStateRecalculationSlot())
	}
	if cState.LastJustifiedSlot() != 2*params.GetBeaconConfig().CycleLength-1 {
		t.Fatalf("expected justified slot to equal %d: got %d", 2*params.GetBeaconConfig().CycleLength-1, cState.LastJustifiedSlot())
	}
	if cState.JustifiedStreak() != 2*params.GetBeaconConfig().CycleLength {
		t.Fatalf("expected justified streak to equal %d: got %d", 2*params.GetBeaconConfig().CycleLength, cState.JustifiedStreak())
	}
	if cState.LastFinalizedSlot() != params.GetBeaconConfig().CycleLength-2 {
		t.Fatalf("expected finalized slot to equal %d: got %d", params.GetBeaconConfig().CycleLength-2, cState.LastFinalizedSlot())
	}

	cState, err = cState.NewStateRecalculations(aState, block)
	if err != nil {
		t.Fatalf("failed to derive crystallized state: %v", err)
	}
	if cState.LastStateRecalculationSlot() != 4*params.GetBeaconConfig().CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", 3*params.GetBeaconConfig().CycleLength, cState.LastStateRecalculationSlot())
	}
	if cState.LastJustifiedSlot() != 3*params.GetBeaconConfig().CycleLength-1 {
		t.Fatalf("expected justified slot to equal %d: got %d", 3*params.GetBeaconConfig().CycleLength-1, cState.LastJustifiedSlot())
	}
	if cState.JustifiedStreak() != 3*params.GetBeaconConfig().CycleLength {
		t.Fatalf("expected justified streak to equal %d: got %d", 3*params.GetBeaconConfig().CycleLength, cState.JustifiedStreak())
	}
	if cState.LastFinalizedSlot() != 2*params.GetBeaconConfig().CycleLength-2 {
		t.Fatalf("expected finalized slot to equal %d: got %d", 2*params.GetBeaconConfig().CycleLength-2, cState.LastFinalizedSlot())
	}
}

func TestProcessCrosslinks(t *testing.T) {
	// Set up crosslink record for every shard.
	var clRecords []*pb.CrosslinkRecord
	for i := uint64(0); i < params.GetBeaconConfig().ShardCount; i++ {
		clRecord := &pb.CrosslinkRecord{RecentlyChanged: false, ShardBlockHash: []byte{'A'}, Slot: 1}
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

	// Set up pending attestations.
	pAttestations := []*pb.AggregatedAttestation{
		{
			Slot:             0,
			Shard:            1,
			ShardBlockHash:   []byte{'a'},
			AttesterBitfield: []byte{224},
		},
	}

	// Process crosslinks happened at slot 50.
	shardAndCommitteesForSlots, err := casper.InitialShardAndCommitteesForSlots(validators)
	if err != nil {
		t.Fatalf("failed to initialize indices for slots: %v", err)
	}

	committee := []uint32{0, 4, 6}

	shardAndCommitteesForSlots[0].ArrayShardAndCommittee[0].Committee = committee

	cState := NewCrystallizedState(&pb.CrystallizedState{
		Crosslinks:                 clRecords,
		Validators:                 validators,
		ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
	})
	newCrosslinks, err := cState.processCrosslinks(pAttestations, cState.Validators(), 100)
	if err != nil {
		t.Fatalf("process crosslink failed %v", err)
	}

	if newCrosslinks[1].Slot != params.GetBeaconConfig().CycleLength {
		t.Errorf("Slot did not change for new cross link. Wanted: %d. Got: %d", params.GetBeaconConfig().CycleLength, newCrosslinks[0].Slot)
	}
	if !bytes.Equal(newCrosslinks[1].ShardBlockHash, []byte{'a'}) {
		t.Errorf("ShardBlockHash did not change for new cross link. Wanted a. Got: %s", newCrosslinks[0].ShardBlockHash)
	}
	//TODO(#538) Implement tests on balances of the validators in committee once big.Int is introduced.
}

func TestIsNewValidatorSetTransition(t *testing.T) {
	cState, err := NewGenesisCrystallizedState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize crystallized state: %v", err)
	}
	cState.data.ValidatorSetChangeSlot = 1
	if cState.isValidatorSetChange(0) {
		t.Errorf("Is new validator set change should be false, last changed slot greater than finalized slot")
	}
	cState.data.LastFinalizedSlot = 2
	if cState.isValidatorSetChange(1) {
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
	cState.data.ShardAndCommitteesForSlots = shardCommitteeForSlots

	crosslinks := []*pb.CrosslinkRecord{
		{Slot: 1},
		{Slot: 1},
		{Slot: 1},
	}
	cState.data.Crosslinks = crosslinks

	if cState.isValidatorSetChange(params.GetBeaconConfig().MinValidatorSetChangeInterval + 1) {
		t.Errorf("Is new validator set change should be false, crosslink slot record is higher than current slot")
	}

	crosslinks = []*pb.CrosslinkRecord{
		{Slot: 2},
		{Slot: 2},
		{Slot: 2},
	}
	cState.data.Crosslinks = crosslinks

	if !cState.isValidatorSetChange(params.GetBeaconConfig().MinValidatorSetChangeInterval + 1) {
		t.Errorf("New validator set changen failed should have been true")
	}
}

func TestNewValidatorSetRecalculationsInvalid(t *testing.T) {
	cState, err := NewGenesisCrystallizedState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize crystallized state: %v", err)
	}

	// Negative test case, shuffle validators with more than MaxValidators.
	size := params.GetBeaconConfig().ModuloBias + 1
	validators := make([]*pb.ValidatorRecord, size)
	validator := &pb.ValidatorRecord{Status: uint64(params.Active)}
	for i := uint64(0); i < size; i++ {
		validators[i] = validator
	}
	cState.data.Validators = validators
	if _, err := cState.newValidatorSetRecalculations([32]byte{'A'}); err == nil {
		t.Errorf("new validator set change calculation should have failed with invalid validator count")
	}
}

func TestNewValidatorSetRecalculations(t *testing.T) {
	cState, err := NewGenesisCrystallizedState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize crystallized state: %v", err)
	}

	// Create shard committee for every slot.
	var shardCommitteesForSlot []*pb.ShardAndCommitteeArray
	for i := 0; i < int(params.GetBeaconConfig().CycleLength); i++ {
		// Only 10 shards gets crosslinked by validators this period.
		var shardCommittees []*pb.ShardAndCommittee
		for i := 0; i < 10; i++ {
			shardCommittees = append(shardCommittees, &pb.ShardAndCommittee{Shard: uint64(i)})
		}
		shardCommitteesForSlot = append(shardCommitteesForSlot, &pb.ShardAndCommitteeArray{ArrayShardAndCommittee: shardCommittees})
	}

	cState.data.ShardAndCommitteesForSlots = shardCommitteesForSlot
	cState.data.LastStateRecalculationSlot = 65

	shardCount = 10
	_, err = cState.newValidatorSetRecalculations([32]byte{'A'})
	if err != nil {
		t.Fatalf("New validator set change failed %v", err)
	}
}

func TestPenalizedETH(t *testing.T) {
	cState, err := NewGenesisCrystallizedState(nil)
	if err != nil {
		t.Fatalf("Failed to initialize crystallized state: %v", err)
	}
	cState.data.DepositsPenalizedInPeriod = []uint32{100, 200, 300, 400, 500}
	cState.penalizedETH(2)

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
		if cState.penalizedETH(uint32(tt.a)) != tt.b {
			t.Errorf("PenalizedETH(%d) = %v, want = %d", tt.a, cState.penalizedETH(uint32(tt.a)), tt.b)
		}
	}
}
