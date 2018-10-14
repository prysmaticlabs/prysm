package types

import (
	"bytes"
	"os"
	"strconv"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestGenesisCrystallizedState(t *testing.T) {
	cState1, err1 := NewGenesisCrystallizedState("")
	cState2, err2 := NewGenesisCrystallizedState("")

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

func TestInitialDeriveCrystallizedState(t *testing.T) {
	cState, err := NewGenesisCrystallizedState("")
	if err != nil {
		t.Fatalf("Failed to initialize crystallized state: %v", err)
	}

	var attesterBitfield []byte
	for len(attesterBitfield)*8 < params.GetConfig().BootstrappedValidatorsCount {
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

	newCState, _, err := cState.NewStateRecalculations(aState, block, false, false)
	if err != nil {
		t.Fatalf("failed to derive new crystallized state: %v", err)
	}

	if newCState.LastJustifiedSlot() != 0 {
		t.Fatalf("expected justified slot to equal %d: got %d", 0, newCState.LastJustifiedSlot())
	}

	if newCState.JustifiedStreak() != 0 {
		t.Fatalf("expected justified streak to equal %d: got %d", 0, newCState.JustifiedStreak())
	}

	if newCState.LastStateRecalculationSlot() != params.GetConfig().CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", params.GetConfig().CycleLength, newCState.LastStateRecalculationSlot())
	}

	if newCState.LastFinalizedSlot() != 0 {
		t.Fatalf("xpected finalized slot to equal %d, got %d", 0, newCState.LastFinalizedSlot())
	}
}

func TestNextDeriveCrystallizedSlot(t *testing.T) {
	cState, err := NewGenesisCrystallizedState("")
	if err != nil {
		t.Fatalf("Failed to initialized crystallized state: %v", err)
	}

	aState := NewGenesisActiveState()
	block := NewBlock(nil)

	cState, _, err = cState.NewStateRecalculations(aState, block, false, false)
	if err != nil {
		t.Fatalf("failed to derive next crystallized state: %v", err)
	}

	cState.data.Validators = []*pb.ValidatorRecord{
		{Balance: uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei),
			Status: uint64(params.Active)},
	}

	totalDeposits := cState.TotalDeposits()
	recentShardBlockHashes := make([][]byte, 3*params.GetConfig().CycleLength)
	voteCache := make(map[[32]byte]*VoteCache)
	for i := 0; i < 3*int(params.GetConfig().CycleLength); i++ {
		shardBlockHash := [32]byte{}
		counter := []byte(strconv.Itoa(i))
		copy(shardBlockHash[:], counter)
		recentShardBlockHashes[i] = shardBlockHash[:]
		voteCache[shardBlockHash] = &VoteCache{
			VoteTotalDeposit: totalDeposits * 3 / 4,
		}
	}

	aState = NewActiveState(&pb.ActiveState{
		RecentBlockHashes: recentShardBlockHashes,
	}, voteCache)

	cState, _, err = cState.NewStateRecalculations(aState, block, false, false)
	if err != nil {
		t.Fatalf("failed to derive crystallized state: %v", err)
	}
	if cState.LastStateRecalculationSlot() != 2*params.GetConfig().CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", 2*params.GetConfig().CycleLength, cState.LastStateRecalculationSlot())
	}
	if cState.LastJustifiedSlot() != params.GetConfig().CycleLength-1 {
		t.Fatalf("expected justified slot to equal %d: got %d", params.GetConfig().CycleLength-1, cState.LastJustifiedSlot())
	}
	if cState.JustifiedStreak() != params.GetConfig().CycleLength {
		t.Fatalf("expected justified streak to equal %d: got %d", params.GetConfig().CycleLength, cState.JustifiedStreak())
	}
	if cState.LastFinalizedSlot() != 0 {
		t.Fatalf("expected finalized slot to equal %d: got %d", 0, cState.LastFinalizedSlot())
	}

	cState, _, err = cState.NewStateRecalculations(aState, block, false, false)
	if err != nil {
		t.Fatalf("failed to derive crystallized state: %v", err)
	}
	if cState.LastStateRecalculationSlot() != 3*params.GetConfig().CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", 3*params.GetConfig().CycleLength, cState.LastStateRecalculationSlot())
	}
	if cState.LastJustifiedSlot() != 2*params.GetConfig().CycleLength-1 {
		t.Fatalf("expected justified slot to equal %d: got %d", 2*params.GetConfig().CycleLength-1, cState.LastJustifiedSlot())
	}
	if cState.JustifiedStreak() != 2*params.GetConfig().CycleLength {
		t.Fatalf("expected justified streak to equal %d: got %d", 2*params.GetConfig().CycleLength, cState.JustifiedStreak())
	}
	if cState.LastFinalizedSlot() != params.GetConfig().CycleLength-2 {
		t.Fatalf("expected finalized slot to equal %d: got %d", params.GetConfig().CycleLength-2, cState.LastFinalizedSlot())
	}

	cState, _, err = cState.NewStateRecalculations(aState, block, true, true)
	if err != nil {
		t.Fatalf("failed to derive crystallized state: %v", err)
	}
	if cState.LastStateRecalculationSlot() != 4*params.GetConfig().CycleLength {
		t.Fatalf("expected last state recalc to equal %d: got %d", 3*params.GetConfig().CycleLength, cState.LastStateRecalculationSlot())
	}
	if cState.LastJustifiedSlot() != 2*params.GetConfig().CycleLength-1 {
		t.Fatalf("expected justified slot to equal %d: got %d", 2*params.GetConfig().CycleLength-1, cState.LastJustifiedSlot())
	}
	if cState.JustifiedStreak() != 0 {
		t.Fatalf("expected justified streak to equal %d: got %d", 0, cState.JustifiedStreak())
	}
	if cState.LastFinalizedSlot() != params.GetConfig().CycleLength-2 {
		t.Fatalf("expected finalized slot to equal %d: got %d", params.GetConfig().CycleLength-2, cState.LastFinalizedSlot())
	}
}

func TestProcessCrosslinks(t *testing.T) {
	// Set up crosslink record for every shard.
	var clRecords []*pb.CrosslinkRecord
	for i := 0; i < params.GetConfig().ShardCount; i++ {
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
			AttesterBitfield: []byte{10},
		},
	}

	// Process crosslinks happened at slot 50.
	shardAndCommitteesForSlots, err := initialShardAndCommitteesForSlots(validators)
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
	newCrosslinks, err := cState.processCrosslinks(pAttestations, 50, 100)
	if err != nil {
		t.Fatalf("process crosslink failed %v", err)
	}

	if newCrosslinks[1].Slot != 50 {
		t.Errorf("Slot did not change for new cross link. Wanted: 50. Got: %d", newCrosslinks[0].Slot)
	}
	if !bytes.Equal(newCrosslinks[1].ShardBlockHash, []byte{'a'}) {
		t.Errorf("ShardBlockHash did not change for new cross link. Wanted a. Got: %s", newCrosslinks[0].ShardBlockHash)
	}
	//TODO(#538) Implement tests on balances of the validators in committee once big.Int is introduced.
}

func TestIsNewValidatorSetTransition(t *testing.T) {
	cState, err := NewGenesisCrystallizedState("")
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

	if cState.isValidatorSetChange(params.GetConfig().MinValidatorSetChangeInterval + 1) {
		t.Errorf("Is new validator set change should be false, crosslink slot record is higher than current slot")
	}

	crosslinks = []*pb.CrosslinkRecord{
		{Slot: 2},
		{Slot: 2},
		{Slot: 2},
	}
	cState.data.Crosslinks = crosslinks

	if !cState.isValidatorSetChange(params.GetConfig().MinValidatorSetChangeInterval + 1) {
		t.Errorf("New validator set changen failed should have been true")
	}
}

func TestNewValidatorSetRecalculationsInvalid(t *testing.T) {
	cState, err := NewGenesisCrystallizedState("")
	if err != nil {
		t.Fatalf("Failed to initialize crystallized state: %v", err)
	}

	// Negative test case, shuffle validators with more than MaxValidators.
	size := params.GetConfig().ModuloBias + 1
	validators := make([]*pb.ValidatorRecord, size)
	validator := &pb.ValidatorRecord{Status: uint64(params.Active)}
	for i := 0; i < size; i++ {
		validators[i] = validator
	}
	cState.data.Validators = validators
	if _, err := cState.newValidatorSetRecalculations([32]byte{'A'}); err == nil {
		t.Errorf("new validator set change calculation should have failed with invalid validator count")
	}
}

func TestNewValidatorSetRecalculations(t *testing.T) {
	cState, err := NewGenesisCrystallizedState("")
	if err != nil {
		t.Fatalf("Failed to initialize crystallized state: %v", err)
	}

	// Create shard committee for every slot.
	var shardCommitteesForSlot []*pb.ShardAndCommitteeArray
	for i := 0; i < int(params.GetConfig().CycleLength); i++ {
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

func TestInitGenesisJsonFailure(t *testing.T) {
	fname := "/genesis.json"
	pwd, _ := os.Getwd()
	fnamePath := pwd + fname

	_, err := NewGenesisCrystallizedState(fnamePath)
	if err == nil {
		t.Fatalf("genesis.json should have failed %v", err)
	}
}

func TestInitGenesisJson(t *testing.T) {
	fname := "/genesis.json"
	pwd, _ := os.Getwd()
	fnamePath := pwd + fname
	os.Remove(fnamePath)

	params.SetEnv("demo")
	cStateJSON := &pb.CrystallizedState{
		LastStateRecalculationSlot: 0,
		JustifiedStreak:            1,
		LastFinalizedSlot:          99,
		Validators: []*pb.ValidatorRecord{
			{Pubkey: []byte{}, Balance: 32, Status: uint64(params.Active)},
		},
	}
	os.Create(fnamePath)
	f, err := os.OpenFile(fnamePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		t.Fatalf("can't open file %v", err)
	}

	ma := jsonpb.Marshaler{}
	err = ma.Marshal(f, cStateJSON)
	if err != nil {
		t.Fatalf("can't marshal file %v", err)
	}

	cState, err := NewGenesisCrystallizedState(fnamePath)
	if err != nil {
		t.Fatalf("genesis.json failed %v", err)
	}

	if cState.Validators()[0].Status != 1 {
		t.Errorf("Failed to load of genesis json")
	}
	os.Remove(fnamePath)
}

func TestPenalizedETH(t *testing.T) {
	cState, err := NewGenesisCrystallizedState("")
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
		if cState.penalizedETH(tt.a) != tt.b {
			t.Errorf("PenalizedETH(%d) = %v, want = %d", tt.a, cState.penalizedETH(tt.a), tt.b)
		}
	}
}
