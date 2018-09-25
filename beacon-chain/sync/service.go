// Package sync defines the utilities for the beacon-chain to sync with the network.
package sync

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "sync")

type chainService interface {
	IncomingBlockFeed() *event.Feed
	IncomingAttestationFeed() *event.Feed
}

// DB is the interface for the DB service.
type beaconDB interface {
	GetBlock([32]byte) (*types.Block, error)
	GetBlockBySlot(uint64) (*types.Block, error)
	GetCrystallizedState() (*types.CrystallizedState, error)
	HasInitialState() bool
	HasBlock([32]byte) bool
	HasBlockForSlot(uint64) bool
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
	db                    beaconDB
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
func NewSyncService(ctx context.Context, cfg Config, beaconp2p types.P2P, cs chainService, db beaconDB) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                   ctx,
		cancel:                cancel,
		p2p:                   beaconp2p,
		db:                    db,
		chainService:          cs,
		blockAnnouncementFeed: new(event.Feed),
		announceBlockHashBuf:  make(chan p2p.Message, cfg.BlockHashBufferSize),
		blockBuf:              make(chan p2p.Message, cfg.BlockBufferSize),
		blockRequestBySlot:    make(chan p2p.Message, cfg.BlockRequestBufferSize),
	}
}

// Start begins the block processing goroutine.
func (ss *Service) Start() {
	stored := ss.db.HasInitialState()

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

// receiveBlockHash accepts a block hash.
// New hashes are forwarded to other peers in the network (unimplemented), and
// the contents of the block are requested if the local chain doesn't have the block.
func (ss *Service) receiveBlockHash(data *pb.BeaconBlockHashAnnounce, peer p2p.Peer) error {
	var h [32]byte
	copy(h[:], data.Hash[:32])
	if ss.db.HasBlock(h) {
		return nil
	}
	log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Debug("Received incoming block hash, requesting full block data from sender")
	// Request the full block data from peer that sent the block hash.
	ss.p2p.Send(&pb.BeaconBlockRequest{Hash: h[:]}, peer)
	return nil
}

// run handles incoming block sync.
func (ss *Service) run() {
	announceBlockHashSub := ss.p2p.Subscribe(&pb.BeaconBlockHashAnnounce{}, ss.announceBlockHashBuf)
	blockSub := ss.p2p.Subscribe(&pb.BeaconBlockResponse{}, ss.blockBuf)
	blockRequestSub := ss.p2p.Subscribe(&pb.BeaconBlockRequestBySlotNumber{}, ss.blockRequestBySlot)

	defer announceBlockHashSub.Unsubscribe()
	defer blockSub.Unsubscribe()
	defer blockRequestSub.Unsubscribe()

	for {
		select {
		case <-ss.ctx.Done():
			log.Debug("Exiting goroutine")
			return
		case msg := <-ss.announceBlockHashBuf:
			data := msg.Data.(*pb.BeaconBlockHashAnnounce)
			if err := ss.receiveBlockHash(data, msg.Peer); err != nil {
				log.Errorf("Received block hash failed: %v", err)
			}
		case msg := <-ss.blockBuf:
			response := msg.Data.(*pb.BeaconBlockResponse)
			block := types.NewBlock(response.Block)
			blockHash, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash received block: %v", err)
			}
			if ss.db.HasBlock(blockHash) {
				continue
			}

			// Verify attestation coming from proposer then forward block to the subscribers.
			attestation := types.NewAttestation(response.Attestation)
			cState, err := ss.db.GetCrystallizedState()
			if err != nil {
				log.Errorf("Failed to get crystallized state: %v", err)
			}

			parentBlock, err := ss.db.GetBlock(block.ParentHash())
			if err != nil {
				log.Errorf("Failed to get parent slot: %v", err)
				continue
			}
			if parentBlock == nil {
				continue
			}
			parentSlot := parentBlock.SlotNumber()

			proposerShardID, _, err := casper.GetProposerIndexAndShard(cState.ShardAndCommitteesForSlots(), cState.LastStateRecalc(), parentSlot)
			if err != nil {
				log.Errorf("Failed to get proposer shard ID: %v", err)
				continue
			}
			if err := attestation.VerifyAttestation(proposerShardID); err != nil {
				log.Errorf("Failed to verify proposer attestation: %v", err)
				continue
			}

			log.WithField("blockHash", fmt.Sprintf("0x%x", blockHash)).Debug("Sending newly received block to subscribers")
			ss.chainService.IncomingBlockFeed().Send(block)
			log.WithField("attestationHash", fmt.Sprintf("0x%x", attestation.Key())).Debug("Sending newly received attestation to subscribers")
			ss.chainService.IncomingAttestationFeed().Send(attestation)

		case msg := <-ss.blockRequestBySlot:
			request, ok := msg.Data.(*pb.BeaconBlockRequestBySlotNumber)
			if !ok {
				log.Error("Received malformed beacon block request p2p message")
				continue
			}

			block, err := ss.db.GetBlockBySlot(request.GetSlotNumber())
			if err != nil {
				log.Errorf("Error retrieving block from db %v", err)
				continue
			}
			if block != nil {
				continue
			}

			log.WithField("slotNumber", fmt.Sprintf("%d", request.GetSlotNumber())).Debug("Sending requested block to peer")
			ss.p2p.Send(block.Proto(), msg.Peer)
		}
	}
}
