// Package backend contains utilities for simulating an entire
// ETH 2.0 beacon chain for e2e tests and benchmarking
// purposes.
package backend

import (
	"context"
	"fmt"
	"reflect"
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

	// We create a list of randao hash onions for the given number of slots
	// the simulation will attempt.
	hashOnions := generateSimulatedRandaoHashOnions(testCase.Config.NumSlots)

	// We then generate initial validator deposits for initializing the
	// beacon state based where every validator will use the last layer in the randao
	// onions list as the commitment in the deposit instance.
	lastRandaoLayer := hashOnions[len(hashOnions)-1]
	initialDeposits, err := generateInitialSimulatedDeposits(lastRandaoLayer)
	if err != nil {
		return fmt.Errorf("could not simulate initial validator deposits: %v", err)
	}

	genesisTime := params.BeaconConfig().GenesisTime.Unix()
	beaconState, err := state.InitialBeaconState(initialDeposits, uint64(genesisTime), nil)
	if err != nil {
		return fmt.Errorf("could not initialize simulated beacon state")
	}

	// We do not expect hashing initial beacon state and genesis block to
	// fail, so we can safely ignore the error below.
	// #nosec G104
	encodedState, _ := proto.Marshal(beaconState)
	stateRoot := hashutil.Hash(encodedState)
	genesisBlock := b.NewGenesisBlock(stateRoot[:])
	// #nosec G104
	encodedGenesisBlock, _ := proto.Marshal(genesisBlock)
	genesisBlockRoot := hashutil.Hash(encodedGenesisBlock)

	// We now keep track of generated blocks for each state transition in
	// a slice.
	prevBlockRoots := [][32]byte{genesisBlockRoot}
	prevBlocks := []*pb.BeaconBlock{genesisBlock}

	// We keep track of the randao layers peeled for each proposer index in a map.
	layersPeeledForProposer := make(map[uint32]int, len(beaconState.ValidatorRegistry))
	for idx := range beaconState.ValidatorRegistry {
		layersPeeledForProposer[uint32(idx)] = 0
	}

	startTime := time.Now()
	for i := uint64(0); i < testCase.Config.NumSlots; i++ {
		prevBlockRoot := prevBlockRoots[len(prevBlockRoots)-1]
		prevBlock := prevBlocks[len(prevBlocks)-1]

		proposerIndex, err := findNextSlotProposerIndex(beaconState)
		if err != nil {
			return fmt.Errorf("could not fetch beacon proposer index: %v", err)
		}

		// If the slot is marked as skipped in the configuration options,
		// we simply run the state transition with a nil block argument.
		if contains(i, testCase.Config.SkipSlots) {
			newState, err := state.ExecuteStateTransition(beaconState, nil, prevBlockRoot)
			if err != nil {
				return fmt.Errorf("could not execute state transition: %v", err)
			}
			beaconState = newState
			layersPeeledForProposer[proposerIndex]++
			continue
		}

		layersPeeled := layersPeeledForProposer[proposerIndex]
		blockRandaoReveal := determineSimulatedBlockRandaoReveal(layersPeeled, hashOnions)

		// We generate a new block to pass into the state transition.
		newBlock, newBlockRoot, err := generateSimulatedBlock(
			beaconState,
			prevBlock,
			prevBlockRoot,
			blockRandaoReveal,
		)
		if err != nil {
			return fmt.Errorf("could not generate simulated beacon block %v", err)
		}
		newState, err := state.ExecuteStateTransition(beaconState, newBlock, prevBlockRoot)
		if err != nil {
			return fmt.Errorf("could not execute state transition: %v", err)
		}

		// We then keep track of information about the state after the
		// state transition was applied.
		beaconState = newState
		prevBlockRoots = append(prevBlockRoots, newBlockRoot)
		prevBlocks = append(prevBlocks, newBlock)
		layersPeeledForProposer[proposerIndex]++
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

func contains(item uint64, slice []uint64) bool {
	if len(slice) == 0 {
		return false
	}
	for _, a := range slice {
		if item == a {
			return true
		}
	}
	return false
}
