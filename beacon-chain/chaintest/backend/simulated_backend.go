// Package backend contains utilities for simulating an entire
// ETH 2.0 beacon chain for e2e tests and benchmarking
// purposes.
package backend

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/prysmaticlabs/prysm/shared/trie"

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
	"github.com/prysmaticlabs/prysm/shared/slices"
	log "github.com/sirupsen/logrus"
)

// SimulatedBackend allowing for a programmatic advancement
// of an in-memory beacon chain for client test runs
// and other e2e use cases.
type SimulatedBackend struct {
	chainService *blockchain.ChainService
	beaconDB     *db.BeaconDB
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
		beaconDB:     db,
	}, nil
}

// RunForkChoiceTest uses a parsed set of chaintests from a YAML file
// according to the ETH 2.0 client chain test specification and runs them
// against the simulated backend.
func (sb *SimulatedBackend) RunForkChoiceTest(testCase *ForkChoiceTestCase) error {
	defer teardownDB(sb.beaconDB)
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
	defer teardownDB(sb.beaconDB)
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
	defer teardownDB(sb.beaconDB)
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

	// We keep track of the randao layers peeled for each proposer index in a map.
	layersPeeledForProposer := make(map[uint32]int, len(beaconState.ValidatorRegistry))
	for idx := range beaconState.ValidatorRegistry {
		layersPeeledForProposer[uint32(idx)] = 0
	}

	depositsTrie := trie.NewDepositTrie()
	averageTimesPerTransition := []time.Duration{}
	for i := uint64(0); i < testCase.Config.NumSlots; i++ {
		prevBlockRoot := prevBlockRoots[len(prevBlockRoots)-1]

		proposerIndex, err := findNextSlotProposerIndex(beaconState)
		if err != nil {
			return fmt.Errorf("could not fetch beacon proposer index: %v", err)
		}

		// If the slot is marked as skipped in the configuration options,
		// we simply run the state transition with a nil block argument.
		if slices.IsInUint64(i, testCase.Config.SkipSlots) {
			newState, err := state.ExecuteStateTransition(beaconState, nil, prevBlockRoot)
			if err != nil {
				return fmt.Errorf("could not execute state transition: %v", err)
			}
			beaconState = newState
			layersPeeledForProposer[proposerIndex]++
			continue
		}

		// If the slot is not skipped, we check if we are simulating a deposit at the current slot.
		var simulatedDeposit *StateTestDeposit
		for _, deposit := range testCase.Config.Deposits {
			if deposit.Slot == i {
				simulatedDeposit = deposit
				break
			}
		}
		var simulatedProposerSlashing *StateTestProposerSlashing
		for _, pSlashing := range testCase.Config.ProposerSlashings {
			if pSlashing.Slot == i {
				simulatedProposerSlashing = pSlashing
				break
			}
		}
		var simulatedCasperSlashing *StateTestCasperSlashing
		for _, cSlashing := range testCase.Config.CasperSlashings {
			if cSlashing.Slot == i {
				simulatedCasperSlashing = cSlashing
				break
			}
		}
		var simulatedValidatorExit *StateTestValidatorExit
		for _, exit := range testCase.Config.ValidatorExits {
			if exit.Slot == i {
				simulatedValidatorExit = exit
				break
			}
		}

		layersPeeled := layersPeeledForProposer[proposerIndex]
		blockRandaoReveal := determineSimulatedBlockRandaoReveal(layersPeeled, hashOnions)

		// We generate a new block to pass into the state transition.
		newBlock, newBlockRoot, err := generateSimulatedBlock(
			beaconState,
			prevBlockRoot,
			blockRandaoReveal,
			lastRandaoLayer,
			simulatedDeposit,
			depositsTrie,
			simulatedProposerSlashing,
			simulatedCasperSlashing,
			simulatedValidatorExit,
		)
		if err != nil {
			return fmt.Errorf("could not generate simulated beacon block %v", err)
		}
		latestRoot := depositsTrie.Root()
		beaconState.LatestDepositRootHash32 = latestRoot[:]

		startTime := time.Now()
		newState, err := state.ExecuteStateTransition(beaconState, newBlock, prevBlockRoot)
		if err != nil {
			return fmt.Errorf("could not execute state transition: %v", err)
		}
		endTime := time.Now()
		averageTimesPerTransition = append(averageTimesPerTransition, endTime.Sub(startTime))

		// We then keep track of information about the state after the
		// state transition was applied.
		beaconState = newState
		prevBlockRoots = append(prevBlockRoots, newBlockRoot)
		layersPeeledForProposer[proposerIndex]++
	}

	log.Infof(
		"with %d initial deposits, each state transition took average time = %v",
		testCase.Config.DepositsForChainStart,
		averageDuration(averageTimesPerTransition),
	)

	if beaconState.Slot != testCase.Results.Slot {
		return fmt.Errorf(
			"incorrect state slot after %d state transitions without blocks, wanted %d, received %d",
			testCase.Config.NumSlots,
			testCase.Config.NumSlots,
			testCase.Results.Slot,
		)
	}
	if len(beaconState.ValidatorRegistry) != testCase.Results.NumValidators {
		return fmt.Errorf(
			"incorrect num validators after %d state transitions without blocks, wanted %d, received %d",
			testCase.Config.NumSlots,
			testCase.Results.NumValidators,
			len(beaconState.ValidatorRegistry),
		)
	}
	for _, penalized := range testCase.Results.PenalizedValidators {
		if beaconState.ValidatorRegistry[penalized].PenalizedSlot == params.BeaconConfig().FarFutureSlot {
			return fmt.Errorf(
				"expected validator at index %d to have been penalized",
				penalized,
			)
		}
	}
	for _, exited := range testCase.Results.ExitedValidators {
		if beaconState.ValidatorRegistry[exited].StatusFlags != pb.ValidatorRecord_INITIATED_EXIT {
			return fmt.Errorf(
				"expected validator at index %d to have exited",
				exited,
			)
		}
	}
	return nil
}

func averageDuration(times []time.Duration) time.Duration {
	sum := int64(0)
	for _, t := range times {
		sum += t.Nanoseconds()
	}
	return time.Duration(sum / int64(len(times)))
}
