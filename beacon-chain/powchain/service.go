// Package powchain defines the services that interact with the PoWChain of Ethereum.
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

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	contracts "github.com/prysmaticlabs/prysm/contracts/validator-registration-contract"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/trie"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "powchain")

// Reader defines a struct that can fetch latest header events from a web3 endpoint.
type Reader interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error)
}

// POWBlockFetcher defines a struct that can retrieve mainchain blocks.
type POWBlockFetcher interface {
	BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error)
}

// Client defines a struct that combines all relevant PoW mainchain interactions required
// by the beacon chain node.
type Client interface {
	Reader
	POWBlockFetcher
	bind.ContractFilterer
	bind.ContractCaller
}

// Web3Service fetches important information about the canonical
// Ethereum PoW chain via a web3 endpoint using an ethclient. The Random
// Beacon Chain requires synchronization with the PoW chain's current
// blockhash, block number, and access to logs within the
// Validator Registration Contract on the PoW chain to kick off the beacon
// chain's validator registration process.
type Web3Service struct {
	ctx          context.Context
	cancel       context.CancelFunc
	client       Client
	headerChan   chan *gethTypes.Header
	logChan      chan gethTypes.Log
	endpoint     string
	vrcAddress   common.Address
	reader       Reader
	logger       bind.ContractFilterer
	blockNumber  *big.Int    // the latest PoW chain blockNumber.
	blockHash    common.Hash // the latest PoW chain blockHash.
	vrcCaller    *contracts.ValidatorRegistrationCaller
	depositCount uint64
	depositRoot  []byte
	depositTrie  *trie.DepositTrie
}

// Web3ServiceConfig defines a config struct for web3 service to use through its life cycle.
type Web3ServiceConfig struct {
	Endpoint        string
	VrcAddr         common.Address
	Client          Client
	Reader          Reader
	Logger          bind.ContractFilterer
	ContractBackend bind.ContractBackend
}

var (
	depositEventSignature    = []byte("Deposit(bytes,bytes,bytes)")
	chainStartEventSignature = []byte("ChainStart(bytes,bytes,bytes)")
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

	vrcCaller, err := contracts.NewValidatorRegistrationCaller(config.VrcAddr, config.ContractBackend)
	if err != nil {
		return nil, fmt.Errorf("could not create VRC caller %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	return &Web3Service{
		ctx:         ctx,
		cancel:      cancel,
		headerChan:  make(chan *gethTypes.Header),
		logChan:     make(chan gethTypes.Log),
		endpoint:    config.Endpoint,
		blockNumber: nil,
		blockHash:   common.BytesToHash([]byte{}),
		vrcAddress:  config.VrcAddr,
		client:      config.Client,
		reader:      config.Reader,
		logger:      config.Logger,
		vrcCaller:   vrcCaller,
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

// Status always returns nil.
// TODO(1204): Add service health checks.
func (w *Web3Service) Status() error {
	return nil
}

// initDataFromVRC calls the vrc contract and finds the deposit count
// and deposit root.
func (w *Web3Service) initDataFromVRC() error {
	depositCount, err := w.vrcCaller.DepositCount(&bind.CallOpts{})
	if err != nil {
		return fmt.Errorf("could not retrieve deposit count %v", err)
	}

	w.depositCount = depositCount.Uint64()

	root, err := w.vrcCaller.GetDepositRoot(&bind.CallOpts{})
	if err != nil {
		return fmt.Errorf("could not retrieve deposit root %v", err)
	}

	w.depositRoot = root
	w.depositTrie = trie.NewDepositTrie()

	return nil
}

// run subscribes to all the services for the powchain.
func (w *Web3Service) run(done <-chan struct{}) {

	if err := w.initDataFromVRC(); err != nil {
		log.Errorf("Unable to retrieve data from VRC %v", err)
		return
	}

	headSub, err := w.reader.SubscribeNewHead(w.ctx, w.headerChan)
	if err != nil {
		log.Errorf("Unable to subscribe to incoming PoW chain headers: %v", err)
		return
	}
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			w.vrcAddress,
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
			log.Debug("Powchain service context closed, exiting goroutine")
			return
		case <-headSub.Err():
			log.Debug("Unsubscribed to head events, exiting goroutine")
			return
		case <-logSub.Err():
			log.Debug("Unsubscribed to log events, exiting goroutine")
			return
		case header := <-w.headerChan:
			w.blockNumber = header.Number
			w.blockHash = header.Hash()
			log.WithFields(logrus.Fields{
				"blockNumber": w.blockNumber,
				"blockHash":   w.blockHash.Hex(),
			}).Debug("Latest web3 chain event")
		case VRClog := <-w.logChan:
			w.processLog(VRClog)

		}
	}
}

func (w *Web3Service) processLog(VRClog gethTypes.Log) {
	// Process logs according to their event signature.
	if VRClog.Topics[0] == hashutil.Hash(depositEventSignature) {
		w.processDepositLog(VRClog)
		return
	}

	if VRClog.Topics[0] == hashutil.Hash(chainStartEventSignature) {
		w.processChainStartLog(VRClog)
		return
	}

	log.Debugf("Log is not of a valid event signature %#x", VRClog.Topics[0])
}

// processDepositLog processes the log which had been received from
// the POW chain by trying to ascertain which participant deposited
// in the contract.
func (w *Web3Service) processDepositLog(VRClog gethTypes.Log) {

	merkleRoot := VRClog.Topics[1]
	depositData, MerkleTreeIndex, err := w.unPackDepositLogData(VRClog.Data)
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

	index := binary.BigEndian.Uint64(MerkleTreeIndex)

	log.WithFields(logrus.Fields{
		"publicKey":         depositInput.Pubkey,
		"merkle tree index": index,
	}).Info("Validator registered in VRC with public key and index")

}

// processChainStartLog processes the log which had been received from
// the POW chain by trying to determine when to start the beacon chain.
func (w *Web3Service) processChainStartLog(VRClog gethTypes.Log) {
	receiptRoot := VRClog.Topics[1]

	timestampData, err := w.unPackChainStartLogData(VRClog.Data)
	if err != nil {
		log.Errorf("Unable to unpack ChainStart log data %v", err)
		return
	}

	if w.depositTrie.Root() != receiptRoot {
		log.Errorf("Receipt root from log doesn't match the root saved in memory %#x", receiptRoot)
		return
	}

	timestamp := binary.BigEndian.Uint64(timestampData)

	if uint64(time.Now().Unix()) < timestamp {
		log.Errorf("Invalid timestamp from log %v", err)
	}

	chainStartTime := time.Unix(int64(timestamp), 0)

	log.WithFields(logrus.Fields{
		"ChainStartTime": chainStartTime,
	}).Info("Minimum Number of Validators Reached for beacon-chain to start")

}

// unPackDepositLogData unpacks the data from a log using the abi decoder.
func (w *Web3Service) unPackDepositLogData(data []byte) ([]byte, []byte, error) {

	reader := bytes.NewReader([]byte(contracts.ValidatorRegistrationABI))
	contractAbi, err := abi.JSON(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate contract abi %v", err)
	}

	unpackedLogs := []*[]byte{
		&[]byte{},
		&[]byte{},
	}

	if err := contractAbi.Unpack(&unpackedLogs, "Deposit", data); err != nil {
		return nil, nil, fmt.Errorf("unable to unpack logs %v", err)
	}

	depositData := *unpackedLogs[0]
	merkleTreeIndex := *unpackedLogs[1]

	return depositData, merkleTreeIndex, nil
}

func (w *Web3Service) unPackChainStartLogData(data []byte) ([]byte, error) {

	reader := bytes.NewReader([]byte(contracts.ValidatorRegistrationABI))
	contractAbi, err := abi.JSON(reader)
	if err != nil {
		return nil, fmt.Errorf("unable to generate contract abi %v", err)
	}

	unpackedLogs := []*[]byte{
		&[]byte{},
	}

	if err := contractAbi.Unpack(&unpackedLogs, "Deposit", data); err != nil {
		return nil, fmt.Errorf("unable to unpack logs %v", err)
	}

	timestamp := *unpackedLogs[0]

	return timestamp, nil
}

// saveInTrie saves in the in-memory deposit trie.
func (w *Web3Service) saveInTrie(depositData []byte, merkleRoot common.Hash) error {
	if w.depositTrie.Root() != merkleRoot {
		return errors.New("saved root in trie is unequal to root received from log")
	}

	w.depositTrie.UpdateDepositTrie(depositData)
	return nil
}

func (w *Web3Service) processPastLogs(query ethereum.FilterQuery) error {
	logs, err := w.logger.FilterLogs(w.ctx, query)
	if err != nil {
		return err
	}

	for _, log := range logs {
		w.processLog(log)
	}
	return nil
}

// LatestBlockNumber in the PoWChain.
func (w *Web3Service) LatestBlockNumber() *big.Int {
	return w.blockNumber
}

// LatestBlockHash in the PoWChain.
func (w *Web3Service) LatestBlockHash() common.Hash {
	return w.blockHash
}

// Client for interacting with the PoWChain.
func (w *Web3Service) Client() Client {
	return w.client
}
