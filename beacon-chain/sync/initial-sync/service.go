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
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
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
	p2p.Broadcaster
	p2p.Sender
	Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription
}

type powChainService interface {
	BlockExists(ctx context.Context, hash common.Hash) (bool, *big.Int, error)
}

type chainService interface {
	ReceiveBlock(ctx context.Context, block *pb.BeaconBlock, cfg *blockchain.ReceiveBlockConfig) (*pb.BeaconState, error)
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
	ctx                            context.Context
	cancel                         context.CancelFunc
	p2p                            p2pAPI
	syncService                    syncService
	chainService                   chainService
	db                             *db.BeaconDB
	powchain                       powChainService
	blockAnnounceBuf               chan p2p.Message
	batchedBlockBuf                chan p2p.Message
	blockBuf                       chan p2p.Message
	stateBuf                       chan p2p.Message
	currentSlot                    uint64
	highestObservedSlot            uint64
	beaconStateSlot                uint64
	syncPollingInterval            time.Duration
	inMemoryBlocks                 map[uint64]*pb.BeaconBlock
	syncedFeed                     *event.Feed
	stateReceived                  bool
	latestSyncedBlock              *pb.BeaconBlock
	mutex                          *sync.Mutex
	blocksAboveHighestObservedSlot []*pb.BeaconBlock
	highestObservedCanonicalState *pb.BeaconState
	pendingBlockAnnouncements int
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
		ctx:                            ctx,
		cancel:                         cancel,
		p2p:                            cfg.P2P,
		syncService:                    cfg.SyncService,
		db:                             cfg.BeaconDB,
		powchain:                       cfg.PowChain,
		chainService:                   cfg.ChainService,
		currentSlot:                    params.BeaconConfig().GenesisSlot,
		highestObservedSlot:            params.BeaconConfig().GenesisSlot,
		beaconStateSlot:                params.BeaconConfig().GenesisSlot,
		blockBuf:                       blockBuf,
		stateBuf:                       stateBuf,
		batchedBlockBuf:                batchedBlockBuf,
		blockAnnounceBuf:               blockAnnounceBuf,
		syncPollingInterval:            cfg.SyncPollingInterval,
		inMemoryBlocks:                 map[uint64]*pb.BeaconBlock{},
		blocksAboveHighestObservedSlot: []*pb.BeaconBlock{},
		pendingBlockAnnouncements: 0,
		syncedFeed:                     new(event.Feed),
		stateReceived:                  false,
		mutex:                          new(sync.Mutex),
	}
}

// Start begins the goroutine.
func (s *InitialSync) Start() {
	cHead, err := s.db.ChainHead()
	if err != nil {
		log.Errorf("Unable to get chain head %v", err)
	}
	s.currentSlot = cHead.Slot

	go func() {
		ticker := time.NewTicker(s.syncPollingInterval)
		s.run(ticker.C)
		ticker.Stop()
	}()
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

// SyncedFeed returns a feed which fires a message once the node is synced
func (s *InitialSync) SyncedFeed() *event.Feed {
	return s.syncedFeed
}

// checkSyncStatus verifies if the beacon node is correctly synced with its peers up to their
// latest canonical head. If not, then it requests batched blocks up to the highest observed slot.
func (s *InitialSync) checkSyncStatus() bool {
	if s.currentSlot == s.highestObservedSlot+uint64(s.pendingBlockAnnouncements) {
		if err := s.exitInitialSync(s.ctx); err != nil {
			log.Errorf("Could not exit initial sync: %v", err)
			return false
		}
		return true
	}
	if s.stateReceived {
		// requests multiple blocks so as to save and sync quickly.
		s.requestBatchedBlocks(s.currentSlot+1, s.highestObservedSlot)
	}
	return false
}

func (s *InitialSync) exitInitialSync(ctx context.Context) error {
	state := s.highestObservedCanonicalState
	var err error
	if err := s.db.SaveBlock(s.latestSyncedBlock); err != nil {
		return fmt.Errorf("could not save block: %v", err)
	}
	if err := s.db.UpdateChainHead(s.latestSyncedBlock, state); err != nil {
		return fmt.Errorf("could not update chain head: %v", err)
	}
	if err := s.db.SaveHistoricalState(state); err != nil {
		return fmt.Errorf("could not save state: %v", err)
	}
	log.Infof("Updated chain head block slot: %d, state slot: %d", s.latestSyncedBlock.Slot-params.BeaconConfig().GenesisSlot, state.Slot-params.BeaconConfig().GenesisSlot)
	// If there were any blocks received above the highest observed slot
	// during the process of performing initial sync, we run state transitions on those blocks.
	log.Infof("Processing %d blocks above high observed slot", len(s.blocksAboveHighestObservedSlot))
	for _, block := range s.blocksAboveHighestObservedSlot {
		log.Infof("Slot: %d", block.Slot-params.BeaconConfig().GenesisSlot)
		if err = s.db.SaveBlock(block); err != nil {
			return fmt.Errorf("could not save block: %v", err)
		}
		state, err = s.chainService.ReceiveBlock(s.ctx, block, &blockchain.ReceiveBlockConfig{
			EnableP2P: false,
			EnableLogging: false,
			EnableOperationsCleanup: false,
		})
		if err != nil {
			return fmt.Errorf("could not receive block in chain service: %v", err)
		}
		if err := s.db.UpdateChainHead(block, state); err != nil {
			return fmt.Errorf("could not update chain head: %v", err)
		}
		log.Infof("Updated chain head block slot: %d, state slot: %d", block.Slot-params.BeaconConfig().GenesisSlot, state.Slot-params.BeaconConfig().GenesisSlot)
	}
	canonicalState, err := s.db.State(ctx)
	if err != nil {
		return fmt.Errorf("could not get state: %v", err)
	}
	log.Infof("Canonical state slot: %d", canonicalState.Slot-params.BeaconConfig().GenesisSlot)
	log.Info("Exiting initial sync and starting normal sync")
	s.syncedFeed.Send(s.currentSlot)
	s.syncService.ResumeSync()
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
				s.processBlock(s.ctx, block, p2p.AnyPeer)
			}
			s.mutex.Unlock()
		}
	}
}

// run is the main goroutine for the initial sync service.
// delayChan is explicitly passed into this function to facilitate tests that don't require a timeout.
// It is assumed that the goroutine `run` is only called once per instance.
func (s *InitialSync) run(delayChan <-chan time.Time) {
	blockSub := s.p2p.Subscribe(&pb.BeaconBlockResponse{}, s.blockBuf)
	batchedBlocksub := s.p2p.Subscribe(&pb.BatchedBeaconBlockResponse{}, s.batchedBlockBuf)
	blockAnnounceSub := s.p2p.Subscribe(&pb.BeaconBlockAnnounce{}, s.blockAnnounceBuf)
	beaconStateSub := s.p2p.Subscribe(&pb.BeaconStateResponse{}, s.stateBuf)
	defer func() {
		blockSub.Unsubscribe()
		blockAnnounceSub.Unsubscribe()
		beaconStateSub.Unsubscribe()
		batchedBlocksub.Unsubscribe()
		close(s.batchedBlockBuf)
		close(s.blockBuf)
		close(s.stateBuf)
	}()

	if err := s.requestStateFromPeer(s.ctx, p2p.AnyPeer); err != nil {
		log.Errorf("Could not request state from peer %v", err)
	}

	for {
		select {
		case <-s.ctx.Done():
			log.Debug("Exiting goroutine")
			return
		case <-delayChan:
			if s.checkSyncStatus() {
				return
			}
		case msg := <-s.blockAnnounceBuf:
			safelyHandleMessage(s.processBlockAnnounce, msg)
		case msg := <-s.blockBuf:
			safelyHandleMessage(func(message p2p.Message) {
				data := message.Data.(*pb.BeaconBlockResponse)
				s.processBlock(message.Ctx, data.Block, message.Peer)
			}, msg)
		case msg := <-s.stateBuf:
			safelyHandleMessage(s.processState, msg)
		case msg := <-s.batchedBlockBuf:
			safelyHandleMessage(s.processBatchedBlocks, msg)
		}
	}
}
