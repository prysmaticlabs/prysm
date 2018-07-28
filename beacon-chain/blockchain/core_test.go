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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/prysmaticlabs/prysm/beacon-chain/database"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
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

func startInMemoryBeaconChain(t *testing.T) (*BeaconChain, *database.BeaconDB) {
	config := &database.BeaconDBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewBeaconDB(config)
	if err != nil {
		t.Fatalf("unable to setup db: %v", err)
	}
	db.Start()
	beaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("unable to setup beacon chain: %v", err)
	}

	return beaconChain, db
}

func TestNewBeaconChain(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Stop()

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
	defer db.Stop()

	active := &types.ActiveState{
		TotalAttesterDeposits: 4096,
		AttesterBitfields:     []byte{'A', 'B', 'C'},
	}
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
	if active.TotalAttesterDeposits != newBeaconChain.state.ActiveState.TotalAttesterDeposits {
		t.Errorf("active state height incorrect. wanted %v, got %v", active.TotalAttesterDeposits, newBeaconChain.state.ActiveState.TotalAttesterDeposits)
	}
	if !bytes.Equal(active.AttesterBitfields, newBeaconChain.state.ActiveState.AttesterBitfields) {
		t.Errorf("active state randao incorrect. wanted %v, got %v", active.AttesterBitfields, newBeaconChain.state.ActiveState.AttesterBitfields)
	}
}

func TestMutateCrystallizedState(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Stop()

	currentCheckpoint := common.BytesToHash([]byte("checkpoint"))
	crystallized := &types.CrystallizedState{
		Dynasty:           3,
		CurrentCheckpoint: currentCheckpoint,
	}
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
	if crystallized.Dynasty != newBeaconChain.state.CrystallizedState.Dynasty {
		t.Errorf("crystallized state dynasty incorrect. wanted %v, got %v", crystallized.Dynasty, newBeaconChain.state.CrystallizedState.Dynasty)
	}
	if crystallized.CurrentCheckpoint.Hex() != newBeaconChain.state.CrystallizedState.CurrentCheckpoint.Hex() {
		t.Errorf("crystallized state current checkpoint incorrect. wanted %v, got %v", crystallized.CurrentCheckpoint.Hex(), newBeaconChain.state.CrystallizedState.CurrentCheckpoint.Hex())
	}
}

func TestGetAttestersProposer(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Stop()

	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Could not generate key: %v", err)
	}

	var validators []types.ValidatorRecord
	// Create 1000 validators in ActiveValidators.
	for i := 0; i < 1000; i++ {
		validator := types.ValidatorRecord{WithdrawalAddress: common.Address{'A'}, PubKey: enr.Secp256k1(priv.PublicKey)}
		validators = append(validators, validator)
	}

	beaconChain.MutateCrystallizedState(&types.CrystallizedState{ActiveValidators: validators})

	attesters, propser, err := beaconChain.getAttestersProposer(common.Hash{'A'})
	if err != nil {
		t.Errorf("GetAttestersProposer function failed: %v", err)
	}

	validatorList, err := utils.ShuffleIndices(common.Hash{'A'}, len(validators))
	if err != nil {
		t.Errorf("Shuffle function function failed: %v", err)
	}

	if !reflect.DeepEqual(propser, validatorList[len(validatorList)-1]) {
		t.Errorf("Get proposer failed, expected: %v got: %v", validatorList[len(validatorList)-1], propser)
	}
	if !reflect.DeepEqual(attesters, validatorList[:len(attesters)]) {
		t.Errorf("Get attesters failed, expected: %v got: %v", validatorList[:len(attesters)], attesters)
	}
}

func TestCanProcessBlock(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Stop()

	block := types.NewBlock(1)
	// Using a faulty fetcher should throw an error.
	if _, err := beaconChain.CanProcessBlock(&faultyFetcher{}, block); err == nil {
		t.Errorf("Using a faulty fetcher should throw an error, received nil")
	}
	activeState := &types.ActiveState{TotalAttesterDeposits: 10000}
	beaconChain.state.ActiveState = activeState

	activeHash, err := hashActiveState(*activeState)
	if err != nil {
		t.Fatalf("Cannot hash active state: %v", err)
	}

	block.InsertActiveHash(activeHash)

	crystallizedHash, err := hashCrystallizedState(types.CrystallizedState{})
	if err != nil {
		t.Fatalf("Compute crystallized state hash failed: %v", err)
	}
	block.InsertCrystallizedHash(crystallizedHash)
	canProcess, err := beaconChain.CanProcessBlock(&mockFetcher{}, block)
	if err != nil {
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if !canProcess {
		t.Errorf("Should be able to process block, could not")
	}

	// Attempting to try a block with that fails the timestamp validity
	// condition.
	block = types.NewBlock(1000000)
	block.InsertActiveHash(activeHash)
	block.InsertCrystallizedHash(crystallizedHash)
	canProcess, err = beaconChain.CanProcessBlock(&mockFetcher{}, block)
	if err != nil {
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if canProcess {
		t.Errorf("Should not be able to process block with invalid timestamp condition")
	}
}

func TestProcessBlockWithBadHashes(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Stop()

	// Test negative scenario where active state hash is different than node's compute
	block := types.NewBlock(1)
	activeState := &types.ActiveState{TotalAttesterDeposits: 10000}
	stateHash, err := hashActiveState(*activeState)
	if err != nil {
		t.Fatalf("Cannot hash active state: %v", err)
	}
	block.InsertActiveHash(stateHash)

	beaconChain.state.ActiveState = &types.ActiveState{TotalAttesterDeposits: 9999}

	canProcess, err := beaconChain.CanProcessBlock(&mockFetcher{}, block)
	if err == nil {
		t.Fatalf("CanProcessBlocks should have failed with diff state hashes")
	}
	if canProcess {
		t.Errorf("CanProcessBlocks should have returned false")
	}

	// Test negative scenario where crystallized state hash is different than node's compute
	crystallizedState := &types.CrystallizedState{CurrentEpoch: 10000}
	stateHash, err = hashCrystallizedState(*crystallizedState)
	if err != nil {
		t.Fatalf("Cannot hash crystallized state: %v", err)
	}
	block.InsertCrystallizedHash(stateHash)

	beaconChain.state.CrystallizedState = &types.CrystallizedState{CurrentEpoch: 9999}

	canProcess, err = beaconChain.CanProcessBlock(&mockFetcher{}, block)
	if err == nil {
		t.Fatalf("CanProcessBlocks should have failed with diff state hashes")
	}
	if canProcess {
		t.Errorf("CanProcessBlocks should have returned false")
	}
}

func TestRotateValidatorSet(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Stop()

	activeValidators := []types.ValidatorRecord{
		{Balance: 10000, WithdrawalAddress: common.Address{'A'}},
		{Balance: 15000, WithdrawalAddress: common.Address{'B'}},
		{Balance: 20000, WithdrawalAddress: common.Address{'C'}},
		{Balance: 25000, WithdrawalAddress: common.Address{'D'}},
		{Balance: 30000, WithdrawalAddress: common.Address{'E'}},
	}

	queuedValidators := []types.ValidatorRecord{
		{Balance: params.DefaultBalance, WithdrawalAddress: common.Address{'F'}},
		{Balance: params.DefaultBalance, WithdrawalAddress: common.Address{'G'}},
		{Balance: params.DefaultBalance, WithdrawalAddress: common.Address{'H'}},
		{Balance: params.DefaultBalance, WithdrawalAddress: common.Address{'I'}},
		{Balance: params.DefaultBalance, WithdrawalAddress: common.Address{'J'}},
	}

	exitedValidators := []types.ValidatorRecord{
		{Balance: 99999, WithdrawalAddress: common.Address{'K'}},
		{Balance: 99999, WithdrawalAddress: common.Address{'L'}},
		{Balance: 99999, WithdrawalAddress: common.Address{'M'}},
		{Balance: 99999, WithdrawalAddress: common.Address{'N'}},
		{Balance: 99999, WithdrawalAddress: common.Address{'O'}},
	}

	beaconChain.CrystallizedState().ActiveValidators = activeValidators
	beaconChain.CrystallizedState().QueuedValidators = queuedValidators
	beaconChain.CrystallizedState().ExitedValidators = exitedValidators

	if beaconChain.ActiveValidatorCount() != 5 {
		t.Errorf("Get active validator count failed, wanted 5, got %v", beaconChain.ActiveValidatorCount())
	}
	if beaconChain.QueuedValidatorCount() != 5 {
		t.Errorf("Get queued validator count failed, wanted 5, got %v", beaconChain.QueuedValidatorCount())
	}
	if beaconChain.ExitedValidatorCount() != 5 {
		t.Errorf("Get exited validator count failed, wanted 5, got %v", beaconChain.ExitedValidatorCount())
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

func TestIsEpochTransition(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Stop()

	beaconChain.state.CrystallizedState.CurrentEpoch = 1
	if !beaconChain.isEpochTransition(30) {
		t.Errorf("there was supposed to be an epoch transition but there isn't one now")
	}
	if beaconChain.isEpochTransition(8) {
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
	defer db.Stop()

	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Could not generate key: %v", err)
	}

	balance1 := uint64(10000)
	balance2 := uint64(15000)
	balance3 := uint64(20000)
	balance4 := uint64(25000)
	balance5 := uint64(30000)

	activeValidators := &types.CrystallizedState{ActiveValidators: []types.ValidatorRecord{
		{Balance: balance1, WithdrawalAddress: common.Address{'A'}, PubKey: enr.Secp256k1(priv.PublicKey)},
		{Balance: balance2, WithdrawalAddress: common.Address{'B'}, PubKey: enr.Secp256k1(priv.PublicKey)},
		{Balance: balance3, WithdrawalAddress: common.Address{'C'}, PubKey: enr.Secp256k1(priv.PublicKey)},
		{Balance: balance4, WithdrawalAddress: common.Address{'D'}, PubKey: enr.Secp256k1(priv.PublicKey)},
		{Balance: balance5, WithdrawalAddress: common.Address{'E'}, PubKey: enr.Secp256k1(priv.PublicKey)},
	}}

	if err := beaconChain.MutateCrystallizedState(activeValidators); err != nil {
		t.Fatalf("unable to mutate crystallizedstate: %v", err)
	}

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

	if expectedBalance1 != beaconChain.state.CrystallizedState.ActiveValidators[0].Balance {
		t.Errorf("rewards and penalties were not able to be applied correctly:%d , %d", expectedBalance1, beaconChain.state.CrystallizedState.ActiveValidators[0].Balance)
	}

	if expectedBalance2 != beaconChain.state.CrystallizedState.ActiveValidators[1].Balance {
		t.Errorf("rewards and penalties were not able to be applied correctly:%d , %d", expectedBalance2, beaconChain.state.CrystallizedState.ActiveValidators[1].Balance)
	}

	if expectedBalance3 != beaconChain.state.CrystallizedState.ActiveValidators[2].Balance {
		t.Errorf("rewards and penalties were not able to be applied correctly:%d , %d", expectedBalance3, beaconChain.state.CrystallizedState.ActiveValidators[2].Balance)
	}

	if expectedBalance4 != beaconChain.state.CrystallizedState.ActiveValidators[3].Balance {
		t.Errorf("rewards and penalties were not able to be applied correctly:%d , %d", expectedBalance4, beaconChain.state.CrystallizedState.ActiveValidators[3].Balance)
	}

	if expectedBalance5 != beaconChain.state.CrystallizedState.ActiveValidators[4].Balance {
		t.Errorf("rewards and penalties were not able to be applied correctly:%d , %d", expectedBalance5, beaconChain.state.CrystallizedState.ActiveValidators[4].Balance)
	}

}

func TestResetAttesterBitfields(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Stop()

	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Could not generate key: %v", err)
	}

	// Testing validator set sizes from 1 to 100.

	for j := 1; j <= 100; j++ {

		var validators []types.ValidatorRecord

		for i := 0; i < j; i++ {
			validator := types.ValidatorRecord{WithdrawalAddress: common.Address{'A'}, PubKey: enr.Secp256k1(priv.PublicKey)}
			validators = append(validators, validator)
		}

		if err := beaconChain.MutateCrystallizedState(&types.CrystallizedState{ActiveValidators: validators}); err != nil {
			t.Fatalf("unable to mutate crystallizedstate: %v", err)
		}

		testAttesterBitfield := []byte{2, 4, 6, 9}

		if err := beaconChain.MutateActiveState(&types.ActiveState{AttesterBitfields: testAttesterBitfield}); err != nil {
			t.Fatal("unable to mutate active state")
		}
		if err := beaconChain.resetAttesterBitfields(); err != nil {
			t.Fatalf("unable to reset Attester Bitfields")
		}

		if bytes.Equal(testAttesterBitfield, beaconChain.state.ActiveState.AttesterBitfields) {
			t.Fatalf("attester bitfields have not been able to be reset: %v", testAttesterBitfield)
		}

		bitfieldLength := j / 8
		if j%8 != 0 {
			bitfieldLength += 1
		}

		if !bytes.Equal(beaconChain.state.ActiveState.AttesterBitfields, make([]byte, bitfieldLength)) {
			t.Fatalf("attester bitfields are not zeroed out: %v", beaconChain.state.ActiveState.AttesterBitfields)
		}

	}

}

func TestResetTotalDeposit(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Stop()

	ActiveState := &types.ActiveState{TotalAttesterDeposits: 10000}
	if err := beaconChain.MutateActiveState(ActiveState); err != nil {
		t.Fatalf("unable to Mutate Active state: %v", err)
	}

	if beaconChain.state.ActiveState.TotalAttesterDeposits != uint64(10000) {
		t.Fatalf("attester deposit was not saved: %d", beaconChain.state.ActiveState.TotalAttesterDeposits)
	}

	if err := beaconChain.resetTotalDeposit(); err != nil {
		t.Fatalf("unable to reset total attester deposit: %v", err)
	}

	if beaconChain.state.ActiveState.TotalAttesterDeposits != uint64(0) {
		t.Fatalf("attester deposit was not able to be reset: %d", beaconChain.state.ActiveState.TotalAttesterDeposits)
	}
}

func TestUpdateJustifiedEpoch(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Stop()

	CrystallizedState := &types.CrystallizedState{CurrentEpoch: 5, LastJustifiedEpoch: 4, LastFinalizedEpoch: 3}
	if err := beaconChain.MutateCrystallizedState(CrystallizedState); err != nil {
		t.Fatalf("unable to Mutate Active state: %v", err)
	}

	if beaconChain.state.CrystallizedState.LastFinalizedEpoch != uint64(3) ||
		beaconChain.state.CrystallizedState.LastJustifiedEpoch != uint64(4) ||
		beaconChain.state.CrystallizedState.CurrentEpoch != uint64(5) {
		t.Fatal("crystallized state unable to be saved")
	}

	if err := beaconChain.updateJustifiedEpoch(); err != nil {
		t.Fatalf("unable to update justified epoch: %v", err)
	}

	if beaconChain.state.CrystallizedState.LastJustifiedEpoch != uint64(5) {
		t.Fatalf("unable to update last justified epoch: %d", beaconChain.state.CrystallizedState.LastJustifiedEpoch)
	}

	if beaconChain.state.CrystallizedState.LastFinalizedEpoch != uint64(4) {
		t.Fatalf("unable to update last finalized epoch: %d", beaconChain.state.CrystallizedState.LastFinalizedEpoch)
	}

	CrystallizedState = &types.CrystallizedState{CurrentEpoch: 8, LastJustifiedEpoch: 4, LastFinalizedEpoch: 3}
	if err := beaconChain.MutateCrystallizedState(CrystallizedState); err != nil {
		t.Fatalf("unable to Mutate Active state: %v", err)
	}

	if beaconChain.state.CrystallizedState.LastFinalizedEpoch != uint64(3) ||
		beaconChain.state.CrystallizedState.LastJustifiedEpoch != uint64(4) ||
		beaconChain.state.CrystallizedState.CurrentEpoch != uint64(8) {
		t.Fatal("crystallized state unable to be saved")
	}

	if err := beaconChain.updateJustifiedEpoch(); err != nil {
		t.Fatalf("unable to update justified epoch: %v", err)
	}

	if beaconChain.state.CrystallizedState.LastJustifiedEpoch != uint64(8) {
		t.Fatalf("unable to update last justified epoch: %d", beaconChain.state.CrystallizedState.LastJustifiedEpoch)
	}

	if beaconChain.state.CrystallizedState.LastFinalizedEpoch != uint64(3) {
		t.Fatalf("unable to update last finalized epoch: %d", beaconChain.state.CrystallizedState.LastFinalizedEpoch)
	}
}

func TestUpdateRewardsAndPenalties(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Stop()
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Could not generate key: %v", err)
	}

	var validators []types.ValidatorRecord

	for i := 0; i < 40; i++ {
		validator := types.ValidatorRecord{Balance: 1000, WithdrawalAddress: common.Address{'A'}, PubKey: enr.Secp256k1(priv.PublicKey)}
		validators = append(validators, validator)
	}

	if err := beaconChain.MutateCrystallizedState(&types.CrystallizedState{ActiveValidators: validators}); err != nil {
		t.Fatalf("unable to mutate crystallizedstate: %v", err)
	}

	//Binary Representation of Bitfield: 00010110 00101011 00101110 01001111 01010000

	testAttesterBitfield := []byte{22, 43, 46, 79, 80}

	if err := beaconChain.MutateActiveState(&types.ActiveState{AttesterBitfields: testAttesterBitfield}); err != nil {
		t.Fatalf("unable to mutate active state: %v", err)
	}
	// Test Validator with index 10 would refer to the 11th bit in the bitfield

	if err := beaconChain.updateRewardsAndPenalties(10); err != nil {
		t.Fatalf("unable to update rewards and penalties: %v", err)
	}

	if beaconChain.state.CrystallizedState.ActiveValidators[10].Balance != uint64(1001) {
		t.Fatal("validator balance not updated")
	}

	// Test Validator with index 15 would refer to the 16th bit in the bitfield
	if err := beaconChain.updateRewardsAndPenalties(15); err != nil {
		t.Fatalf("unable to update rewards and penalties: %v", err)
	}

	if beaconChain.state.CrystallizedState.ActiveValidators[15].Balance != uint64(1001) {
		t.Fatal("validator balance not updated")
	}

	// Test Validator with index 23 would refer to the 24th bit in the bitfield
	if err := beaconChain.updateRewardsAndPenalties(23); err != nil {
		t.Fatalf("unable to update rewards and penalties: %v", err)
	}

	if beaconChain.state.CrystallizedState.ActiveValidators[23].Balance == uint64(1001) {
		t.Fatal("validator balance not updated")
	}

	err = beaconChain.updateRewardsAndPenalties(230)

	if err == nil {
		t.Fatal("no error displayed when there is supposed to be one")

	}

	if err.Error() != "attester index does not exist" {
		t.Fatalf("incorrect error message: ` %s ` is displayed when it should have been: attester index does not exist", err.Error())
	}

}
