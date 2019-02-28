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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/prysmaticlabs/prysm/shared/hashutil"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "initial-sync")

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
}

// DefaultConfig provides the default configuration for a sync service.
// SyncPollingInterval determines how frequently the service checks that initial sync is complete.
// BlockBufferSize determines that buffer size of the `blockBuf` channel.
// CrystallizedStateBufferSize determines the buffer size of thhe `crystallizedStateBuf` channel.
func DefaultConfig() *Config {
	return &Config{
		SyncPollingInterval:     time.Duration(params.BeaconConfig().SyncPollingInterval) * time.Second,
		BlockBufferSize:         100,
		BatchedBlockBufferSize:  100,
		BlockAnnounceBufferSize: 100,
		StateBufferSize:         100,
	}
}

type p2pAPI interface {
	Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription
	Send(msg proto.Message, peer p2p.Peer)
	Broadcast(msg proto.Message)
}

type chainService interface {
	IncomingBlockFeed() *event.Feed
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
	blockAnnounceBuf               chan p2p.Message
	batchedBlockBuf                chan p2p.Message
	blockBuf                       chan p2p.Message
	stateBuf                       chan p2p.Message
	currentSlot                    uint64
	highestObservedSlot            uint64
	syncPollingInterval            time.Duration
	inMemoryBlocks                 map[uint64]*pb.BeaconBlock
	syncedFeed                     *event.Feed
	atGenesis                      bool
	stateRootOfHighestObservedSlot [32]byte
	mutex                          *sync.Mutex
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
		chainService:                   cfg.ChainService,
		db:                             cfg.BeaconDB,
		currentSlot:                    params.BeaconConfig().GenesisSlot,
		highestObservedSlot:            params.BeaconConfig().GenesisSlot,
		blockBuf:                       blockBuf,
		stateBuf:                       stateBuf,
		batchedBlockBuf:                batchedBlockBuf,
		blockAnnounceBuf:               blockAnnounceBuf,
		syncPollingInterval:            cfg.SyncPollingInterval,
		inMemoryBlocks:                 map[uint64]*pb.BeaconBlock{},
		syncedFeed:                     new(event.Feed),
		atGenesis:                      false,
		stateRootOfHighestObservedSlot: [32]byte{},
		mutex:                          new(sync.Mutex),
	}
}

// Start begins the goroutine.
func (s *InitialSync) Start() {
	cHead, err := s.db.ChainHead()
	if err != nil {
		log.Errorf("Unable to get chain head %v", err)
	}

	var atGenesis bool
	// setting genesis bool
	if cHead.Slot == params.BeaconConfig().GenesisSlot {
		atGenesis = true
	}
	s.atGenesis = atGenesis
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

// InitializeStateRoot sets the state root of the highest observed slot.
func (s *InitialSync) InitializeStateRoot(root [32]byte) {
	s.stateRootOfHighestObservedSlot = root
}

// SyncedFeed returns a feed which fires a message once the node is synced
func (s *InitialSync) SyncedFeed() *event.Feed {
	return s.syncedFeed
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

	// Send out a batch request
	s.requestBatchedBlocks(s.currentSlot+1, s.highestObservedSlot)

	for {
		select {
		case <-s.ctx.Done():
			log.Debug("Exiting goroutine")
			return
		case <-delayChan:
			if s.atGenesis {
				if err := s.requestStateFromPeer(s.stateRootOfHighestObservedSlot[:], p2p.Peer{}); err != nil {
					log.Errorf("Could not request state from peer %v", err)
				}
				continue
			}
			if s.highestObservedSlot == s.currentSlot {
				log.Info("Exiting initial sync and starting normal sync")
				s.syncedFeed.Send(s.currentSlot)
				s.syncService.ResumeSync()
				return
			}

			// requests multiple blocks so as to save and sync quickly.
			s.requestBatchedBlocks(s.currentSlot+1, s.highestObservedSlot)
		case msg := <-s.blockAnnounceBuf:
			data := msg.Data.(*pb.BeaconBlockAnnounce)

			if s.atGenesis {
				if err := s.requestStateFromPeer(s.stateRootOfHighestObservedSlot[:], p2p.Peer{}); err != nil {
					log.Errorf("Could not request state from peer %v", err)
				}
				continue
			}

			if data.SlotNumber > s.highestObservedSlot {
				s.highestObservedSlot = data.SlotNumber
			}

			s.requestBatchedBlocks(s.currentSlot+1, s.highestObservedSlot)
			log.Debugf("Successfully requested the next block with slot: %d", data.SlotNumber)
		case msg := <-s.blockBuf:
			data := msg.Data.(*pb.BeaconBlockResponse)
			s.processBlock(context.TODO(), data.Block, msg.Peer)
		case msg := <-s.stateBuf:
			data := msg.Data.(*pb.BeaconStateResponse)
			beaconState := data.BeaconState

			if s.currentSlot > beaconState.FinalizedEpoch*params.BeaconConfig().SlotsPerEpoch {
				continue
			}

			if err := s.db.SaveState(beaconState); err != nil {
				log.Errorf("Unable to set beacon state for initial sync %v", err)
			}

			h, err := hashutil.HashProto(beaconState)
			if err != nil {
				log.Error(err)
				continue
			}

			if h == s.stateRootOfHighestObservedSlot {
				s.atGenesis = false
			}

			// sets the current slot to the last finalized slot of the
			// beacon state to begin our sync from.
			s.currentSlot = beaconState.FinalizedEpoch * params.BeaconConfig().SlotsPerEpoch
			log.Debugf("Successfully saved crystallized state with the last finalized slot: %d", beaconState.FinalizedEpoch*params.BeaconConfig().SlotsPerEpoch)

			s.requestBatchedBlocks(s.currentSlot+1, s.highestObservedSlot)

		case msg := <-s.batchedBlockBuf:
			s.processBatchedBlocks(context.TODO(), msg)
		}
	}
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
				s.processBlock(context.TODO(), block, p2p.Peer{})
			}
			s.mutex.Unlock()
		}
	}
}

// processBlock is the main method that validates each block which is received
// for initial sync. It checks if the blocks are valid and then will continue to
// process and save it into the db.
func (s *InitialSync) processBlock(ctx context.Context, block *pb.BeaconBlock, peer p2p.Peer) {
	if block.Slot > s.highestObservedSlot {
		s.highestObservedSlot = block.Slot
		s.stateRootOfHighestObservedSlot = bytesutil.ToBytes32(block.StateRootHash32)
	}

	if block.Slot < s.currentSlot {
		return
	}

	// requesting beacon state if there is no saved state.
	if s.atGenesis {
		if err := s.requestStateFromPeer(block.StateRootHash32, peer); err != nil {
			log.Errorf("Could not request beacon state from peer: %v", err)
		}
		return
	}
	// if it isn't the block in the next slot it saves it in memory.
	if block.Slot != (s.currentSlot + 1) {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		if _, ok := s.inMemoryBlocks[block.Slot]; !ok {
			s.inMemoryBlocks[block.Slot] = block
		}
		return
	}

	if err := s.validateAndSaveNextBlock(ctx, block); err != nil {
		log.Errorf("Unable to save block: %v", err)
	}
	s.requestNextBlockBySlot(ctx, s.currentSlot+1)
}

// processBatchedBlocks processes all the received blocks from
// the p2p message.
func (s *InitialSync) processBatchedBlocks(ctx context.Context, msg p2p.Message) {
	log.Debug("Processing batched block response")

	response := msg.Data.(*pb.BatchedBeaconBlockResponse)
	batchedBlocks := response.BatchedBlocks

	for _, block := range batchedBlocks {
		s.processBlock(ctx, block, msg.Peer)
	}
	log.Debug("Finished processing batched blocks")
}

// requestStateFromPeer sends a request to a peer for the corresponding state
// for a beacon block.
func (s *InitialSync) requestStateFromPeer(stateRoot []byte, peer p2p.Peer) error {
	log.Debugf("Successfully processed incoming block with state hash: %#x", stateRoot)
	s.p2p.Send(&pb.BeaconStateRequest{Hash: stateRoot}, peer)
	return nil
}

// requestNextBlock broadcasts a request for a block with the entered slotnumber.
func (s *InitialSync) requestNextBlockBySlot(ctx context.Context, slotNumber uint64) {
	log.Debugf("Requesting block %d ", slotNumber)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if block, ok := s.inMemoryBlocks[slotNumber]; ok {
		s.processBlock(ctx, block, p2p.Peer{})
		return
	}
	s.p2p.Broadcast(&pb.BeaconBlockRequestBySlotNumber{SlotNumber: slotNumber})
}

// requestBatchedBlocks sends out a request for multiple blocks till a
// specified bound slot number.
func (s *InitialSync) requestBatchedBlocks(startSlot uint64, endSlot uint64) {
	blockLimit := params.BeaconConfig().BatchBlockLimit
	if startSlot+blockLimit < endSlot {
		endSlot = startSlot + blockLimit
	}
	log.Debugf("Requesting batched blocks from slot %d to %d", startSlot, endSlot)
	s.p2p.Broadcast(&pb.BatchedBeaconBlockRequest{
		StartSlot: startSlot,
		EndSlot:   endSlot,
	})
}

// validateAndSaveNextBlock will validate whether blocks received from the blockfetcher
// routine can be added to the chain.
func (s *InitialSync) validateAndSaveNextBlock(ctx context.Context, block *pb.BeaconBlock) error {
	root, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return err
	}

	if (s.currentSlot + 1) == block.Slot {

		if err := s.checkBlockValidity(ctx, block); err != nil {
			return err
		}

		log.Infof("Saved block with root %#x and slot %d for initial sync", root, block.Slot)
		s.currentSlot = block.Slot

		s.mutex.Lock()
		defer s.mutex.Unlock()
		// delete block from memory
		if _, ok := s.inMemoryBlocks[block.Slot]; ok {
			delete(s.inMemoryBlocks, block.Slot)
		}

		// Send block to main chain service to be processed
		s.chainService.IncomingBlockFeed().Send(block)
	}
	return nil
}

func (s *InitialSync) checkBlockValidity(ctx context.Context, block *pb.BeaconBlock) error {
	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return fmt.Errorf("could not tree hash received block: %v", err)
	}

	log.Debugf("Processing response to block request: %#x", blockRoot)

	if s.db.HasBlock(blockRoot) {
		return errors.New("received a block that already exists. Exiting")
	}

	beaconState, err := s.db.State(ctx)
	if err != nil {
		return fmt.Errorf("failed to get beacon state: %v", err)
	}

	if block.Slot < beaconState.FinalizedEpoch*params.BeaconConfig().SlotsPerEpoch {
		return errors.New("discarding received block with a slot number smaller than the last finalized slot")
	}
	// Attestation from proposer not verified as, other nodes only store blocks not proposer
	// attestations.
	return nil
}
