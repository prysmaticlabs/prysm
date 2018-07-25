package sync

import (
	"context"
	"hash"

	"github.com/ethereum/go-ethereum/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/shared/p2p"
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
	ctx                  context.Context
	cancel               context.CancelFunc
	networkService       types.NetworkService
	chainService         types.ChainService
	announceBlockHashBuf chan p2p.Message
	blockBuf             chan p2p.Message
	announceBlockHashSub event.Subscription
	blockSub             event.Subscription
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

// NewSyncService accepts a context and returns a new Service.
func NewSyncService(ctx context.Context, cfg Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                  ctx,
		cancel:               cancel,
		announceBlockHashBuf: make(chan p2p.Message, cfg.HashBufferSize),
		blockBuf:             make(chan p2p.Message, cfg.BlockBufferSize),
	}
}

// SetNetworkService sets a concrete value for the p2p layer.
func (ss *Service) SetNetworkService(ps types.NetworkService) {
	ss.networkService = ps
}

// SetChainService sets a concrete value for the local beacon chain.
func (ss *Service) SetChainService(cs types.ChainService) {
	ss.chainService = cs
}

// Start begins the block processing goroutine.
func (ss *Service) Start() {
	log.Info("Starting service")
	ss.announceBlockHashSub = s.p2p.Feed(pb.BeaconBlockHashAnnounce{}).Subscribe(ss.announceBlockHashBuf)
	ss.blockSub = s.p2p.Feed(pb.BeaconBlockResponse{}).Subscribe(ss.blockBuf)
	go ss.run(ss.networkService, ss.chainService, ss.ctx.Done())
}

// Stop kills the block processing goroutine, but does not wait until the goroutine exits.
func (ss *Service) Stop() error {
	log.Info("Stopping service")
	ss.cancel()
	ss.announceBlockHashSub.Unsubscribe()
	ss.blockSub.Unsubscribe()
	return nil
}

// ReceiveBlockHash accepts a block hash.
// New hashes are forwarded to other peers in the network (unimplemented), and
// the contents of the block are requested if the local chain doesn't have the block.
func (ss *Service) ReceiveBlockHash(h hash.Hash) {
	if ss.chainService.ContainsBlock(h) {
		return
	}
	ss.networkService.BroadcastBlockHash(h)
	ss.networkService.RequestBlock(h)
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
	ss.networkService.BroadcastBlock(b)
	ss.chainService.ProcessBlock(b)
	return nil
}

func (ss *Service) run(ps types.NetworkService, cs types.ChainService, done <-chan struct{}) {
	for {
		select {
		case <-done:
			log.Infof("exiting goroutine")
			return
		case h := <-ss.announceBlockHashBuf:
			ss.ReceiveBlockHash(h)
		case b := <-ss.announceBlockBuf:
			cs.ReceiveBlock(b)
		case <-ss.announceBlockHashSub.Err():
			log.Debugf("Subscriber failed")
			return
		}
		case <-ss.blockSub.Err():
			log.Debugf("Subscriber failed")
			return
		}
	}
}
