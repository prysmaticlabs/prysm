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
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "initial-sync")

// Config defines the configurable properties of InitialSync.
//
type Config struct {
	SyncPollingInterval         time.Duration
	BlockBufferSize             int
	CrystallizedStateBufferSize int
	BeaconDB                    beaconDB
	P2P                         p2pAPI
	SyncService                 syncService
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

type p2pAPI interface {
	Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription
	Send(msg proto.Message, peer p2p.Peer)
	Broadcast(msg proto.Message)
}

type beaconDB interface {
	HasStoredState() (bool, error)
	SaveBlock(*types.Block) error
}

// SyncService is the interface for the Sync service.
// InitialSync calls `Start` when initial sync completes.
type syncService interface {
	Start()
}

// InitialSync defines the main class in this package.
// See the package comments for a general description of the service's functions.
type InitialSync struct {
	ctx                          context.Context
	cancel                       context.CancelFunc
	p2p                          p2pAPI
	syncService                  syncService
	db                           beaconDB
	blockBuf                     chan p2p.Message
	crystallizedStateBuf         chan p2p.Message
	currentSlot                  uint64
	syncPollingInterval          time.Duration
	initialCrystallizedStateRoot [32]byte
}

// NewInitialSyncService constructs a new InitialSyncService.
// This method is normally called by the main node.
func NewInitialSyncService(ctx context.Context,
	cfg Config,
) *InitialSync {
	ctx, cancel := context.WithCancel(ctx)

	blockBuf := make(chan p2p.Message, cfg.BlockBufferSize)
	crystallizedStateBuf := make(chan p2p.Message, cfg.CrystallizedStateBufferSize)

	return &InitialSync{
		ctx:                  ctx,
		cancel:               cancel,
		p2p:                  cfg.P2P,
		syncService:          cfg.SyncService,
		db:                   cfg.BeaconDB,
		blockBuf:             blockBuf,
		crystallizedStateBuf: crystallizedStateBuf,
		syncPollingInterval:  cfg.SyncPollingInterval,
	}
}

// Start begins the goroutine.
func (s *InitialSync) Start() {
	stored, err := s.db.HasStoredState()
	if err != nil {
		log.Errorf("error retrieving stored state: %v", err)
		return
	}

	if stored {
		// TODO(555): Bail out of the sync service if the chain is only partially synced.
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
			if highestObservedSlot == s.currentSlot {
				log.Info("Exiting initial sync and starting normal sync")
				// TODO(#426): Resume sync after completion of initial sync.
				// See comment in Sync service's Start function for explanation.
				return
			}
		case msg := <-s.blockBuf:
			data := msg.Data.(*pb.BeaconBlockResponse)

			if data.Block.GetSlot() > highestObservedSlot {
				highestObservedSlot = data.Block.GetSlot()
			}

			if s.currentSlot == 0 {
				if s.initialCrystallizedStateRoot != [32]byte{} {
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

			if data.Block.GetSlot() != (s.currentSlot + 1) {
				continue
			}

			if err := s.validateAndSaveNextBlock(data); err != nil {
				log.Errorf("Unable to save block: %v", err)
			}
			s.requestNextBlock()
		case msg := <-s.crystallizedStateBuf:
			data := msg.Data.(*pb.CrystallizedStateResponse)

			if s.initialCrystallizedStateRoot == [32]byte{} {
				continue
			}

			crystallizedState := types.NewCrystallizedState(data.CrystallizedState)
			hash, err := crystallizedState.Hash()
			if err != nil {
				log.Errorf("Unable to hash crytsallized state: %v", err)
			}

			if hash != s.initialCrystallizedStateRoot {
				continue
			}

			s.currentSlot = crystallizedState.LastFinalizedSlot()
			s.requestNextBlock()
			crystallizedStateSub.Unsubscribe()
		}
	}
}

// requestCrystallizedStateFromPeer sends a request to a peer for the corresponding crystallized state
// for a beacon block.
func (s *InitialSync) requestCrystallizedStateFromPeer(data *pb.BeaconBlockResponse, peer p2p.Peer) error {
	block := types.NewBlock(data.Block)
	h := block.CrystallizedStateRoot()
	log.Debugf("Successfully processed incoming block with crystallized state hash: %#x", h)
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
	log.WithField("blockhash", fmt.Sprintf("%#x", h)).Debug("Crystallized state hash exists locally")

	if err := s.writeBlockToDB(block); err != nil {
		return err
	}

	s.initialCrystallizedStateRoot = block.CrystallizedStateRoot()

	log.Infof("Saved block with hash %#x for initial sync", h)
	return nil
}

// requestNextBlock broadcasts a request for a block with the next slotnumber.
func (s *InitialSync) requestNextBlock() {
	s.p2p.Broadcast(&pb.BeaconBlockRequestBySlotNumber{SlotNumber: (s.currentSlot + 1)})
}

// validateAndSaveNextBlock will validate whether blocks received from the blockfetcher
// routine can be added to the chain.
func (s *InitialSync) validateAndSaveNextBlock(data *pb.BeaconBlockResponse) error {
	block := types.NewBlock(data.Block)

	if s.currentSlot == uint64(0) {
		return errors.New("invalid slot number for syncing")
	}

	if (s.currentSlot + 1) == block.SlotNumber() {

		if err := s.writeBlockToDB(block); err != nil {
			return err
		}
		s.currentSlot = block.SlotNumber()
	}
	return nil
}

// writeBlockToDB saves the corresponding block to the local DB.
func (s *InitialSync) writeBlockToDB(block *types.Block) error {
	return s.db.SaveBlock(block)
}
