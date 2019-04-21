// Package powchain defines the services that interact with the ETH1.0 of Ethereum.
package powchain

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "powchain")

var (
	validDepositsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powchain_valid_deposits_received",
		Help: "The number of valid deposits received in the deposit contract",
	})
	chainStartCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powchain_chainstart_logs",
		Help: "The number of chainstart logs received from the deposit contract",
	})
	blockNumberGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powchain_block_number",
		Help: "The current block number in the proof-of-work chain",
	})
)

// Reader defines a struct that can fetch latest header events from a web3 endpoint.
type Reader interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error)
}

// POWBlockFetcher defines a struct that can retrieve mainchain blocks.
type POWBlockFetcher interface {
	BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*gethTypes.Block, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*gethTypes.Header, error)
}

// Client defines a struct that combines all relevant ETH1.0 mainchain interactions required
// by the beacon chain node.
type Client interface {
	Reader
	POWBlockFetcher
	bind.ContractFilterer
	bind.ContractCaller
}

// Web3Service fetches important information about the canonical
// Ethereum ETH1.0 chain via a web3 endpoint using an ethclient. The Random
// Beacon Chain requires synchronization with the ETH1.0 chain's current
// blockhash, block number, and access to logs within the
// Validator Registration Contract on the ETH1.0 chain to kick off the beacon
// chain's validator registration process.
type Web3Service struct {
	ctx                     context.Context
	cancel                  context.CancelFunc
	client                  Client
	headerChan              chan *gethTypes.Header
	endpoint                string
	depositContractAddress  common.Address
	chainStartFeed          *event.Feed
	reader                  Reader
	logger                  bind.ContractFilterer
	httpLogger              bind.ContractFilterer
	blockFetcher            POWBlockFetcher
	blockHeight             *big.Int    // the latest ETH1.0 chain blockHeight.
	blockHash               common.Hash // the latest ETH1.0 chain blockHash.
	blockTime               time.Time   // the latest ETH1.0 chain blockTime.
	blockCache              *blockCache // cache to store block hash/block height.
	depositContractCaller   *contracts.DepositContractCaller
	depositRoot             []byte
	depositTrie             *trieutil.MerkleTrie
	chainStartDeposits      [][]byte
	chainStarted            bool
	chainStartETH1Data      *pb.Eth1Data
	beaconDB                *db.BeaconDB
	lastReceivedMerkleIndex int64 // Keeps track of the last received index to prevent log spam.
	isRunning               bool
	runError                error
	lastRequestedBlock      *big.Int
}

// Web3ServiceConfig defines a config struct for web3 service to use through its life cycle.
type Web3ServiceConfig struct {
	Endpoint        string
	DepositContract common.Address
	Client          Client
	Reader          Reader
	Logger          bind.ContractFilterer
	HTTPLogger      bind.ContractFilterer
	BlockFetcher    POWBlockFetcher
	ContractBackend bind.ContractBackend
	BeaconDB        *db.BeaconDB
}

// NewWeb3Service sets up a new instance with an ethclient when
// given a web3 endpoint as a string in the config.
func NewWeb3Service(ctx context.Context, config *Web3ServiceConfig) (*Web3Service, error) {
	if !strings.HasPrefix(config.Endpoint, "ws") && !strings.HasPrefix(config.Endpoint, "ipc") {
		return nil, fmt.Errorf(
			"powchain service requires either an IPC or WebSocket endpoint, provided %s",
			config.Endpoint,
		)
	}

	depositContractCaller, err := contracts.NewDepositContractCaller(config.DepositContract, config.ContractBackend)
	if err != nil {
		return nil, fmt.Errorf("could not create deposit contract caller %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	depositTrie, err := trieutil.GenerateTrieFromItems([][]byte{{}}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not setup deposit trie: %v", err)
	}
	return &Web3Service{
		ctx:                     ctx,
		cancel:                  cancel,
		headerChan:              make(chan *gethTypes.Header),
		endpoint:                config.Endpoint,
		blockHeight:             nil,
		blockHash:               common.BytesToHash([]byte{}),
		blockCache:              newBlockCache(),
		depositContractAddress:  config.DepositContract,
		chainStartFeed:          new(event.Feed),
		client:                  config.Client,
		depositTrie:             depositTrie,
		reader:                  config.Reader,
		logger:                  config.Logger,
		httpLogger:              config.HTTPLogger,
		blockFetcher:            config.BlockFetcher,
		depositContractCaller:   depositContractCaller,
		chainStartDeposits:      [][]byte{},
		beaconDB:                config.BeaconDB,
		lastReceivedMerkleIndex: -1,
		lastRequestedBlock:      big.NewInt(0),
		chainStartETH1Data:      &pb.Eth1Data{},
	}, nil
}

// Start a web3 service's main event loop.
func (w *Web3Service) Start() {
	log.WithFields(logrus.Fields{
		"endpoint": w.endpoint,
	}).Info("Starting service")
	go w.run(w.ctx.Done())
}

// Stop the web3 service's main event loop and associated goroutines.
func (w *Web3Service) Stop() error {
	if w.cancel != nil {
		defer w.cancel()
	}
	if w.headerChan != nil {
		defer close(w.headerChan)
	}
	log.Info("Stopping service")
	return nil
}

// ChainStartFeed returns a feed that is written to
// whenever the deposit contract fires a ChainStart log.
func (w *Web3Service) ChainStartFeed() *event.Feed {
	return w.chainStartFeed
}

// ChainStartDeposits returns a slice of validator deposit data processed
// by the deposit contract and cached in the powchain service.
func (w *Web3Service) ChainStartDeposits() [][]byte {
	return w.chainStartDeposits
}

// ChainStartETH1Data returns the eth1 data at chainstart.
func (w *Web3Service) ChainStartETH1Data() *pb.Eth1Data {
	return w.chainStartETH1Data
}

// Status is service health checks. Return nil or error.
func (w *Web3Service) Status() error {
	// Web3Service don't start
	if !w.isRunning {
		return nil
	}
	// get error from run function
	if w.runError != nil {
		return w.runError
	}
	// use a 5 minutes timeout for block time, because the max mining time is 278 sec (block 7208027)
	// (analyzed the time of the block from 2018-09-01 to 2019-02-13)
	fiveMinutesTimeout := time.Now().Add(-5 * time.Minute)
	// check that web3 client is syncing
	if w.blockTime.Before(fiveMinutesTimeout) {
		return errors.New("eth1 client is not syncing")
	}
	return nil
}

// DepositRoot returns the Merkle root of the latest deposit trie
// from the ETH1.0 deposit contract.
func (w *Web3Service) DepositRoot() [32]byte {
	return w.depositTrie.Root()
}

// DepositTrie returns the sparse Merkle trie used for storing
// deposits from the ETH1.0 deposit contract.
func (w *Web3Service) DepositTrie() *trieutil.MerkleTrie {
	return w.depositTrie
}

// LatestBlockHeight in the ETH1.0 chain.
func (w *Web3Service) LatestBlockHeight() *big.Int {
	return w.blockHeight
}

// LatestBlockHash in the ETH1.0 chain.
func (w *Web3Service) LatestBlockHash() common.Hash {
	return w.blockHash
}

// Client for interacting with the ETH1.0 chain.
func (w *Web3Service) Client() Client {
	return w.client
}

// initDataFromContract calls the deposit contract and finds the deposit count
// and deposit root.
func (w *Web3Service) initDataFromContract() error {
	root, err := w.depositContractCaller.GetDepositRoot(&bind.CallOpts{})
	if err != nil {
		return fmt.Errorf("could not retrieve deposit root %v", err)
	}
	w.depositRoot = root[:]
	return nil
}

// processSubscribedHeaders adds a newly observed eth1 block to the block cache and
// updates the latest blockHeight, blockHash, and blockTime properties of the service.
func (w *Web3Service) processSubscribedHeaders(header *gethTypes.Header) {
	defer safelyHandlePanic()
	blockNumberGauge.Set(float64(header.Number.Int64()))
	w.blockHeight = header.Number
	w.blockHash = header.Hash()
	w.blockTime = time.Unix(int64(header.Time), 0)
	log.WithFields(logrus.Fields{
		"blockNumber": w.blockHeight,
		"blockHash":   w.blockHash.Hex(),
	}).Debug("Latest eth1 chain event")

	if err := w.blockCache.AddBlock(gethTypes.NewBlockWithHeader(header)); err != nil {
		w.runError = err
		log.Errorf("Unable to add block data to cache %v", err)
	}
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

func (w *Web3Service) handleDelayTicker() {
	defer safelyHandlePanic()
	// If the last requested block has not changed,
	// we do not request batched logs as this means there are no new
	// logs for the powchain service to process.
	if w.lastRequestedBlock.Cmp(w.blockHeight) == 0 {
		return
	}
	if err := w.requestBatchedLogs(); err != nil {
		w.runError = err
		log.Error(err)
	}
}

// run subscribes to all the services for the ETH1.0 chain.
func (w *Web3Service) run(done <-chan struct{}) {
	w.isRunning = true
	w.runError = nil
	if err := w.initDataFromContract(); err != nil {
		log.Errorf("Unable to retrieve data from deposit contract %v", err)
		return
	}

	headSub, err := w.reader.SubscribeNewHead(w.ctx, w.headerChan)
	if err != nil {
		log.Errorf("Unable to subscribe to incoming ETH1.0 chain headers: %v", err)
		w.runError = err
		return
	}

	header, err := w.blockFetcher.HeaderByNumber(w.ctx, nil)
	if err != nil {
		log.Errorf("Unable to retrieve latest ETH1.0 chain header: %v", err)
		w.runError = err
		return
	}

	w.blockHeight = header.Number
	w.blockHash = header.Hash()

	if err := w.processPastLogs(); err != nil {
		log.Errorf("Unable to process past logs %v", err)
		w.runError = err
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer headSub.Unsubscribe()
	defer ticker.Stop()

	for {
		select {
		case <-done:
			w.isRunning = false
			w.runError = nil
			log.Debug("ETH1.0 chain service context closed, exiting goroutine")
			return
		case w.runError = <-headSub.Err():
			log.Debugf("Unsubscribed to head events, exiting goroutine: %v", w.runError)
			return
		case header := <-w.headerChan:
			w.processSubscribedHeaders(header)
		case <-ticker.C:
			w.handleDelayTicker()
		}
	}
}
