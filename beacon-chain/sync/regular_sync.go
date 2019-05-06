// Package sync defines the utilities for the beacon-chain to sync with the network.
package sync

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
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
	blockchain.BlockReceiver
	blockchain.BlockProcessor
	blockchain.ForkChoice
	blockchain.ChainFeeds
}

type attsService interface {
	IncomingAttestationFeed() *event.Feed
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
	ctx                          context.Context
	cancel                       context.CancelFunc
	p2p                          p2pAPI
	chainService                 chainService
	attsService                  attsService
	operationsService            operations.OperationFeeds
	db                           *db.BeaconDB
	blockAnnouncementFeed        *event.Feed
	announceBlockBuf             chan p2p.Message
	blockBuf                     chan p2p.Message
	blockRequestBySlot           chan p2p.Message
	blockRequestByHash           chan p2p.Message
	batchedRequestBuf            chan p2p.Message
	stateRequestBuf              chan p2p.Message
	chainHeadReqBuf              chan p2p.Message
	attestationBuf               chan p2p.Message
	attestationReqByHashBuf      chan p2p.Message
	announceAttestationBuf       chan p2p.Message
	exitBuf                      chan p2p.Message
	canonicalBuf                 chan *pb.BeaconBlockAnnounce
	highestObservedSlot          uint64
	blocksAwaitingProcessing     map[[32]byte]p2p.Message
	blocksAwaitingProcessingLock sync.RWMutex
	blockProcessingLock          sync.RWMutex
	blockAnnouncements           map[uint64][]byte
	blockAnnouncementsLock       sync.RWMutex
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
	AttsService                 attsService
	BeaconDB                    *db.BeaconDB
	P2P                         p2pAPI
}

// DefaultRegularSyncConfig provides the default configuration for a sync service.
func DefaultRegularSyncConfig() *RegularSyncConfig {
	return &RegularSyncConfig{
		BlockAnnounceBufferSize:     params.BeaconConfig().DefaultBufferSize,
		BlockBufferSize:             params.BeaconConfig().DefaultBufferSize,
		BlockReqSlotBufferSize:      params.BeaconConfig().DefaultBufferSize,
		BlockReqHashBufferSize:      params.BeaconConfig().DefaultBufferSize,
		BatchedBufferSize:           params.BeaconConfig().DefaultBufferSize,
		StateReqBufferSize:          params.BeaconConfig().DefaultBufferSize,
		ChainHeadReqBufferSize:      params.BeaconConfig().DefaultBufferSize,
		AttestationBufferSize:       params.BeaconConfig().DefaultBufferSize,
		AttestationReqHashBufSize:   params.BeaconConfig().DefaultBufferSize,
		AttestationsAnnounceBufSize: params.BeaconConfig().DefaultBufferSize,
		ExitBufferSize:              params.BeaconConfig().DefaultBufferSize,
		CanonicalBufferSize:         params.BeaconConfig().DefaultBufferSize,
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
		attsService:              cfg.AttsService,
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
		blocksAwaitingProcessing: make(map[[32]byte]p2p.Message),
		blockAnnouncements:       make(map[uint64][]byte),
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
			go safelyHandleMessage(rs.receiveBlockAnnounce, msg)
		case msg := <-rs.attestationBuf:
			go safelyHandleMessage(rs.receiveAttestation, msg)
		case msg := <-rs.attestationReqByHashBuf:
			go safelyHandleMessage(rs.handleAttestationRequestByHash, msg)
		case msg := <-rs.announceAttestationBuf:
			go safelyHandleMessage(rs.handleAttestationAnnouncement, msg)
		case msg := <-rs.exitBuf:
			go safelyHandleMessage(rs.receiveExitRequest, msg)
		case msg := <-rs.blockBuf:
			go safelyHandleMessage(rs.receiveBlock, msg)
		case msg := <-rs.blockRequestBySlot:
			go safelyHandleMessage(rs.handleBlockRequestBySlot, msg)
		case msg := <-rs.blockRequestByHash:
			go safelyHandleMessage(rs.handleBlockRequestByHash, msg)
		case msg := <-rs.batchedRequestBuf:
			go safelyHandleMessage(rs.handleBatchedBlockRequest, msg)
		case msg := <-rs.stateRequestBuf:
			go safelyHandleMessage(rs.handleStateRequest, msg)
		case msg := <-rs.chainHeadReqBuf:
			go safelyHandleMessage(rs.handleChainHeadRequest, msg)
		case blockAnnounce := <-rs.canonicalBuf:
			go rs.broadcastCanonicalBlock(rs.ctx, blockAnnounce)
		}
	}
}

// safelyHandleMessage will recover and log any panic that occurs from the
// function argument.
func safelyHandleMessage(fn func(p2p.Message) error, msg p2p.Message) {
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
	if err := fn(msg); err != nil {
		// Report any error to the span, if one exists.
		if span := trace.FromContext(msg.Ctx); span != nil {
			span.SetStatus(trace.Status{
				Code:    trace.StatusCodeInternal,
				Message: err.Error(),
			})
		}
	}
}

// handleBlockRequestBySlot processes a block request from the p2p layer.
// if found, the block is sent to the requesting peer.
func (rs *RegularSync) handleBlockRequestBySlot(msg p2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleBlockRequestBySlot")
	defer span.End()
	blockReqSlot.Inc()

	request, ok := msg.Data.(*pb.BeaconBlockRequestBySlotNumber)
	if !ok {
		log.Error("Received malformed beacon block request p2p message")
		return errors.New("incoming message is not type *pb.BeaconBlockRequestBySlotNumber")
	}

	block, err := rs.db.BlockBySlot(ctx, request.SlotNumber)
	if err != nil || block == nil {
		if block == nil {
			log.WithField("slot", request.SlotNumber-params.BeaconConfig().GenesisSlot).Debug(
				"block does not exist")
			return errors.New("block does not exist")
		}
		log.Errorf("Error retrieving block from db: %v", err)
		return err
	}

	log.WithField("slot",
		fmt.Sprintf("%d", request.SlotNumber-params.BeaconConfig().GenesisSlot)).Debug("Sending requested block to peer")

	defer sentBlocks.Inc()
	if err := rs.p2p.Send(ctx, &pb.BeaconBlockResponse{
		Block: block,
	}, msg.Peer); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (rs *RegularSync) handleStateRequest(msg p2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleStateRequest")
	defer span.End()
	stateReq.Inc()
	req, ok := msg.Data.(*pb.BeaconStateRequest)
	if !ok {
		log.Error("Message is of the incorrect type")
		return errors.New("incoming message is not *pb.BeaconStateRequest")
	}
	fState, err := rs.db.FinalizedState()
	if err != nil {
		log.Errorf("Unable to retrieve beacon state, %v", err)
		return err
	}
	root, err := hashutil.HashProto(fState)
	if err != nil {
		log.Errorf("unable to marshal the beacon state: %v", err)
		return err
	}
	if root != bytesutil.ToBytes32(req.FinalizedStateRootHash32S) {
		log.WithFields(logrus.Fields{
			"requested": fmt.Sprintf("%#x", req.FinalizedStateRootHash32S),
			"local":     fmt.Sprintf("%#x", root)},
		).Debug("Requested state root is diff than local state root")
		return err
	}
	log.WithField(
		"beaconState", fmt.Sprintf("%#x", root),
	).Debug("Sending finalized, justified, and canonical states to peer")
	defer sentState.Inc()
	resp := &pb.BeaconStateResponse{
		FinalizedState: fState,
	}
	if err := rs.p2p.Send(ctx, resp, msg.Peer); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (rs *RegularSync) handleChainHeadRequest(msg p2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleChainHeadRequest")
	defer span.End()
	chainHeadReq.Inc()
	if _, ok := msg.Data.(*pb.ChainHeadRequest); !ok {
		log.Error("message is of the incorrect type")
		return errors.New("incoming message is not *pb.ChainHeadRequest")
	}

	block, err := rs.db.ChainHead()
	if err != nil {
		log.Errorf("Could not retrieve chain head %v", err)
		return err
	}

	stateRoot := rs.db.HeadStateRoot()
	finalizedState, err := rs.db.FinalizedState()
	if err != nil {
		log.Errorf("Could not retrieve finalized state %v", err)
		return err
	}

	finalizedRoot, err := hashutil.HashProto(finalizedState)
	if err != nil {
		log.Errorf("Could not tree hash block %v", err)
		return err
	}

	req := &pb.ChainHeadResponse{
		CanonicalSlot:             block.Slot,
		CanonicalStateRootHash32:  stateRoot[:],
		FinalizedStateRootHash32S: finalizedRoot[:],
	}
	ctx, ChainHead := trace.StartSpan(ctx, "sendChainHead")
	defer ChainHead.End()
	defer sentChainHead.Inc()
	if err := rs.p2p.Send(ctx, req, msg.Peer); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// receiveAttestation accepts an broadcasted attestation from the p2p layer,
// discard the attestation if we have gotten before, send it to attestation
// pool if we have not.
func (rs *RegularSync) receiveAttestation(msg p2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.receiveAttestation")
	defer span.End()
	recAttestation.Inc()

	resp := msg.Data.(*pb.AttestationResponse)
	attestation := resp.Attestation
	attestationRoot, err := hashutil.HashProto(attestation)
	if err != nil {
		log.Errorf("Could not hash received attestation: %v", err)
		return err
	}
	log.WithFields(logrus.Fields{
		"headRoot":       fmt.Sprintf("%#x", bytesutil.Trunc(attestation.Data.BeaconBlockRootHash32)),
		"justifiedEpoch": attestation.Data.JustifiedEpoch - params.BeaconConfig().GenesisEpoch,
	}).Debug("Received an attestation")

	// Skip if attestation has been seen before.
	hasAttestation := rs.db.HasAttestation(attestationRoot)
	span.AddAttributes(trace.BoolAttribute("hasAttestation", hasAttestation))
	if hasAttestation {
		log.WithField("attestationRoot", fmt.Sprintf("%#x", bytesutil.Trunc(attestationRoot[:]))).
			Debug("Received, skipping attestation")
		return nil
	}

	// Skip if attestation slot is older than last finalized slot in state.
	highestSlot := rs.db.HighestBlockSlot()

	span.AddAttributes(
		trace.Int64Attribute("attestation.Data.Slot", int64(attestation.Data.Slot)),
		trace.Int64Attribute("finalized state slot", int64(highestSlot-params.BeaconConfig().SlotsPerEpoch)),
	)
	if attestation.Data.Slot < highestSlot-params.BeaconConfig().SlotsPerEpoch {
		log.WithFields(logrus.Fields{
			"receivedSlot": attestation.Data.Slot,
			"epochSlot":    highestSlot - params.BeaconConfig().SlotsPerEpoch},
		).Debug("Skipping received attestation with slot smaller than one epoch ago")
		return nil
	}

	_, sendAttestationSpan := trace.StartSpan(ctx, "beacon-chain.sync.sendAttestation")
	log.Debug("Sending newly received attestation to subscribers")
	rs.operationsService.IncomingAttFeed().Send(attestation)
	rs.attsService.IncomingAttestationFeed().Send(attestation)
	sentAttestation.Inc()
	sendAttestationSpan.End()
	return nil
}

// receiveExitRequest accepts an broadcasted exit from the p2p layer,
// discard the exit if we have gotten before, send it to operation
// service if we have not.
func (rs *RegularSync) receiveExitRequest(msg p2p.Message) error {
	_, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.receiveExitRequest")
	defer span.End()
	recExit.Inc()
	exit := msg.Data.(*pb.VoluntaryExit)
	h, err := hashutil.HashProto(exit)
	if err != nil {
		log.Errorf("Could not hash incoming exit request: %v", err)
		return err
	}

	hasExit := rs.db.HasExit(h)
	span.AddAttributes(trace.BoolAttribute("hasExit", hasExit))
	if hasExit {
		log.WithField("exitRoot", fmt.Sprintf("%#x", h)).
			Debug("Received, skipping exit request")
		return nil
	}
	log.WithField("exitReqHash", fmt.Sprintf("%#x", h)).
		Debug("Forwarding validator exit request to subscribed services")
	rs.operationsService.IncomingExitFeed().Send(exit)
	sentExit.Inc()
	return nil
}

func (rs *RegularSync) handleBlockRequestByHash(msg p2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleBlockRequestByHash")
	defer span.End()
	blockReqHash.Inc()

	data := msg.Data.(*pb.BeaconBlockRequest)
	root := bytesutil.ToBytes32(data.Hash)
	block, err := rs.db.Block(root)
	if err != nil {
		log.Error(err)
		return err
	}
	if block == nil {
		return errors.New("block does not exist")
	}

	defer sentBlocks.Inc()
	if err := rs.p2p.Send(ctx, &pb.BeaconBlockResponse{
		Block: block,
	}, msg.Peer); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// handleBatchedBlockRequest receives p2p messages which consist of requests for batched blocks
// which are bounded by a start slot and end slot.
func (rs *RegularSync) handleBatchedBlockRequest(msg p2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleBatchedBlockRequest")
	defer span.End()
	batchedBlockReq.Inc()
	data := msg.Data.(*pb.BatchedBeaconBlockRequest)
	startSlot, endSlot := data.StartSlot, data.EndSlot

	block, err := rs.db.ChainHead()
	if err != nil {
		log.Errorf("Could not retrieve chain head %v", err)
		return err
	}

	bState, err := rs.db.HeadState(ctx)
	if err != nil {
		log.Errorf("Could not retrieve last finalized slot %v", err)
		return err
	}

	finalizedSlot := helpers.StartSlot(bState.FinalizedEpoch)

	currentSlot := block.Slot
	if currentSlot < startSlot || finalizedSlot > endSlot {
		log.WithFields(logrus.Fields{
			"currentSlot":   currentSlot,
			"startSlot":     startSlot,
			"endSlot":       endSlot,
			"finalizedSlot": finalizedSlot},
		).Debug("invalid batch request: current slot < start slot || finalized slot > end slot")
		return err
	}

	blockRange := endSlot - startSlot
	// Handle overflows
	if startSlot > endSlot {
		// Do not process requests with invalid slot ranges
		log.WithFields(logrus.Fields{
			"slotSlot": startSlot - params.BeaconConfig().GenesisSlot,
			"endSlot":  endSlot - params.BeaconConfig().GenesisSlot},
		).Debug("Invalid batched block range")
		return err
	}

	response := make([]*pb.BeaconBlock, 0, blockRange)
	for i := startSlot; i <= endSlot; i++ {
		retBlock, err := rs.chainService.CanonicalBlock(i)
		if err != nil {
			log.Errorf("Unable to retrieve canonical block %v", err)
			continue
		}
		if retBlock == nil {
			log.WithField("slot", i).
				Debug("Canonical block does not exist")
			continue
		}
		response = append(response, retBlock)
	}

	log.WithField("peer", msg.Peer).
		Debug("Sending response for batch blocks")

	defer sentBatchedBlocks.Inc()
	if err := rs.p2p.Send(ctx, &pb.BatchedBeaconBlockResponse{
		BatchedBlocks: response,
	}, msg.Peer); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (rs *RegularSync) handleAttestationRequestByHash(msg p2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleAttestationRequestByHash")
	defer span.End()
	attestationReq.Inc()

	req := msg.Data.(*pb.AttestationRequest)
	root := bytesutil.ToBytes32(req.Hash)
	att, err := rs.db.Attestation(root)
	if err != nil {
		log.Error(err)
		return err
	}
	span.AddAttributes(trace.BoolAttribute("hasAttestation", att == nil))
	if att == nil {
		log.WithField("attestationRoot", fmt.Sprintf("%#x", bytesutil.Trunc(root[:]))).
			Debug("Attestation not in db")
		return nil
	}

	log.WithFields(logrus.Fields{
		"attestationRoot": fmt.Sprintf("%#x", bytesutil.Trunc(root[:])),
		"peer":            msg.Peer},
	).Debug("Sending attestation to peer")
	if err := rs.p2p.Send(ctx, &pb.AttestationResponse{
		Attestation: att,
	}, msg.Peer); err != nil {
		log.Error(err)
		return err
	}
	sentAttestation.Inc()
	return nil
}

// handleAttestationAnnouncement will process the incoming p2p message. The
// behavior here is that we've just received an announcement of a new
// attestation and we're given the hash of that new attestation. If we don't
// have this attestation yet in our database, request the attestation from the
// sending peer.
func (rs *RegularSync) handleAttestationAnnouncement(msg p2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleAttestationAnnouncement")
	defer span.End()
	data, ok := msg.Data.(*pb.AttestationAnnounce)
	if !ok {
		log.Errorf("message is of the incorrect type")
		return errors.New("incoming message is not of type *pb.AttestationAnnounce")
	}

	hasAttestation := rs.db.HasAttestation(bytesutil.ToBytes32(data.Hash))
	span.AddAttributes(trace.BoolAttribute("hasAttestation", hasAttestation))
	if hasAttestation {
		return nil
	}

	log.WithField("peer", msg.Peer).
		Debug("Sending request for attestation")
	if err := rs.p2p.Send(ctx, &pb.AttestationRequest{
		Hash: data.Hash,
	}, msg.Peer); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (rs *RegularSync) broadcastCanonicalBlock(ctx context.Context, announce *pb.BeaconBlockAnnounce) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.sync.broadcastCanonicalBlock")
	defer span.End()
	log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(announce.Hash))).
		Debug("Announcing canonical block")
	rs.p2p.Broadcast(ctx, announce)
	sentBlockAnnounce.Inc()
}
