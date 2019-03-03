// Package sync defines the utilities for the beacon-chain to sync with the network.
package sync

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
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
	StateInitializedFeed() *event.Feed
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
	ctx                      context.Context
	cancel                   context.CancelFunc
	p2p                      p2pAPI
	chainService             chainService
	operationsService        operationService
	db                       *db.BeaconDB
	blockAnnouncementFeed    *event.Feed
	announceBlockBuf         chan p2p.Message
	blockBuf                 chan p2p.Message
	blockRequestBySlot       chan p2p.Message
	blockRequestByHash       chan p2p.Message
	batchedRequestBuf        chan p2p.Message
	stateRequestBuf          chan p2p.Message
	chainHeadReqBuf          chan p2p.Message
	attestationBuf           chan p2p.Message
	attestationReqByHashBuf  chan p2p.Message
	unseenAttestationsReqBuf chan p2p.Message
	exitBuf                  chan p2p.Message
}

// RegularSyncConfig allows the channel's buffer sizes to be changed.
type RegularSyncConfig struct {
	BlockAnnounceBufferSize      int
	BlockBufferSize              int
	BlockReqSlotBufferSize       int
	BlockReqHashBufferSize       int
	BatchedBufferSize            int
	StateReqBufferSize           int
	AttestationBufferSize        int
	AttestationReqHashBufSize    int
	UnseenAttestationsReqBufSize int
	ExitBufferSize               int
	ChainHeadReqBufferSize       int
	ChainService                 chainService
	OperationService             operationService
	BeaconDB                     *db.BeaconDB
	P2P                          p2pAPI
}

// DefaultRegularSyncConfig provides the default configuration for a sync service.
func DefaultRegularSyncConfig() *RegularSyncConfig {
	return &RegularSyncConfig{
		BlockAnnounceBufferSize:      100,
		BlockBufferSize:              100,
		BlockReqSlotBufferSize:       100,
		BlockReqHashBufferSize:       100,
		BatchedBufferSize:            100,
		StateReqBufferSize:           100,
		ChainHeadReqBufferSize:       100,
		AttestationBufferSize:        100,
		AttestationReqHashBufSize:    100,
		UnseenAttestationsReqBufSize: 100,
		ExitBufferSize:               100,
	}
}

// NewRegularSyncService accepts a context and returns a new Service.
func NewRegularSyncService(ctx context.Context, cfg *RegularSyncConfig) *RegularSync {
	ctx, cancel := context.WithCancel(ctx)
	return &RegularSync{
		ctx:                      ctx,
		cancel:                   cancel,
		p2p:                      cfg.P2P,
		chainService:             cfg.ChainService,
		db:                       cfg.BeaconDB,
		operationsService:        cfg.OperationService,
		blockAnnouncementFeed:    new(event.Feed),
		announceBlockBuf:         make(chan p2p.Message, cfg.BlockAnnounceBufferSize),
		blockBuf:                 make(chan p2p.Message, cfg.BlockBufferSize),
		blockRequestBySlot:       make(chan p2p.Message, cfg.BlockReqSlotBufferSize),
		blockRequestByHash:       make(chan p2p.Message, cfg.BlockReqHashBufferSize),
		batchedRequestBuf:        make(chan p2p.Message, cfg.BatchedBufferSize),
		stateRequestBuf:          make(chan p2p.Message, cfg.StateReqBufferSize),
		attestationBuf:           make(chan p2p.Message, cfg.AttestationBufferSize),
		attestationReqByHashBuf:  make(chan p2p.Message, cfg.AttestationReqHashBufSize),
		unseenAttestationsReqBuf: make(chan p2p.Message, cfg.UnseenAttestationsReqBufSize),
		exitBuf:                  make(chan p2p.Message, cfg.ExitBufferSize),
		chainHeadReqBuf:          make(chan p2p.Message, cfg.ChainHeadReqBufferSize),
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
	batchedBlockRequestSub := rs.p2p.Subscribe(&pb.BatchedBeaconBlockRequest{}, rs.batchedRequestBuf)
	stateRequestSub := rs.p2p.Subscribe(&pb.BeaconStateRequest{}, rs.stateRequestBuf)
	attestationSub := rs.p2p.Subscribe(&pb.AttestationResponse{}, rs.attestationBuf)
	attestationReqSub := rs.p2p.Subscribe(&pb.AttestationRequest{}, rs.attestationReqByHashBuf)
	unseenAttestationsReqSub := rs.p2p.Subscribe(&pb.UnseenAttestationsRequest{}, rs.unseenAttestationsReqBuf)
	exitSub := rs.p2p.Subscribe(&pb.VoluntaryExit{}, rs.exitBuf)
	chainHeadReqSub := rs.p2p.Subscribe(&pb.ChainHeadRequest{}, rs.chainHeadReqBuf)

	defer announceBlockSub.Unsubscribe()
	defer blockSub.Unsubscribe()
	defer blockRequestSub.Unsubscribe()
	defer blockRequestHashSub.Unsubscribe()
	defer batchedBlockRequestSub.Unsubscribe()
	defer stateRequestSub.Unsubscribe()
	defer chainHeadReqSub.Unsubscribe()
	defer attestationSub.Unsubscribe()
	defer attestationReqSub.Unsubscribe()
	defer unseenAttestationsReqSub.Unsubscribe()
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
		case msg := <-rs.attestationReqByHashBuf:
			rs.handleAttestationRequestByHash(msg)
		case msg := <-rs.unseenAttestationsReqBuf:
			rs.handleUnseenAttestationsRequest(msg)
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
		case msg := <-rs.stateRequestBuf:
			rs.handleStateRequest(msg)
		case msg := <-rs.chainHeadReqBuf:
			rs.handleChainHeadRequest(msg)
		}
	}
}

// receiveBlockAnnounce accepts a block hash.
// TODO(#175): New hashes are forwarded to other peers in the network, and
// the contents of the block are requested if the local chain doesn't have the block.
func (rs *RegularSync) receiveBlockAnnounce(msg p2p.Message) {
	ctx, receiveBlockSpan := trace.StartSpan(msg.Ctx, "RegularSync_receiveBlockRoot")
	defer receiveBlockSpan.End()

	data := msg.Data.(*pb.BeaconBlockAnnounce)
	h := bytesutil.ToBytes32(data.Hash[:32])

	if rs.db.HasBlock(h) {
		log.Debugf("Received a root for a block that has already been processed: %#x", h)
		return
	}

	log.WithField("blockRoot", fmt.Sprintf("%#x", h)).Debug("Received incoming block root, requesting full block data from sender")
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
	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		log.Errorf("Could not hash received block: %v", err)
	}

	log.Debugf("Processing response to block request: %#x", blockRoot)

	if rs.db.HasBlock(blockRoot) {
		log.Debug("Received a block that already exists. Exiting...")
		return
	}

	beaconState, err := rs.db.State(msg.Ctx)
	if err != nil {
		log.Errorf("Failed to get beacon state: %v", err)
		return
	}

	if block.Slot < beaconState.FinalizedEpoch*params.BeaconConfig().SlotsPerEpoch {
		log.Debug("Discarding received block with a slot number smaller than the last finalized slot")
		return
	}

	_, sendBlockSpan := trace.StartSpan(ctx, "sendBlock")
	log.WithField("blockRoot", fmt.Sprintf("%#x", blockRoot)).Debug("Sending newly received block to subscribers")
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
	log.WithField("slotNumber",
		fmt.Sprintf("%d", request.SlotNumber-params.BeaconConfig().GenesisSlot)).Debug("Sending requested block to peer")
	rs.p2p.Send(&pb.BeaconBlockResponse{
		Block: block,
	}, msg.Peer)
	sendBlockSpan.End()
}

func (rs *RegularSync) handleStateRequest(msg p2p.Message) {
	ctx, handleStateReqSpan := trace.StartSpan(msg.Ctx, "RegularSync_handleStateReq")
	defer handleStateReqSpan.End()
	req, ok := msg.Data.(*pb.BeaconStateRequest)
	if !ok {
		log.Errorf("Message is of the incorrect type")
		return
	}
	state, err := rs.db.State(msg.Ctx)
	if err != nil {
		log.Errorf("Unable to retrieve beacon state, %v", err)
		return
	}
	root, err := hashutil.HashProto(state)
	if err != nil {
		log.Errorf("unable to marshal the beacon state: %v", err)
		return
	}
	if root != bytesutil.ToBytes32(req.Hash) {
		log.Debugf("Requested state root is different from locally stored state root %#x", req.Hash)
		return
	}
	_, sendStateSpan := trace.StartSpan(ctx, "sendState")
	log.WithField("beaconState", fmt.Sprintf("%#x", root)).Debug("Sending beacon state to peer")
	rs.p2p.Send(&pb.BeaconStateResponse{BeaconState: state}, msg.Peer)
	sendStateSpan.End()
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

	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		log.Errorf("Could not tree hash block %v", err)
		return
	}

	req := &pb.ChainHeadResponse{
		Slot:  block.Slot,
		Hash:  blockRoot[:],
		Block: block,
	}

	rs.p2p.Send(req, msg.Peer)
}

// receiveAttestation accepts an broadcasted attestation from the p2p layer,
// discard the attestation if we have gotten before, send it to attestation
// pool if we have not.
func (rs *RegularSync) receiveAttestation(msg p2p.Message) {
	ctx, receiveAttestationSpan := trace.StartSpan(msg.Ctx, "RegularSync_receiveAttestation")
	defer receiveAttestationSpan.End()

	attestation := msg.Data.(*pb.Attestation)
	attestationRoot, err := hashutil.HashProto(attestation)
	if err != nil {
		log.Errorf("Could not hash received attestation: %v", err)
	}

	// Skip if attestation has been seen before.
	if rs.db.HasAttestation(attestationRoot) {
		log.Debugf("Received, skipping attestation #%x", attestationRoot)
		return
	}

	// Skip if attestation slot is older than last finalized slot in state.
	beaconState, err := rs.db.State(msg.Ctx)
	if err != nil {
		log.Errorf("Failed to get beacon state: %v", err)
		return
	}

	finalizedSlot := beaconState.FinalizedEpoch * params.BeaconConfig().SlotsPerEpoch
	if attestation.Data.Slot < finalizedSlot {
		log.Debugf("Skipping received attestation with slot smaller than last finalized slot, %d < %d",
			attestation.Data.Slot, finalizedSlot)
		return
	}

	_, sendAttestationSpan := trace.StartSpan(ctx, "sendAttestation")
	log.WithField("attestationHash", fmt.Sprintf("%#x", attestationRoot)).Debug("Sending newly received attestation to subscribers")
	rs.operationsService.IncomingAttFeed().Send(attestation)
	sendAttestationSpan.End()
}

// receiveExitRequest accepts an broadcasted exit from the p2p layer,
// discard the exit if we have gotten before, send it to operation
// service if we have not.
func (rs *RegularSync) receiveExitRequest(msg p2p.Message) {
	exit := msg.Data.(*pb.VoluntaryExit)
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

	root := bytesutil.ToBytes32(data.Hash)

	block, err := rs.db.Block(root)
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
		retBlock, err := rs.db.BlockBySlot(i)
		if err != nil {
			log.Errorf("Unable to retrieve block from db %v", err)
			continue
		}

		if retBlock == nil {
			log.Debug("Block does not exist in db")
			continue
		}

		response = append(response, retBlock)
	}

	log.Debugf("Sending response for batch blocks to peer %v", msg.Peer)
	rs.p2p.Send(&pb.BatchedBeaconBlockResponse{
		BatchedBlocks: response,
	}, msg.Peer)
}

func (rs *RegularSync) handleAttestationRequestByHash(msg p2p.Message) {
	ctx, respondAttestationSpan := trace.StartSpan(msg.Ctx, "RegularSync_respondAttestation")
	defer respondAttestationSpan.End()
	req := msg.Data.(*pb.AttestationRequest)

	root := bytesutil.ToBytes32(req.Hash)

	att, err := rs.db.Attestation(root)
	if err != nil {
		log.Error(err)
		return
	}
	if att == nil {
		log.Debugf("Attestation %#x is not in db", root)
		return
	}

	_, sendAttestationSpan := trace.StartSpan(ctx, "sendAttestation")
	log.Debugf("Sending attestation %#x to peer %v", root, msg.Peer)
	rs.p2p.Send(&pb.AttestationResponse{
		Attestation: att,
	}, msg.Peer)
	sendAttestationSpan.End()
}

func (rs *RegularSync) handleUnseenAttestationsRequest(msg p2p.Message) {
	ctx, respondAttestationxSpan := trace.StartSpan(msg.Ctx, "RegularSync_respondUnseenAttestations")
	defer respondAttestationxSpan.End()
	if _, ok := msg.Data.(*pb.UnseenAttestationsRequest); !ok {
		log.Errorf("message is of the incorrect type")
		return
	}

	atts, err := rs.db.Attestations()
	if err != nil {
		log.Error(err)
		return
	}

	if len(atts) == 0 {
		log.Debug("There's no unseen attestation in db")
		return
	}

	_, sendAttestationsSpan := trace.StartSpan(ctx, "sendAttestation")
	log.Debugf("Sending response for batched unseen attestations to peer %v", msg.Peer)
	rs.p2p.Send(&pb.UnseenAttestationResponse{
		Attestations: atts,
	}, msg.Peer)
	sendAttestationsSpan.End()
}
