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
	"time"

	"github.com/golang/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
	StateBufferSize         int
	BeaconDB                *db.BeaconDB
	P2P                     p2pAPI
	SyncService             syncService
}

// DefaultConfig provides the default configuration for a sync service.
// SyncPollingInterval determines how frequently the service checks that initial sync is complete.
// BlockBufferSize determines that buffer size of the `blockBuf` channel.
// CrystallizedStateBufferSize determines the buffer size of thhe `crystallizedStateBuf` channel.
func DefaultConfig() *Config {
	return &Config{
		SyncPollingInterval:     time.Duration(params.BeaconConfig().SyncPollingInterval) * time.Second,
		BlockBufferSize:         100,
		BlockAnnounceBufferSize: 100,
		StateBufferSize:         100,
	}
}

type p2pAPI interface {
	Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription
	Send(msg proto.Message, peer p2p.Peer)
	Broadcast(msg proto.Message)
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
	ctx                    context.Context
	cancel                 context.CancelFunc
	p2p                    p2pAPI
	syncService            syncService
	db                     *db.BeaconDB
	blockAnnounceBuf       chan p2p.Message
	blockBuf               chan p2p.Message
	stateBuf               chan p2p.Message
	currentSlot            uint64
	highestObservedSlot    uint64
	syncPollingInterval    time.Duration
	initialStateRootHash32 [32]byte
	inMemoryBlocks         map[uint64]*pb.BeaconBlockResponse
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

	return &InitialSync{
		ctx:                 ctx,
		cancel:              cancel,
		p2p:                 cfg.P2P,
		syncService:         cfg.SyncService,
		db:                  cfg.BeaconDB,
		currentSlot:         0,
		highestObservedSlot: 0,
		blockBuf:            blockBuf,
		stateBuf:            stateBuf,
		blockAnnounceBuf:    blockAnnounceBuf,
		syncPollingInterval: cfg.SyncPollingInterval,
		inMemoryBlocks:      map[uint64]*pb.BeaconBlockResponse{},
	}
}

// Start begins the goroutine.
func (s *InitialSync) Start() {
	go func() {
		ticker := time.NewTicker(s.syncPollingInterval)
		s.run(ticker.C)
		ticker.Stop()
	}()
}

// Stop kills the initial sync goroutine.
func (s *InitialSync) Stop() error {
	log.Info("Stopping service")
	s.cancel()
	return nil
}

// run is the main goroutine for the initial sync service.
// delayChan is explicitly passed into this function to facilitate tests that don't require a timeout.
// It is assumed that the goroutine `run` is only called once per instance.
func (s *InitialSync) run(delayChan <-chan time.Time) {

	blockSub := s.p2p.Subscribe(&pb.BeaconBlockResponse{}, s.blockBuf)
	blockAnnounceSub := s.p2p.Subscribe(&pb.BeaconBlockAnnounce{}, s.blockAnnounceBuf)
	beaconStateSub := s.p2p.Subscribe(&pb.BeaconStateResponse{}, s.stateBuf)
	defer func() {
		blockSub.Unsubscribe()
		blockAnnounceSub.Unsubscribe()
		beaconStateSub.Unsubscribe()
		close(s.blockBuf)
		close(s.stateBuf)
	}()

	for {
		select {
		case <-s.ctx.Done():
			log.Debug("Exiting goroutine")
			return
		case <-delayChan:
			if s.currentSlot == 0 {
				continue
			}
			if s.highestObservedSlot == s.currentSlot {
				log.Info("Exiting initial sync and starting normal sync")
				s.syncService.ResumeSync()
				// TODO(#661): Resume sync after completion of initial sync.
				return
			}

			// requests multiple blocks so as to save and sync quickly.
			s.requestBatchedBlocks(s.highestObservedSlot)
		case msg := <-s.blockAnnounceBuf:
			data := msg.Data.(*pb.BeaconBlockAnnounce)

			if data.GetSlotNumber() > s.highestObservedSlot {
				s.highestObservedSlot = data.GetSlotNumber()
			}

			s.requestBatchedBlocks(s.highestObservedSlot)
			log.Debugf("Successfully requested the next block with slot: %d", data.GetSlotNumber())
		case msg := <-s.blockBuf:
			data := msg.Data.(*pb.BeaconBlockResponse)

			if data.Block.GetSlot() > s.highestObservedSlot {
				s.highestObservedSlot = data.Block.GetSlot()
			}

			if s.currentSlot == 0 {
				if s.initialStateRootHash32 != [32]byte{} {
					continue
				}
				if data.GetBlock().GetSlot() != 1 {

					// saves block in memory if it isn't the initial block.
					if _, ok := s.inMemoryBlocks[data.Block.GetSlot()]; !ok {
						s.inMemoryBlocks[data.Block.GetSlot()] = data
					}
					s.requestNextBlockBySlot(1)
					continue
				}
				if err := s.setBlockForInitialSync(data); err != nil {
					log.Errorf("Could not set block for initial sync: %v", err)
				}
				if err := s.requestStateFromPeer(data, msg.Peer); err != nil {
					log.Errorf("Could not request beacon state from peer: %v", err)
				}

				continue
			}
			// if it isn't the block in the next slot it saves it in memory.
			if data.Block.GetSlot() != (s.currentSlot + 1) {
				if _, ok := s.inMemoryBlocks[data.Block.GetSlot()]; !ok {
					s.inMemoryBlocks[data.Block.GetSlot()] = data
				}
				continue
			}

			if err := s.validateAndSaveNextBlock(data); err != nil {
				log.Errorf("Unable to save block: %v", err)
			}
			s.requestNextBlockBySlot(s.currentSlot + 1)
		case msg := <-s.stateBuf:
			data := msg.Data.(*pb.BeaconStateResponse)

			if s.initialStateRootHash32 == [32]byte{} {
				continue
			}

			beaconState := types.NewBeaconState(data.BeaconState)
			hash, err := beaconState.Hash()
			if err != nil {
				log.Errorf("Unable to hash beacon state: %v", err)
			}

			if hash != s.initialStateRootHash32 {
				continue
			}

			if err := s.db.SaveState(beaconState); err != nil {
				log.Errorf("Unable to set beacon state for initial sync %v", err)
			}

			log.Debug("Successfully saved beacon state to the db")

			if s.currentSlot >= beaconState.LastFinalizedSlot() {
				continue
			}

			// sets the current slot to the last finalized slot of the
			// crystallized state to begin our sync from.
			s.currentSlot = beaconState.LastFinalizedSlot()
			log.Debugf("Successfully saved crystallized state with the last finalized slot: %d", beaconState.LastFinalizedSlot())

			s.requestNextBlockBySlot(s.currentSlot + 1)
			beaconStateSub.Unsubscribe()
		}
	}
}

// requestStateFromPeer sends a request to a peer for the corresponding state
// for a beacon block.
func (s *InitialSync) requestStateFromPeer(data *pb.BeaconBlockResponse, peer p2p.Peer) error {
	block := data.Block
	h := block.GetStateRootHash32()
	log.Debugf("Successfully processed incoming block with state hash: %#x", h)
	s.p2p.Send(&pb.BeaconStateRequest{Hash: h[:]}, peer)
	return nil
}

// setBlockForInitialSync sets the first received block as the base finalized
// block for initial sync.
func (s *InitialSync) setBlockForInitialSync(data *pb.BeaconBlockResponse) error {
	block := data.Block

	h, err := b.Hash(block)
	if err != nil {
		return err
	}
	log.WithField("blockhash", fmt.Sprintf("%#x", h)).Debug("State state hash exists locally")

	if err := s.writeBlockToDB(block); err != nil {
		return err
	}

	var blockStateRoot [32]byte
	copy(blockStateRoot[:], block.GetStateRootHash32())
	s.initialStateRootHash32 = blockStateRoot

	log.Infof("Saved block with hash %#x for initial sync", h)
	s.currentSlot = block.GetSlot()
	s.requestNextBlockBySlot(s.currentSlot + 1)
	return nil
}

// requestNextBlock broadcasts a request for a block with the entered slotnumber.
func (s *InitialSync) requestNextBlockBySlot(slotNumber uint64) {
	log.Debugf("Requesting block %d ", slotNumber)
	if _, ok := s.inMemoryBlocks[slotNumber]; ok {
		s.blockBuf <- p2p.Message{
			Data: s.inMemoryBlocks[slotNumber],
		}
		return
	}
	s.p2p.Broadcast(&pb.BeaconBlockRequestBySlotNumber{SlotNumber: slotNumber})
}

// requestBatchedBlocks sends out multiple requests for blocks till a
// specified bound slot number.
func (s *InitialSync) requestBatchedBlocks(endSlot uint64) {
	log.Debug("Requesting batched blocks")
	for i := s.currentSlot + 1; i <= endSlot; i++ {
		s.requestNextBlockBySlot(i)
	}
}

// validateAndSaveNextBlock will validate whether blocks received from the blockfetcher
// routine can be added to the chain.
func (s *InitialSync) validateAndSaveNextBlock(data *pb.BeaconBlockResponse) error {
	block := data.Block
	h, err := b.Hash(block)
	if err != nil {
		return err
	}

	if s.currentSlot == uint64(0) {
		return errors.New("invalid slot number for syncing")
	}

	if (s.currentSlot + 1) == block.GetSlot() {
		if err := s.writeBlockToDB(block); err != nil {
			return err
		}

		log.Infof("Saved block with hash %#x and slot %d for initial sync", h, block.GetSlot())
		s.currentSlot = block.GetSlot()

		// delete block from memory
		if _, ok := s.inMemoryBlocks[block.GetSlot()]; ok {
			delete(s.inMemoryBlocks, block.GetSlot())
		}
	}
	return nil
}

// writeBlockToDB saves the corresponding block to the local DB.
func (s *InitialSync) writeBlockToDB(block *pb.BeaconBlock) error {
	return s.db.SaveBlock(block)
}
