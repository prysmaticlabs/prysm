// Package sync defines the utilities for the beacon-chain to sync with the network.
package sync

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "sync")

type chainService interface {
	ContainsBlock(h [32]byte) bool
	HasStoredState() (bool, error)
	IncomingBlockFeed() *event.Feed
	CheckForCanonicalBlockBySlot(slotnumber uint64) (bool, error)
	GetCanonicalBlockBySlotNumber(slotnumber uint64) (*types.Block, error)
}

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
//     *  Throttle incoming requests
type Service struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	p2p                   types.P2P
	chainService          chainService
	blockAnnouncementFeed *event.Feed
	announceBlockHashBuf  chan p2p.Message
	blockBuf              chan p2p.Message
	blockRequestBySlot    chan p2p.Message
}

// Config allows the channel's buffer sizes to be changed.
type Config struct {
	BlockHashBufferSize    int
	BlockBufferSize        int
	BlockRequestBufferSize int
}

// DefaultConfig provides the default configuration for a sync service.
func DefaultConfig() Config {
	return Config{
		BlockHashBufferSize:    100,
		BlockBufferSize:        100,
		BlockRequestBufferSize: 100,
	}
}

// NewSyncService accepts a context and returns a new Service.
func NewSyncService(ctx context.Context, cfg Config, beaconp2p types.P2P, cs chainService) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                   ctx,
		cancel:                cancel,
		p2p:                   beaconp2p,
		chainService:          cs,
		blockAnnouncementFeed: new(event.Feed),
		announceBlockHashBuf:  make(chan p2p.Message, cfg.BlockHashBufferSize),
		blockBuf:              make(chan p2p.Message, cfg.BlockBufferSize),
		blockRequestBySlot:    make(chan p2p.Message, cfg.BlockRequestBufferSize),
	}
}

// Start begins the block processing goroutine.
func (ss *Service) Start() {
	stored, err := ss.chainService.HasStoredState()
	if err != nil {
		log.Errorf("error retrieving stored state: %v", err)
		return
	}

	if !stored {
		// TODO: Resume sync after completion of initial sync.
		// Currently, `Simulator` only supports sync from genesis block, therefore
		// new nodes with a fresh database must skip InitialSync and immediately run the Sync goroutine.
		log.Info("Empty chain state, but continue sync")
	}

	go ss.run()
}

// Stop kills the block processing goroutine, but does not wait until the goroutine exits.
func (ss *Service) Stop() error {
	log.Info("Stopping service")
	ss.cancel()
	return nil
}

// BlockAnnouncementFeed returns an event feed processes can subscribe to for
// newly received, incoming p2p blocks.
func (ss *Service) BlockAnnouncementFeed() *event.Feed {
	return ss.blockAnnouncementFeed
}

// ReceiveBlockHash accepts a block hash.
// New hashes are forwarded to other peers in the network (unimplemented), and
// the contents of the block are requested if the local chain doesn't have the block.
func (ss *Service) ReceiveBlockHash(data *pb.BeaconBlockHashAnnounce, peer p2p.Peer) {
	var h [32]byte
	copy(h[:], data.Hash[:32])
	if ss.chainService.ContainsBlock(h) {
		return
	}
	log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Debug("Received incoming block hash, requesting full block data from sender")
	// Request the full block data from peer that sent the block hash.
	ss.p2p.Send(&pb.BeaconBlockRequest{Hash: h[:]}, peer)
}

// run handles incoming block sync.
func (ss *Service) run() {
	announceBlockHashSub := ss.p2p.Subscribe(pb.BeaconBlockHashAnnounce{}, ss.announceBlockHashBuf)
	blockSub := ss.p2p.Subscribe(pb.BeaconBlockResponse{}, ss.blockBuf)
	blockRequestSub := ss.p2p.Subscribe(pb.BeaconBlockRequestBySlotNumber{}, ss.blockRequestBySlot)

	defer announceBlockHashSub.Unsubscribe()
	defer blockSub.Unsubscribe()
	defer blockRequestSub.Unsubscribe()

	for {
		select {
		case <-ss.ctx.Done():
			log.Debug("Exiting goroutine")
			return
		case msg := <-ss.announceBlockHashBuf:
			data, ok := msg.Data.(*pb.BeaconBlockHashAnnounce)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Error("Received malformed beacon block hash announcement p2p message")
				continue
			}
			ss.ReceiveBlockHash(data, msg.Peer)
		case msg := <-ss.blockBuf:
			response, ok := msg.Data.(*pb.BeaconBlockResponse)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed beacon block p2p message")
				continue
			}
			block := types.NewBlock(response.Block)
			h, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash received block: %v", err)
			}
			if ss.chainService.ContainsBlock(h) {
				continue
			}
			log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Debug("Sending newly received block to subscribers")
			// We send out a message over a feed.
			ss.chainService.IncomingBlockFeed().Send(block)
		case msg := <-ss.blockRequestBySlot:
			request, ok := msg.Data.(*pb.BeaconBlockRequestBySlotNumber)
			if !ok {
				log.Error("Received malformed beacon block request p2p message")
				continue
			}

			blockExists, err := ss.chainService.CheckForCanonicalBlockBySlot(request.GetSlotNumber())
			if err != nil {
				log.Errorf("Error checking db for block %v", err)
				continue
			}
			if !blockExists {
				continue
			}

			block, err := ss.chainService.GetCanonicalBlockBySlotNumber(request.GetSlotNumber())
			if err != nil {
				log.Errorf("Error retrieving block from db %v", err)
				continue
			}

			log.WithField("slotnumber", fmt.Sprintf("%d", request.GetSlotNumber())).Debug("Sending requested block to peer")
			ss.p2p.Send(block.Proto(), msg.Peer)
		}
	}
}
