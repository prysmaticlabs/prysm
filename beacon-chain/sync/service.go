// Package sync defines the utilities for the beacon-chain to sync with the network.
package sync

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "sync")

type chainService interface {
	ContainsBlock(h [32]byte) (bool, error)
	HasStoredState() (bool, error)
	IncomingBlockFeed() *event.Feed
	IncomingAttestationFeed() *event.Feed
	CheckForCanonicalBlockBySlot(slotnumber uint64) (bool, error)
	GetCanonicalBlockBySlotNumber(slotnumber uint64) (*types.Block, error)
	GetBlockSlotNumber(h [32]byte) (uint64, error)
	CurrentCrystallizedState() *types.CrystallizedState
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
	p2p                   shared.P2P
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
func NewSyncService(ctx context.Context, cfg Config, beaconp2p shared.P2P, cs chainService) *Service {
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
			ss.receiveBlockHash(msg)
		case msg := <-ss.blockBuf:
			ss.receiveBlock(msg)
		case msg := <-ss.blockRequestBySlot:
			ss.handleBlockRequestBySlot(msg)
		}
	}
}

// receiveBlockHash accepts a block hash.
// New hashes are forwarded to other peers in the network (unimplemented), and
// the contents of the block are requested if the local chain doesn't have the block.
func (ss *Service) receiveBlockHash(msg p2p.Message) {
	ctx, receiveBlockSpan := trace.StartSpan(msg.Ctx, "receiveBlockHash")
	defer receiveBlockSpan.End()

	data := msg.Data.(*pb.BeaconBlockHashAnnounce)
	var h [32]byte
	copy(h[:], data.Hash[:32])

	ctx, containsBlockSpan := trace.StartSpan(ctx, "containsBlock")
	blockExists, err := ss.chainService.ContainsBlock(h)
	containsBlockSpan.End()
	if err != nil {
		log.Errorf("Received block hash failed: %v", err)
	}
	if blockExists {
		return
	}

	log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Debug("Received incoming block hash, requesting full block data from sender")
	// Request the full block data from peer that sent the block hash.
	_, sendBlockRequestSpan := trace.StartSpan(ctx, "sendBlockRequest")
	ss.p2p.Send(&pb.BeaconBlockRequest{Hash: h[:]}, msg.Peer)
	sendBlockRequestSpan.End()
}

// receiveBlock processes a block from the p2p layer.
func (ss *Service) receiveBlock(msg p2p.Message) {
	ctx, receiveBlockSpan := trace.StartSpan(msg.Ctx, "receiveBlock")
	defer receiveBlockSpan.End()

	response := msg.Data.(*pb.BeaconBlockResponse)
	block := types.NewBlock(response.Block)
	blockHash, err := block.Hash()
	if err != nil {
		log.Errorf("Could not hash received block: %v", err)
	}

	ctx, containsBlockSpan := trace.StartSpan(ctx, "containsBlock")
	blockExists, err := ss.chainService.ContainsBlock(blockHash)
	containsBlockSpan.End()
	if err != nil {
		log.Errorf("Can not check for block in DB: %v", err)
		return
	}
	if blockExists {
		return
	}

	// Verify attestation coming from proposer then forward block to the subscribers.
	attestation := types.NewAttestation(response.Attestation)
	cState := ss.chainService.CurrentCrystallizedState()
	parentSlot, err := ss.chainService.GetBlockSlotNumber(block.ParentHash())
	if err != nil {
		log.Errorf("Failed to get parent slot: %v", err)
		return
	}
	proposerShardID, _, err := casper.GetProposerIndexAndShard(cState.ShardAndCommitteesForSlots(), cState.LastStateRecalc(), parentSlot)
	if err != nil {
		log.Errorf("Failed to get proposer shard ID: %v", err)
		return
	}
	if err := attestation.VerifyAttestation(proposerShardID); err != nil {
		log.Errorf("Failed to verify proposer attestation: %v", err)
		return
	}

	_, sendBlockSpan := trace.StartSpan(ctx, "sendBlock")
	log.WithField("blockHash", fmt.Sprintf("0x%x", blockHash)).Debug("Sending newly received block to subscribers")
	ss.chainService.IncomingBlockFeed().Send(block)
	sendBlockSpan.End()

	_, sendAttestationSpan := trace.StartSpan(ctx, "sendAttestation")
	log.WithField("attestationHash", fmt.Sprintf("0x%x", attestation.Key())).Debug("Sending newly received attestation to subscribers")
	ss.chainService.IncomingAttestationFeed().Send(attestation)
	sendAttestationSpan.End()
}

// handleBlockRequestBySlot processes a block request from the p2p layer.
// if found, the block is sent to the requesting peer.
func (ss *Service) handleBlockRequestBySlot(msg p2p.Message) {
	ctx, blockRequestSpan := trace.StartSpan(msg.Ctx, "blockRequestBySlot")
	defer blockRequestSpan.End()

	request, ok := msg.Data.(*pb.BeaconBlockRequestBySlotNumber)
	// TODO: Handle this at p2p layer.
	if !ok {
		log.Error("Received malformed beacon block request p2p message")
		return
	}

	ctx, checkForBlockSpan := trace.StartSpan(ctx, "checkForBlockBySlot")
	blockExists, err := ss.chainService.CheckForCanonicalBlockBySlot(request.GetSlotNumber())
	checkForBlockSpan.End()
	if err != nil {
		log.Errorf("Error checking db for block %v", err)
		return
	}
	if !blockExists {
		return
	}

	ctx, getBlockSpan := trace.StartSpan(ctx, "getBlockBySlot")
	block, err := ss.chainService.GetCanonicalBlockBySlotNumber(request.GetSlotNumber())
	getBlockSpan.End()
	if err != nil {
		log.Errorf("Error retrieving block from db %v", err)
		return
	}

	_, sendBlockSpan := trace.StartSpan(ctx, "sendBlock")
	log.WithField("slotNumber", fmt.Sprintf("%d", request.GetSlotNumber())).Debug("Sending requested block to peer")
	ss.p2p.Send(block.Proto(), msg.Peer)
	sendBlockSpan.End()
}
