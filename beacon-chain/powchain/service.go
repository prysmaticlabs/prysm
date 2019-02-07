// Package powchain defines the services that interact with the ETH1.0 of Ethereum.
package powchain

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "powchain")

var (
	validDepositsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powchain_valid_deposits_received",
		Help: "The number of valid deposits received in the deposit contract",
	})
	totalDepositsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powchain_deposit_logs",
		Help: "The total number of deposits received in the deposit contract",
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
	ctx                    context.Context
	cancel                 context.CancelFunc
	client                 Client
	headerChan             chan *gethTypes.Header
	logChan                chan gethTypes.Log
	endpoint               string
	depositContractAddress common.Address
	chainStartFeed         *event.Feed
	reader                 Reader
	logger                 bind.ContractFilterer
	blockNumber            *big.Int    // the latest ETH1.0 chain blockNumber.
	blockHash              common.Hash // the latest ETH1.0 chain blockHash.
	vrcCaller              *contracts.DepositContractCaller
	depositRoot            []byte
	depositTrie            *trieutil.DepositTrie
	chainStartDeposits     []*pb.Deposit
	chainStarted           bool
	beaconDB               *db.BeaconDB
}

// Web3ServiceConfig defines a config struct for web3 service to use through its life cycle.
type Web3ServiceConfig struct {
	Endpoint        string
	DepositContract common.Address
	Client          Client
	Reader          Reader
	Logger          bind.ContractFilterer
	ContractBackend bind.ContractBackend
	BeaconDB        *db.BeaconDB
}

var (
	depositEventSignature    = []byte("Deposit(bytes32,bytes,bytes)")
	chainStartEventSignature = []byte("ChainStart(bytes32,bytes)")
)

// NewWeb3Service sets up a new instance with an ethclient when
// given a web3 endpoint as a string in the config.
func NewWeb3Service(ctx context.Context, config *Web3ServiceConfig) (*Web3Service, error) {
	if !strings.HasPrefix(config.Endpoint, "ws") && !strings.HasPrefix(config.Endpoint, "ipc") {
		return nil, fmt.Errorf(
			"web3service requires either an IPC or WebSocket endpoint, provided %s",
			config.Endpoint,
		)
	}

	vrcCaller, err := contracts.NewDepositContractCaller(config.DepositContract, config.ContractBackend)
	if err != nil {
		return nil, fmt.Errorf("could not create VRC caller %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	return &Web3Service{
		ctx:                    ctx,
		cancel:                 cancel,
		headerChan:             make(chan *gethTypes.Header),
		logChan:                make(chan gethTypes.Log),
		endpoint:               config.Endpoint,
		blockNumber:            nil,
		blockHash:              common.BytesToHash([]byte{}),
		depositContractAddress: config.DepositContract,
		chainStartFeed:         new(event.Feed),
		client:                 config.Client,
		reader:                 config.Reader,
		logger:                 config.Logger,
		vrcCaller:              vrcCaller,
		chainStartDeposits:     []*pb.Deposit{},
		beaconDB:               config.BeaconDB,
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

// LatestBlockNumber in the ETH1.0 chain.
func (w *Web3Service) LatestBlockNumber() *big.Int {
	return w.blockNumber
}

// LatestBlockHash in the ETH1.0 chain.
func (w *Web3Service) LatestBlockHash() common.Hash {
	return w.blockHash
}

// Client for interacting with the ETH1.0 chain.
func (w *Web3Service) Client() Client {
	return w.client
}

// HasChainStartLogOccurred queries all logs in the deposit contract to verify
// if ChainStart has occurred. If so, it returns true alongside the ChainStart timestamp.
func (w *Web3Service) HasChainStartLogOccurred() (bool, uint64, error) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			w.depositContractAddress,
		},
	}
	logs, err := w.logger.FilterLogs(w.ctx, query)
	if err != nil {
		return false, 0, fmt.Errorf("could not filter deposit contract logs: %v", err)
	}
	for _, log := range logs {
		if log.Topics[0] == hashutil.Hash(chainStartEventSignature) {
			_, timestampData, err := contracts.UnpackChainStartLogData(log.Data)
			if err != nil {
				return false, 0, fmt.Errorf("unable to unpack ChainStart log data %v", err)
			}
			timestamp := binary.BigEndian.Uint64(timestampData)
			if uint64(time.Now().Unix()) < timestamp {
				return false, 0, fmt.Errorf(
					"invalid timestamp from log expected %d > %d",
					time.Now().Unix(),
					timestamp,
				)
			}
			return true, timestamp, nil
		}
	}
	return false, 0, nil
}

// ProcessLog is the main method which handles the processing of all
// logs from the deposit contract on the ETH1.0 chain.
func (w *Web3Service) ProcessLog(VRClog gethTypes.Log) {
	// Process logs according to their event signature.
	if VRClog.Topics[0] == hashutil.Hash(depositEventSignature) {
		w.ProcessDepositLog(VRClog)
		return
	}

	if VRClog.Topics[0] == hashutil.Hash(chainStartEventSignature) {
		w.ProcessChainStartLog(VRClog)
		return
	}

	log.Debugf("Log is not of a valid event signature %#x", VRClog.Topics[0])
}

// ProcessDepositLog processes the log which had been received from
// the ETH1.0 chain by trying to ascertain which participant deposited
// in the contract.
func (w *Web3Service) ProcessDepositLog(VRClog gethTypes.Log) {
	totalDepositsCount.Inc()
	merkleRoot, depositData, MerkleTreeIndex, err := contracts.UnpackDepositLogData(VRClog.Data)
	if err != nil {
		log.Errorf("Could not unpack log %v", err)
		return
	}
	if err := w.saveInTrie(depositData, merkleRoot); err != nil {
		log.Errorf("Could not save in trie %v", err)
		return
	}
	depositInput, err := blocks.DecodeDepositInput(depositData)
	if err != nil {
		log.Errorf("Could not decode deposit input  %v", err)
		return
	}
	deposit := &pb.Deposit{
		DepositData: depositData,
	}
	if !w.chainStarted {
		w.chainStartDeposits = append(w.chainStartDeposits, deposit)
	} else {
		w.beaconDB.InsertPendingDeposit(w.ctx, deposit, big.NewInt(int64(VRClog.BlockNumber)))
	}
	index := binary.BigEndian.Uint64(MerkleTreeIndex)
	log.WithFields(logrus.Fields{
		"publicKey":       fmt.Sprintf("%#x", depositInput.Pubkey),
		"merkleTreeIndex": index,
	}).Info("Validator registered in deposit contract")

	validDepositsCount.Inc()
}

// ProcessChainStartLog processes the log which had been received from
// the ETH1.0 chain by trying to determine when to start the beacon chain.
func (w *Web3Service) ProcessChainStartLog(VRClog gethTypes.Log) {
	chainStartCount.Inc()
	receiptRoot, timestampData, err := contracts.UnpackChainStartLogData(VRClog.Data)
	if err != nil {
		log.Errorf("Unable to unpack ChainStart log data %v", err)
		return
	}
	if w.depositTrie.Root() != receiptRoot {
		log.Errorf("Receipt root from log doesn't match the root saved in memory,"+
			" want %#x but got %#x", w.depositTrie.Root(), receiptRoot)
		return
	}

	timestamp := binary.BigEndian.Uint64(timestampData)
	if uint64(time.Now().Unix()) < timestamp {
		log.Errorf("Invalid timestamp from log expected %d > %d", time.Now().Unix(), timestamp)
	}
	w.chainStarted = true
	chainStartTime := time.Unix(int64(timestamp), 0)
	log.WithFields(logrus.Fields{
		"ChainStartTime": chainStartTime,
	}).Info("Minimum Number of Validators Reached for beacon-chain to start")
	w.chainStartFeed.Send(chainStartTime)
}

// run subscribes to all the services for the ETH1.0 chain.
func (w *Web3Service) run(done <-chan struct{}) {
	if err := w.initDataFromVRC(); err != nil {
		log.Errorf("Unable to retrieve data from VRC %v", err)
		return
	}

	headSub, err := w.reader.SubscribeNewHead(w.ctx, w.headerChan)
	if err != nil {
		log.Errorf("Unable to subscribe to incoming ETH1.0 chain headers: %v", err)
		return
	}
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			w.depositContractAddress,
		},
	}
	logSub, err := w.logger.SubscribeFilterLogs(w.ctx, query, w.logChan)
	if err != nil {
		log.Errorf("Unable to query logs from VRC: %v", err)
		return
	}
	if err := w.processPastLogs(query); err != nil {
		log.Errorf("Unable to process past logs %v", err)
		return
	}
	defer logSub.Unsubscribe()
	defer headSub.Unsubscribe()

	for {
		select {
		case <-done:
			log.Debug("ETH1.0 chain service context closed, exiting goroutine")
			return
		case <-headSub.Err():
			log.Debug("Unsubscribed to head events, exiting goroutine")
			return
		case <-logSub.Err():
			log.Debug("Unsubscribed to log events, exiting goroutine")
			return
		case header := <-w.headerChan:
			blockNumberGauge.Set(float64(header.Number.Int64()))
			w.blockNumber = header.Number
			w.blockHash = header.Hash()
			log.WithFields(logrus.Fields{
				"blockNumber": w.blockNumber,
				"blockHash":   w.blockHash.Hex(),
			}).Debug("Latest web3 chain event")
		case VRClog := <-w.logChan:
			log.Info("Received deposit contract log")
			w.ProcessLog(VRClog)

		}
	}
}

// initDataFromVRC calls the vrc contract and finds the deposit count
// and deposit root.
func (w *Web3Service) initDataFromVRC() error {
	root, err := w.vrcCaller.GetDepositRoot(&bind.CallOpts{})
	if err != nil {
		return fmt.Errorf("could not retrieve deposit root %v", err)
	}

	w.depositRoot = root[:]
	w.depositTrie = trieutil.NewDepositTrie()

	return nil
}

// saveInTrie saves in the in-memory deposit trie.
func (w *Web3Service) saveInTrie(depositData []byte, merkleRoot common.Hash) error {
	if w.depositTrie.Root() != merkleRoot {
		return errors.New("saved root in trie is unequal to root received from log")
	}

	w.depositTrie.UpdateDepositTrie(depositData)
	return nil
}

// processPastLogs processes all the past logs from the deposit contract and
// updates the deposit trie with the data from each individual log.
func (w *Web3Service) processPastLogs(query ethereum.FilterQuery) error {
	logs, err := w.logger.FilterLogs(w.ctx, query)
	if err != nil {
		return err
	}

	for _, log := range logs {
		w.ProcessLog(log)
	}
	return nil
}
