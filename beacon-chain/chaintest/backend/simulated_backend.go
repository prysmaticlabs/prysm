// Package backend contains utilities for simulating an entire
// ETH 2.0 beacon chain for e2e tests and benchmarking
// purposes.
package backend

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
)

// SimulatedBackend allowing for a programmatic advancement
// of an in-memory beacon chain for client test runs
// and other e2e use cases.
type SimulatedBackend struct {
	chainService *blockchain.ChainService
	db           *db.BeaconDB
}

// NewSimulatedBackend creates an instance by initializing a chain service
// utilizing a mockDB which will act according to test run parameters specified
// in the common ETH 2.0 client test YAML format.
func NewSimulatedBackend() (*SimulatedBackend, error) {
	db, err := setupDB()
	if err != nil {
		return nil, fmt.Errorf("could not setup simulated backend db: %v", err)
	}
	cs, err := blockchain.NewChainService(context.Background(), &blockchain.Config{
		BeaconDB:         db,
		IncomingBlockBuf: 0,
		EnablePOWChain:   false,
	})
	if err != nil {
		return nil, err
	}
	return &SimulatedBackend{
		chainService: cs,
		db:           db,
	}, nil
}

// RunChainTest uses a parsed set of chaintests from a YAML file
// according to the ETH 2.0 client chain test specification and runs them
// against the simulated backend.
func (sb *SimulatedBackend) RunChainTest(testCase *ChainTestCase) error {
	defer teardownDB(sb.db)
	// Utilize the config parameters in the test case to setup
	// the DB and set global config parameters accordingly.
	// Config parameters include: ValidatorCount, ShardCount,
	// CycleLength, MinCommitteeSize, and more based on the YAML
	// test language specification.
	c := params.BeaconConfig()
	c.ShardCount = testCase.Config.ShardCount
	c.EpochLength = testCase.Config.CycleLength
	c.TargetCommitteeSize = testCase.Config.MinCommitteeSize
	params.OverrideBeaconConfig(c)

	// Then, we create the validators based on the custom test config.
	randaoPreCommit := [32]byte{}
	randaoReveal := hashutil.Hash(randaoPreCommit[:])
	validators := make([]*pb.ValidatorRecord, testCase.Config.ValidatorCount)
	for i := uint64(0); i < testCase.Config.ValidatorCount; i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitSlot:               params.BeaconConfig().EntryExitDelay,
			Balance:                c.MaxDeposit * c.Gwei,
			Pubkey:                 []byte{},
			RandaoCommitmentHash32: randaoReveal[:],
		}
	}
	// TODO(#718): Next step is to update and save the blocks specified
	// in the case case into the DB.
	//
	// Then, we call the updateHead routine and confirm the
	// chain's head is the expected result from the test case.
	return nil
}

// RunShuffleTest uses validator set specified from a YAML file, runs the validator shuffle
// algorithm, then compare the output with the expected output from the YAML file.
func (sb *SimulatedBackend) RunShuffleTest(testCase *ShuffleTestCase) error {
	defer teardownDB(sb.db)

	seed := common.BytesToHash([]byte(testCase.Seed))
	output, err := utils.ShuffleIndices(seed, testCase.Input)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(output, testCase.Output) {
		return fmt.Errorf("shuffle result error: expected %v, actual %v", testCase.Output, output)
	}
	return nil
}

// RunStateTransitionTest advances a beacon chain state transition an N amount of
// slots from a genesis state, with a block being processed at every iteration
// of the state transition function.
func (sb *SimulatedBackend) RunStateTransitionTest(testCase *StateTestCase) error {
	// We setup the initial configuration for running state
	// transition tests below.
	c := params.BeaconConfig()
	c.EpochLength = testCase.Config.EpochLength
	c.DepositsForChainStart = testCase.Config.DepositsForChainStart
	params.OverrideBeaconConfig(c)

	genesisTime := params.BeaconConfig().GenesisTime.Unix()
	deposits := make([]*pb.Deposit, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey:                 []byte(strconv.Itoa(i)),
			RandaoCommitmentHash32: []byte("simulated"),
		}
		depositData, err := b.EncodeDepositData(
			depositInput,
			params.BeaconConfig().MaxDepositInGwei,
			genesisTime,
		)
		if err != nil {
			return fmt.Errorf("could not encode initial block deposits: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}

	beaconState, err := state.InitialBeaconState(deposits, uint64(genesisTime), nil)
	if err != nil {
		return err
	}

	// We do not expect hashing initial beacon state and genesis block to
	// fail, so we can safely ignore the error below.
	// #nosec G104
	encodedState, _ := proto.Marshal(beaconState)
	stateRoot := hashutil.Hash(encodedState)
	genesisBlock := b.NewGenesisBlock(stateRoot[:])
	// #nosec G104
	encodedGenesisBlock, _ := proto.Marshal(genesisBlock)
	prevBlockRoot := hashutil.Hash(encodedGenesisBlock)

	startTime := time.Now()
	for i := uint64(0); i < testCase.Config.NumSlots; i++ {
		newState, err := state.ExecuteStateTransition(beaconState, nil, prevBlockRoot)
		if err != nil {
			return fmt.Errorf("could not execute state transition: %v", err)
		}
		beaconState = newState
	}
	endTime := time.Now()
	log.Infof(
		"%d state transitions with %d deposits finished in %v",
		testCase.Config.NumSlots,
		testCase.Config.DepositsForChainStart,
		endTime.Sub(startTime),
	)

	if beaconState.GetSlot() != testCase.Results.Slot {
		return fmt.Errorf(
			"incorrect state slot after %d state transitions without blocks, wanted %d, received %d",
			testCase.Config.NumSlots,
			testCase.Config.NumSlots,
			testCase.Results.Slot,
		)
	}
	return nil
}
