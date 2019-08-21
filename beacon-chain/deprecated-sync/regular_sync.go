// Package sync defines the utilities for the beacon-chain to sync with the network.
package sync

import (
	"bytes"
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	p2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	deprecatedp2p "github.com/prysmaticlabs/prysm/shared/deprecated-p2p"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/logutil"
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
	p2p.DeprecatedSubscriber
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
	announceBlockBuf             chan deprecatedp2p.Message
	blockBuf                     chan deprecatedp2p.Message
	blockRequestByHash           chan deprecatedp2p.Message
	batchedRequestBuf            chan deprecatedp2p.Message
	stateRequestBuf              chan deprecatedp2p.Message
	chainHeadReqBuf              chan deprecatedp2p.Message
	attestationBuf               chan deprecatedp2p.Message
	exitBuf                      chan deprecatedp2p.Message
	canonicalBuf                 chan *pb.BeaconBlockAnnounce
	highestObservedSlot          uint64
	blocksAwaitingProcessing     map[[32]byte]deprecatedp2p.Message
	blocksAwaitingProcessingLock sync.RWMutex
	blockProcessingLock          sync.RWMutex
	blockAnnouncements           map[uint64][]byte
	blockAnnouncementsLock       sync.RWMutex
}

// RegularSyncConfig allows the channel's buffer sizes to be changed.
type RegularSyncConfig struct {
	BlockAnnounceBufferSize int
	BlockBufferSize         int
	BlockReqHashBufferSize  int
	BatchedBufferSize       int
	StateReqBufferSize      int
	AttestationBufferSize   int
	ExitBufferSize          int
	ChainHeadReqBufferSize  int
	CanonicalBufferSize     int
	ChainService            chainService
	OperationService        operations.OperationFeeds
	AttsService             attsService
	BeaconDB                *db.BeaconDB
	P2P                     p2pAPI
}

// DefaultRegularSyncConfig provides the default configuration for a sync service.
func DefaultRegularSyncConfig() *RegularSyncConfig {
	return &RegularSyncConfig{
		BlockAnnounceBufferSize: params.BeaconConfig().DefaultBufferSize,
		BlockBufferSize:         params.BeaconConfig().DefaultBufferSize,
		BlockReqHashBufferSize:  params.BeaconConfig().DefaultBufferSize,
		BatchedBufferSize:       params.BeaconConfig().DefaultBufferSize,
		StateReqBufferSize:      params.BeaconConfig().DefaultBufferSize,
		ChainHeadReqBufferSize:  params.BeaconConfig().DefaultBufferSize,
		AttestationBufferSize:   params.BeaconConfig().DefaultBufferSize,
		ExitBufferSize:          params.BeaconConfig().DefaultBufferSize,
		CanonicalBufferSize:     params.BeaconConfig().DefaultBufferSize,
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
		announceBlockBuf:         make(chan deprecatedp2p.Message, cfg.BlockAnnounceBufferSize),
		blockBuf:                 make(chan deprecatedp2p.Message, cfg.BlockBufferSize),
		blockRequestByHash:       make(chan deprecatedp2p.Message, cfg.BlockReqHashBufferSize),
		batchedRequestBuf:        make(chan deprecatedp2p.Message, cfg.BatchedBufferSize),
		stateRequestBuf:          make(chan deprecatedp2p.Message, cfg.StateReqBufferSize),
		attestationBuf:           make(chan deprecatedp2p.Message, cfg.AttestationBufferSize),
		exitBuf:                  make(chan deprecatedp2p.Message, cfg.ExitBufferSize),
		chainHeadReqBuf:          make(chan deprecatedp2p.Message, cfg.ChainHeadReqBufferSize),
		canonicalBuf:             make(chan *pb.BeaconBlockAnnounce, cfg.CanonicalBufferSize),
		blocksAwaitingProcessing: make(map[[32]byte]deprecatedp2p.Message),
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
	attestationSub := rs.p2p.Subscribe(&ethpb.Attestation{}, rs.attestationBuf)
	exitSub := rs.p2p.Subscribe(&ethpb.VoluntaryExit{}, rs.exitBuf)
	chainHeadReqSub := rs.p2p.Subscribe(&pb.ChainHeadRequest{}, rs.chainHeadReqBuf)
	canonicalBlockSub := rs.chainService.CanonicalBlockFeed().Subscribe(rs.canonicalBuf)

	defer announceBlockSub.Unsubscribe()
	defer blockSub.Unsubscribe()
	defer blockRequestHashSub.Unsubscribe()
	defer batchedBlockRequestSub.Unsubscribe()
	defer stateRequestSub.Unsubscribe()
	defer chainHeadReqSub.Unsubscribe()
	defer attestationSub.Unsubscribe()
	defer exitSub.Unsubscribe()
	defer canonicalBlockSub.Unsubscribe()

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
		case blockAnnounce := <-rs.canonicalBuf:
			go rs.broadcastCanonicalBlock(rs.ctx, blockAnnounce)
		}
	}
}

// safelyHandleMessage will recover and log any panic that occurs from the
// function argument.
func safelyHandleMessage(fn func(deprecatedp2p.Message) error, msg deprecatedp2p.Message) {
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

		log.WithField("method", logutil.FunctionName(fn)).Error(err)
	}
}

func (rs *RegularSync) handleStateRequest(msg deprecatedp2p.Message) error {
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
	finalizedBlk, err := rs.db.FinalizedBlock()
	if err != nil {
		log.Error("could not get finalized block")
		return err
	}

	log.WithField(
		"beaconState", fmt.Sprintf("%#x", root),
	).Debug("Sending finalized state and block to peer")
	defer sentState.Inc()
	resp := &pb.BeaconStateResponse{
		FinalizedState: fState,
		FinalizedBlock: finalizedBlk,
	}
	if err := rs.p2p.Send(ctx, resp, msg.Peer); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (rs *RegularSync) handleChainHeadRequest(msg deprecatedp2p.Message) error {
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
	headBlkRoot, err := ssz.SigningRoot(head)
	if err != nil {
		log.Errorf("Could not hash chain head: %v", err)
	}
	finalizedBlk, err := rs.db.FinalizedBlock()
	if err != nil {
		log.Errorf("Could not retrieve finalized block: %v", err)
		return err
	}
	finalizedBlkRoot, err := ssz.SigningRoot(finalizedBlk)
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
func (rs *RegularSync) receiveAttestation(msg deprecatedp2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.receiveAttestation")
	defer span.End()
	recAttestation.Inc()

	attestation := msg.Data.(*ethpb.Attestation)
	attestationDataHash, err := hashutil.HashProto(attestation.Data)
	if err != nil {
		log.Errorf("Could not hash received attestation: %v", err)
		return err
	}
	log.WithFields(logrus.Fields{
		"headRoot":       fmt.Sprintf("%#x", bytesutil.Trunc(attestation.Data.BeaconBlockRoot)),
		"justifiedEpoch": attestation.Data.Source.Epoch,
	}).Debug("Received an attestation")

	// Skip if attestation has been seen before.
	hasAttestation := rs.db.HasAttestationDeprecated(attestationDataHash)
	span.AddAttributes(trace.BoolAttribute("hasAttestation", hasAttestation))
	if hasAttestation {
		dbAttestation, err := rs.db.AttestationDeprecated(attestationDataHash)
		if err != nil {
			return err
		}
		if dbAttestation.AggregationBits.Contains(attestation.AggregationBits) {
			log.WithField("attestationDataHash", fmt.Sprintf("%#x", bytesutil.Trunc(attestationDataHash[:]))).
				Debug("Received, skipping attestation")
			return nil
		}
	}

	// Skip if attestation slot is older than last finalized slot in state.
	headState, err := rs.db.HeadState(rs.ctx)
	if err != nil {
		return err
	}

	attTargetEpoch := attestation.Data.Target.Epoch
	headFinalizedEpoch := headState.FinalizedCheckpoint.Epoch
	span.AddAttributes(
		trace.Int64Attribute("attestation.target.epoch", int64(attTargetEpoch)),
		trace.Int64Attribute("finalized.epoch", int64(headFinalizedEpoch)),
	)

	if attTargetEpoch < headFinalizedEpoch {
		log.WithFields(logrus.Fields{
			"receivedTargetEpoch": attTargetEpoch,
			"finalizedEpoch":      headFinalizedEpoch},
		).Debug("Skipping received attestation with target epoch less than current finalized epoch")
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
func (rs *RegularSync) receiveExitRequest(msg deprecatedp2p.Message) error {
	_, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.receiveExitRequest")
	defer span.End()
	recExit.Inc()
	exit := msg.Data.(*ethpb.VoluntaryExit)
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

func (rs *RegularSync) handleBlockRequestByHash(msg deprecatedp2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleBlockRequestByHash")
	defer span.End()
	blockReqHash.Inc()

	data := msg.Data.(*pb.BeaconBlockRequest)
	root := bytesutil.ToBytes32(data.Hash)
	block, err := rs.db.BlockDeprecated(root)
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
func (rs *RegularSync) handleBatchedBlockRequest(msg deprecatedp2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.handleBatchedBlockRequest")
	defer span.End()
	batchedBlockReq.Inc()
	req := msg.Data.(*pb.BatchedBeaconBlockRequest)

	// To prevent circuit in the chain and the potentiality peer can bomb a node building block list.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	response, err := rs.respondBatchedBlocks(ctx, req.FinalizedRoot, req.CanonicalRoot)
	cancel()
	if err != nil {
		return errors.Wrap(err, "could not build canonical block list")
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

func (rs *RegularSync) broadcastCanonicalBlock(ctx context.Context, announce *pb.BeaconBlockAnnounce) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.sync.broadcastCanonicalBlock")
	defer span.End()
	log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(announce.Hash))).
		Debug("Announcing canonical block")
	rs.p2p.Broadcast(ctx, announce)
	sentBlockAnnounce.Inc()
}

// respondBatchedBlocks returns the requested block list inclusive of head block but not inclusive of the finalized block.
// the return should look like (finalizedBlock... headBlock].
func (rs *RegularSync) respondBatchedBlocks(ctx context.Context, finalizedRoot []byte, headRoot []byte) ([]*ethpb.BeaconBlock, error) {
	// if head block was the same as the finalized block.
	if bytes.Equal(headRoot, finalizedRoot) {
		return nil, nil
	}

	b, err := rs.db.BlockDeprecated(bytesutil.ToBytes32(headRoot))
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("nil block %#x from db", bytesutil.Trunc(headRoot))
	}

	bList := []*ethpb.BeaconBlock{b}
	parentRoot := b.ParentRoot
	for !bytes.Equal(parentRoot, finalizedRoot) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		b, err = rs.db.BlockDeprecated(bytesutil.ToBytes32(parentRoot))
		if err != nil {
			return nil, err
		}
		if b == nil {
			return nil, fmt.Errorf("nil parent block %#x from db", bytesutil.Trunc(parentRoot[:]))
		}

		// Prepend parent to the beginning of the list.
		bList = append([]*ethpb.BeaconBlock{b}, bList...)

		parentRoot = b.ParentRoot
	}
	return bList, nil
}
