// Package sync defines the utilities for the beacon-chain to sync with the network.
package sync

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	log                           = logrus.WithField("prefix", "regular-sync")
	blocksAwaitingProcessingGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "regsync_blocks_awaiting_processing",
		Help: "Number of blocks which do not have a parent and are awaiting processing by the chain service",
	})
)

type chainService interface {
	blockchain.BlockProcessor
	blockchain.ForkChoice
	blockchain.ChainFeeds
}

type p2pAPI interface {
	p2p.Broadcaster
	p2p.Sender
	p2p.Subscriber
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
	operationsService        operations.OperationFeeds
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
	announceAttestationBuf   chan p2p.Message
	exitBuf                  chan p2p.Message
	canonicalBuf             chan *pb.BeaconBlockAnnounce
	highestObservedSlot      uint64
	blocksAwaitingProcessing map[[32]byte]*pb.BeaconBlock
}

// RegularSyncConfig allows the channel's buffer sizes to be changed.
type RegularSyncConfig struct {
	BlockAnnounceBufferSize     int
	BlockBufferSize             int
	BlockReqSlotBufferSize      int
	BlockReqHashBufferSize      int
	BatchedBufferSize           int
	StateReqBufferSize          int
	AttestationBufferSize       int
	AttestationReqHashBufSize   int
	AttestationsAnnounceBufSize int
	ExitBufferSize              int
	ChainHeadReqBufferSize      int
	CanonicalBufferSize         int
	ChainService                chainService
	OperationService            operations.OperationFeeds
	BeaconDB                    *db.BeaconDB
	P2P                         p2pAPI
}

// DefaultRegularSyncConfig provides the default configuration for a sync service.
func DefaultRegularSyncConfig() *RegularSyncConfig {
	return &RegularSyncConfig{
		BlockAnnounceBufferSize:     100,
		BlockBufferSize:             100,
		BlockReqSlotBufferSize:      100,
		BlockReqHashBufferSize:      100,
		BatchedBufferSize:           100,
		StateReqBufferSize:          100,
		ChainHeadReqBufferSize:      100,
		AttestationBufferSize:       100,
		AttestationReqHashBufSize:   100,
		AttestationsAnnounceBufSize: 100,
		ExitBufferSize:              100,
		CanonicalBufferSize:         100,
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
		announceAttestationBuf:   make(chan p2p.Message, cfg.AttestationsAnnounceBufSize),
		exitBuf:                  make(chan p2p.Message, cfg.ExitBufferSize),
		chainHeadReqBuf:          make(chan p2p.Message, cfg.ChainHeadReqBufferSize),
		canonicalBuf:             make(chan *pb.BeaconBlockAnnounce, cfg.CanonicalBufferSize),
		blocksAwaitingProcessing: make(map[[32]byte]*pb.BeaconBlock),
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
	announceAttestationSub := rs.p2p.Subscribe(&pb.AttestationAnnounce{}, rs.announceAttestationBuf)
	exitSub := rs.p2p.Subscribe(&pb.VoluntaryExit{}, rs.exitBuf)
	chainHeadReqSub := rs.p2p.Subscribe(&pb.ChainHeadRequest{}, rs.chainHeadReqBuf)
	canonicalBlockSub := rs.chainService.CanonicalBlockFeed().Subscribe(rs.canonicalBuf)

	defer announceBlockSub.Unsubscribe()
	defer blockSub.Unsubscribe()
	defer blockRequestSub.Unsubscribe()
	defer blockRequestHashSub.Unsubscribe()
	defer batchedBlockRequestSub.Unsubscribe()
	defer stateRequestSub.Unsubscribe()
	defer chainHeadReqSub.Unsubscribe()
	defer attestationSub.Unsubscribe()
	defer attestationReqSub.Unsubscribe()
	defer announceAttestationSub.Unsubscribe()
	defer exitSub.Unsubscribe()
	defer canonicalBlockSub.Unsubscribe()

	for {
		select {
		case <-rs.ctx.Done():
			log.Debug("Exiting goroutine")
			return
		case msg := <-rs.announceBlockBuf:
			safelyHandleMessage(rs.receiveBlockAnnounce, msg)
		case msg := <-rs.attestationBuf:
			safelyHandleMessage(rs.receiveAttestation, msg)
		case msg := <-rs.attestationReqByHashBuf:
			safelyHandleMessage(rs.handleAttestationRequestByHash, msg)
		case msg := <-rs.announceAttestationBuf:
			safelyHandleMessage(rs.handleAttestationAnnouncement, msg)
		case msg := <-rs.exitBuf:
			safelyHandleMessage(rs.receiveExitRequest, msg)
		case msg := <-rs.blockBuf:
			safelyHandleMessage(rs.receiveBlock, msg)
		case msg := <-rs.blockRequestBySlot:
			safelyHandleMessage(rs.handleBlockRequestBySlot, msg)
		case msg := <-rs.blockRequestByHash:
			safelyHandleMessage(rs.handleBlockRequestByHash, msg)
		case msg := <-rs.batchedRequestBuf:
			safelyHandleMessage(rs.handleBatchedBlockRequest, msg)
		case msg := <-rs.stateRequestBuf:
			safelyHandleMessage(rs.handleStateRequest, msg)
		case msg := <-rs.chainHeadReqBuf:
			safelyHandleMessage(rs.handleChainHeadRequest, msg)
		case blockAnnounce := <-rs.canonicalBuf:
			rs.broadcastCanonicalBlock(rs.ctx, blockAnnounce)
		}
	}
	log.Info("Exiting regular sync run()")
}

// safelyHandleMessage will recover and log any panic that occurs from the
// function argument.
func safelyHandleMessage(fn func(p2p.Message), msg p2p.Message) {
	defer func() {
		if r := recover(); r != nil {
			printedMsg := "message contains no data"
			if msg.Data != nil {
				printedMsg = proto.MarshalTextString(msg.Data)
			}
			log.WithFields(logrus.Fields{
				"r":   r,
				"msg": printedMsg,
			}).Error("Panicked when handling p2p message! Recovering...")

			debug.PrintStack()

			if msg.Ctx == nil {
				return
			}
			if span := trace.FromContext(msg.Ctx); span != nil {
				span.SetStatus(trace.Status{
					Code:    trace.StatusCodeInternal,
					Message: fmt.Sprintf("Panic: %v", r),
				})
			}
		}
	}()

	// Fingers crossed that it doesn't panic...
	fn(msg)
}

// receiveBlockAnnounce accepts a block hash.
// TODO(#175): New hashes are forwarded to other peers in the network, and
// the contents of the block are requested if the local chain doesn't have the block.
func (rs *RegularSync) receiveBlockAnnounce(msg p2p.Message) {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.receiveBlockAnnounce")
	defer span.End()
	recBlockAnnounce.Inc()

	data := msg.Data.(*pb.BeaconBlockAnnounce)
	h := bytesutil.ToBytes32(data.Hash[:32])

	if rs.db.HasBlock(h) {
		log.Debugf("Received a root for a block that has already been processed: %#x", h)
		return
	}

	log.WithField("blockRoot", fmt.Sprintf("%#x", h)).Debug("Received incoming block root, requesting full block data from sender")
	// Request the full block data from peer that sent the block hash.
	ctx, sendBlockRequestSpan := trace.StartSpan(ctx, "beacon-chain.sync.sendBlockRequest")
	if err := rs.p2p.Send(ctx, &pb.BeaconBlockRequest{Hash: h[:]}, msg.Peer); err != nil {
		log.Error(err)
	}
	sentBlockReq.Inc()
	sendBlockRequestSpan.End()
}

// receiveBlock processes a block from the p2p layer.
func (rs *RegularSync) receiveBlock(msg p2p.Message) {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.receiveBlock")
	defer span.End()
	recBlock.Inc()

	response := msg.Data.(*pb.BeaconBlockResponse)
	block := response.Block
	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		log.Errorf("Could not hash received block: %v", err)
		return
	}

	log.Debugf("Processing response to block request: %#x", blockRoot)
	if rs.db.HasBlock(blockRoot) {
		log.Debug("Received a block that already exists. Exiting...")
		return
	}

	beaconState, err := rs.db.State(ctx)
	if err != nil {
		log.Errorf("Failed to get beacon state: %v", err)
		return
	}

	if block.Slot < beaconState.FinalizedEpoch*params.BeaconConfig().SlotsPerEpoch {
		log.Debug("Discarding received block with a slot number smaller than the last finalized slot")
		return
	}

	// We check if we have the block's parents saved locally, if not, we store the block in a
	// pending processing map by hash and once we receive the parent, we process said parent AND then
	// we process the received block.
	parentRoot := bytesutil.ToBytes32(block.ParentRootHash32)
	if !rs.db.HasBlock(parentRoot) {
		rs.blocksAwaitingProcessing[parentRoot] = block
		blocksAwaitingProcessingGauge.Inc()
		rs.p2p.Broadcast(ctx, &pb.BeaconBlockRequest{Hash: parentRoot[:]})
		// We update the last observed slot to the received canonical block's slot.
		if block.Slot > rs.highestObservedSlot {
			rs.highestObservedSlot = block.Slot
		}
		return
	}

	if block.Slot < rs.highestObservedSlot {
		// If we receive a block from the past AND it corresponds to
		// a parent block of a block stored in the processing cache, that means we are
		// receiving a parent block which was missing from our db.
		if childBlock, ok := rs.blocksAwaitingProcessing[blockRoot]; ok {
			log.WithField("blockRoot", fmt.Sprintf("%#x", blockRoot)).Debug("Received missing block parent")
			delete(rs.blocksAwaitingProcessing, blockRoot)
			blocksAwaitingProcessingGauge.Dec()
			beaconState, err = rs.chainService.ReceiveBlock(ctx, block)
			if err != nil {
				log.Errorf("could not process beacon block: %v", err)
				return
			}
			if err := rs.chainService.ApplyForkChoiceRule(ctx, block, beaconState); err != nil {
				log.Errorf("could not apply fork choice rule: %v", err)
				return
			}
			beaconState, err = rs.chainService.ReceiveBlock(ctx, childBlock)
			if err != nil {
				log.Errorf("could not process beacon block: %v", err)
				return
			}
			if err := rs.chainService.ApplyForkChoiceRule(ctx, childBlock, beaconState); err != nil {
				log.Errorf("could not apply fork choice rule: %v", err)
				return
			}
			log.Debug("Sent missing block parent and child to chain service for processing")
			return
		}
	}

	ctx, sendBlockSpan := trace.StartSpan(ctx, "beacon-chain.sync.sendBlock")
	log.WithField("blockRoot", fmt.Sprintf("%#x", blockRoot)).Debug("Sending newly received block to chain service")
	beaconState, err = rs.chainService.ReceiveBlock(ctx, block)
	if err != nil {
		log.Errorf("could not process beacon block: %v", err)
		return
	}
	if err := rs.chainService.ApplyForkChoiceRule(ctx, block, beaconState); err != nil {
		log.Errorf("could not apply fork choice rule: %v", err)
		return
	}
	sentBlocks.Inc()
	sendBlockSpan.End()
	// We update the last observed slot to the received canonical block's slot.
	if block.Slot > rs.highestObservedSlot {
		rs.highestObservedSlot = block.Slot
	}
}

// handleBlockRequestBySlot processes a block request from the p2p layer.
// if found, the block is sent to the requesting peer.
func (rs *RegularSync) handleBlockRequestBySlot(msg p2p.Message) {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleBlockRequestBySlot")
	defer span.End()
	blockReqSlot.Inc()

	request, ok := msg.Data.(*pb.BeaconBlockRequestBySlotNumber)
	if !ok {
		log.Error("Received malformed beacon block request p2p message")
		return
	}

	ctx, getBlockSpan := trace.StartSpan(ctx, "getBlockBySlot")
	block, err := rs.db.BlockBySlot(request.SlotNumber)
	getBlockSpan.End()
	if err != nil || block == nil {
		if block == nil {
			log.Debugf("Block with slot %d does not exist", request.SlotNumber)
			return
		}
		log.Errorf("Error retrieving block from db: %v", err)
		return
	}

	ctx, sendBlockSpan := trace.StartSpan(ctx, "sendBlock")
	log.WithField("slotNumber",
		fmt.Sprintf("%d", request.SlotNumber-params.BeaconConfig().GenesisSlot)).Debug("Sending requested block to peer")
	if err := rs.p2p.Send(ctx, &pb.BeaconBlockResponse{
		Block: block,
	}, msg.Peer); err != nil {
		log.Error(err)
	}
	sentBlocks.Inc()
	sendBlockSpan.End()
}

func (rs *RegularSync) handleStateRequest(msg p2p.Message) {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleStateRequest")
	defer span.End()
	stateReq.Inc()
	req, ok := msg.Data.(*pb.BeaconStateRequest)
	if !ok {
		log.Errorf("Message is of the incorrect type")
		return
	}
	fState, err := rs.db.FinalizedState()
	if err != nil {
		log.Errorf("Unable to retrieve beacon state, %v", err)
		return
	}
	root, err := hashutil.HashProto(fState)
	if err != nil {
		log.Errorf("unable to marshal the beacon state: %v", err)
		return
	}
	if root != bytesutil.ToBytes32(req.FinalizedStateRootHash32S) {
		log.Debugf("Requested state root is different from locally stored state root %#x", req.FinalizedStateRootHash32S)
		return
	}
	ctx, sendStateSpan := trace.StartSpan(ctx, "beacon-chain.sync.sendState")
	log.WithField("beaconState", fmt.Sprintf("%#x", root)).Debug("Sending beacon state to peer")
	if err := rs.p2p.Send(ctx, &pb.BeaconStateResponse{BeaconState: fState}, msg.Peer); err != nil {
		log.Error(err)
	}
	sentState.Inc()
	sendStateSpan.End()
}

func (rs *RegularSync) handleChainHeadRequest(msg p2p.Message) {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleChainHeadRequest")
	defer span.End()
	chainHeadReq.Inc()
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

	finalizedState, err := rs.db.FinalizedState()
	if err != nil {
		log.Errorf("Could not retrieve finalized state %v", err)
		return
	}

	finalizedRoot, err := hashutil.HashProto(finalizedState)
	if err != nil {
		log.Errorf("Could not tree hash block %v", err)
		return
	}

	req := &pb.ChainHeadResponse{
		Slot:                      block.Slot,
		Hash:                      blockRoot[:],
		FinalizedStateRootHash32S: finalizedRoot[:],
	}
	ctx, ChainHead := trace.StartSpan(ctx, "sendChainHead")
	defer ChainHead.End()
	if err := rs.p2p.Send(ctx, req, msg.Peer); err != nil {
		log.Error(err)
	}
	sentChainHead.Inc()
}

// receiveAttestation accepts an broadcasted attestation from the p2p layer,
// discard the attestation if we have gotten before, send it to attestation
// pool if we have not.
func (rs *RegularSync) receiveAttestation(msg p2p.Message) {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.receiveAttestation")
	defer span.End()
	recAttestation.Inc()

	resp := msg.Data.(*pb.AttestationResponse)
	attestation := resp.Attestation
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
	beaconState, err := rs.db.State(ctx)
	if err != nil {
		log.Errorf("Failed to get beacon state: %v", err)
		return
	}

	if attestation.Data.Slot < beaconState.Slot-params.BeaconConfig().SlotsPerEpoch {
		log.Debugf("Skipping received attestation with slot smaller than one epoch ago, %d < %d",
			attestation.Data.Slot, beaconState.Slot-params.BeaconConfig().SlotsPerEpoch)
		return
	}

	ctx, sendAttestationSpan := trace.StartSpan(ctx, "beacon-chain.sync.sendAttestation")
	log.WithField("attestationHash", fmt.Sprintf("%#x", attestationRoot)).Debug("Sending newly received attestation to subscribers")
	rs.operationsService.IncomingAttFeed().Send(attestation)
	sentAttestation.Inc()
	sendAttestationSpan.End()
}

// receiveExitRequest accepts an broadcasted exit from the p2p layer,
// discard the exit if we have gotten before, send it to operation
// service if we have not.
func (rs *RegularSync) receiveExitRequest(msg p2p.Message) {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.receiveExitRequest")
	defer span.End()
	recExit.Inc()
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
	ctx, sendExitReqSpan := trace.StartSpan(ctx, "sendExitRequest")
	log.WithField("exitReqHash", fmt.Sprintf("%#x", h)).
		Debug("Forwarding validator exit request to subscribed services")
	rs.operationsService.IncomingExitFeed().Send(exit)
	sentExit.Inc()
	sendExitReqSpan.End()
}

func (rs *RegularSync) handleBlockRequestByHash(msg p2p.Message) {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleBlockRequestByHash")
	defer span.End()
	blockReqHash.Inc()

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

	ctx, sendBlockSpan := trace.StartSpan(ctx, "sendBlock")
	defer sendBlockSpan.End()
	if err := rs.p2p.Send(ctx, &pb.BeaconBlockResponse{
		Block: block,
	}, msg.Peer); err != nil {
		log.Error(err)
	}
	sentBlocks.Inc()
}

// handleBatchedBlockRequest receives p2p messages which consist of requests for batched blocks
// which are bounded by a start slot and end slot.
func (rs *RegularSync) handleBatchedBlockRequest(msg p2p.Message) {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleBatchedBlockRequest")
	defer span.End()
	batchedBlockReq.Inc()
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

	blockRange := endSlot - startSlot
	// Handle overflows
	if startSlot > endSlot {
		// Do not process requests with invalid slot ranges
		log.Debugf("Batched block range is invalid, start slot %d , end slot %d", startSlot, endSlot)
		return
	}

	response := make([]*pb.BeaconBlock, 0, blockRange)
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

	ctx, sendBatchedBlockSpan := trace.StartSpan(ctx, "sendBatchedBlocks")
	defer sendBatchedBlockSpan.End()
	log.Debugf("Sending response for batch blocks to peer %v", msg.Peer)
	if err := rs.p2p.Send(ctx, &pb.BatchedBeaconBlockResponse{
		BatchedBlocks: response,
	}, msg.Peer); err != nil {
		log.Error(err)
	}
	sentBatchedBlocks.Inc()
}

func (rs *RegularSync) handleAttestationRequestByHash(msg p2p.Message) {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleAttestationRequestByHash")
	defer span.End()
	attestationReq.Inc()

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

	ctx, sendAttestationSpan := trace.StartSpan(ctx, "sendAttestation")
	defer sendAttestationSpan.End()
	log.Debugf("Sending attestation %#x to peer %v", root, msg.Peer)
	if err := rs.p2p.Send(ctx, &pb.AttestationResponse{
		Attestation: att,
	}, msg.Peer); err != nil {
		log.Error(err)
	}
	sentAttestation.Inc()
}

// handleAttestationAnnouncement will process the incoming p2p message. The
// behavior here is that we've just received an announcement of a new
// attestation and we're given the hash of that new attestation. If we don't
// have this attestation yet in our database, request the attestation from the
// sending peer.
func (rs *RegularSync) handleAttestationAnnouncement(msg p2p.Message) {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleAttestationAnnouncement")
	defer span.End()
	data, ok := msg.Data.(*pb.AttestationAnnounce)
	if !ok {
		log.Errorf("message is of the incorrect type")
		return
	}

	if rs.db.HasAttestation(bytesutil.ToBytes32(data.Hash)) {
		return
	}

	log.Debugf("Sending request for attestation to peer %v", msg.Peer)
	if err := rs.p2p.Send(ctx, &pb.AttestationRequest{
		Hash: data.Hash,
	}, msg.Peer); err != nil {
		log.Error(err)
	}
}

func (rs *RegularSync) broadcastCanonicalBlock(ctx context.Context, announce *pb.BeaconBlockAnnounce) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.sync.broadcastCanonicalBlock")
	defer span.End()
	log.Debugf("Announcing canonical block %#x", announce.Hash)
	rs.p2p.Broadcast(ctx, announce)
	sentBlockAnnounce.Inc()
}
