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
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	log "github.com/sirupsen/logrus"
)

// SimulatedBackend allowing for a programmatic advancement
// of an in-memory beacon chain for client test runs
// and other e2e use cases.
type SimulatedBackend struct {
	chainService            *blockchain.ChainService
	beaconDB                *db.BeaconDB
	state                   *pb.BeaconState
	hashOnions              [][32]byte
	lastRandaoLayer         [32]byte
	layersPeeledForProposer map[uint64]int
	prevBlockRoots          [][32]byte
	inMemoryBlocks          []*pb.BeaconBlock
	depositTrie             *trieutil.DepositTrie
}

// SimulatedObjects is a container to hold the
// required primitives for generation of a beacon
// block.
type SimulatedObjects struct {
	simDeposit          *StateTestDeposit
	simProposerSlashing *StateTestProposerSlashing
	simAttesterSlashing *StateTestAttesterSlashing
	simValidatorExit    *StateTestValidatorExit
}

// Number of slots to have the randao reveal hashed for.
var lengthOfOnion uint64 = 1000

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

// InitializeChain sets up the whole backend to be able to run a mock
// beacon state and chain.
func (sb *SimulatedBackend) InitializeChain() error {
	// Generating hash onion for 1000 slots.
	sb.hashOnions = generateSimulatedRandaoHashOnions(lengthOfOnion)

	sb.lastRandaoLayer = sb.hashOnions[len(sb.hashOnions)-1]
	initialDeposits, err := generateInitialSimulatedDeposits(sb.lastRandaoLayer)
	if err != nil {
		return fmt.Errorf("could not simulate initial validator deposits: %v", err)
	}

	if err := sb.setupBeaconStateAndGenesisBlock(initialDeposits); err != nil {
		return err
	}

	// We keep track of the randao layers peeled for each proposer index in a map.
	sb.layersPeeledForProposer = make(map[uint64]int, len(sb.state.ValidatorRegistry))
	for idx := range sb.state.ValidatorRegistry {
		sb.layersPeeledForProposer[uint64(idx)] = 0
	}

	sb.depositTrie = trieutil.NewDepositTrie()
	sb.inMemoryBlocks = make([]*pb.BeaconBlock, 1000)

	return nil
}

// GenerateBlockAndAdvanceChain generates a simulated block and runs that block though
// state transition.
func (sb *SimulatedBackend) GenerateBlockAndAdvanceChain(objects *SimulatedObjects) error {
	slotToGenerate := sb.state.Slot + 1
	prevBlockRoot := sb.prevBlockRoots[len(sb.prevBlockRoots)-1]

	proposerIndex, err := validators.BeaconProposerIdx(sb.state, slotToGenerate)
	if err != nil {
		return fmt.Errorf("could not compute proposer index %v", err)
	}

	layersPeeled := sb.layersPeeledForProposer[proposerIndex]
	blockRandaoReveal := determineSimulatedBlockRandaoReveal(layersPeeled, sb.hashOnions)

	// We generate a new block to pass into the state transition.
	newBlock, newBlockRoot, err := generateSimulatedBlock(
		sb.state,
		prevBlockRoot,
		blockRandaoReveal,
		sb.lastRandaoLayer,
		sb.depositTrie,
		objects,
	)
	if err != nil {
		return fmt.Errorf("could not generate simulated beacon block %v", err)
	}
	latestRoot := sb.depositTrie.Root()

	sb.state.LatestEth1Data = &pb.Eth1Data{
		DepositRootHash32: latestRoot[:],
		BlockHash32:       []byte{},
	}

	newState, err := state.ExecuteStateTransition(
		sb.state,
		newBlock,
		prevBlockRoot,
		false, /*  no sig verify */
	)
	if err != nil {
		return fmt.Errorf("could not execute state transition: %v", err)
	}

	sb.state = newState
	sb.prevBlockRoots = append(sb.prevBlockRoots, newBlockRoot)
	sb.layersPeeledForProposer[proposerIndex]++
	sb.inMemoryBlocks = append(sb.inMemoryBlocks, newBlock)

	return nil
}

// GenerateNilBlockAndAdvanceChain would trigger a state transition with a nil block.
func (sb *SimulatedBackend) GenerateNilBlockAndAdvanceChain() error {
	slotToGenerate := sb.state.Slot + 1
	prevBlockRoot := sb.prevBlockRoots[len(sb.prevBlockRoots)-1]

	proposerIndex, err := validators.BeaconProposerIdx(sb.state, slotToGenerate)
	if err != nil {
		return fmt.Errorf("could not compute proposer index %v", err)
	}

	newState, err := state.ExecuteStateTransition(
		sb.state,
		nil,
		prevBlockRoot,
		false, /* no sig verify */
	)
	if err != nil {
		return fmt.Errorf("could not execute state transition: %v", err)
	}
	sb.state = newState
	sb.layersPeeledForProposer[proposerIndex]++
	return nil
}

// Shutdown closes the db associated with the simulated backend.
func (sb *SimulatedBackend) Shutdown() error {
	return sb.beaconDB.Close()
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
	validators := make([]*pb.Validator, testCase.Config.ValidatorCount)
	for i := uint64(0); i < testCase.Config.ValidatorCount; i++ {
		validators[i] = &pb.Validator{
			ExitEpoch:              params.BeaconConfig().EntryExitDelay,
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
	setTestConfig(testCase)

	if err := sb.initializeStateTest(testCase); err != nil {
		return fmt.Errorf("could not initialize state test %v", err)
	}

	sb.depositTrie = trieutil.NewDepositTrie()
	averageTimesPerTransition := []time.Duration{}
	for i := uint64(0); i < testCase.Config.NumSlots; i++ {

		// If the slot is marked as skipped in the configuration options,
		// we simply run the state transition with a nil block argument.
		if sliceutil.IsInUint64(i, testCase.Config.SkipSlots) {
			if err := sb.GenerateNilBlockAndAdvanceChain(); err != nil {
				return fmt.Errorf("could not advance the chain with a nil block %v", err)
			}
			continue
		}

		simulatedObjects := sb.generateSimulatedObjects(testCase, i)
		startTime := time.Now()

		if err := sb.GenerateBlockAndAdvanceChain(simulatedObjects); err != nil {
			return fmt.Errorf("could not generate the block and advance the chain %v", err)
		}

		endTime := time.Now()
		averageTimesPerTransition = append(averageTimesPerTransition, endTime.Sub(startTime))
	}

	log.Infof(
		"with %d initial deposits, each state transition took average time = %v",
		testCase.Config.DepositsForChainStart,
		averageDuration(averageTimesPerTransition),
	)

	if err := sb.compareTestCase(testCase); err != nil {
		return err
	}

	return nil
}

// initializeStateTest sets up the environment by generating all the required objects in order
// to proceed with the state test.
func (sb *SimulatedBackend) initializeStateTest(testCase *StateTestCase) error {

	// We create a list of randao hash onions for the given number of slots
	// the simulation will attempt.
	sb.hashOnions = generateSimulatedRandaoHashOnions(testCase.Config.NumSlots)

	// We then generate initial validator deposits for initializing the
	// beacon state based where every validator will use the last layer in the randao
	// onions list as the commitment in the deposit instance.
	sb.lastRandaoLayer = sb.hashOnions[len(sb.hashOnions)-1]
	initialDeposits, err := generateInitialSimulatedDeposits(sb.lastRandaoLayer)
	if err != nil {
		return fmt.Errorf("could not simulate initial validator deposits: %v", err)
	}

	if err := sb.setupBeaconStateAndGenesisBlock(initialDeposits); err != nil {
		return fmt.Errorf("could not set up beacon state and initialize genesis block %v", err)
	}

	// We keep track of the randao layers peeled for each proposer index in a map.
	sb.layersPeeledForProposer = make(map[uint64]int, len(sb.state.ValidatorRegistry))
	for idx := range sb.state.ValidatorRegistry {
		sb.layersPeeledForProposer[uint64(idx)] = 0
	}

	return nil
}

// setupBeaconStateAndGenesisBlock creates the initial beacon state and genesis block in order to
// proceed with the test.
func (sb *SimulatedBackend) setupBeaconStateAndGenesisBlock(initialDeposits []*pb.Deposit) error {
	var err error
	genesisTime := params.BeaconConfig().GenesisTime.Unix()
	sb.state, err = state.InitialBeaconState(initialDeposits, uint64(genesisTime), nil)
	if err != nil {
		return fmt.Errorf("could not initialize simulated beacon state")
	}

	// We do not expect hashing initial beacon state and genesis block to
	// fail, so we can safely ignore the error below.
	// #nosec G104
	encodedState, _ := proto.Marshal(sb.state)
	stateRoot := hashutil.Hash(encodedState)
	genesisBlock := b.NewGenesisBlock(stateRoot[:])
	// #nosec G104
	encodedGenesisBlock, _ := proto.Marshal(genesisBlock)
	genesisBlockRoot := hashutil.Hash(encodedGenesisBlock)

	// We now keep track of generated blocks for each state transition in
	// a slice.
	sb.prevBlockRoots = [][32]byte{genesisBlockRoot}

	return nil
}

// generateSimulatedObjects generates the simulated objects depending on the testcase and current slot.
func (sb *SimulatedBackend) generateSimulatedObjects(testCase *StateTestCase, slotNumber uint64) *SimulatedObjects {

	// If the slot is not skipped, we check if we are simulating a deposit at the current slot.
	var simulatedDeposit *StateTestDeposit
	for _, deposit := range testCase.Config.Deposits {
		if deposit.Slot == slotNumber {
			simulatedDeposit = deposit
			break
		}
	}
	var simulatedProposerSlashing *StateTestProposerSlashing
	for _, pSlashing := range testCase.Config.ProposerSlashings {
		if pSlashing.Slot == slotNumber {
			simulatedProposerSlashing = pSlashing
			break
		}
	}
	var simulatedAttesterSlashing *StateTestAttesterSlashing
	for _, cSlashing := range testCase.Config.AttesterSlashings {
		if cSlashing.Slot == slotNumber {
			simulatedAttesterSlashing = cSlashing
			break
		}
	}
	var simulatedValidatorExit *StateTestValidatorExit
	for _, exit := range testCase.Config.ValidatorExits {
		if exit.Slot == slotNumber {
			simulatedValidatorExit = exit
			break
		}
	}

	return &SimulatedObjects{
		simDeposit:          simulatedDeposit,
		simProposerSlashing: simulatedProposerSlashing,
		simAttesterSlashing: simulatedAttesterSlashing,
		simValidatorExit:    simulatedValidatorExit,
	}
}

// compareTestCase compares the state in the simulated backend against the values in inputted test case. If
// there are any discrepancies it returns an error.
func (sb *SimulatedBackend) compareTestCase(testCase *StateTestCase) error {

	if sb.state.Slot != testCase.Results.Slot {
		return fmt.Errorf(
			"incorrect state slot after %d state transitions without blocks, wanted %d, received %d",
			testCase.Config.NumSlots,
			testCase.Config.NumSlots,
			testCase.Results.Slot,
		)
	}
	if len(sb.state.ValidatorRegistry) != testCase.Results.NumValidators {
		return fmt.Errorf(
			"incorrect num validators after %d state transitions without blocks, wanted %d, received %d",
			testCase.Config.NumSlots,
			testCase.Results.NumValidators,
			len(sb.state.ValidatorRegistry),
		)
	}
	for _, penalized := range testCase.Results.PenalizedValidators {
		if sb.state.ValidatorRegistry[penalized].PenalizedEpoch == params.BeaconConfig().FarFutureEpoch {
			return fmt.Errorf(
				"expected validator at index %d to have been penalized",
				penalized,
			)
		}
	}
	for _, exited := range testCase.Results.ExitedValidators {
		if sb.state.ValidatorRegistry[exited].StatusFlags != pb.Validator_INITIATED_EXIT {
			return fmt.Errorf(
				"expected validator at index %d to have exited",
				exited,
			)
		}
	}
	return nil
}

func setTestConfig(testCase *StateTestCase) {
	// We setup the initial configuration for running state
	// transition tests below.
	c := params.BeaconConfig()
	c.EpochLength = testCase.Config.EpochLength
	c.DepositsForChainStart = testCase.Config.DepositsForChainStart
	params.OverrideBeaconConfig(c)
}

func averageDuration(times []time.Duration) time.Duration {
	sum := int64(0)
	for _, t := range times {
		sum += t.Nanoseconds()
	}
	return time.Duration(sum / int64(len(times)))
}
