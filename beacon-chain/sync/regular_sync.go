// Package sync defines the utilities for the beacon-chain to sync with the network.
package sync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

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
	p2p.ReputationManager
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
	regSyncLock                  sync.Mutex
	chainService                 chainService
	ancestorVerifier             blockchain.AncestorVerifier
	attsService                  attsService
	operationsService            operations.OperationFeeds
	db                           *db.BeaconDB
	blockAnnouncementFeed        *event.Feed
	announceBlockBuf             chan p2p.Message
	blockBuf                     chan p2p.Message
	blockRequestByHash           chan p2p.Message
	batchedRequestBuf            chan p2p.Message
	stateRequestBuf              chan p2p.Message
	chainHeadReqBuf              chan p2p.Message
	attestationBuf               chan p2p.Message
	attestationReqByHashBuf      chan p2p.Message
	announceAttestationBuf       chan p2p.Message
	finalizedAnnouncementBuf     chan p2p.Message
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
	BlockAnnounceBufferSize      int
	BlockBufferSize              int
	BlockReqHashBufferSize       int
	BatchedBufferSize            int
	StateReqBufferSize           int
	AttestationBufferSize        int
	AttestationReqHashBufSize    int
	AttestationsAnnounceBufSize  int
	FinalizedAnnouncementBufSize int
	ExitBufferSize               int
	ChainHeadReqBufferSize       int
	CanonicalBufferSize          int
	ChainService                 chainService
	AncestorVerifier             blockchain.AncestorVerifier
	OperationService             operations.OperationFeeds
	AttsService                  attsService
	BeaconDB                     *db.BeaconDB
	P2P                          p2pAPI
}

// DefaultRegularSyncConfig provides the default configuration for a sync service.
func DefaultRegularSyncConfig() *RegularSyncConfig {
	return &RegularSyncConfig{
		BlockAnnounceBufferSize:      params.BeaconConfig().DefaultBufferSize,
		BlockBufferSize:              params.BeaconConfig().DefaultBufferSize,
		BlockReqHashBufferSize:       params.BeaconConfig().DefaultBufferSize,
		BatchedBufferSize:            params.BeaconConfig().DefaultBufferSize,
		StateReqBufferSize:           params.BeaconConfig().DefaultBufferSize,
		ChainHeadReqBufferSize:       params.BeaconConfig().DefaultBufferSize,
		AttestationBufferSize:        params.BeaconConfig().DefaultBufferSize,
		AttestationReqHashBufSize:    params.BeaconConfig().DefaultBufferSize,
		AttestationsAnnounceBufSize:  params.BeaconConfig().DefaultBufferSize,
		ExitBufferSize:               params.BeaconConfig().DefaultBufferSize,
		CanonicalBufferSize:          params.BeaconConfig().DefaultBufferSize,
		FinalizedAnnouncementBufSize: params.BeaconConfig().DefaultBufferSize,
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
		ancestorVerifier:         cfg.AncestorVerifier,
		db:                       cfg.BeaconDB,
		operationsService:        cfg.OperationService,
		attsService:              cfg.AttsService,
		blockAnnouncementFeed:    new(event.Feed),
		announceBlockBuf:         make(chan p2p.Message, cfg.BlockAnnounceBufferSize),
		blockBuf:                 make(chan p2p.Message, cfg.BlockBufferSize),
		blockRequestByHash:       make(chan p2p.Message, cfg.BlockReqHashBufferSize),
		batchedRequestBuf:        make(chan p2p.Message, cfg.BatchedBufferSize),
		stateRequestBuf:          make(chan p2p.Message, cfg.StateReqBufferSize),
		attestationBuf:           make(chan p2p.Message, cfg.AttestationBufferSize),
		attestationReqByHashBuf:  make(chan p2p.Message, cfg.AttestationReqHashBufSize),
		announceAttestationBuf:   make(chan p2p.Message, cfg.AttestationsAnnounceBufSize),
		finalizedAnnouncementBuf: make(chan p2p.Message, cfg.FinalizedAnnouncementBufSize),
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
	blockRequestHashSub := rs.p2p.Subscribe(&pb.BeaconBlockRequest{}, rs.blockRequestByHash)
	batchedBlockRequestSub := rs.p2p.Subscribe(&pb.BatchedBeaconBlockRequest{}, rs.batchedRequestBuf)
	stateRequestSub := rs.p2p.Subscribe(&pb.BeaconStateRequest{}, rs.stateRequestBuf)
	attestationSub := rs.p2p.Subscribe(&pb.AttestationResponse{}, rs.attestationBuf)
	attestationReqSub := rs.p2p.Subscribe(&pb.AttestationRequest{}, rs.attestationReqByHashBuf)
	announceAttestationSub := rs.p2p.Subscribe(&pb.AttestationAnnounce{}, rs.announceAttestationBuf)
	exitSub := rs.p2p.Subscribe(&pb.VoluntaryExit{}, rs.exitBuf)
	chainHeadReqSub := rs.p2p.Subscribe(&pb.ChainHeadRequest{}, rs.chainHeadReqBuf)
	canonicalBlockSub := rs.chainService.CanonicalBlockFeed().Subscribe(rs.canonicalBuf)
	finalizedAnnouncementSub := rs.p2p.Subscribe(&pb.FinalizedStateAnnounce{}, rs.finalizedAnnouncementBuf)

	defer announceBlockSub.Unsubscribe()
	defer blockSub.Unsubscribe()
	defer blockRequestHashSub.Unsubscribe()
	defer batchedBlockRequestSub.Unsubscribe()
	defer stateRequestSub.Unsubscribe()
	defer chainHeadReqSub.Unsubscribe()
	defer attestationSub.Unsubscribe()
	defer attestationReqSub.Unsubscribe()
	defer announceAttestationSub.Unsubscribe()
	defer exitSub.Unsubscribe()
	defer canonicalBlockSub.Unsubscribe()
	defer finalizedAnnouncementSub.Unsubscribe()

	log.Info("Listening for regular sync messages from peers")

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
		case msg := <-rs.blockRequestByHash:
			go safelyHandleMessage(rs.handleBlockRequestByHash, msg)
		case msg := <-rs.batchedRequestBuf:
			go safelyHandleMessage(rs.handleBatchedBlockRequest, msg)
		case msg := <-rs.stateRequestBuf:
			go safelyHandleMessage(rs.handleStateRequest, msg)
		case msg := <-rs.chainHeadReqBuf:
			go safelyHandleMessage(rs.handleChainHeadRequest, msg)
		case msg := <-rs.finalizedAnnouncementBuf:
			go safelyHandleMessage(rs.handleFinalizedStateAnnouncement, msg)
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

	// messages received in sync should be processed in a timely manner
	ctx, cancel := context.WithTimeout(msg.Ctx, 30*time.Second)
	defer cancel()
	msg.Ctx = ctx

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

func (rs *RegularSync) handleFinalizedStateAnnouncement(msg p2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleFinalizedStateAnnouncement")
	defer span.End()
	rs.regSyncLock.Lock()
	defer rs.regSyncLock.Unlock()

	announce, ok := msg.Data.(*pb.FinalizedStateAnnounce)
	if !ok {
		log.Error("Message is of the incorrect type")
		return errors.New("incoming message is not *pb.FinalizedStateAnnounce")
	}
	chainHead, err := rs.db.ChainHead()
	if err != nil {
		log.Errorf("Unable to retrieve chain head: %v", err)
		return err
	}
	announcedBlock, err := rs.db.Block(bytesutil.ToBytes32(announce.BlockRoot))
	if err != nil {
		log.Errorf("Unable to retrieve block: %v", err)
		return err
	}

	isDescendant, err := rs.ancestorVerifier.IsDescendant(chainHead, announcedBlock)
	if err != nil {
		log.Errorf("Unable to verify if block is descendant: %v", err)
	}
	// If the announced finalized block is a descendant of our chain head, then we just return,
	// as it will be correctly processed by the rest of the regular sync runtime.
	if isDescendant {
		return nil
	}
	log.Info("Announced finalized block is not a descendant of the current chain, reorging...")
	// If the announced finalized block is NOT a descendant of our chain head, then it means
	// we have been building on a forked chain and we need to roll all the way back
	// to our current finalized state, and then process blocks up on the correct branch of the block tree.
	fState, err := rs.db.FinalizedState()
	if err != nil {
		log.Errorf("Unable to retrieve beacon state: %v", err)
		return err
	}
	fBlock, err := rs.db.FinalizedBlock()
	if err != nil {
		log.Errorf("Unable to retrieve finalized block: %v", err)
	}
	fRoot, err := hashutil.HashProto(fBlock)
	if err != nil {
		log.Errorf("Unable to marshal the beacon state: %v", err)
		return err
	}
	blocks, err := rs.blockParentsToFinalized(ctx, fRoot[:], announce.BlockRoot)
	if err != nil {
		log.Errorf("Could not get block parents: %v", err)
		return err
	}

	currentState := fState
	for _, block := range blocks {
		beaconState, err := rs.chainService.ReceiveBlock(ctx, block)
		if err != nil {
			log.Errorf("Could not process beacon block: %v", err)
			return err
		}
		if err := rs.db.UpdateChainHead(ctx, block, beaconState); err != nil {
			log.Errorf("Could not update chain head: %v", err)
		}
		currentState = beaconState
	}
	stateRoot, err := hashutil.HashProto(currentState)
	if err != nil {
		log.Errorf("Could not hash beacon state: %v", err)
		return err
	}
	if !bytes.Equal(stateRoot[:], announce.StateRoot) {
		return fmt.Errorf("state root mismatched: wanted %#x, received %#x", announce.StateRoot, stateRoot)
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

	head, err := rs.db.ChainHead()
	if err != nil {
		log.Errorf("Could not retrieve chain head: %v", err)
		return err
	}
	headBlkRoot, err := hashutil.HashBeaconBlock(head)
	if err != nil {
		log.Errorf("Could not hash chain head: %v", err)
	}
	finalizedBlk, err := rs.db.FinalizedBlock()
	if err != nil {
		log.Errorf("Could not retrieve finalized block: %v", err)
		return err
	}
	finalizedBlkRoot, err := hashutil.HashBeaconBlock(finalizedBlk)
	if err != nil {
		log.Errorf("Could not hash finalized block: %v", err)
	}

	stateRoot := rs.db.HeadStateRoot()
	finalizedState, err := rs.db.FinalizedState()
	if err != nil {
		log.Errorf("Could not retrieve finalized state: %v", err)
		return err
	}
	finalizedRoot, err := hashutil.HashProto(finalizedState)
	if err != nil {
		log.Errorf("Could not tree hash block: %v", err)
		return err
	}

	req := &pb.ChainHeadResponse{
		CanonicalSlot:             head.Slot,
		CanonicalStateRootHash32:  stateRoot[:],
		FinalizedStateRootHash32S: finalizedRoot[:],
		CanonicalBlockRoot:        headBlkRoot[:],
		FinalizedBlockRoot:        finalizedBlkRoot[:],
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
	rs.p2p.Reputation(msg.Peer, p2p.RepRewardValidAttestation)
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
	req := msg.Data.(*pb.BatchedBeaconBlockRequest)

	// To prevent circuit in the chain and the potentiality peer can bomb a node building block list.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	response, err := rs.blockParentsToFinalized(ctx, req.FinalizedRoot, req.CanonicalRoot)
	cancel()
	if err != nil {
		return fmt.Errorf("could not build canonical block list %v", err)
	}
	log.WithField("peer", msg.Peer).Debug("Sending response for batch blocks")

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

// blockParentsToFinalized returns the requested block list inclusive of head block but not inclusive of the finalized block.
// the return should look like (finalizedBlock... headBlock].
func (rs *RegularSync) blockParentsToFinalized(ctx context.Context, finalizedRoot []byte, headRoot []byte) ([]*pb.BeaconBlock, error) {
	// if head block was the same as the finalized block.
	if bytes.Equal(headRoot, finalizedRoot) {
		return nil, nil
	}

	b, err := rs.db.Block(bytesutil.ToBytes32(headRoot))
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("nil block %#x from db", bytesutil.Trunc(headRoot))
	}

	bList := []*pb.BeaconBlock{b}
	parentRoot := b.ParentRootHash32
	for !bytes.Equal(parentRoot, finalizedRoot) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		b, err = rs.db.Block(bytesutil.ToBytes32(parentRoot))
		if err != nil {
			return nil, err
		}
		if b == nil {
			return nil, fmt.Errorf("nil parent block %#x from db", bytesutil.Trunc(parentRoot[:]))
		}

		// Prepend parent to the beginning of the list.
		bList = append([]*pb.BeaconBlock{b}, bList...)

		parentRoot = b.ParentRootHash32
	}
	return bList, nil
}
