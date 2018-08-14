package initialsync

import (
	"context"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "sync")

// Mode refers to the type for the sync mode of the client.
type Mode int

// This specifies the different sync modes.
const (
	SyncModeInitial Mode = 0
	SyncModeDefault Mode = 1
)

// Config allows the channel's buffer sizes to be changed.
type Config struct {
	SyncMode            Mode
	CurrentSlotNumber   uint64
	SyncPollingInterval time.Duration
}

// DefaultConfig provides the default configuration for a sync service.
func DefaultConfig() Config {
	return Config{
		SyncMode:            SyncModeDefault,
		CurrentSlotNumber:   0,
		SyncPollingInterval: time.Second,
	}
}

type ChainService interface {
	HasStoredState() (bool, error)
	SaveBlock(*types.Block) error
}

type Service struct {
	ctx                          context.Context
	cancel                       context.CancelFunc
	p2p                          types.P2P
	chainService                 ChainService
	currentSlotNumber            uint64
	syncPollingInterval          time.Duration
	initialCrystallizedStateHash [32]byte
}

func NewInitialSyncService(ctx context.Context, cfg Config, beaconp2p types.P2P, cs ChainService) *Service {
	ctx, cancel := context.WithCancel(ctx)
	stored, err := cs.HasStoredState()

	if err != nil {
		log.Errorf("error retrieving stored state: %v", err)
	}

	if !stored {
		cfg.SyncMode = SyncModeInitial
	}

	return &Service{
		ctx:                 ctx,
		cancel:              cancel,
		p2p:                 beaconp2p,
		chainService:        cs,
		syncPollingInterval: cfg.SyncPollingInterval,
	}
}

func (s *Service) Start() {
	go func() {
		ticker := time.NewTicker(s.syncPollingInterval)
		blockBuf := make(chan p2p.Message)
		crystallizedStateBuf := make(chan p2p.Message)
		s.run(ticker.C, make(chan p2p.Message), make(chan p2p.Message))
		close(crystallizedStateBuf)
		close(blockBuf)
		ticker.Stop()
	}()
}

// run is the main goroutine for the intial sync service
// delayChan is explicitly passed into this function to facilitate tests that don't require a timeout
func (s *Service) run(delaychan <-chan time.Time, blockBuf chan p2p.Message, crystallizedStateBuf chan p2p.Message) {
	blockSub := s.p2p.Subscribe(pb.BeaconBlockResponse{}, blockBuf)
	crystallizedStateSub := s.p2p.Subscribe(pb.CrystallizedStateResponse{}, crystallizedStateBuf)
	defer blockSub.Unsubscribe()
	defer blockSub.Unsubscribe()

	highestObservedSlot := uint64(0)

	for {
		select {
		case <-s.ctx.Done():
			log.Infof("Exiting goroutine")
			return
		case <-delaychan:
			if highestObservedSlot == s.currentSlotNumber {
				log.Infof("Exiting initial sync and starting normal sync")
				// TODO: Start normal sync
				return
			}
		case msg := <-blockBuf:
			data, ok := msg.Data.(*pb.BeaconBlockResponse)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed beacon block p2p message")
				continue
			}

			if data.Block.GetSlotNumber() > highestObservedSlot {
				highestObservedSlot = data.Block.GetSlotNumber()
			}

			if s.currentSlotNumber == 0 {
				if s.initialCrystallizedStateHash != [32]byte{} {
					continue
				}
				if err := s.SetBlockForInitialSync(data); err != nil {
					log.Errorf("Could not set block for initial sync: %v", err)
				}
				if err := s.RequestCrystallizedStateFromPeer(data, msg.Peer); err != nil {
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
		case msg := <-crystallizedStateBuf:
			data, ok := msg.Data.(*pb.CrystallizedStateResponse)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed crystallized state p2p message")
				continue
			}

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

// RequestCrystallizedStateFromPeer sends a request to a peer for the corresponding crystallized state
// for a beacon block.
func (s *Service) RequestCrystallizedStateFromPeer(data *pb.BeaconBlockResponse, peer p2p.Peer) error {
	block, err := types.NewBlock(data.Block)
	if err != nil {
		return fmt.Errorf("could not instantiate new block from proto: %v", err)
	}
	h := block.CrystallizedStateHash()
	log.Debugf("Successfully processed incoming block with crystallized state hash: %x", h)
	s.p2p.Send(&pb.CrystallizedStateRequest{Hash: h[:]}, peer)
	return nil
}

// SetBlockForInitialSync sets the first received block as the base finalized
// block for initial sync.
func (s *Service) SetBlockForInitialSync(data *pb.BeaconBlockResponse) error {
	block, err := types.NewBlock(data.Block)
	if err != nil {
		return fmt.Errorf("could not instantiate new block from proto: %v", err)
	}

	h, err := block.Hash()
	if err != nil {
		return err
	}
	log.WithField("Block received with hash", fmt.Sprintf("0x%x", h)).Debug("Crystallized state hash exists locally")

	if err := s.writeBlockToDB(block); err != nil {
		return err
	}

	s.initialCrystallizedStateHash = block.CrystallizedStateHash()

	log.Infof("Saved block with hash 0%x for initial sync", h)
	return nil
}

// requestNextBlock broadcasts a request for a block with the next slotnumber.
func (s *Service) requestNextBlock() {
	s.p2p.Broadcast(&pb.BeaconBlockRequestBySlotNumber{SlotNumber: (s.currentSlotNumber + 1)})
}

// validateAndSaveNextBlock will validate whether blocks received from the blockfetcher
// routine can be added to the chain.
func (s *Service) validateAndSaveNextBlock(data *pb.BeaconBlockResponse) error {
	block, err := types.NewBlock(data.Block)
	if err != nil {
		return fmt.Errorf("could not instantiate new block from proto: %v", err)
	}

	if s.currentSlotNumber == uint64(0) {
		return fmt.Errorf("invalid slot number for syncing")
	}

	if (s.currentSlotNumber + 1) == block.SlotNumber() {

		if err := s.writeBlockToDB(block); err != nil {
			return err
		}
		s.currentSlotNumber = block.SlotNumber()
	}
	return nil
}

// writeBlockToDB saves the corresponding block to the local DB.
func (s *Service) writeBlockToDB(block *types.Block) error {
	return s.chainService.SaveBlock(block)
}
