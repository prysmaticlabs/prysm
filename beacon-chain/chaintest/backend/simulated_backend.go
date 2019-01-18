// Package backend contains utilities for simulating an entire
// ETH 2.0 beacon chain for e2e tests and benchmarking
// purposes.
package backend

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/prysmaticlabs/prysm/shared/trie"
	logTest "github.com/sirupsen/logrus/hooks/test"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	contracts "github.com/prysmaticlabs/prysm/contracts/validator-registration-contract"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slices"
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

func (sb *SimulatedBackend) Close() {
	teardownDB(sb.beaconDB)
}

// RunForkChoiceTest uses a parsed set of chaintests from a YAML file
// according to the ETH 2.0 client chain test specification and runs them
// against the simulated backend.
func (sb *SimulatedBackend) RunForkChoiceTest(testCase *ForkChoiceTestCase) error {
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

	hook := logTest.NewGlobal()
	endpoint := "ws://127.0.0.1"
	testAcc, err := setupPOWChainAccount()
	if err != nil {
		return fmt.Errorf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := powchain.NewWeb3Service(context.Background(), &powchain.Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		return fmt.Errorf("unable to setup web3 PoW chain service: %v", err)
	}

	testAcc.backend.Commit()
	testAcc.backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	web3Service.DepositTrie = trie.NewDepositTrie()
	currentRoot := web3Service.DepositTrie.Root()

	var stub [48]byte
	copy(stub[:], []byte("testing"))

	data := &pb.DepositInput{
		Pubkey:                      stub[:],
		ProofOfPossession:           stub[:],
		WithdrawalCredentialsHash32: []byte("withdraw"),
		RandaoCommitmentHash32:      []byte("randao"),
		CustodyCommitmentHash32:     []byte("custody"),
	}

	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		fmt.Errorf("Could not serialize data %v", err)
	}

	for i := 0; i < 8; i++ {
		testAcc.txOpts.Value = amount32Eth
		if _, err := testAcc.contract.Deposit(testAcc.txOpts, serializedData.Bytes()); err != nil {
			return fmt.Errorf("Could not deposit to VRC %v", err)
		}
		testAcc.backend.Commit()
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
            testAcc.contractAddr,
		},
	}

	logs, err := testAcc.backend.FilterLogs(context.Background(), query)
	if err != nil {
		return fmt.Errorf("Unable to retrieve logs %v", err)
	}

	logs[len(logs)-1].Topics[1] = currentRoot

	web3Service.ProcessLog(logs[len(logs)-1])

	if err := assertLogsDoNotContain(hook, "Unable to unpack ChainStart log data"); err != nil {
		return fmt.Errorf("chain start log failure: %v", err)
	}
	if err := assertLogsDoNotContain(hook, "Reciept root from log doesn't match the root saved in memory"); err != nil {
		return fmt.Errorf("chain start log failure: %v", err)
	}
	if err := assertLogsDoNotContain(hook, "Invalid timestamp from log"); err != nil {
		return fmt.Errorf("chain start log failure: %v", err)
	}
	if err := assertLogsContain(hook, "Minimum Number of Validators Reached for beacon-chain to start"); err != nil {
		return fmt.Errorf("chain start log failure: %v", err)
	}
	hook.Reset()

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

	// TODO: Use the deposits from the contract instead to initialize the state.
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
	genesisBlockRoot, err := hashutil.HashBeaconBlock(genesisBlock)
	if err != nil {
		return fmt.Errorf("could not hash genesis block: %v", err)
	}
	if err := sb.beaconDB.SaveBlock(genesisBlock); err != nil {
		return fmt.Errorf("could not save genesis block: %v", err)
	}
	if err := sb.beaconDB.UpdateChainHead(genesisBlock, beaconState); err != nil {
		return fmt.Errorf("could not save initial beacon state: %v", err)
	}

	// We now keep track of generated blocks for each state transition in
	// a slice.
	prevBlockRoots := [][32]byte{genesisBlockRoot}

	// We keep track of the randao layers peeled for each proposer index in a map.
	layersPeeledForProposer := make(map[uint32]int, len(beaconState.ValidatorRegistry))
	for idx := range beaconState.ValidatorRegistry {
		layersPeeledForProposer[uint32(idx)] = 0
	}

	// We then initialize a ChainService we can effectively use to test the runtime
	// of the node after the ChainStart log has been triggered.
	chainService, err := blockchain.NewChainService(context.Background(), &blockchain.Config{
		BeaconBlockBuf: 0,
		BeaconDB:       sb.beaconDB,
		Web3Service:    web3Service,
		EnablePOWChain: false,
	})
	if err != nil {
		return fmt.Errorf("unable to setup chain service: %v", err)
	}
	exitRoutine := make(chan bool)
	go func() {
		chainService.Start()
		<-exitRoutine
	}()

	depositsTrie := trie.NewDepositTrie()
	for i := uint64(1); i <= testCase.Config.NumSlots; i++ {
		time.Sleep(time.Second*2)
		currentState, err := sb.beaconDB.GetState()
		if err != nil {
			return fmt.Errorf("could not get current beacon state: %v", err)
		}

		prevBlockRoot := prevBlockRoots[len(prevBlockRoots)-1]
		proposerIndex, err := findNextSlotProposerIndex(currentState)
		if err != nil {
			return fmt.Errorf("could not fetch beacon proposer index: %v", err)
		}

		// If the slot is marked as skipped in the configuration options,
		// we simply run the state transition with a nil block argument.
		if slices.IsInUint64(i, testCase.Config.SkipSlots) {
			layersPeeledForProposer[proposerIndex]++
			continue
		}

		// If the slot is not skipped, we check if we are simulating a deposit at the current slot.
		// TODO: Use the POWChain service to do this and have an aggregator of deposits in the chain service.
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
            currentState,
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

		// We trigger a new block in the blockchain service.
		chainService.IncomingBlockFeed.Send(newBlock)

		// We then keep track of information about the state after the
		// state transition was applied.
		prevBlockRoots = append(prevBlockRoots, newBlockRoot)
		layersPeeledForProposer[proposerIndex]++
	}

	time.Sleep(time.Second*2)
	chainService.Cancel()
	exitRoutine <- true

	finalState, err := sb.beaconDB.GetState()
	if err != nil {
		return fmt.Errorf("could not get final beacon state: %v", err)
	}

	if finalState.Slot != testCase.Results.Slot {
		return fmt.Errorf(
			"incorrect state slot after %d state transitions without blocks, wanted %d, received %d",
			testCase.Config.NumSlots,
			testCase.Results.Slot,
            finalState.Slot,
		)
	}
	if len(finalState.ValidatorRegistry) != testCase.Results.NumValidators {
		return fmt.Errorf(
			"incorrect num validators after %d state transitions without blocks, wanted %d, received %d",
			testCase.Config.NumSlots,
			testCase.Results.NumValidators,
			len(finalState.ValidatorRegistry),
		)
	}
	for _, penalized := range testCase.Results.PenalizedValidators {
		if finalState.ValidatorRegistry[penalized].PenalizedSlot == params.BeaconConfig().FarFutureSlot {
			return fmt.Errorf(
				"expected validator at index %d to have been penalized",
				penalized,
			)
		}
	}
	for _, exited := range testCase.Results.ExitedValidators {
		if finalState.ValidatorRegistry[exited].StatusFlags != pb.ValidatorRecord_INITIATED_EXIT {
			return fmt.Errorf(
				"expected validator at index %d to have exited",
				exited,
			)
		}
	}
	return nil
}

var amount32Eth, _ = new(big.Int).SetString("32000000000000000000", 10)

type goodReader struct{}

func (g *goodReader) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

type goodLogger struct{}

func (g *goodLogger) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- gethTypes.Log) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

func (g *goodLogger) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]gethTypes.Log, error) {
	logs := make([]gethTypes.Log, 3)
	for i := 0; i < len(logs); i++ {
		logs[i].Address = common.Address{}
		logs[i].Topics = make([]common.Hash, 5)
		logs[i].Topics[0] = common.Hash{'a'}
		logs[i].Topics[1] = common.Hash{'b'}
		logs[i].Topics[2] = common.Hash{'c'}

	}
	return logs, nil
}

type testAccount struct {
	addr         common.Address
	contract     *contracts.ValidatorRegistration
	contractAddr common.Address
	backend      *backends.SimulatedBackend
	txOpts       *bind.TransactOpts
}

func setupPOWChainAccount() (*testAccount, error) {
	genesis := make(core.GenesisAlloc)
	privKey, _ := crypto.GenerateKey()
	pubKeyECDSA, ok := privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}

	// strip off the 0x and the first 2 characters 04 which is always the EC prefix and is not required.
	publicKeyBytes := crypto.FromECDSAPub(pubKeyECDSA)[4:]
	var pubKey = make([]byte, 48)
	copy(pubKey[:], []byte(publicKeyBytes))

	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	txOpts := bind.NewKeyedTransactor(privKey)
	startingBalance, _ := new(big.Int).SetString("900000000000000000000000", 10)
	genesis[addr] = core.GenesisAccount{Balance: startingBalance}
	backend := backends.NewSimulatedBackend(genesis, 2100000)

	contractAddr, _, contract, err := contracts.DeployValidatorRegistration(txOpts, backend)
	if err != nil {
		return nil, err
	}
	return &testAccount{addr, contract, contractAddr, backend, txOpts}, nil
}

func averageDuration(times []time.Duration) time.Duration {
	sum := int64(0)
	for _, t := range times {
		sum += t.Nanoseconds()
	}
	return time.Duration(sum / int64(len(times)))
}

// assertLogsContain checks that the desired string is a subset of the current log output.
// Set exitOnFail to true to immediately exit the test on failure
func assertLogsContain(hook *logTest.Hook, want string) error {
	return assertLogs(hook, want, true)
}

// assertLogsDoNotContain is the inverse check of AssertLogsContain
func assertLogsDoNotContain(hook *logTest.Hook, want string) error {
	return assertLogs(hook, want, false)
}

func assertLogs(hook *logTest.Hook, want string, flag bool) error {
	entries := hook.AllEntries()
	match := false
	for _, e := range entries {
		if strings.Contains(e.Message, want) {
			match = true
		}
	}

	if flag && !match {
		return fmt.Errorf("log not found: %s", want)
	} else if !flag && match {
		return fmt.Errorf("unwanted log found: %s", want)
	}
	return nil
}

