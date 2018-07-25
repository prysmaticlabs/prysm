package sync

import (
	"context"
	"hash"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "sync")

// Service is the gateway and the bridge between the p2p network and the local beacon chain.
// In broad terms, a new block is synced in 4 steps:
//     1. Receive a block hash from a peer
//     2. Request the block for the hash from the network
//     3. Receive the block
//     4. Forward block to the beacon service for full validation
//
//  In addition, Service will handle the following responsibilities:
//     *  Decide which messages are forwarded to other peers
//     *  Filter redundant data and unwanted data
//     *  Drop peers that send invalid data
//     *  Trottle incoming requests
type Service struct {
	ctx            context.Context
	cancel         context.CancelFunc
	networkService NetworkService
	chainService   ChainService
	hashBuf        chan hash.Hash
	blockBuf       chan *types.Block
}

// Config allows the channel's buffer sizes to be changed
type Config struct {
	HashBufferSize  int
	BlockBufferSize int
}

// DefaultConfig provides the default configuration for a sync service
func DefaultConfig() Config {
	return Config{100, 100}
}

// NetworkService is the interface for the p2p network.
type NetworkService interface {
	BroadcastBlockHash(hash.Hash) error
	BroadcastBlock(*types.Block) error
	RequestBlock(hash.Hash) error
}

// ChainService is the interface for the local beacon chain.
type ChainService interface {
	ProcessBlock(*types.Block) error
	ContainsBlock(hash.Hash) bool
}

// NewSyncService accepts a context and returns a new Service.
func NewSyncService(ctx context.Context, cfg Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:      ctx,
		cancel:   cancel,
		hashBuf:  make(chan hash.Hash, cfg.HashBufferSize),
		blockBuf: make(chan *types.Block, cfg.BlockBufferSize),
	}
}

// SetNetworkService sets a concrete value for the p2p layer.
func (ss *Service) SetNetworkService(ps NetworkService) {
	ss.networkService = ps
}

// SetChainService sets a concrete value for the local beacon chain.
func (ss *Service) SetChainService(cs ChainService) {
	ss.chainService = cs
}

// Start begins the block processing goroutine.
func (ss *Service) Start() {
	log.Info("Starting service")
	go run(ss.ctx.Done(), ss.hashBuf, ss.blockBuf, ss.networkService, ss.chainService)
}

// Stop kills the block processing goroutine, but does not wait until the goroutine exits.
func (ss *Service) Stop() error {
	log.Info("Stopping service")
	ss.cancel()
	return nil
}

// ReceiveBlockHash accepts a block hash.
// New hashes are forwarded to other peers in the network (unimplemented), and
// the contents of the block are requested if the local chain doesn't have the block.
func (ss *Service) ReceiveBlockHash(h hash.Hash) {
	if ss.chainService.ContainsBlock(h) {
		return
	}

	ss.hashBuf <- h
	ss.networkService.BroadcastBlockHash(h)
}

// ReceiveBlock accepts a block to potentially be included in the local chain.
// The service will filter blocks that have not been requested (unimplemented).
func (ss *Service) ReceiveBlock(b *types.Block) error {
	h, err := b.Hash()
	if err != nil {
		return err
	}

	if ss.chainService.ContainsBlock(h) {
		return nil
	}

	ss.blockBuf <- b
	ss.networkService.BroadcastBlock(b)

	return nil
}

func run(done <-chan struct{}, hashBuf <-chan hash.Hash, blockBuf <-chan *types.Block, ps NetworkService, cs ChainService) {
	for {
		select {
		case <-done:
			log.Infof("exiting goroutine")
			return
		case h := <-hashBuf:
			ps.RequestBlock(h)
		case b := <-blockBuf:
			cs.ProcessBlock(b)
		}
	}
}
