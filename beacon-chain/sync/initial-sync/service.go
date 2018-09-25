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

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "initialsync")

// Config defines the configurable properties of InitialSync.
//
type Config struct {
	SyncPollingInterval         time.Duration
	BlockBufferSize             int
	CrystallizedStateBufferSize int
}

// DefaultConfig provides the default configuration for a sync service.
// SyncPollingInterval determines how frequently the service checks that initial sync is complete.
// BlockBufferSize determines that buffer size of the `blockBuf` channel.
// CrystallizedStateBufferSize determines the buffer size of thhe `crystallizedStateBuf` channel.
func DefaultConfig() Config {
	return Config{
		SyncPollingInterval:         1 * time.Second,
		BlockBufferSize:             100,
		CrystallizedStateBufferSize: 100,
	}
}

// DB is the interface for the DB service.
type DB interface {
	HasInitialState() bool
	SaveBlock(*types.Block) error
}

// SyncService is the interface for the Sync service.
// InitialSync calls `Start` when initial sync completes.
type SyncService interface {
	Start()
}

// InitialSync defines the main class in this package.
// See the package comments for a general description of the service's functions.
type InitialSync struct {
	ctx                          context.Context
	cancel                       context.CancelFunc
	db 				DB
	p2p                          shared.P2P
	syncService                  SyncService
	blockBuf                     chan p2p.Message
	crystallizedStateBuf         chan p2p.Message
	currentSlotNumber            uint64
	syncPollingInterval          time.Duration
	initialCrystallizedStateHash [32]byte
}

// NewInitialSyncService constructs a new InitialSyncService.
// This method is normally called by the main node.
func NewInitialSyncService(ctx context.Context,
	cfg Config,
	db DB,
	beaconp2p shared.P2P,
	syncService SyncService,
) *InitialSync {
	ctx, cancel := context.WithCancel(ctx)

	blockBuf := make(chan p2p.Message, cfg.BlockBufferSize)
	crystallizedStateBuf := make(chan p2p.Message, cfg.CrystallizedStateBufferSize)

	return &InitialSync{
		ctx:                  ctx,
		cancel:               cancel,
		p2p:                  beaconp2p,
		db: 		db,
		syncService:          syncService,
		blockBuf:             blockBuf,
		crystallizedStateBuf: crystallizedStateBuf,
		syncPollingInterval:  cfg.SyncPollingInterval,
	}
}

// Start begins the goroutine.
func (s *InitialSync) Start() {

	stored := s.db.HasInitialState()

	if stored {
		// TODO: Bail out of the sync service if the chain is only partially synced.
		log.Info("Chain state detected, exiting initial sync")
		return
	}

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
func (s *InitialSync) run(delaychan <-chan time.Time) {
	blockSub := s.p2p.Subscribe(&pb.BeaconBlockResponse{}, s.blockBuf)
	crystallizedStateSub := s.p2p.Subscribe(&pb.CrystallizedStateResponse{}, s.crystallizedStateBuf)
	defer func() {
		blockSub.Unsubscribe()
		crystallizedStateSub.Unsubscribe()
		close(s.blockBuf)
		close(s.crystallizedStateBuf)
	}()

	highestObservedSlot := uint64(0)

	for {
		select {
		case <-s.ctx.Done():
			log.Debug("Exiting goroutine")
			return
		case <-delaychan:
			if highestObservedSlot == s.currentSlotNumber {
				log.Info("Exiting initial sync and starting normal sync")
				// TODO: Resume sync after completion of initial sync.
				// See comment in Sync service's Start function for explanation.
				return
			}
		case msg := <-s.blockBuf:
			data := msg.Data.(*pb.BeaconBlockResponse)

			if data.Block.GetSlotNumber() > highestObservedSlot {
				highestObservedSlot = data.Block.GetSlotNumber()
			}

			if s.currentSlotNumber == 0 {
				if s.initialCrystallizedStateHash != [32]byte{} {
					continue
				}
				if err := s.setBlockForInitialSync(data); err != nil {
					log.Errorf("Could not set block for initial sync: %v", err)
				}
				if err := s.requestCrystallizedStateFromPeer(data, msg.Peer); err != nil {
					log.Errorf("Could not request crystallized state from peer: %v", err)
				}

				continue
			}

			if data.Block.GetSlotNumber() != (s.currentSlotNumber + 1) {
				continue
			}

			if err := s.validateAndSaveNextBlock(data); err != nil {
				log.Errorf("Unable to save block: %v", err)
			}
			s.requestNextBlock()
		case msg := <-s.crystallizedStateBuf:
			data := msg.Data.(*pb.CrystallizedStateResponse)

			if s.initialCrystallizedStateHash == [32]byte{} {
				continue
			}

			crystallizedState := types.NewCrystallizedState(data.CrystallizedState)
			hash, err := crystallizedState.Hash()
			if err != nil {
				log.Errorf("Unable to hash crytsallized state: %v", err)
			}

			if hash != s.initialCrystallizedStateHash {
				continue
			}

			s.currentSlotNumber = crystallizedState.LastFinalizedSlot()
			s.requestNextBlock()
			crystallizedStateSub.Unsubscribe()
		}
	}
}

// requestCrystallizedStateFromPeer sends a request to a peer for the corresponding crystallized state
// for a beacon block.
func (s *InitialSync) requestCrystallizedStateFromPeer(data *pb.BeaconBlockResponse, peer p2p.Peer) error {
	block := types.NewBlock(data.Block)
	h := block.CrystallizedStateHash()
	log.Debugf("Successfully processed incoming block with crystallized state hash: %x", h)
	s.p2p.Send(&pb.CrystallizedStateRequest{Hash: h[:]}, peer)
	return nil
}

// setBlockForInitialSync sets the first received block as the base finalized
// block for initial sync.
func (s *InitialSync) setBlockForInitialSync(data *pb.BeaconBlockResponse) error {
	block := types.NewBlock(data.Block)

	h, err := block.Hash()
	if err != nil {
		return err
	}
	log.WithField("blockhash", fmt.Sprintf("%x", h)).Debug("Crystallized state hash exists locally")

	if err := s.db.SaveBlock(block); err != nil {
		return err
	}

	s.initialCrystallizedStateHash = block.CrystallizedStateHash()

	log.Infof("Saved block with hash %x for initial sync", h)
	return nil
}

// requestNextBlock broadcasts a request for a block with the next slotnumber.
func (s *InitialSync) requestNextBlock() {
	s.p2p.Broadcast(&pb.BeaconBlockRequestBySlotNumber{SlotNumber: (s.currentSlotNumber + 1)})
}

// validateAndSaveNextBlock will validate whether blocks received from the blockfetcher
// routine can be added to the chain.
func (s *InitialSync) validateAndSaveNextBlock(data *pb.BeaconBlockResponse) error {
	block := types.NewBlock(data.Block)

	if s.currentSlotNumber == uint64(0) {
		return errors.New("invalid slot number for syncing")
	}

	if (s.currentSlotNumber + 1) == block.SlotNumber() {
		if err := s.db.SaveBlock(block); err != nil {
			return err
		}
		s.currentSlotNumber = block.SlotNumber()
	}
	return nil
}
