// Package powchain defines a runtime service which is tasked with
// communicating with an eth1 endpoint, processing logs from a deposit
// contract, and the latest eth1 data headers for usage in the beacon node.
package powchain

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	protodb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "powchain")

var (
	validDepositsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powchain_valid_deposits_received",
		Help: "The number of valid deposits received in the deposit contract",
	})
	blockNumberGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powchain_block_number",
		Help: "The current block number in the proof-of-work chain",
	})
	missedDepositLogsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powchain_missed_deposit_logs",
		Help: "The number of times a missed deposit log is detected",
	})
)

// time to wait before trying to reconnect with the eth1 node.
var backOffPeriod = 6 * time.Second

// Amount of times before we log the status of the eth1 dial attempt.
var logThreshold = 20

// ChainStartFetcher retrieves information pertaining to the chain start event
// of the beacon chain for usage across various services.
type ChainStartFetcher interface {
	ChainStartDeposits() []*ethpb.Deposit
	ChainStartEth1Data() *ethpb.Eth1Data
	PreGenesisState() *stateTrie.BeaconState
	ClearPreGenesisData()
}

// ChainInfoFetcher retrieves information about eth1 metadata at the eth2 genesis time.
type ChainInfoFetcher interface {
	Eth2GenesisPowchainInfo() (uint64, *big.Int)
	IsConnectedToETH1() bool
}

// POWBlockFetcher defines a struct that can retrieve mainchain blocks.
type POWBlockFetcher interface {
	BlockTimeByHeight(ctx context.Context, height *big.Int) (uint64, error)
	BlockNumberByTimestamp(ctx context.Context, time uint64) (*big.Int, error)
	BlockHashByHeight(ctx context.Context, height *big.Int) (common.Hash, error)
	BlockExists(ctx context.Context, hash common.Hash) (bool, *big.Int, error)
}

// Chain defines a standard interface for the powchain service in Prysm.
type Chain interface {
	ChainStartFetcher
	ChainInfoFetcher
	POWBlockFetcher
}

// Client defines a struct that combines all relevant ETH1.0 mainchain interactions required
// by the beacon chain node.
type Client interface {
	RPCDataFetcher
	bind.ContractFilterer
	bind.ContractCaller
}

// RPCDataFetcher defines a subset of methods conformed to by ETH1.0 RPC clients for
// fetching eth1 data from the clients.
type RPCDataFetcher interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*gethTypes.Header, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*gethTypes.Block, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error)
	SyncProgress(ctx context.Context) (*ethereum.SyncProgress, error)
}

// RPCClient defines the rpc methods required to interact with the eth1 node.
type RPCClient interface {
	BatchCall(b []gethRPC.BatchElem) error
}

// Service fetches important information about the canonical
// Ethereum ETH1.0 chain via a web3 endpoint using an ethclient. The Random
// Beacon Chain requires synchronization with the ETH1.0 chain's current
// blockhash, block number, and access to logs within the
// Validator Registration Contract on the ETH1.0 chain to kick off the beacon
// chain's validator registration process.
type Service struct {
	requestingOldLogs       bool
	connectedETH1           bool
	isRunning               bool
	depositContractAddress  common.Address
	processingLock          sync.RWMutex
	ctx                     context.Context
	cancel                  context.CancelFunc
	headerChan              chan *gethTypes.Header
	headTicker              *time.Ticker
	httpEndpoint            string
	stateNotifier           statefeed.Notifier
	httpLogger              bind.ContractFilterer
	eth1DataFetcher         RPCDataFetcher
	rpcClient               RPCClient
	blockCache              *blockCache // cache to store block hash/block height.
	latestEth1Data          *protodb.LatestETH1Data
	depositContractCaller   *contracts.DepositContractCaller
	depositRoot             []byte
	depositTrie             *trieutil.SparseMerkleTrie
	chainStartData          *protodb.ChainStartData
	beaconDB                db.HeadAccessDatabase // Circular dep if using HeadFetcher.
	depositCache            *depositcache.DepositCache
	lastReceivedMerkleIndex int64 // Keeps track of the last received index to prevent log spam.
	runError                error
	preGenesisState         *stateTrie.BeaconState
}

// Web3ServiceConfig defines a config struct for web3 service to use through its life cycle.
type Web3ServiceConfig struct {
	HTTPEndPoint    string
	DepositContract common.Address
	BeaconDB        db.HeadAccessDatabase
	DepositCache    *depositcache.DepositCache
	StateNotifier   statefeed.Notifier
}

// NewService sets up a new instance with an ethclient when
// given a web3 endpoint as a string in the config.
func NewService(ctx context.Context, config *Web3ServiceConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	_ = cancel // govet fix for lost cancel. Cancel is handled in service.Stop()
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "could not setup deposit trie")
	}
	genState, err := state.EmptyGenesisState()
	if err != nil {
		return nil, errors.Wrap(err, "could not setup genesis state")
	}

	s := &Service{
		ctx:          ctx,
		cancel:       cancel,
		headerChan:   make(chan *gethTypes.Header),
		httpEndpoint: config.HTTPEndPoint,
		latestEth1Data: &protodb.LatestETH1Data{
			BlockHeight:        0,
			BlockTime:          0,
			BlockHash:          []byte{},
			LastRequestedBlock: 0,
		},
		blockCache:             newBlockCache(),
		depositContractAddress: config.DepositContract,
		stateNotifier:          config.StateNotifier,
		depositTrie:            depositTrie,
		chainStartData: &protodb.ChainStartData{
			Eth1Data:           &ethpb.Eth1Data{},
			ChainstartDeposits: make([]*ethpb.Deposit, 0),
		},
		beaconDB:                config.BeaconDB,
		depositCache:            config.DepositCache,
		lastReceivedMerkleIndex: -1,
		preGenesisState:         genState,
		headTicker:              time.NewTicker(time.Duration(params.BeaconConfig().SecondsPerETH1Block) * time.Second),
	}

	eth1Data, err := config.BeaconDB.PowchainData(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to retrieve eth1 data")
	}
	if eth1Data != nil {
		s.depositTrie = trieutil.CreateTrieFromProto(eth1Data.Trie)
		s.chainStartData = eth1Data.ChainstartData
		if !reflect.ValueOf(eth1Data.BeaconState).IsZero() {
			s.preGenesisState, err = stateTrie.InitializeFromProto(eth1Data.BeaconState)
			if err != nil {
				return nil, errors.Wrap(err, "Could not initialize state trie")
			}
		}
		s.latestEth1Data = eth1Data.CurrentEth1Data
		s.lastReceivedMerkleIndex = int64(len(s.depositTrie.Items()) - 1)
		if err := s.initDepositCaches(ctx, eth1Data.DepositContainers); err != nil {
			return nil, errors.Wrap(err, "could not initialize caches")
		}
	}
	return s, nil
}

// Start a web3 service's main event loop.
func (s *Service) Start() {
	go func() {
		s.waitForConnection()
		s.run(s.ctx.Done())
	}()
}

// Stop the web3 service's main event loop and associated goroutines.
func (s *Service) Stop() error {
	if s.cancel != nil {
		defer s.cancel()
	}
	if s.headerChan != nil {
		defer close(s.headerChan)
	}
	return nil
}

// ChainStartDeposits returns a slice of validator deposit data processed
// by the deposit contract and cached in the powchain service.
func (s *Service) ChainStartDeposits() []*ethpb.Deposit {
	return s.chainStartData.ChainstartDeposits
}

// ClearPreGenesisData clears out the stored chainstart deposits and beacon state.
func (s *Service) ClearPreGenesisData() {
	s.chainStartData.ChainstartDeposits = []*ethpb.Deposit{}
	s.preGenesisState = &stateTrie.BeaconState{}
}

// ChainStartEth1Data returns the eth1 data at chainstart.
func (s *Service) ChainStartEth1Data() *ethpb.Eth1Data {
	return s.chainStartData.Eth1Data
}

// PreGenesisState returns a state that contains
// pre-chainstart deposits.
func (s *Service) PreGenesisState() *stateTrie.BeaconState {
	return s.preGenesisState
}

// Status is service health checks. Return nil or error.
func (s *Service) Status() error {
	// Service don't start
	if !s.isRunning {
		return nil
	}
	// get error from run function
	if s.runError != nil {
		return s.runError
	}
	return nil
}

// IsConnectedToETH1 checks if the beacon node is connected to a ETH1 Node.
func (s *Service) IsConnectedToETH1() bool {
	return s.connectedETH1
}

// DepositRoot returns the Merkle root of the latest deposit trie
// from the ETH1.0 deposit contract.
func (s *Service) DepositRoot() [32]byte {
	return s.depositTrie.Root()
}

// DepositTrie returns the sparse Merkle trie used for storing
// deposits from the ETH1.0 deposit contract.
func (s *Service) DepositTrie() *trieutil.SparseMerkleTrie {
	return s.depositTrie
}

// LatestBlockHeight in the ETH1.0 chain.
func (s *Service) LatestBlockHeight() *big.Int {
	return big.NewInt(int64(s.latestEth1Data.BlockHeight))
}

// LatestBlockHash in the ETH1.0 chain.
func (s *Service) LatestBlockHash() common.Hash {
	return bytesutil.ToBytes32(s.latestEth1Data.BlockHash)
}

// AreAllDepositsProcessed determines if all the logs from the deposit contract
// are processed.
func (s *Service) AreAllDepositsProcessed() (bool, error) {
	s.processingLock.RLock()
	defer s.processingLock.RUnlock()
	countByte, err := s.depositContractCaller.GetDepositCount(&bind.CallOpts{})
	if err != nil {
		return false, errors.Wrap(err, "could not get deposit count")
	}
	count := bytesutil.FromBytes8(countByte)
	deposits := s.depositCache.AllDeposits(context.TODO(), nil)
	if count != uint64(len(deposits)) {
		return false, nil
	}
	return true, nil
}

// refers to the latest eth1 block which follows the condition: eth1_timestamp +
// SECONDS_PER_ETH1_BLOCK * ETH1_FOLLOW_DISTANCE <= current_unix_time
func (s *Service) followBlockHeight(ctx context.Context) (uint64, error) {
	latestValidBlock := uint64(0)
	if s.latestEth1Data.BlockHeight > params.BeaconConfig().Eth1FollowDistance {
		latestValidBlock = s.latestEth1Data.BlockHeight - params.BeaconConfig().Eth1FollowDistance
	}
	blockTime, err := s.BlockTimeByHeight(ctx, big.NewInt(int64(latestValidBlock)))
	if err != nil {
		return 0, err
	}
	followTime := func(t uint64) uint64 {
		return t + params.BeaconConfig().Eth1FollowDistance*params.BeaconConfig().SecondsPerETH1Block
	}
	for followTime(blockTime) > s.latestEth1Data.BlockTime && latestValidBlock > 0 {
		// reduce block height to get eth1 block which
		// fulfills stated condition
		latestValidBlock--
		blockTime, err = s.BlockTimeByHeight(ctx, big.NewInt(int64(latestValidBlock)))
		if err != nil {
			return 0, err
		}
	}
	return latestValidBlock, nil
}

func (s *Service) connectToPowChain() error {
	httpClient, rpcClient, err := s.dialETH1Nodes()
	if err != nil {
		return errors.Wrap(err, "could not dial eth1 nodes")
	}

	depositContractCaller, err := contracts.NewDepositContractCaller(s.depositContractAddress, httpClient)
	if err != nil {
		return errors.Wrap(err, "could not create deposit contract caller")
	}

	s.initializeConnection(httpClient, rpcClient, depositContractCaller)
	return nil
}

func (s *Service) dialETH1Nodes() (*ethclient.Client, *gethRPC.Client, error) {
	httpRPCClient, err := gethRPC.Dial(s.httpEndpoint)
	if err != nil {
		return nil, nil, err
	}
	httpClient := ethclient.NewClient(httpRPCClient)

	// Make a simple call to ensure we are actually connected to a working node.
	cID, err := httpClient.ChainID(s.ctx)
	if err != nil {
		return nil, nil, err
	}
	nID, err := httpClient.NetworkID(s.ctx)
	if err != nil {
		return nil, nil, err
	}
	if cID.Uint64() != params.BeaconNetworkConfig().ChainID {
		return nil, nil, fmt.Errorf("eth1 node using incorrect chain id, %d != %d", cID.Uint64(), params.BeaconNetworkConfig().ChainID)
	}
	if nID.Uint64() != params.BeaconNetworkConfig().NetworkID {
		return nil, nil, fmt.Errorf("eth1 node using incorrect network id, %d != %d", nID.Uint64(), params.BeaconNetworkConfig().NetworkID)
	}

	return httpClient, httpRPCClient, nil
}

func (s *Service) initializeConnection(
	httpClient *ethclient.Client,
	rpcClient *gethRPC.Client,
	contractCaller *contracts.DepositContractCaller,
) {
	s.httpLogger = httpClient
	s.eth1DataFetcher = httpClient
	s.depositContractCaller = contractCaller
	s.rpcClient = rpcClient
}

func (s *Service) waitForConnection() {
	errConnect := s.connectToPowChain()
	if errConnect == nil {
		synced, errSynced := s.isEth1NodeSynced()
		// Resume if eth1 node is synced.
		if synced {
			s.connectedETH1 = true
			log.WithFields(logrus.Fields{
				"endpoint": s.httpEndpoint,
			}).Info("Connected to eth1 proof-of-work chain")
			return
		}
		if errSynced != nil {
			log.WithError(errSynced).Error("Could not check sync status of eth1 chain")
		}
	}
	if errConnect != nil {
		log.WithError(errConnect).Error("Could not connect to powchain endpoint")
	}
	// Use a custom logger to only log errors
	// once in  a while.
	logCounter := 0
	errorLogger := func(err error, msg string) {
		if logCounter > logThreshold {
			log.WithError(err).Error(msg)
			logCounter = 0
		}
		logCounter++
	}
	ticker := time.NewTicker(backOffPeriod)
	for {
		select {
		case <-ticker.C:
			errConnect := s.connectToPowChain()
			if errConnect != nil {
				errorLogger(errConnect, "Could not connect to powchain endpoint")
				continue
			}
			synced, errSynced := s.isEth1NodeSynced()
			if errSynced != nil {
				errorLogger(errSynced, "Could not check sync status of eth1 chain")
				continue
			}
			if synced {
				s.connectedETH1 = true
				log.WithFields(logrus.Fields{
					"endpoint": s.httpEndpoint,
				}).Info("Connected to eth1 proof-of-work chain")
				ticker.Stop()
				return
			}
			log.Debug("Eth1 node is currently syncing")
		case <-s.ctx.Done():
			ticker.Stop()
			log.Debug("Received cancelled context,closing existing powchain service")
			return
		}
	}
}

// checks if the eth1 node is healthy and ready to serve before
// fetching data from  it.
func (s *Service) isEth1NodeSynced() (bool, error) {
	syncProg, err := s.eth1DataFetcher.SyncProgress(s.ctx)
	if err != nil {
		return false, err
	}
	return syncProg == nil, nil
}

// Reconnect to eth1 node in case of any failure.
func (s *Service) retryETH1Node(err error) {
	s.runError = err
	s.connectedETH1 = false
	// Back off for a while before
	// resuming dialing the eth1 node.
	time.Sleep(backOffPeriod)
	s.waitForConnection()
	// Reset run error in the event of a successful connection.
	s.runError = nil
}

// initDataFromContract calls the deposit contract and finds the deposit count
// and deposit root.
func (s *Service) initDataFromContract() error {
	root, err := s.depositContractCaller.GetDepositRoot(&bind.CallOpts{})
	if err != nil {
		return errors.Wrap(err, "could not retrieve deposit root")
	}
	s.depositRoot = root[:]
	return nil
}

func (s *Service) initDepositCaches(ctx context.Context, ctrs []*protodb.DepositContainer) error {
	if len(ctrs) == 0 {
		return nil
	}
	s.depositCache.InsertDepositContainers(ctx, ctrs)
	currentState, err := s.beaconDB.HeadState(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head state")
	}
	// do not add to pending cache
	// if no state exists.
	if currentState == nil {
		validDepositsCount.Add(float64(s.preGenesisState.Eth1DepositIndex() + 1))
		return nil
	}
	currIndex := currentState.Eth1DepositIndex()
	validDepositsCount.Add(float64(currIndex + 1))

	// Only add pending deposits if the container slice length
	// is more than the current index in state.
	if uint64(len(ctrs)) > currIndex {
		for _, c := range ctrs[currIndex:] {
			s.depositCache.InsertPendingDeposit(ctx, c.Deposit, c.Eth1BlockHeight, c.Index, bytesutil.ToBytes32(c.DepositRoot))
		}
	}
	return nil
}

// processBlockHeader adds a newly observed eth1 block to the block cache and
// updates the latest blockHeight, blockHash, and blockTime properties of the service.
func (s *Service) processBlockHeader(header *gethTypes.Header) {
	defer safelyHandlePanic()
	blockNumberGauge.Set(float64(header.Number.Int64()))
	s.latestEth1Data.BlockHeight = header.Number.Uint64()
	s.latestEth1Data.BlockHash = header.Hash().Bytes()
	s.latestEth1Data.BlockTime = header.Time
	log.WithFields(logrus.Fields{
		"blockNumber": s.latestEth1Data.BlockHeight,
		"blockHash":   hexutil.Encode(s.latestEth1Data.BlockHash),
	}).Debug("Latest eth1 chain event")

	if err := s.blockCache.AddBlock(gethTypes.NewBlockWithHeader(header)); err != nil {
		s.runError = err
		log.Errorf("Unable to add block data to cache %v", err)
	}
}

// batchRequestHeaders requests the block range specified in the arguments. Instead of requesting
// each block in one call, it batches all requests into a single rpc call.
func (s *Service) batchRequestHeaders(startBlock uint64, endBlock uint64) ([]*gethTypes.Header, error) {
	requestRange := (endBlock - startBlock) + 1
	elems := make([]gethRPC.BatchElem, 0, requestRange)
	headers := make([]*gethTypes.Header, 0, requestRange)
	errors := make([]error, 0, requestRange)
	if requestRange == 0 {
		return headers, nil
	}
	for i := startBlock; i <= endBlock; i++ {
		header := &gethTypes.Header{}
		err := error(nil)
		elems = append(elems, gethRPC.BatchElem{
			Method: "eth_getBlockByNumber",
			Args:   []interface{}{hexutil.EncodeBig(big.NewInt(int64(i))), true},
			Result: header,
			Error:  err,
		})
		headers = append(headers, header)
		errors = append(errors, err)
	}
	ioErr := s.rpcClient.BatchCall(elems)
	if ioErr != nil {
		return nil, ioErr
	}
	for _, e := range errors {
		if e != nil {
			return nil, e
		}
	}
	for _, h := range headers {
		if h != nil {
			if err := s.blockCache.AddBlock(gethTypes.NewBlockWithHeader(h)); err != nil {
				return nil, err
			}
		}
	}
	return headers, nil
}

// safelyHandleHeader will recover and log any panic that occurs from the
// block
func safelyHandlePanic() {
	if r := recover(); r != nil {
		log.WithFields(logrus.Fields{
			"r": r,
		}).Error("Panicked when handling data from ETH 1.0 Chain! Recovering...")

		debug.PrintStack()
	}
}

func (s *Service) handleETH1FollowDistance() {
	defer safelyHandlePanic()

	// use a 5 minutes timeout for block time, because the max mining time is 278 sec (block 7208027)
	// (analyzed the time of the block from 2018-09-01 to 2019-02-13)
	fiveMinutesTimeout := roughtime.Now().Add(-5 * time.Minute)
	// check that web3 client is syncing
	if time.Unix(int64(s.latestEth1Data.BlockTime), 0).Before(fiveMinutesTimeout) {
		log.Warn("eth1 client is not syncing")
	}
	if !s.chainStartData.Chainstarted {
		if err := s.checkBlockNumberForChainStart(context.Background(), big.NewInt(int64(s.latestEth1Data.LastRequestedBlock))); err != nil {
			s.runError = err
			log.Error(err)
			return
		}
	}
	// If the last requested block has not changed,
	// we do not request batched logs as this means there are no new
	// logs for the powchain service to process.
	if s.latestEth1Data.LastRequestedBlock == s.latestEth1Data.BlockHeight {
		return
	}
	if err := s.requestBatchedLogs(context.Background()); err != nil {
		s.runError = err
		log.Error(err)
		return
	}
	// Reset the Status.
	if s.runError != nil {
		s.runError = nil
	}
}

func (s *Service) initPOWService() {

	// Run in a select loop to retry in the event of any failures.
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			err := s.initDataFromContract()
			if err != nil {
				log.Errorf("Unable to retrieve data from deposit contract %v", err)
				s.retryETH1Node(err)
				continue
			}

			header, err := s.eth1DataFetcher.HeaderByNumber(context.Background(), nil)
			if err != nil {
				log.Errorf("Unable to retrieve latest ETH1.0 chain header: %v", err)
				s.retryETH1Node(err)
				continue
			}

			s.latestEth1Data.BlockHeight = header.Number.Uint64()
			s.latestEth1Data.BlockHash = header.Hash().Bytes()
			s.latestEth1Data.BlockTime = header.Time

			if err := s.processPastLogs(context.Background()); err != nil {
				log.Errorf("Unable to process past logs %v", err)
				s.retryETH1Node(err)
				continue
			}
			return
		}
	}
}

// run subscribes to all the services for the ETH1.0 chain.
func (s *Service) run(done <-chan struct{}) {
	s.isRunning = true
	s.runError = nil

	s.initPOWService()

	for {
		select {
		case <-done:
			s.isRunning = false
			s.runError = nil
			s.connectedETH1 = false
			log.Debug("Context closed, exiting goroutine")
			return
		case <-s.headTicker.C:
			head, err := s.eth1DataFetcher.HeaderByNumber(s.ctx, nil)
			if err != nil {
				log.WithError(err).Debug("Could not fetch latest eth1 header")
				s.retryETH1Node(err)
				continue
			}
			s.processBlockHeader(head)
			s.handleETH1FollowDistance()
		}
	}
}
