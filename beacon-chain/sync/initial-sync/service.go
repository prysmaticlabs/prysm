// Package initialsync is run by the beacon node when the local chain is
// behind the network's longest chain. Initial sync works as follows:
// The node requests for the slot number of the most recent finalized block.
// The node then builds from the most recent finalized block by requesting for subsequent
// blocks by slot number. Once the service detects that the local chain is caught up with
// the network, the service hands over control to the regular sync service.
// Note: The behavior of initialsync will likely change as the specification changes.
// The most significant and highly probable change will be determining where to sync from.
// The beacon chain may sync from a block in the pasts X months in order to combat long-range attacks
// (see here: https://github.com/ethereum/wiki/wiki/Proof-of-Stake-FAQs#what-is-weak-subjectivity)
package initialsync

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "initial-sync")
var debugError = "debug:"

// Config defines the configurable properties of InitialSync.
//
type Config struct {
	SyncPollingInterval     time.Duration
	BlockBufferSize         int
	BlockAnnounceBufferSize int
	BatchedBlockBufferSize  int
	StateBufferSize         int
	BeaconDB                *db.BeaconDB
	P2P                     p2pAPI
	SyncService             syncService
	ChainService            chainService
	PowChain                powChainService
}

// DefaultConfig provides the default configuration for a sync service.
// SyncPollingInterval determines how frequently the service checks that initial sync is complete.
// BlockBufferSize determines that buffer size of the `blockBuf` channel.
// StateBufferSize determines the buffer size of the `stateBuf` channel.
func DefaultConfig() *Config {
	return &Config{
		SyncPollingInterval:     time.Duration(params.BeaconConfig().SyncPollingInterval) * time.Second,
		BlockBufferSize:         params.BeaconConfig().DefaultBufferSize,
		BatchedBlockBufferSize:  params.BeaconConfig().DefaultBufferSize,
		BlockAnnounceBufferSize: params.BeaconConfig().DefaultBufferSize,
		StateBufferSize:         params.BeaconConfig().DefaultBufferSize,
	}
}

type p2pAPI interface {
	p2p.Sender
	Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription
}

type powChainService interface {
	BlockExists(ctx context.Context, hash common.Hash) (bool, *big.Int, error)
}

type chainService interface {
	blockchain.BlockProcessor
	blockchain.ForkChoice
}

// SyncService is the interface for the Sync service.
// InitialSync calls `Start` when initial sync completes.
type syncService interface {
	Start()
	ResumeSync()
}

// InitialSync defines the main class in this package.
// See the package comments for a general description of the service's functions.
type InitialSync struct {
	ctx                 context.Context
	cancel              context.CancelFunc
	p2p                 p2pAPI
	syncService         syncService
	chainService        chainService
	db                  *db.BeaconDB
	powchain            powChainService
	blockAnnounceBuf    chan p2p.Message
	batchedBlockBuf     chan p2p.Message
	blockBuf            chan p2p.Message
	stateBuf            chan p2p.Message
	currentSlot         uint64
	highestObservedSlot uint64
	highestObservedRoot [32]byte
	beaconStateSlot     uint64
	syncPollingInterval time.Duration
	inMemoryBlocks      map[uint64]*pb.BeaconBlock
	syncedFeed          *event.Feed
	stateReceived       bool
	lastRequestedSlot   uint64
	finalizedStateRoot  [32]byte
	mutex               *sync.Mutex
	nodeIsSynced        bool
	bestPeer            peer.ID
}

// NewInitialSyncService constructs a new InitialSyncService.
// This method is normally called by the main node.
func NewInitialSyncService(ctx context.Context,
	cfg *Config,
) *InitialSync {
	ctx, cancel := context.WithCancel(ctx)

	blockBuf := make(chan p2p.Message, cfg.BlockBufferSize)
	stateBuf := make(chan p2p.Message, cfg.StateBufferSize)
	blockAnnounceBuf := make(chan p2p.Message, cfg.BlockAnnounceBufferSize)
	batchedBlockBuf := make(chan p2p.Message, cfg.BatchedBlockBufferSize)

	return &InitialSync{
		ctx:                 ctx,
		cancel:              cancel,
		p2p:                 cfg.P2P,
		syncService:         cfg.SyncService,
		db:                  cfg.BeaconDB,
		powchain:            cfg.PowChain,
		chainService:        cfg.ChainService,
		currentSlot:         params.BeaconConfig().GenesisSlot,
		highestObservedSlot: params.BeaconConfig().GenesisSlot,
		beaconStateSlot:     params.BeaconConfig().GenesisSlot,
		blockBuf:            blockBuf,
		stateBuf:            stateBuf,
		batchedBlockBuf:     batchedBlockBuf,
		blockAnnounceBuf:    blockAnnounceBuf,
		syncPollingInterval: cfg.SyncPollingInterval,
		inMemoryBlocks:      map[uint64]*pb.BeaconBlock{},
		syncedFeed:          new(event.Feed),
		stateReceived:       false,
		mutex:               new(sync.Mutex),
	}
}

// Start begins the goroutine.
func (s *InitialSync) Start() {
	cHead, err := s.db.ChainHead()
	if err != nil {
		log.Errorf("Unable to get chain head %v", err)
	}
	s.currentSlot = cHead.Slot
	go s.run()
	go s.checkInMemoryBlocks()
}

// Stop kills the initial sync goroutine.
func (s *InitialSync) Stop() error {
	log.Info("Stopping service")
	s.cancel()
	return nil
}

// InitializeObservedSlot sets the highest observed slot.
func (s *InitialSync) InitializeObservedSlot(slot uint64) {
	s.highestObservedSlot = slot
}

// InitializeObservedStateRoot sets the highest observed state root.
func (s *InitialSync) InitializeObservedStateRoot(root [32]byte) {
	s.highestObservedRoot = root
}

// InitializeFinalizedStateRoot sets the state root of the last finalized state.
func (s *InitialSync) InitializeFinalizedStateRoot(root [32]byte) {
	s.finalizedStateRoot = root
}

// InitializeBestPeer sets the peer ID of the highest observed peer.
func (s *InitialSync) InitializeBestPeer(p peer.ID) {
	s.bestPeer = p
}

// HighestObservedSlot returns the highest observed slot.
func (s *InitialSync) HighestObservedSlot() uint64 {
	return s.highestObservedSlot
}

// NodeIsSynced checks that the node has been caught up with the network.
func (s *InitialSync) NodeIsSynced() (bool, uint64) {
	return s.nodeIsSynced, s.currentSlot
}

func (s *InitialSync) exitInitialSync(ctx context.Context, block *pb.BeaconBlock) error {
	if s.nodeIsSynced {
		return nil
	}
	state, err := s.db.HeadState(ctx)
	if err != nil {
		return err
	}
	if err := s.chainService.VerifyBlockValidity(ctx, block, state); err != nil {
		return err
	}
	if err := s.db.SaveBlock(block); err != nil {
		return err
	}
	root, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return fmt.Errorf("failed to tree hash block: %v", err)
	}
	if err := s.db.SaveAttestationTarget(ctx, &pb.AttestationTarget{
		Slot:       block.Slot,
		BlockRoot:  root[:],
		ParentRoot: block.ParentRootHash32,
	}); err != nil {
		return fmt.Errorf("failed to save attestation target: %v", err)
	}
	state, err = s.chainService.ApplyBlockStateTransition(ctx, block, state)
	if err != nil {
		log.Error("OH NO - looks like you synced with a bad peer, try restarting your node!")
		switch err.(type) {
		case *blockchain.BlockFailedProcessingErr:
			// If the block fails processing, we delete it from our DB.
			if err := s.db.DeleteBlock(block); err != nil {
				return fmt.Errorf("could not delete bad block from db: %v", err)
			}
			return fmt.Errorf("could not apply block state transition: %v", err)
		default:
			return fmt.Errorf("could not apply block state transition: %v", err)
		}
	}
	if err := s.chainService.CleanupBlockOperations(ctx, block); err != nil {
		return err
	}
	if err := s.db.UpdateChainHead(ctx, block, state); err != nil {
		return err
	}

	stateRoot := s.db.HeadStateRoot()

	if stateRoot != s.highestObservedRoot {
		// TODO(#2155): Instead of a fatal call, drop the peer and restart the initial sync service.
		log.Error("OH NO - looks like you synced with a bad peer, try restarting your node!")
		log.Fatalf(
			"Canonical state root %#x does not match highest observed root from peer %#x",
			stateRoot,
			s.highestObservedRoot,
		)
	}
	log.WithField("canonicalStateSlot", state.Slot-params.BeaconConfig().GenesisSlot).Info("Exiting init sync and starting regular sync")
	s.syncService.ResumeSync()
	s.cancel()
	s.nodeIsSynced = true
	return nil
}

// checkInMemoryBlocks is another routine which will run concurrently with the
// main routine for initial sync, where it checks the blocks saved in memory regularly
// to see if the blocks are valid enough to be processed.
func (s *InitialSync) checkInMemoryBlocks() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			if s.currentSlot == s.highestObservedSlot {
				return
			}
			s.mutex.Lock()
			if block, ok := s.inMemoryBlocks[s.currentSlot+1]; ok && s.currentSlot+1 <= s.highestObservedSlot {
				s.processBlock(s.ctx, block)
			}
			s.mutex.Unlock()
		}
	}
}

// run is the main goroutine for the initial sync service.
// delayChan is explicitly passed into this function to facilitate tests that don't require a timeout.
// It is assumed that the goroutine `run` is only called once per instance.
func (s *InitialSync) run() {
	blockSub := s.p2p.Subscribe(&pb.BeaconBlockResponse{}, s.blockBuf)
	batchedBlocksub := s.p2p.Subscribe(&pb.BatchedBeaconBlockResponse{}, s.batchedBlockBuf)
	beaconStateSub := s.p2p.Subscribe(&pb.BeaconStateResponse{}, s.stateBuf)
	defer func() {
		blockSub.Unsubscribe()
		beaconStateSub.Unsubscribe()
		batchedBlocksub.Unsubscribe()
		close(s.batchedBlockBuf)
		close(s.blockBuf)
		close(s.stateBuf)
	}()

	// We send out a state request to all peers.
	if err := s.requestStateFromPeer(s.ctx, s.finalizedStateRoot); err != nil {
		log.Errorf("Could not request state from peer %v", err)
	}

	for {
		select {
		case <-s.ctx.Done():
			log.Debug("Exiting goroutine")
			return
		case msg := <-s.blockBuf:
			safelyHandleMessage(func(message p2p.Message) {
				data := message.Data.(*pb.BeaconBlockResponse)
				s.processBlock(message.Ctx, data.Block)
			}, msg)
		case msg := <-s.stateBuf:
			safelyHandleMessage(s.processState, msg)
		case msg := <-s.batchedBlockBuf:
			safelyHandleMessage(s.processBatchedBlocks, msg)
		}
	}
}
