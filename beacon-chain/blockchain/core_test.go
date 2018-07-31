package blockchain

import (
	"bytes"
	"context"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"github.com/prysmaticlabs/prysm/shared/database"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type faultyFetcher struct{}

func (f *faultyFetcher) BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error) {
	return nil, errors.New("cannot fetch block")
}

type mockFetcher struct{}

func (m *mockFetcher) BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error) {
	block := gethTypes.NewBlock(&gethTypes.Header{}, nil, nil, nil)
	return block, nil
}

func startInMemoryBeaconChain(t *testing.T) (*BeaconChain, *database.DB) {
	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("unable to setup db: %v", err)
	}
	beaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("unable to setup beacon chain: %v", err)
	}

	return beaconChain, db
}

func TestNewBeaconChain(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	msg := hook.LastEntry().Message
	want := "No chainstate found on disk, initializing beacon from genesis"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	hook.Reset()
	active, crystallized := types.NewGenesisStates()
	if !reflect.DeepEqual(beaconChain.ActiveState(), active) {
		t.Errorf("active states not equal. received: %v, wanted: %v", beaconChain.ActiveState(), active)
	}
	if !reflect.DeepEqual(beaconChain.CrystallizedState(), crystallized) {
		t.Errorf("crystallized states not equal. received: %v, wanted: %v", beaconChain.CrystallizedState(), crystallized)
	}
}

func TestMutateActiveState(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	data := &pb.ActiveStateResponse{
		TotalAttesterDeposits: 4096,
		AttesterBitfield:      []byte{'A', 'B', 'C'},
	}
	active := types.NewActiveState(data)

	if err := beaconChain.MutateActiveState(active); err != nil {
		t.Fatalf("unable to mutate active state: %v", err)
	}
	if !reflect.DeepEqual(beaconChain.state.ActiveState, active) {
		t.Errorf("active state was not updated. wanted %v, got %v", active, beaconChain.state.ActiveState)
	}

	// Initializing a new beacon chain should deserialize persisted state from disk.
	newBeaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("unable to setup second beacon chain: %v", err)
	}

	// The active state should still be the one we mutated and persited earlier.
	if active.TotalAttesterDeposits() != newBeaconChain.state.ActiveState.TotalAttesterDeposits() {
		t.Errorf("active state height incorrect. wanted %v, got %v", active.TotalAttesterDeposits(), newBeaconChain.state.ActiveState.TotalAttesterDeposits())
	}
	if !bytes.Equal(active.AttesterBitfield(), newBeaconChain.state.ActiveState.AttesterBitfield()) {
		t.Errorf("active state randao incorrect. wanted %v, got %v", active.AttesterBitfield(), newBeaconChain.state.ActiveState.AttesterBitfield())
	}
}

func TestMutateCrystallizedState(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	data := &pb.CrystallizedStateResponse{
		CurrentDynasty:    3,
		CurrentCheckPoint: []byte("checkpoint"),
	}
	crystallized := types.NewCrystallizedState(data)

	if err := beaconChain.MutateCrystallizedState(crystallized); err != nil {
		t.Fatalf("unable to mutate crystallized state: %v", err)
	}
	if !reflect.DeepEqual(beaconChain.state.CrystallizedState, crystallized) {
		t.Errorf("crystallized state was not updated. wanted %v, got %v", crystallized, beaconChain.state.CrystallizedState)
	}

	// Initializing a new beacon chain should deserialize persisted state from disk.
	newBeaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("unable to setup second beacon chain: %v", err)
	}

	// The crystallized state should still be the one we mutated and persited earlier.
	if crystallized.CurrentDynasty() != newBeaconChain.state.CrystallizedState.CurrentDynasty() {
		t.Errorf("crystallized state dynasty incorrect. wanted %v, got %v", crystallized.CurrentDynasty(), newBeaconChain.state.CrystallizedState.CurrentDynasty())
	}
	if crystallized.CurrentCheckPoint().Hex() != newBeaconChain.state.CrystallizedState.CurrentCheckPoint().Hex() {
		t.Errorf("crystallized state current checkpoint incorrect. wanted %v, got %v", crystallized.CurrentCheckPoint().Hex(), newBeaconChain.state.CrystallizedState.CurrentCheckPoint().Hex())
	}
}

func TestGetAttestersProposer(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	var validators []*pb.ValidatorRecord
	// Create 1000 validators in ActiveValidators.
	for i := 0; i < 1000; i++ {
		validator := &pb.ValidatorRecord{WithdrawalAddress: []byte{'A'}, PublicKey: 0}
		validators = append(validators, validator)
	}

	_, crystallized := types.NewGenesisStates()
	crystallized.UpdateActiveValidators(validators)
	beaconChain.MutateCrystallizedState(crystallized)

	attesters, proposer, err := beaconChain.getAttestersProposer(common.Hash{'A'})
	if err != nil {
		t.Errorf("GetAttestersProposer function failed: %v", err)
	}

	validatorList, err := utils.ShuffleIndices(common.Hash{'A'}, len(validators))
	if err != nil {
		t.Errorf("Shuffle function function failed: %v", err)
	}

	if !reflect.DeepEqual(proposer, validatorList[len(validatorList)-1]) {
		t.Errorf("Get proposer failed, expected: %v got: %v", validatorList[len(validatorList)-1], proposer)
	}
	if !reflect.DeepEqual(attesters, validatorList[:len(attesters)]) {
		t.Errorf("Get attesters failed, expected: %v got: %v", validatorList[:len(attesters)], attesters)
	}
}

func TestCanProcessBlock(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	// Initialize a parent block
	parentBlock := NewBlock(t, &pb.BeaconBlockResponse{
		SlotNumber: 1,
	})
	parentHash, err := parentBlock.Hash()
	if err != nil {
		t.Fatalf("Failed to compute parent block's hash: %v", err)
	}
	if err = db.DB().Put(parentHash[:], []byte{}); err != nil {
		t.Fatalf("Failed to put parent block on db: %v", err)
	}

	// Using a faulty fetcher should throw an error.
	block := NewBlock(t, &pb.BeaconBlockResponse{
		SlotNumber: 2,
	})
	if _, err := beaconChain.CanProcessBlock(&faultyFetcher{}, block); err == nil {
		t.Error("Using a faulty fetcher should throw an error, received nil")
	}

	// Initialize initial state
	activeState := types.NewActiveState(&pb.ActiveStateResponse{TotalAttesterDeposits: 10000})
	beaconChain.state.ActiveState = activeState
	activeHash, err := activeState.Hash()
	if err != nil {
		t.Fatalf("Cannot hash active state: %v", err)
	}

	crystallized := types.NewCrystallizedState(&pb.CrystallizedStateResponse{CurrentEpoch: 5})
	beaconChain.state.CrystallizedState = crystallized
	crystallizedHash, err := crystallized.Hash()
	if err != nil {
		t.Fatalf("Compute crystallized state hash failed: %v", err)
	}

	block = NewBlock(t, &pb.BeaconBlockResponse{
		SlotNumber:            2,
		ActiveStateHash:       activeHash[:],
		CrystallizedStateHash: crystallizedHash[:],
		ParentHash:            parentHash[:],
	})

	// A properly initialize block should not fail
	canProcess, err := beaconChain.CanProcessBlock(&mockFetcher{}, block)
	if err != nil {
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if !canProcess {
		t.Error("Should be able to process block, could not")
	}

	// Test timestamp validity condition
	block = NewBlock(t, &pb.BeaconBlockResponse{
		SlotNumber:            1000000,
		ActiveStateHash:       activeHash[:],
		CrystallizedStateHash: crystallizedHash[:],
		ParentHash:            parentHash[:],
	})
	canProcess, err = beaconChain.CanProcessBlock(&mockFetcher{}, block)
	if err != nil {
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if canProcess {
		t.Error("Should not be able to process block with invalid timestamp condition")
	}
}

func TestProcessBlockWithBadHashes(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	// Test negative scenario where active state hash is different than node's compute
	parentBlock := NewBlock(t, nil)
	parentHash, err := parentBlock.Hash()
	if err != nil {
		t.Fatalf("Failed to compute parent block's hash: %v", err)
	}
	if err = db.DB().Put(parentHash[:], []byte{}); err != nil {
		t.Fatalf("Failed to put parent block on db: %v", err)
	}

	// Initialize state
	active := types.NewActiveState(&pb.ActiveStateResponse{TotalAttesterDeposits: 10000})
	activeStateHash, err := active.Hash()
	if err != nil {
		t.Fatalf("Cannot hash active state: %v", err)
	}
	crystallized := types.NewCrystallizedState(&pb.CrystallizedStateResponse{CurrentEpoch: 10000})
	crystallizedStateHash, err := crystallized.Hash()
	if err != nil {
		t.Fatalf("Cannot hash crystallized state: %v", err)
	}

	block := NewBlock(t, &pb.BeaconBlockResponse{
		SlotNumber:            1,
		ActiveStateHash:       activeStateHash[:],
		CrystallizedStateHash: crystallizedStateHash[:],
		ParentHash:            parentHash[:],
	})

	// Test negative scenario where active state hash is different than node's compute
	beaconChain.state.ActiveState = types.NewActiveState(&pb.ActiveStateResponse{TotalAttesterDeposits: 9999})

	canProcess, err := beaconChain.CanProcessBlock(&mockFetcher{}, block)
	if err == nil {
		t.Fatal("CanProcessBlocks should have failed with diff state hashes")
	}
	if canProcess {
		t.Error("CanProcessBlocks should have returned false")
	}

	// Test negative scenario where crystallized state hash is different than node's compute
	beaconChain.state.CrystallizedState = types.NewCrystallizedState(&pb.CrystallizedStateResponse{CurrentEpoch: 9999})

	canProcess, err = beaconChain.CanProcessBlock(&mockFetcher{}, block)
	if err == nil {
		t.Fatal("CanProcessBlocks should have failed with diff state hashes")
	}
	if canProcess {
		t.Error("CanProcessBlocks should have returned false")
	}
}

func TestProcessBlockWithInvalidParent(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	// If parent hash is non-existent, processing block should fail.
	active := types.NewActiveState(&pb.ActiveStateResponse{TotalAttesterDeposits: 10000})
	activeStateHash, err := active.Hash()
	if err != nil {
		t.Fatalf("Cannot hash active state: %v", err)
	}
	beaconChain.state.ActiveState = active

	crystallized := types.NewCrystallizedState(&pb.CrystallizedStateResponse{CurrentEpoch: 10000})
	crystallizedStateHash, err := crystallized.Hash()
	if err != nil {
		t.Fatalf("Cannot hash crystallized state: %v", err)
	}
	beaconChain.state.CrystallizedState = crystallized

	// Test that block processing is invalid without a parent hash
	block := NewBlock(t, &pb.BeaconBlockResponse{
		SlotNumber:            2,
		ActiveStateHash:       activeStateHash[:],
		CrystallizedStateHash: crystallizedStateHash[:],
	})
	if _, err = beaconChain.CanProcessBlock(&mockFetcher{}, block); err == nil {
		t.Error("Processing without a valid parent hash should fail")
	}

	// If parent hash is not stored in db, processing block should fail.
	parentBlock := NewBlock(t, &pb.BeaconBlockResponse{
		SlotNumber: 1,
	})
	parentHash, err := parentBlock.Hash()
	if err != nil {
		t.Fatalf("Failed to compute parent block's hash: %v", err)
	}
	block = NewBlock(t, &pb.BeaconBlockResponse{
		SlotNumber:            2,
		ActiveStateHash:       activeStateHash[:],
		CrystallizedStateHash: crystallizedStateHash[:],
		ParentHash:            parentHash[:],
	})

	if _, err := beaconChain.CanProcessBlock(&mockFetcher{}, block); err == nil {
		t.Error("Processing block should fail when parent hash is not in db")
	}

	if err = db.DB().Put(parentHash[:], nil); err != nil {
		t.Fatalf("Failed to put parent block on db: %v", err)
	}
	_, err = beaconChain.CanProcessBlock(&mockFetcher{}, block)
	if err == nil {
		t.Error("Processing block should fail when parent hash points to nil in db")
	}
	want := "parent hash points to nil in beaconDB"
	if err.Error() != want {
		t.Errorf("invalid log, expected \"%s\", got \"%s\"", want, err.Error())
	}

	if err = db.DB().Put(parentHash[:], []byte{}); err != nil {
		t.Fatalf("Failed to put parent block on db: %v", err)
	}

	canProcess, err := beaconChain.CanProcessBlock(&mockFetcher{}, block)
	if err != nil {
		t.Errorf("Should have been able to process block: %v", err)
	}
	if !canProcess {
		t.Error("Should have been able to process block")
	}
}

func TestRotateValidatorSet(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	activeValidators := []*pb.ValidatorRecord{
		{Balance: 10000, WithdrawalAddress: []byte{'A'}},
		{Balance: 15000, WithdrawalAddress: []byte{'B'}},
		{Balance: 20000, WithdrawalAddress: []byte{'C'}},
		{Balance: 25000, WithdrawalAddress: []byte{'D'}},
		{Balance: 30000, WithdrawalAddress: []byte{'E'}},
	}

	queuedValidators := []*pb.ValidatorRecord{
		{Balance: params.DefaultBalance, WithdrawalAddress: []byte{'F'}},
		{Balance: params.DefaultBalance, WithdrawalAddress: []byte{'G'}},
		{Balance: params.DefaultBalance, WithdrawalAddress: []byte{'H'}},
		{Balance: params.DefaultBalance, WithdrawalAddress: []byte{'I'}},
		{Balance: params.DefaultBalance, WithdrawalAddress: []byte{'J'}},
	}

	exitedValidators := []*pb.ValidatorRecord{
		{Balance: 99999, WithdrawalAddress: []byte{'K'}},
		{Balance: 99999, WithdrawalAddress: []byte{'L'}},
		{Balance: 99999, WithdrawalAddress: []byte{'M'}},
		{Balance: 99999, WithdrawalAddress: []byte{'N'}},
		{Balance: 99999, WithdrawalAddress: []byte{'O'}},
	}

	beaconChain.CrystallizedState().UpdateActiveValidators(activeValidators)
	beaconChain.CrystallizedState().UpdateQueuedValidators(queuedValidators)
	beaconChain.CrystallizedState().UpdateExitedValidators(exitedValidators)

	if beaconChain.CrystallizedState().ActiveValidatorsLength() != 5 {
		t.Errorf("Get active validator count failed, wanted 5, got %v", beaconChain.CrystallizedState().ActiveValidatorsLength())
	}
	if beaconChain.CrystallizedState().QueuedValidatorsLength() != 5 {
		t.Errorf("Get queued validator count failed, wanted 5, got %v", beaconChain.CrystallizedState().QueuedValidatorsLength())
	}
	if beaconChain.CrystallizedState().ExitedValidatorsLength() != 5 {
		t.Errorf("Get exited validator count failed, wanted 5, got %v", beaconChain.CrystallizedState().ExitedValidatorsLength())
	}

	newQueuedValidators, newActiveValidators, newExitedValidators := beaconChain.RotateValidatorSet()

	if len(newActiveValidators) != 4 {
		t.Errorf("Get active validator count failed, wanted 5, got %v", len(newActiveValidators))
	}
	if len(newQueuedValidators) != 4 {
		t.Errorf("Get queued validator count failed, wanted 4, got %v", len(newQueuedValidators))
	}
	if len(newExitedValidators) != 7 {
		t.Errorf("Get exited validator count failed, wanted 6, got %v", len(newExitedValidators))
	}
}

func TestCutOffValidatorSet(t *testing.T) {
	// Test scenario #1: Assume there's enough validators to fill in all the heights.
	validatorCount := params.EpochLength * params.MinCommiteeSize
	cutoffsValidators := GetCutoffs(validatorCount)

	// The length of cutoff list should be 65. Since there is 64 heights per epoch,
	// it means during every height, a new set of 128 validators will form a committee.
	expectedCount := int(math.Ceil(float64(validatorCount)/params.MinCommiteeSize)) + 1
	if len(cutoffsValidators) != expectedCount {
		t.Errorf("Incorrect count for cutoffs validator. Wanted: %v, Got: %v", expectedCount, len(cutoffsValidators))
	}

	// Verify each cutoff is an increment of MinCommiteeSize, it means 128 validators forms a
	// a committee and get to attest per height.
	count := 0
	for _, cutoff := range cutoffsValidators {
		if cutoff != count {
			t.Errorf("cutoffsValidators did not get 128 increment. Wanted: count, Got: %v", cutoff)
		}
		count += params.MinCommiteeSize
	}

	// Test scenario #2: Assume there's not enough validators to fill in all the heights.
	validatorCount = 1000
	cutoffsValidators = unique(GetCutoffs(validatorCount))
	// With 1000 validators, we can't attest every height. Given min committee size is 128,
	// we can only attest 7 heights. round down 1000 / 128 equals to 7, means the length is 8.
	expectedCount = int(math.Ceil(float64(validatorCount) / params.MinCommiteeSize))
	if len(unique(cutoffsValidators)) != expectedCount {
		t.Errorf("Incorrect count for cutoffs validator. Wanted: %v, Got: %v", expectedCount, validatorCount/params.MinCommiteeSize)
	}

	// Verify each cutoff is an increment of 142~143 (1000 / 7).
	count = 0
	for _, cutoff := range cutoffsValidators {
		num := count * validatorCount / (len(cutoffsValidators) - 1)
		if cutoff != num {
			t.Errorf("cutoffsValidators did not get correct increment. Wanted: %v, Got: %v", num, cutoff)
		}
		count++
	}
}

func TestIsEpochTransition(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	if err := beaconChain.MutateCrystallizedState(types.NewCrystallizedState(&pb.CrystallizedStateResponse{CurrentEpoch: 1})); err != nil {
		t.Fatalf("unable to mutate crystallizedstate: %v", err)
	}
	if !beaconChain.isEpochTransition(128) {
		t.Errorf("there was supposed to be an epoch transition but there isn't one now")
	}
	if beaconChain.isEpochTransition(80) {
		t.Errorf("there is not supposed to be an epoch transition but there is one now")
	}
}

func TestHasVoted(t *testing.T) {
	for i := 0; i < 8; i++ {
		testfield := int(math.Pow(2, float64(i)))
		bitfields := []byte{byte(testfield), 0, 0}
		attesterBlock := 1
		attesterFieldIndex := (8 - i)

		voted := hasVoted(bitfields, attesterBlock, attesterFieldIndex)
		if !voted {
			t.Fatalf("attester was supposed to have voted but the test shows they have not, this is their bitfield and index: %b :%d", bitfields[0], attesterFieldIndex)
		}
	}
}

func TestApplyRewardAndPenalty(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	balance1 := uint64(10000)
	balance2 := uint64(15000)
	balance3 := uint64(20000)
	balance4 := uint64(25000)
	balance5 := uint64(30000)

	activeValidators := []*pb.ValidatorRecord{
		{Balance: balance1, WithdrawalAddress: []byte{'A'}, PublicKey: 0},
		{Balance: balance2, WithdrawalAddress: []byte{'B'}, PublicKey: 1},
		{Balance: balance3, WithdrawalAddress: []byte{'C'}, PublicKey: 2},
		{Balance: balance4, WithdrawalAddress: []byte{'D'}, PublicKey: 3},
		{Balance: balance5, WithdrawalAddress: []byte{'E'}, PublicKey: 4},
	}

	beaconChain.CrystallizedState().UpdateActiveValidators(activeValidators)

	beaconChain.applyRewardAndPenalty(0, true)
	beaconChain.applyRewardAndPenalty(1, false)
	beaconChain.applyRewardAndPenalty(2, true)
	beaconChain.applyRewardAndPenalty(3, false)
	beaconChain.applyRewardAndPenalty(4, true)

	expectedBalance1 := balance1 + params.AttesterReward
	expectedBalance2 := balance2 - params.AttesterReward
	expectedBalance3 := balance3 + params.AttesterReward
	expectedBalance4 := balance4 - params.AttesterReward
	expectedBalance5 := balance5 + params.AttesterReward

	if expectedBalance1 != beaconChain.CrystallizedState().ActiveValidators()[0].Balance {
		t.Errorf("rewards and penalties were not able to be applied correctly:%d , %d", expectedBalance1, beaconChain.CrystallizedState().ActiveValidators()[0].Balance)
	}
	if expectedBalance2 != beaconChain.CrystallizedState().ActiveValidators()[1].Balance {
		t.Errorf("rewards and penalties were not able to be applied correctly:%d , %d", expectedBalance2, beaconChain.CrystallizedState().ActiveValidators()[1].Balance)
	}
	if expectedBalance3 != beaconChain.CrystallizedState().ActiveValidators()[2].Balance {
		t.Errorf("rewards and penalties were not able to be applied correctly:%d , %d", expectedBalance3, beaconChain.CrystallizedState().ActiveValidators()[2].Balance)
	}
	if expectedBalance4 != beaconChain.CrystallizedState().ActiveValidators()[3].Balance {
		t.Errorf("rewards and penalties were not able to be applied correctly:%d , %d", expectedBalance4, beaconChain.CrystallizedState().ActiveValidators()[3].Balance)
	}
	if expectedBalance5 != beaconChain.CrystallizedState().ActiveValidators()[4].Balance {
		t.Errorf("rewards and penalties were not able to be applied correctly:%d , %d", expectedBalance5, beaconChain.CrystallizedState().ActiveValidators()[4].Balance)
	}
}

func TestResetAttesterBitfields(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	// Testing validator set sizes from 1 to 100.
	for j := 1; j <= 100; j++ {
		var validators []*pb.ValidatorRecord

		for i := 0; i < j; i++ {
			validator := &pb.ValidatorRecord{WithdrawalAddress: []byte{}, PublicKey: 0}
			validators = append(validators, validator)
		}

		beaconChain.CrystallizedState().UpdateActiveValidators(validators)

		testAttesterBitfield := []byte{2, 4, 6, 9}
		if err := beaconChain.MutateActiveState(types.NewActiveState(&pb.ActiveStateResponse{AttesterBitfield: testAttesterBitfield})); err != nil {
			t.Fatal("unable to mutate active state")
		}
		if err := beaconChain.resetAttesterBitfields(); err != nil {
			t.Fatalf("unable to reset Attester Bitfields")
		}
		if bytes.Equal(testAttesterBitfield, beaconChain.state.ActiveState.AttesterBitfield()) {
			t.Fatalf("attester bitfields have not been able to be reset: %v", testAttesterBitfield)
		}

		bitfieldLength := j / 8
		if j%8 != 0 {
			bitfieldLength++
		}
		if !bytes.Equal(beaconChain.state.ActiveState.AttesterBitfield(), make([]byte, bitfieldLength)) {
			t.Fatalf("attester bitfields are not zeroed out: %v", beaconChain.state.ActiveState.AttesterBitfield())
		}
	}
}

func TestResetTotalAttesterDeposit(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	active := types.NewActiveState(&pb.ActiveStateResponse{TotalAttesterDeposits: 10000})
	if err := beaconChain.MutateActiveState(active); err != nil {
		t.Fatalf("unable to Mutate Active state: %v", err)
	}
	if beaconChain.state.ActiveState.TotalAttesterDeposits() != uint64(10000) {
		t.Fatalf("attester deposit was not saved: %d", beaconChain.state.ActiveState.TotalAttesterDeposits())
	}
	if err := beaconChain.resetTotalAttesterDeposit(); err != nil {
		t.Fatalf("unable to reset total attester deposit: %v", err)
	}
	if beaconChain.state.ActiveState.TotalAttesterDeposits() != uint64(0) {
		t.Fatalf("attester deposit was not able to be reset: %d", beaconChain.state.ActiveState.TotalAttesterDeposits())
	}
}

func TestUpdateJustifiedEpoch(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	data := &pb.CrystallizedStateResponse{CurrentEpoch: 5, LastJustifiedEpoch: 4, LastFinalizedEpoch: 3}
	beaconChain.MutateCrystallizedState(types.NewCrystallizedState(data))

	if beaconChain.state.CrystallizedState.LastFinalizedEpoch() != uint64(3) ||
		beaconChain.state.CrystallizedState.LastJustifiedEpoch() != uint64(4) ||
		beaconChain.state.CrystallizedState.CurrentEpoch() != uint64(5) {
		t.Fatal("crystallized state unable to be saved")
	}

	if err := beaconChain.updateJustifiedEpoch(); err != nil {
		t.Fatalf("unable to update justified epoch: %v", err)
	}
	if beaconChain.state.CrystallizedState.LastJustifiedEpoch() != uint64(5) {
		t.Fatalf("unable to update last justified epoch: %d", beaconChain.state.CrystallizedState.LastJustifiedEpoch())
	}
	if beaconChain.state.CrystallizedState.LastFinalizedEpoch() != uint64(4) {
		t.Fatalf("unable to update last finalized epoch: %d", beaconChain.state.CrystallizedState.LastFinalizedEpoch())
	}

	data = &pb.CrystallizedStateResponse{CurrentEpoch: 8, LastJustifiedEpoch: 4, LastFinalizedEpoch: 3}
	beaconChain.MutateCrystallizedState(types.NewCrystallizedState(data))

	if beaconChain.state.CrystallizedState.LastFinalizedEpoch() != uint64(3) ||
		beaconChain.state.CrystallizedState.LastJustifiedEpoch() != uint64(4) ||
		beaconChain.state.CrystallizedState.CurrentEpoch() != uint64(8) {
		t.Fatal("crystallized state unable to be saved")
	}

	if err := beaconChain.updateJustifiedEpoch(); err != nil {
		t.Fatalf("unable to update justified epoch: %v", err)
	}
	if beaconChain.state.CrystallizedState.LastJustifiedEpoch() != uint64(8) {
		t.Fatalf("unable to update last justified epoch: %d", beaconChain.state.CrystallizedState.LastJustifiedEpoch())
	}
	if beaconChain.state.CrystallizedState.LastFinalizedEpoch() != uint64(3) {
		t.Fatalf("unable to update last finalized epoch: %d", beaconChain.state.CrystallizedState.LastFinalizedEpoch())
	}
}

func TestUpdateRewardsAndPenalties(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	var validators []*pb.ValidatorRecord

	for i := 0; i < 40; i++ {
		validator := &pb.ValidatorRecord{Balance: 1000, WithdrawalAddress: []byte{'A'}, PublicKey: 0}
		validators = append(validators, validator)
	}
	beaconChain.state.CrystallizedState.UpdateActiveValidators(validators)

	//Binary Representation of Bitfield: 00010110 00101011 00101110 01001111 01010000
	testAttesterBitfield := []byte{22, 43, 46, 79, 80}
	if err := beaconChain.MutateActiveState(types.NewActiveState(&pb.ActiveStateResponse{AttesterBitfield: testAttesterBitfield})); err != nil {
		t.Fatalf("unable to mutate active state: %v", err)
	}
	// Test Validator with index 10 would refer to the 11th bit in the bitfield

	if err := beaconChain.updateRewardsAndPenalties(10); err != nil {
		t.Fatalf("unable to update rewards and penalties: %v", err)
	}
	if beaconChain.CrystallizedState().ActiveValidators()[10].Balance != uint64(1001) {
		t.Fatal("validator balance not updated")
	}

	// Test Validator with index 15 would refer to the 16th bit in the bitfield
	if err := beaconChain.updateRewardsAndPenalties(15); err != nil {
		t.Fatalf("unable to update rewards and penalties: %v", err)
	}
	if beaconChain.CrystallizedState().ActiveValidators()[15].Balance != uint64(1001) {
		t.Fatal("validator balance not updated")
	}

	// Test Validator with index 23 would refer to the 24th bit in the bitfield
	if err := beaconChain.updateRewardsAndPenalties(23); err != nil {
		t.Fatalf("unable to update rewards and penalties: %v", err)
	}
	if beaconChain.CrystallizedState().ActiveValidators()[23].Balance == uint64(1001) {
		t.Fatal("validator balance not updated")
	}

	err := beaconChain.updateRewardsAndPenalties(230)
	if err == nil {
		t.Fatal("no error displayed when there is supposed to be one")

	}
	if err.Error() != "attester index does not exist" {
		t.Fatalf("incorrect error message: ` %s ` is displayed when it should have been: attester index does not exist", err.Error())
	}
}

func TestComputeValidatorRewardsAndPenalties(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	var validators []*pb.ValidatorRecord
	for i := 0; i < 40; i++ {
		validator := &pb.ValidatorRecord{Balance: 1000, WithdrawalAddress: []byte{'A'}, PublicKey: 0}
		validators = append(validators, validator)
	}

	data := &pb.CrystallizedStateResponse{
		ActiveValidators:   validators,
		CurrentCheckPoint:  []byte("checkpoint"),
		TotalDeposits:      10000,
		CurrentEpoch:       5,
		LastJustifiedEpoch: 4,
		LastFinalizedEpoch: 3,
	}
	if err := beaconChain.MutateCrystallizedState(types.NewCrystallizedState(data)); err != nil {
		t.Fatalf("unable to mutate crystallizedstate: %v", err)
	}

	//Binary representation of bitfield: 11001000 10010100 10010010 10110011 00110001
	testAttesterBitfield := []byte{200, 148, 146, 179, 49}
	types.NewActiveState(&pb.ActiveStateResponse{AttesterBitfield: testAttesterBitfield})
	ActiveState := types.NewActiveState(&pb.ActiveStateResponse{TotalAttesterDeposits: 8000, AttesterBitfield: testAttesterBitfield})
	if err := beaconChain.MutateActiveState(ActiveState); err != nil {
		t.Fatalf("unable to Mutate Active state: %v", err)
	}
	if err := beaconChain.computeValidatorRewardsAndPenalties(); err != nil {
		t.Fatalf("could not compute validator rewards and penalties: %v", err)
	}
	if beaconChain.state.CrystallizedState.LastJustifiedEpoch() != uint64(5) {
		t.Fatalf("unable to update last justified epoch: %d", beaconChain.state.CrystallizedState.LastJustifiedEpoch())
	}
	if beaconChain.state.CrystallizedState.LastFinalizedEpoch() != uint64(4) {
		t.Fatalf("unable to update last finalized epoch: %d", beaconChain.state.CrystallizedState.LastFinalizedEpoch())
	}
	if beaconChain.CrystallizedState().ActiveValidators()[0].Balance != uint64(1001) {
		t.Fatalf("validator balance not updated: %d", beaconChain.CrystallizedState().ActiveValidators()[1].Balance)
	}
	if beaconChain.CrystallizedState().ActiveValidators()[7].Balance != uint64(999) {
		t.Fatalf("validator balance not updated: %d", beaconChain.CrystallizedState().ActiveValidators()[1].Balance)
	}
	if beaconChain.CrystallizedState().ActiveValidators()[29].Balance != uint64(999) {
		t.Fatalf("validator balance not updated: %d", beaconChain.CrystallizedState().ActiveValidators()[1].Balance)
	}
	if beaconChain.state.ActiveState.TotalAttesterDeposits() != uint64(0) {
		t.Fatalf("attester deposit was not able to be reset: %d", beaconChain.state.ActiveState.TotalAttesterDeposits())
	}
	if !bytes.Equal(beaconChain.state.ActiveState.AttesterBitfield(), make([]byte, 5)) {
		t.Fatalf("attester bitfields are not zeroed out: %v", beaconChain.state.ActiveState.AttesterBitfield())
	}
}

// helper function to remove duplicates in a int slice.
func unique(ints []int) []int {
	keys := make(map[int]bool)
	list := []int{}
	for _, int := range ints {
		if _, value := keys[int]; !value {
			keys[int] = true
			list = append(list, int)
		}
	}
	return list

}

// NewBlock is a helper method to create blocks with valid defaults.
// For a generic block, use NewBlock(t, nil)
func NewBlock(t *testing.T, b *pb.BeaconBlockResponse) *types.Block {
	if b == nil {
		b = &pb.BeaconBlockResponse{}
	}
	if b.ActiveStateHash == nil {
		b.ActiveStateHash = make([]byte, 32)
	}
	if b.CrystallizedStateHash == nil {
		b.CrystallizedStateHash = make([]byte, 32)
	}
	if b.ParentHash == nil {
		b.ParentHash = make([]byte, 32)
	}

	blk, err := types.NewBlock(b)
	if err != nil {
		t.Fatalf("failed to instantiate block with slot number: %v", err)
	}

	return blk
}
