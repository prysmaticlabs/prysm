// Package sync defines the utilities for the beacon-chain to sync with the network.
package sync

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	att "github.com/prysmaticlabs/prysm/beacon-chain/core/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "regular-sync")

type chainService interface {
	IncomingBlockFeed() *event.Feed
}

type operationService interface {
	IncomingExitFeed() *event.Feed
	IncomingAttFeed() *event.Feed
}

type p2pAPI interface {
	Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription
	Send(msg proto.Message, peer p2p.Peer)
	Broadcast(msg proto.Message)
}

// RegularSync is the gateway and the bridge between the p2p network and the local beacon chain.
// In broad terms, a new block is synced in 4 steps:
//     1. Receive a block hash from a peer
//     2. Request the block for the hash from the network
//     3. Receive the block
//     4. Forward block to the beacon service for full validation
//
//  In addition, RegularSync will handle the following responsibilities:
//     *  Decide which messages are forwarded to other peers
//     *  Filter redundant data and unwanted data
//     *  Drop peers that send invalid data
//     *  Throttle incoming requests
type RegularSync struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	p2p                   p2pAPI
	chainService          chainService
	operationsService     operationService
	db                    *db.BeaconDB
	blockAnnouncementFeed *event.Feed
	announceBlockBuf      chan p2p.Message
	blockBuf              chan p2p.Message
	blockRequestBySlot    chan p2p.Message
	blockRequestByHash    chan p2p.Message
	batchedRequestBuf     chan p2p.Message
	chainHeadReqBuf       chan p2p.Message
	attestationBuf        chan p2p.Message
	exitBuf               chan p2p.Message
}

// RegularSyncConfig allows the channel's buffer sizes to be changed.
type RegularSyncConfig struct {
	BlockAnnounceBufferSize int
	BlockBufferSize         int
	BlockReqSlotBufferSize  int
	BlockReqHashBufferSize  int
	BatchedBufferSize       int
	AttestationBufferSize   int
	ExitBufferSize          int
	ChainHeadReqBufferSize  int
	ChainService            chainService
	OperationService        operationService
	BeaconDB                *db.BeaconDB
	P2P                     p2pAPI
}

// DefaultRegularSyncConfig provides the default configuration for a sync service.
func DefaultRegularSyncConfig() *RegularSyncConfig {
	return &RegularSyncConfig{
		BlockAnnounceBufferSize: 100,
		BlockBufferSize:         100,
		BlockReqSlotBufferSize:  100,
		BlockReqHashBufferSize:  100,
		BatchedBufferSize:       100,
		ChainHeadReqBufferSize:  100,
		AttestationBufferSize:   100,
		ExitBufferSize:          100,
	}
}

// NewRegularSyncService accepts a context and returns a new Service.
func NewRegularSyncService(ctx context.Context, cfg *RegularSyncConfig) *RegularSync {
	ctx, cancel := context.WithCancel(ctx)
	return &RegularSync{
		ctx:                   ctx,
		cancel:                cancel,
		p2p:                   cfg.P2P,
		chainService:          cfg.ChainService,
		db:                    cfg.BeaconDB,
		operationsService:     cfg.OperationService,
		blockAnnouncementFeed: new(event.Feed),
		announceBlockBuf:      make(chan p2p.Message, cfg.BlockAnnounceBufferSize),
		blockBuf:              make(chan p2p.Message, cfg.BlockBufferSize),
		blockRequestBySlot:    make(chan p2p.Message, cfg.BlockReqSlotBufferSize),
		blockRequestByHash:    make(chan p2p.Message, cfg.BlockReqHashBufferSize),
		batchedRequestBuf:     make(chan p2p.Message, cfg.BatchedBufferSize),
		attestationBuf:        make(chan p2p.Message, cfg.AttestationBufferSize),
		exitBuf:               make(chan p2p.Message, cfg.ExitBufferSize),
		chainHeadReqBuf:       make(chan p2p.Message, cfg.ChainHeadReqBufferSize),
	}
}

// Start begins the block processing goroutine.
func (rs *RegularSync) Start() {
	go rs.run()
}

// ResumeSync resumes normal sync after initial sync is complete.
func (rs *RegularSync) ResumeSync() {
	go rs.run()
}

// Stop kills the block processing goroutine, but does not wait until the goroutine exits.
func (rs *RegularSync) Stop() error {
	log.Info("Stopping service")
	rs.cancel()
	return nil
}

// BlockAnnouncementFeed returns an event feed processes can subscribe to for
// newly received, incoming p2p blocks.
func (rs *RegularSync) BlockAnnouncementFeed() *event.Feed {
	return rs.blockAnnouncementFeed
}

// run handles incoming block sync.
func (rs *RegularSync) run() {
	announceBlockSub := rs.p2p.Subscribe(&pb.BeaconBlockAnnounce{}, rs.announceBlockBuf)
	blockSub := rs.p2p.Subscribe(&pb.BeaconBlockResponse{}, rs.blockBuf)
	blockRequestSub := rs.p2p.Subscribe(&pb.BeaconBlockRequestBySlotNumber{}, rs.blockRequestBySlot)
	blockRequestHashSub := rs.p2p.Subscribe(&pb.BeaconBlockRequest{}, rs.blockRequestByHash)
	batchedRequestSub := rs.p2p.Subscribe(&pb.BatchedBeaconBlockRequest{}, rs.batchedRequestBuf)
	attestationSub := rs.p2p.Subscribe(&pb.Attestation{}, rs.attestationBuf)
	exitSub := rs.p2p.Subscribe(&pb.Exit{}, rs.exitBuf)
	chainHeadReqSub := rs.p2p.Subscribe(&pb.ChainHeadRequest{}, rs.chainHeadReqBuf)

	defer announceBlockSub.Unsubscribe()
	defer blockSub.Unsubscribe()
	defer blockRequestSub.Unsubscribe()
	defer blockRequestHashSub.Unsubscribe()
	defer batchedRequestSub.Unsubscribe()
	defer chainHeadReqSub.Unsubscribe()
	defer attestationSub.Unsubscribe()
	defer exitSub.Unsubscribe()

	for {
		select {
		case <-rs.ctx.Done():
			log.Debug("Exiting goroutine")
			return
		case msg := <-rs.announceBlockBuf:
			rs.receiveBlockAnnounce(msg)
		case msg := <-rs.attestationBuf:
			rs.receiveAttestation(msg)
		case msg := <-rs.exitBuf:
			rs.receiveExitRequest(msg)
		case msg := <-rs.blockBuf:
			rs.receiveBlock(msg)
		case msg := <-rs.blockRequestBySlot:
			rs.handleBlockRequestBySlot(msg)
		case msg := <-rs.blockRequestByHash:
			rs.handleBlockRequestByHash(msg)
		case msg := <-rs.batchedRequestBuf:
			rs.handleBatchedBlockRequest(msg)
		case msg := <-rs.chainHeadReqBuf:
			rs.handleChainHeadRequest(msg)
		}
	}
}

// receiveBlockAnnounce accepts a block hash.
// TODO(#175): New hashes are forwarded to other peers in the network, and
// the contents of the block are requested if the local chain doesn't have the block.
func (rs *RegularSync) receiveBlockAnnounce(msg p2p.Message) {
	ctx, receiveBlockSpan := trace.StartSpan(msg.Ctx, "RegularSync_receiveBlockHash")
	defer receiveBlockSpan.End()

	data := msg.Data.(*pb.BeaconBlockAnnounce)
	h := bytesutil.ToBytes32(data.Hash[:32])

	if rs.db.HasBlock(h) {
		log.Debugf("Received a hash for a block that has already been processed: %#x", h)
		return
	}

	log.WithField("blockHash", fmt.Sprintf("%#x", h)).Debug("Received incoming block hash, requesting full block data from sender")
	// Request the full block data from peer that sent the block hash.
	_, sendBlockRequestSpan := trace.StartSpan(ctx, "sendBlockRequest")
	rs.p2p.Send(&pb.BeaconBlockRequest{Hash: h[:]}, msg.Peer)
	sendBlockRequestSpan.End()
}

// receiveBlock processes a block from the p2p layer.
func (rs *RegularSync) receiveBlock(msg p2p.Message) {
	ctx, receiveBlockSpan := trace.StartSpan(msg.Ctx, "RegularSync_receiveBlock")
	defer receiveBlockSpan.End()

	response := msg.Data.(*pb.BeaconBlockResponse)
	block := response.Block
	blockHash, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		log.Errorf("Could not hash received block: %v", err)
	}

	log.Debugf("Processing response to block request: %#x", blockHash)

	if rs.db.HasBlock(blockHash) {
		log.Debug("Received a block that already exists. Exiting...")
		return
	}

	beaconState, err := rs.db.State()
	if err != nil {
		log.Errorf("Failed to get beacon state: %v", err)
		return
	}

	if block.Slot < beaconState.FinalizedEpoch*params.BeaconConfig().EpochLength {
		log.Debug("Discarding received block with a slot number smaller than the last finalized slot")
		return
	}

	_, sendBlockSpan := trace.StartSpan(ctx, "sendBlock")
	log.WithField("blockHash", fmt.Sprintf("%#x", blockHash)).Debug("Sending newly received block to subscribers")
	rs.chainService.IncomingBlockFeed().Send(block)
	sendBlockSpan.End()
}

// handleBlockRequestBySlot processes a block request from the p2p layer.
// if found, the block is sent to the requesting peer.
func (rs *RegularSync) handleBlockRequestBySlot(msg p2p.Message) {
	ctx, blockRequestSpan := trace.StartSpan(msg.Ctx, "RegularSync_blockRequestBySlot")
	defer blockRequestSpan.End()

	request, ok := msg.Data.(*pb.BeaconBlockRequestBySlotNumber)
	if !ok {
		log.Error("Received malformed beacon block request p2p message")
		return
	}

	ctx, getBlockSpan := trace.StartSpan(ctx, "getBlockBySlot")
	block, err := rs.db.BlockBySlot(request.SlotNumber)
	getBlockSpan.End()
	if err != nil || block == nil {
		log.Errorf("Error retrieving block from db: %v", err)
		return
	}

	_, sendBlockSpan := trace.StartSpan(ctx, "sendBlock")
	log.WithField("slotNumber", fmt.Sprintf("%d", request.SlotNumber)).Debug("Sending requested block to peer")
	rs.p2p.Send(&pb.BeaconBlockResponse{
		Block: block,
	}, msg.Peer)
	sendBlockSpan.End()
}

func (rs *RegularSync) handleChainHeadRequest(msg p2p.Message) {
	if _, ok := msg.Data.(*pb.ChainHeadRequest); !ok {
		log.Errorf("message is of the incorrect type")
		return
	}

	block, err := rs.db.ChainHead()
	if err != nil {
		log.Errorf("Could not retrieve chain head %v", err)
		return
	}

	hash, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		log.Errorf("Could not hash block %v", err)
		return
	}

	req := &pb.ChainHeadResponse{
		Slot:  block.Slot,
		Hash:  hash[:],
		Block: block,
	}

	rs.p2p.Send(req, msg.Peer)
}

// receiveAttestation accepts an broadcasted attestation from the p2p layer,
// discard the attestation if we have gotten before, send it to attestation
// service if we have not.
func (rs *RegularSync) receiveAttestation(msg p2p.Message) {
	data := msg.Data.(*pb.Attestation)
	a := data
	h := att.Key(a.Data)

	attestation, err := rs.db.Attestation(h)
	if err != nil {
		log.Errorf("Could not check for attestation in DB: %v", err)
		return
	}
	if rs.db.HasAttestation(h) {
		log.Debugf("Received, skipping attestation #%x", h)
		return
	}

	log.WithField("attestationHash", fmt.Sprintf("%#x", h)).Debug("Forwarding attestation to subscribed services")
	rs.operationsService.IncomingAttFeed().Send(attestation)
}

// receiveExitRequest accepts an broadcasted exit from the p2p layer,
// discard the exit if we have gotten before, send it to operation
// service if we have not.
func (rs *RegularSync) receiveExitRequest(msg p2p.Message) {
	exit := msg.Data.(*pb.Exit)
	h, err := hashutil.HashProto(exit)
	if err != nil {
		log.Errorf("Could not hash incoming exit request: %v", err)
		return
	}

	if rs.db.HasExit(h) {
		log.Debugf("Received, skipping exit request #%x", h)
		return
	}

	log.WithField("exitReqHash", fmt.Sprintf("%#x", h)).
		Debug("Forwarding validator exit request to subscribed services")
	rs.operationsService.IncomingExitFeed().Send(exit)
}

func (rs *RegularSync) handleBlockRequestByHash(msg p2p.Message) {
	data := msg.Data.(*pb.BeaconBlockRequest)

	hash := bytesutil.ToBytes32(data.Hash)

	block, err := rs.db.Block(hash)
	if err != nil {
		log.Error(err)
		return
	}
	if block == nil {
		log.Debug("Block does not exist")
		return
	}

	rs.p2p.Send(&pb.BeaconBlockResponse{
		Block: block,
	}, msg.Peer)

}

// handleBatchedBlockRequest receives p2p messages which consist of requests for batched blocks
// which are bounded by a start slot and end slot.
func (rs *RegularSync) handleBatchedBlockRequest(msg p2p.Message) {
	data := msg.Data.(*pb.BatchedBeaconBlockRequest)
	startSlot, endSlot := data.StartSlot, data.EndSlot

	block, err := rs.db.ChainHead()
	if err != nil {
		log.Errorf("Could not retrieve chain head %v", err)
		return
	}

	finalizedSlot, err := rs.db.CleanedFinalizedSlot()
	if err != nil {
		log.Errorf("Could not retrieve last finalized slot %v", err)
		return
	}

	currentSlot := block.Slot

	if currentSlot < startSlot || finalizedSlot > endSlot {
		log.Debugf(
			"invalid batch request: current slot < start slot || finalized slot > end slot."+
				"currentSlot %d startSlot %d endSlot %d finalizedSlot %d", currentSlot, startSlot, endSlot, finalizedSlot)
		return
	}

	response := make([]*pb.BeaconBlock, 0, endSlot-startSlot)

	for i := startSlot; i <= endSlot; i++ {
		block, err := rs.db.BlockBySlot(i)
		if err != nil {
			log.Errorf("Unable to retrieve block from db %v", err)
			continue
		}

		if block == nil {
			log.Debug("Block does not exist in db")
			continue
		}

		response = append(response, block)
	}

	log.Debugf("Sending response for batch blocks to peer %v", msg.Peer)
	rs.p2p.Send(&pb.BatchedBeaconBlockResponse{
		BatchedBlocks: response,
	}, msg.Peer)

}
