package sync

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/blake2b"
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
	p2p                  types.P2P
	chainService         types.ChainService
	announceBlockHashBuf chan p2p.Message
	blockBuf             chan p2p.Message
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
func NewSyncService(ctx context.Context, cfg Config, beaconp2p types.P2P, cs types.ChainService) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                  ctx,
		cancel:               cancel,
		p2p:                  beaconp2p,
		chainService:         cs,
		announceBlockHashBuf: make(chan p2p.Message, cfg.HashBufferSize),
		blockBuf:             make(chan p2p.Message, cfg.BlockBufferSize),
	}
}

// Start begins the block processing goroutine.
func (ss *Service) Start() {
	log.Info("Starting service")
	go ss.run(ss.ctx.Done())
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
func (ss *Service) ReceiveBlockHash(data *pb.BeaconBlockHashAnnounce) error {
	h, err := blake2b.New256(data.Hash)
	if err != nil {
		return fmt.Errorf("could not calculate blake2b hash of proto hash: %v", err)
	}
	if ss.chainService.ContainsBlock(h) {
		return nil
	}
	log.Info("Requesting full block data from sender")
	// TODO: Request the full block data from peer that sent the block hash.
	return nil
}

// ReceiveBlock accepts a block to potentially be included in the local chain.
// The service will filter blocks that have not been requested (unimplemented).
func (ss *Service) ReceiveBlock(data *pb.BeaconBlockResponse) error {
	block, err := types.NewBlockWithData(data)
	if err != nil {
		return fmt.Errorf("could not instantiate new block from proto: %v", err)
	}
	h, err := block.Hash()
	if err != nil {
		return fmt.Errorf("could not hash block: %v", err)
	}
	if ss.chainService.ContainsBlock(h) {
		return nil
	}
	log.Infof("Broadcasting block hash to peers: %x", h.Sum(nil))
	ss.p2p.Broadcast(&pb.BeaconBlockHashAnnounce{
		Hash: h.Sum(nil),
	})
	ss.chainService.ProcessBlock(block)
	return nil
}

func (ss *Service) run(done <-chan struct{}) {
	announceBlockHashSub := ss.p2p.Feed(pb.BeaconBlockHashAnnounce{}).Subscribe(ss.announceBlockHashBuf)
	blockSub := ss.p2p.Feed(pb.BeaconBlockResponse{}).Subscribe(ss.blockBuf)
	defer announceBlockHashSub.Unsubscribe()
	defer blockSub.Unsubscribe()
	for {
		select {
		case <-done:
			log.Infof("Exiting goroutine")
			return
		case msg := <-ss.announceBlockHashBuf:
			data, ok := msg.Data.(pb.BeaconBlockHashAnnounce)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed beacon block hash announcement p2p message")
				continue
			}
			if err := ss.ReceiveBlockHash(&data); err != nil {
				log.Errorf("Could not receive incoming block hash: %v", err)
			}
		case msg := <-ss.blockBuf:
			data, ok := msg.Data.(pb.BeaconBlockResponse)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed beacon block p2p message")
				continue
			}
			if err := ss.ReceiveBlock(&data); err != nil {
				log.Errorf("Could not receive incoming block: %v", err)
			}
		}
	}
}
