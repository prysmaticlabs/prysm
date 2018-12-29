// Package powchain defines the services that interact with the PoWChain of Ethereum.
package powchain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
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

// Logger defines a struct that subscribes to filtered logs on the PoW chain.
type Logger interface {
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- gethTypes.Log) (ethereum.Subscription, error)
}

// Client defines a struct that combines all relevant PoW mainchain interactions required
// by the beacon chain node.
type Client interface {
	Reader
	POWBlockFetcher
	Logger
}

// Web3Service fetches important information about the canonical
// Ethereum PoW chain via a web3 endpoint using an ethclient. The Random
// Beacon Chain requires synchronization with the PoW chain's current
// blockhash, block number, and access to logs within the
// Validator Registration Contract on the PoW chain to kick off the beacon
// chain's validator registration process.
type Web3Service struct {
	ctx                 context.Context
	cancel              context.CancelFunc
	client              Client
	headerChan          chan *gethTypes.Header
	logChan             chan gethTypes.Log
	pubKey              string
	endpoint            string
	validatorRegistered bool
	vrcAddress          common.Address
	reader              Reader
	logger              Logger
	blockNumber         *big.Int    // the latest PoW chain blockNumber.
	blockHash           common.Hash // the latest PoW chain blockHash.
}

// Web3ServiceConfig defines a config struct for web3 service to use through its life cycle.
type Web3ServiceConfig struct {
	Endpoint string
	Pubkey   string
	VrcAddr  common.Address
	Client   Client
	Reader   Reader
	Logger   Logger
}

// NewWeb3Service sets up a new instance with an ethclient when
// given a web3 endpoint as a string in the config.
func NewWeb3Service(ctx context.Context, config *Web3ServiceConfig) (*Web3Service, error) {
	if !strings.HasPrefix(config.Endpoint, "ws") && !strings.HasPrefix(config.Endpoint, "ipc") {
		return nil, fmt.Errorf(
			"web3service requires either an IPC or WebSocket endpoint, provided %s",
			config.Endpoint,
		)
	}
	ctx, cancel := context.WithCancel(ctx)
	return &Web3Service{
		ctx:                 ctx,
		cancel:              cancel,
		headerChan:          make(chan *gethTypes.Header),
		logChan:             make(chan gethTypes.Log),
		pubKey:              config.Pubkey,
		endpoint:            config.Endpoint,
		validatorRegistered: false,
		blockNumber:         nil,
		blockHash:           common.BytesToHash([]byte{}),
		vrcAddress:          config.VrcAddr,
		client:              config.Client,
		reader:              config.Reader,
		logger:              config.Logger,
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

// run subscribes to all the services for the powchain.
func (w *Web3Service) run(done <-chan struct{}) {
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
			// public key is the second topic from validatorRegistered log.
			pubKeyLog := VRClog.Topics[1].Hex()
			// Support user pubKeys with or without the leading 0x.
			if pubKeyLog == w.pubKey || pubKeyLog[2:] == w.pubKey {
				log.WithFields(logrus.Fields{
					"publicKey": pubKeyLog,
				}).Info("Validator registered in VRC with public key")
				w.validatorRegistered = true
				w.logChan = nil
			}
		}
	}
}

// LatestBlockNumber in the PoWChain.
func (w *Web3Service) LatestBlockNumber() *big.Int {
	return w.blockNumber
}

// LatestBlockHash in the PoWChain.
func (w *Web3Service) LatestBlockHash() common.Hash {
	return w.blockHash
}

// IsValidatorRegistered in the PoWChain.
func (w *Web3Service) IsValidatorRegistered() bool {
	return w.validatorRegistered
}

// Client for interacting with the PoWChain.
func (w *Web3Service) Client() Client {
	return w.client
}
