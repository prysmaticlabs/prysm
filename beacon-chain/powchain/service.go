// Package powchain defines the services that interact with the ETH1.0 of Ethereum.
package powchain

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"

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
	"github.com/prysmaticlabs/prysm/shared/hashutil"
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
	blockFetcher            POWBlockFetcher
	blockHeight             *big.Int    // the latest ETH1.0 chain blockHeight.
	blockHash               common.Hash // the latest ETH1.0 chain blockHash.
	depositContractCaller   *contracts.DepositContractCaller
	depositRoot             []byte
	depositTrie             *trieutil.DepositTrie
	chainStartDeposits      []*pb.Deposit
	chainStarted            bool
	beaconDB                *db.BeaconDB
	lastReceivedMerkleIndex int64 // Keeps track of the last received index to prevent log spam.
	chainStartDelay         uint64
	lastRequestedBlock      *big.Int
}

// Web3ServiceConfig defines a config struct for web3 service to use through its life cycle.
type Web3ServiceConfig struct {
	Endpoint        string
	DepositContract common.Address
	Client          Client
	Reader          Reader
	Logger          bind.ContractFilterer
	BlockFetcher    POWBlockFetcher
	ContractBackend bind.ContractBackend
	BeaconDB        *db.BeaconDB
	ChainStartDelay uint64
}

var (
	depositEventSignature    = []byte("Deposit(bytes32,bytes,bytes,bytes32[32])")
	chainStartEventSignature = []byte("ChainStart(bytes32,bytes)")
)

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
	return &Web3Service{
		ctx:                     ctx,
		cancel:                  cancel,
		headerChan:              make(chan *gethTypes.Header),
		endpoint:                config.Endpoint,
		blockHeight:             nil,
		blockHash:               common.BytesToHash([]byte{}),
		depositContractAddress:  config.DepositContract,
		chainStartFeed:          new(event.Feed),
		client:                  config.Client,
		reader:                  config.Reader,
		logger:                  config.Logger,
		blockFetcher:            config.BlockFetcher,
		depositContractCaller:   depositContractCaller,
		chainStartDeposits:      []*pb.Deposit{},
		beaconDB:                config.BeaconDB,
		lastReceivedMerkleIndex: -1,
		chainStartDelay:         config.ChainStartDelay,
		lastRequestedBlock:      big.NewInt(0),
	}, nil
}

// Start a web3 service's main event loop.
func (w *Web3Service) Start() {
	log.WithFields(logrus.Fields{
		"endpoint": w.endpoint,
	}).Info("Starting service")
	go w.run(w.ctx.Done())

	if w.chainStartDelay > 0 {
		go w.runDelayTimer(w.ctx.Done())
	}
}

// Stop the web3 service's main event loop and associated goroutines.
func (w *Web3Service) Stop() error {
	defer w.cancel()
	defer close(w.headerChan)
	log.Info("Stopping service")
	return nil
}

// ChainStartFeed returns a feed that is written to
// whenever the deposit contract fires a ChainStart log.
func (w *Web3Service) ChainStartFeed() *event.Feed {
	return w.chainStartFeed
}

// ChainStartDeposits returns a slice of validator deposits processed
// by the deposit contract and cached in the powchain service.
func (w *Web3Service) ChainStartDeposits() []*pb.Deposit {
	return w.chainStartDeposits
}

// Status always returns nil.
// TODO(1204): Add service health checks.
func (w *Web3Service) Status() error {
	return nil
}

// DepositRoot returns the Merkle root of the latest deposit trie
// from the ETH1.0 deposit contract.
func (w *Web3Service) DepositRoot() [32]byte {
	return w.depositTrie.Root()
}

// LatestBlockHeight in the ETH1.0 chain.
func (w *Web3Service) LatestBlockHeight() *big.Int {
	return w.blockHeight
}

// LatestBlockHash in the ETH1.0 chain.
func (w *Web3Service) LatestBlockHash() common.Hash {
	return w.blockHash
}

// BlockExists returns true if the block exists, it's height and any possible error encountered.
func (w *Web3Service) BlockExists(hash common.Hash) (bool, *big.Int, error) {
	block, err := w.blockFetcher.BlockByHash(w.ctx, hash)
	if err != nil {
		return false, big.NewInt(0), fmt.Errorf("could not query block with given hash: %v", err)
	}

	return true, block.Number(), nil
}

// BlockHashByHeight returns the block hash of the block at the given height.
func (w *Web3Service) BlockHashByHeight(height *big.Int) (common.Hash, error) {
	block, err := w.blockFetcher.BlockByNumber(w.ctx, height)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not query block with given height: %v", err)
	}

	return block.Hash(), nil
}

// Client for interacting with the ETH1.0 chain.
func (w *Web3Service) Client() Client {
	return w.client
}

// HasChainStartLogOccurred queries all logs in the deposit contract to verify
// if ChainStart has occurred. If so, it returns true alongside the ChainStart timestamp.
func (w *Web3Service) HasChainStartLogOccurred() (bool, uint64, error) {
	genesisTime, err := w.depositContractCaller.GenesisTime(&bind.CallOpts{})
	if err != nil {
		return false, 0, fmt.Errorf("could not query contract to verify chain started: %v", err)
	}
	// If chain has not yet started, the result will be an empty byte slice.
	if bytes.Equal(genesisTime, []byte{}) {
		return false, 0, nil
	}
	timestamp := binary.LittleEndian.Uint64(genesisTime)
	if uint64(time.Now().Unix()) < timestamp {
		return false, 0, fmt.Errorf("invalid timestamp from log expected %d > %d", time.Now().Unix(), timestamp)
	}
	return true, timestamp, nil
}

// ProcessLog is the main method which handles the processing of all
// logs from the deposit contract on the ETH1.0 chain.
func (w *Web3Service) ProcessLog(depositLog gethTypes.Log) {
	// Process logs according to their event signature.
	if depositLog.Topics[0] == hashutil.Hash(depositEventSignature) {
		w.ProcessDepositLog(depositLog)
		return
	}
	if depositLog.Topics[0] == hashutil.Hash(chainStartEventSignature) && !w.chainStarted {
		w.ProcessChainStartLog(depositLog)
		return
	}
	log.Debugf("Log is not of a valid event signature %#x", depositLog.Topics[0])
}

// ProcessDepositLog processes the log which had been received from
// the ETH1.0 chain by trying to ascertain which participant deposited
// in the contract.
func (w *Web3Service) ProcessDepositLog(depositLog gethTypes.Log) {
	merkleRoot, depositData, merkleTreeIndex, _, err := contracts.UnpackDepositLogData(depositLog.Data)
	if err != nil {
		log.Errorf("Could not unpack log %v", err)
		return
	}
	// If we have already seen this Merkle index, skip processing the log.
	// This can happen sometimes when we receive the same log twice from the
	// ETH1.0 network, and prevents us from updating our trie
	// with the same log twice, causing an inconsistent state root.
	index := binary.LittleEndian.Uint64(merkleTreeIndex)
	if int64(index) <= w.lastReceivedMerkleIndex {
		return
	}
	if err := w.saveInTrie(depositData, merkleRoot); err != nil {
		log.Errorf("Could not save in trie %v", err)
		return
	}
	w.lastReceivedMerkleIndex = int64(index)
	depositInput, err := helpers.DecodeDepositInput(depositData)
	if err != nil {
		log.Errorf("Could not decode deposit input  %v", err)
		return
	}
	deposit := &pb.Deposit{
		DepositData: depositData,
	}
	// If chain has not started, do not update the merkle trie
	if !w.chainStarted {
		w.chainStartDeposits = append(w.chainStartDeposits, deposit)
	} else {
		w.beaconDB.InsertPendingDeposit(w.ctx, deposit, big.NewInt(int64(depositLog.BlockNumber)))
	}
	log.WithFields(logrus.Fields{
		"publicKey":       fmt.Sprintf("%#x", depositInput.Pubkey),
		"merkleTreeIndex": index,
	}).Info("Validator registered in deposit contract")
	validDepositsCount.Inc()
}

// ProcessChainStartLog processes the log which had been received from
// the ETH1.0 chain by trying to determine when to start the beacon chain.
func (w *Web3Service) ProcessChainStartLog(depositLog gethTypes.Log) {
	chainStartCount.Inc()
	receiptRoot, timestampData, err := contracts.UnpackChainStartLogData(depositLog.Data)
	if err != nil {
		log.Errorf("Unable to unpack ChainStart log data %v", err)
		return
	}
	if w.depositTrie.Root() != receiptRoot {
		log.Errorf("Receipt root from log doesn't match the root saved in memory,"+
			" want %#x but got %#x", w.depositTrie.Root(), receiptRoot)
		return
	}

	timestamp := binary.LittleEndian.Uint64(timestampData)
	if uint64(time.Now().Unix()) < timestamp {
		log.Errorf("Invalid timestamp from log expected %d > %d", time.Now().Unix(), timestamp)
	}
	w.chainStarted = true
	chainStartTime := time.Unix(int64(timestamp), 0)
	log.WithFields(logrus.Fields{
		"ChainStartTime": chainStartTime,
	}).Info("Minimum number of validators reached for beacon-chain to start")
	w.chainStartFeed.Send(chainStartTime)
}

func (w *Web3Service) runDelayTimer(done <-chan struct{}) {
	timer := time.NewTimer(time.Duration(w.chainStartDelay) * time.Second)

	for {
		select {
		case <-done:
			log.Debug("ETH1.0 chain service context closed, exiting goroutine")
			timer.Stop()
			return
		case currentTime := <-timer.C:

			w.chainStarted = true
			log.WithFields(logrus.Fields{
				"ChainStartTime": currentTime.Unix(),
			}).Info("Minimum number of validators reached for beacon-chain to start")
			w.chainStartFeed.Send(currentTime)
			timer.Stop()
			return
		}
	}
}

// run subscribes to all the services for the ETH1.0 chain.
func (w *Web3Service) run(done <-chan struct{}) {
	if err := w.initDataFromContract(); err != nil {
		log.Errorf("Unable to retrieve data from deposit contract %v", err)
		return
	}

	headSub, err := w.reader.SubscribeNewHead(w.ctx, w.headerChan)
	if err != nil {
		log.Errorf("Unable to subscribe to incoming ETH1.0 chain headers: %v", err)
		return
	}

	header, err := w.blockFetcher.HeaderByNumber(w.ctx, nil)
	if err != nil {
		log.Errorf("Unable to retrieve latest ETH1.0 chain header: %v", err)
		return
	}

	w.blockHeight = header.Number
	w.blockHash = header.Hash()

	// Only process logs if the chain start delay flag is not enabled.
	if w.chainStartDelay == 0 {
		if err := w.processPastLogs(); err != nil {
			log.Errorf("Unable to process past logs %v", err)
			return
		}
	}

	ticker := time.NewTicker(1 * time.Second)
	defer headSub.Unsubscribe()
	defer ticker.Stop()

	for {
		select {
		case <-done:
			log.Debug("ETH1.0 chain service context closed, exiting goroutine")
			return
		case <-headSub.Err():
			log.Debug("Unsubscribed to head events, exiting goroutine")
			return
		case header := <-w.headerChan:
			blockNumberGauge.Set(float64(header.Number.Int64()))
			w.blockHeight = header.Number
			w.blockHash = header.Hash()
			log.WithFields(logrus.Fields{
				"blockNumber": w.blockHeight,
				"blockHash":   w.blockHash.Hex(),
			}).Debug("Latest web3 chain event")
		case <-ticker.C:
			if w.lastRequestedBlock.Cmp(w.blockHeight) == 0 {
				continue
			}
			if err := w.requestBatchedLogs(); err != nil {
				log.Error(err)
			}

		}
	}
}

// initDataFromContract calls the deposit contract and finds the deposit count
// and deposit root.
func (w *Web3Service) initDataFromContract() error {
	root, err := w.depositContractCaller.GetDepositRoot(&bind.CallOpts{})
	if err != nil {
		return fmt.Errorf("could not retrieve deposit root %v", err)
	}
	w.depositRoot = root[:]
	w.depositTrie = trieutil.NewDepositTrie()
	return nil
}

// saveInTrie saves in the in-memory deposit trie.
func (w *Web3Service) saveInTrie(depositData []byte, merkleRoot common.Hash) error {
	w.depositTrie.UpdateDepositTrie(depositData)
	if w.depositTrie.Root() != merkleRoot {
		return errors.New("saved root in trie is unequal to root received from log")
	}
	return nil
}

// processPastLogs processes all the past logs from the deposit contract and
// updates the deposit trie with the data from each individual log.
func (w *Web3Service) processPastLogs() error {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			w.depositContractAddress,
		},
	}

	logs, err := w.logger.FilterLogs(w.ctx, query)
	if err != nil {
		return err
	}

	for _, log := range logs {
		w.ProcessLog(log)
	}
	w.lastRequestedBlock.Set(w.blockHeight)
	return nil
}

// requestBatchedLogs requests and processes all the logs from the period
// last polled to now.
func (w *Web3Service) requestBatchedLogs() error {

	// We request for the nth block behind the current head, in order to have
	// stabilised logs when we retrieve it from the 1.0 chain.
	requestedBlock := big.NewInt(0).Sub(w.blockHeight, big.NewInt(params.BeaconConfig().LogBlockDelay))
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			w.depositContractAddress,
		},
		FromBlock: w.lastRequestedBlock.Add(w.lastRequestedBlock, big.NewInt(1)),
		ToBlock:   requestedBlock,
	}
	logs, err := w.logger.FilterLogs(w.ctx, query)
	if err != nil {
		return err
	}

	// Only process log slices which are larger than zero.
	if len(logs) > 0 {
		log.Debug("Processing Batched Logs")
		for _, log := range logs {
			w.ProcessLog(log)
		}
	}

	w.lastRequestedBlock.Set(requestedBlock)
	return nil
}
