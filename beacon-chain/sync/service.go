package sync

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "sync")

// Service is the gateway and the bridge between the p2p network and the local beacon chain.
// In broad terms, a new block is synced in 4 steps:
//     1. Receive a block hash from a peer
//     2. Request the block for the hash from the network
//     3. Receive the block
//     4. Forward block to the beacon service for full validation
//
//  In addition, Service will handle the following responsibilities:
//     *  Decide which messages are forwarded to other peers
//     *  Filter redundant data and unwanted data
//     *  Drop peers that send invalid data
//     *  Throttle incoming requests
type Service struct {
	ctx                         context.Context
	cancel                      context.CancelFunc
	p2p                         types.P2P
	chainService                types.ChainService
	announceBlockHashBuf        chan p2p.Message
	blockBuf                    chan p2p.Message
	announceCrystallizedHashBuf chan p2p.Message
	crystallizedStateBuf        chan p2p.Message
	announceActiveHashBuf       chan p2p.Message
	activeStateBuf              chan p2p.Message
	storedFinalizedBlocks       map[[32]byte]FinalizedBlock
	syncMode                    Mode
	currentSlotNumber           uint64
}

// Config allows the channel's buffer sizes to be changed.
type Config struct {
	BlockHashBufferSize             int
	BlockBufferSize                 int
	ActiveStateHashBufferSize       int
	ActiveStateBufferSize           int
	CrystallizedStateHashBufferSize int
	CrystallizedStateBufferSize     int
	SyncMode                        Mode
}

// FinalizedBlock stores information regarding a block, its finalized epoch and the contents
// of its crystallized state.
type FinalizedBlock struct {
	BeaconBlock        *types.Block
	LastFinalizedEpoch uint64
	CrystallizedState  *types.CrystallizedState
}

type Mode int

const (
	SYNCMODE_INITIAL Mode = 0
	SYNCMODE_DEFAULT Mode = 1
)

// DefaultConfig provides the default configuration for a sync service.
func DefaultConfig() Config {
	return Config{
		BlockHashBufferSize:             100,
		BlockBufferSize:                 100,
		ActiveStateHashBufferSize:       100,
		ActiveStateBufferSize:           100,
		CrystallizedStateHashBufferSize: 100,
		CrystallizedStateBufferSize:     100,
		SyncMode:                        SYNCMODE_DEFAULT,
	}
}

// NewSyncService accepts a context and returns a new Service.
func NewSyncService(ctx context.Context, cfg Config, beaconp2p types.P2P, cs types.ChainService) *Service {

	ctx, cancel := context.WithCancel(ctx)
	stored, err := cs.HasStoredState()

	if err != nil {
		log.Errorf("error retrieving stored state: %v", err)
		cancel()
		return nil
	}
	if !stored {
		cfg.SyncMode = SYNCMODE_INITIAL
	}

	return &Service{
		ctx:                         ctx,
		cancel:                      cancel,
		p2p:                         beaconp2p,
		chainService:                cs,
		announceBlockHashBuf:        make(chan p2p.Message, cfg.BlockHashBufferSize),
		blockBuf:                    make(chan p2p.Message, cfg.BlockBufferSize),
		announceCrystallizedHashBuf: make(chan p2p.Message, cfg.ActiveStateHashBufferSize),
		crystallizedStateBuf:        make(chan p2p.Message, cfg.ActiveStateBufferSize),
		announceActiveHashBuf:       make(chan p2p.Message, cfg.CrystallizedStateHashBufferSize),
		activeStateBuf:              make(chan p2p.Message, cfg.CrystallizedStateBufferSize),
		storedFinalizedBlocks:       make(map[[32]byte]FinalizedBlock),
		syncMode:                    cfg.SyncMode,
		currentSlotNumber:           0,
	}
}

// Start begins the block processing goroutine.
func (ss *Service) Start() {
	log.Info("Starting service")

	switch ss.syncMode {
	case 0:
		log.Info("Starting initial sync")
		go ss.initialSync(ss.ctx.Done())
	case 1:
		go ss.run(ss.ctx.Done())
	default:
		go ss.run(ss.ctx.Done())

	}

}

// Stop kills the block processing goroutine, but does not wait until the goroutine exits.
func (ss *Service) Stop() error {
	log.Info("Stopping service")
	ss.cancel()
	return nil
}

// isFirstSync checks if this is the first time the node is syncing with the network.
func (ss *Service) isFirstSync() (bool, error) {
	stored, err := ss.chainService.HasStoredState()
	if err != nil {
		return false, fmt.Errorf("error retrieving stored state: %v", err)
	}
	return !stored, nil
}

// setFinalizedBlock stores the contents of a Finalized Block struct with the crystallized
// state hash as a key.
func (ss *Service) setFinalizedBlock(crystallizedStateHash [32]byte, block *types.Block) {
	ss.storedFinalizedBlocks[crystallizedStateHash] = FinalizedBlock{
		BeaconBlock: block,
	}
}

// setFinalizedEpochforMapping saves the finalized epoch for the beacon block so that an
// initial sync can be started.
func (ss *Service) setFinalizedEpochforMapping(crystallizedStateHash [32]byte, block *types.Block, epoch uint64, CrystallizedState *types.CrystallizedState) {
	ss.storedFinalizedBlocks[crystallizedStateHash] = FinalizedBlock{
		BeaconBlock:        block,
		LastFinalizedEpoch: epoch,
		CrystallizedState:  CrystallizedState,
	}
}

// ReceiveBlockHash accepts a block hash.
// New hashes are forwarded to other peers in the network (unimplemented), and
// the contents of the block are requested if the local chain doesn't have the block.
func (ss *Service) ReceiveBlockHash(data *pb.BeaconBlockHashAnnounce, peer p2p.Peer) {
	var h [32]byte
	copy(h[:], data.Hash[:32])
	if ss.chainService.ContainsBlock(h) {
		return
	}
	log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Info("Received incoming block hash, requesting full block data from sender")
	// Request the full block data from peer that sent the block hash.
	ss.p2p.Send(&pb.BeaconBlockRequest{Hash: h[:]}, peer)
}

// ReceiveBlock accepts a block to potentially be included in the local chain.
// The service will filter blocks that have not been requested (unimplemented).
func (ss *Service) ReceiveBlock(data *pb.BeaconBlockResponse) error {
	block, err := types.NewBlock(data)
	if err != nil {
		return fmt.Errorf("could not instantiate new block from proto: %v", err)
	}
	h, err := block.Hash()
	if err != nil {
		return fmt.Errorf("could not hash block: %v", err)
	}
	if ss.chainService.ContainsBlock(h) {
		return nil
	}
	if err := ss.chainService.ProcessBlock(block); err != nil {
		return fmt.Errorf("could not process block: %v", err)
	}
	log.Debugf("Successfully processed incoming block with hash: %x", h)
	return nil
}

// ReceiveCrystallizedStateHash accepts a crystallized state hash.
// New hashes are forwarded to other peers in the network (unimplemented), and
// the contents of the crystallized hash are requested if the local chain doesn't have the hash.
func (ss *Service) ReceiveCrystallizedStateHash(data *pb.CrystallizedStateHashAnnounce, peer p2p.Peer) {
	var h [32]byte
	copy(h[:], data.Hash[:32])
	if ss.chainService.ContainsCrystallizedState(h) {
		log.WithFields(logrus.Fields{"crystallizedStateHash": h}).Debug("Crystallized state hash exists locally")
		return
	}
	log.WithField("crystallizedStateHash", fmt.Sprintf("0x%x", h)).Info("Received crystallized state hash, requesting state data from sender")
	// Request the crystallized hash data from peer that sent the block hash.
	ss.p2p.Send(&pb.CrystallizedStateRequest{Hash: h[:]}, peer)
}

// ReceiveCrystallizedState accepts a crystallized state object to potentially be included in the local chain.
// The service will filter crystallized state objects that have not been requested (unimplemented).
func (ss *Service) ReceiveCrystallizedState(data *pb.CrystallizedStateResponse) error {
	state := types.NewCrystallizedState(data)

	h, err := state.Hash()
	if err != nil {
		return fmt.Errorf("could not hash crystallized state: %v", err)
	}
	if ss.chainService.ContainsCrystallizedState(h) {
		log.WithFields(logrus.Fields{"crystallizedStateHash": h}).Debug("Crystallized state hash exists locally")
		return nil
	}

	if err := ss.chainService.ProcessCrystallizedState(state); err != nil {
		return fmt.Errorf("could not process crystallized state: %v", err)
	}
	log.Debugf("Successfully received incoming crystallized state with hash: %x", h)
	return nil
}

// ReceiveActiveStateHash accepts a active state hash.
// New hashes are forwarded to other peers in the network (unimplemented), and
// the contents of the active hash are requested if the local chain doesn't have the hash.
func (ss *Service) ReceiveActiveStateHash(data *pb.ActiveStateHashAnnounce, peer p2p.Peer) {
	var h [32]byte
	copy(h[:], data.Hash[:32])
	if ss.chainService.ContainsActiveState(h) {
		log.WithFields(logrus.Fields{"activeStateHash": h}).Debug("Active state hash exists locally")
		return
	}
	log.WithField("activeStateHash", fmt.Sprintf("0x%x", h)).Info("Received active state hash, requesting state data from sender")
	// Request the active hash data from peer that sent the block hash.
	ss.p2p.Send(&pb.ActiveStateRequest{Hash: h[:]}, peer)
}

// ReceiveActiveState accepts a active state object to potentially be included in the local chain.
// The service will filter active state objects that have not been requested (unimplemented).
func (ss *Service) ReceiveActiveState(data *pb.ActiveStateResponse) error {
	state := types.NewActiveState(data)

	h, err := state.Hash()
	if err != nil {
		return fmt.Errorf("could not hash active state: %v", err)
	}
	if ss.chainService.ContainsActiveState(h) {
		log.WithFields(logrus.Fields{"activeStateHash": h}).Debug("Active state hash exists locally")
		return nil
	}

	if err := ss.chainService.ProcessActiveState(state); err != nil {
		return fmt.Errorf("could not process active state: %v", err)
	}
	log.Debugf("Successfully received incoming active state with hash: %x", h)
	return nil
}

// GetCrystallizedStateFromPeer sends a request to a peer for the corresponding crystallized state
// for a beacon block.
func (ss *Service) GetCrystallizedStateFromPeer(data *pb.BeaconBlockResponse, peer p2p.Peer) error {
	block, err := types.NewBlock(data)
	if err != nil {
		return fmt.Errorf("could not instantiate new block from proto: %v", err)
	}
	h := block.CrystallizedStateHash()

	log.Debugf("Successfully processed incoming block with crystallized state hash: %x", h)
	ss.p2p.Send(&pb.CrystallizedStateRequest{Hash: h[:]}, peer)
	ss.setFinalizedBlock(h, block)
	return nil
}

// SetFinalizedEpochFromCrystallizedState receives the crystallized state response from the peer
// and sets the finalized epoch for the beacon block.
func (ss *Service) SetFinalizedEpochFromCrystallizedState(data *pb.CrystallizedStateResponse) error {
	state := types.NewCrystallizedState(data)

	h, err := state.Hash()
	if err != nil {
		return fmt.Errorf("could not hash crystallized state: %v", err)
	}
	if ss.chainService.ContainsCrystallizedState(h) {
		log.WithField("crystallizedStateHash", fmt.Sprintf("0x%x", h)).Debug("Crystallized state hash exists locally")
		return nil
	}
	log.Debugf("Successfully received incoming crystallized state with hash: %0x", h)

	finalizedBlock, ok := ss.storedFinalizedBlocks[h]
	if !ok {
		if err := ss.chainService.ProcessCrystallizedState(state); err != nil {
			return fmt.Errorf("could not process crystallized state: %v", err)
		}
		return fmt.Errorf("unable to retrieve finalized block from mapping for hash: %x", h)
	}

	ss.setFinalizedEpochforMapping(h, finalizedBlock.BeaconBlock, state.LastFinalizedEpoch(), state)
	log.Debugf("Saved finalized epoch for block with crystallised state hash: %x", h)
	return nil
}

// validateAndSaveNextBlock will validate whether blocks received from the blockfetcher
// routine can be added to the chain.
func (ss *Service) validateAndSaveNextBlock(data *pb.BeaconBlockResponse) error {
	block, err := types.NewBlock(data)
	if err != nil {
		return fmt.Errorf("could not instantiate new block from proto: %v", err)
	}

	if ss.currentSlotNumber == uint64(0) {
		return fmt.Errorf("invalid slot number for syncing")
	}

	if (ss.currentSlotNumber + 1) == block.SlotNumber() {

		if err := ss.writeBlockToDB(block); err != nil {
			return err
		}
		ss.currentSlotNumber = block.SlotNumber()
	}

	return nil
}

// findLatestFinalizedBlock will retrieve the initial finalized block to be
// saved, after the inital block has been saved syncing will be carried out by
// a separate routine(not implemented yet).
func (ss *Service) findAndSaveLatestFinalizedBlock() error {

	sync, err := ss.isFirstSync()

	if err != nil {
		return err
	}

	if !sync {
		return nil
	}

	if len(ss.storedFinalizedBlocks) > 20 {
		epoch := uint64(0)
		var hash [32]byte
		for k, v := range ss.storedFinalizedBlocks {
			if v.LastFinalizedEpoch > epoch {
				epoch = v.LastFinalizedEpoch
				hash = k
			}
		}
		finalizedBlock, ok := ss.storedFinalizedBlocks[hash]

		if !ok {
			return fmt.Errorf("unable to retrieve finalized block with hash %x", hash)
		}

		if err := ss.writeBlockToDB(finalizedBlock.BeaconBlock); err != nil {
			return err
		}

		ss.currentSlotNumber = finalizedBlock.BeaconBlock.SlotNumber()
		ss.blockFetcher(ss.ctx.Done())
	}
	return nil
}

// writeBlockToDB saves the corresponding block to the local DB.
func (ss *Service) writeBlockToDB(block *types.Block) error {

	if err := ss.chainService.SaveBlock(block); err != nil {
		return err
	}
	return nil
}

func (ss *Service) blockFetcher(done <-chan struct{}) {
	blockSub := ss.p2p.Subscribe(pb.BeaconBlockResponse{}, ss.blockBuf)
	defer blockSub.Unsubscribe()

	for {
		select {
		case <-done:
			log.Infof("Exiting goroutine")
			return
		case msg := <-ss.blockBuf:
			data, ok := msg.Data.(*pb.BeaconBlockResponse)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed beacon block p2p message")
				continue
			}

			if err := ss.validateAndSaveNextBlock(data); err != nil {
				log.Errorf("Could not send request for crystallized state: %v", err)
			}
		}
	}

}

func (ss *Service) initialSync(done <-chan struct{}) {
	blockSub := ss.p2p.Subscribe(pb.BeaconBlockResponse{}, ss.blockBuf)
	crystallizedStateSub := ss.p2p.Subscribe(pb.CrystallizedStateResponse{}, ss.crystallizedStateBuf)

	defer blockSub.Unsubscribe()
	defer crystallizedStateSub.Unsubscribe()
	for {
		select {
		case <-done:
			log.Infof("Exiting goroutine")
			return
		case msg := <-ss.blockBuf:
			data, ok := msg.Data.(*pb.BeaconBlockResponse)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed beacon block p2p message")
				continue
			}

			if err := ss.GetCrystallizedStateFromPeer(data, msg.Peer); err != nil {
				log.Errorf("Could not send request for crystallized state: %v", err)
			}
		case msg := <-ss.crystallizedStateBuf:
			data, ok := msg.Data.(*pb.CrystallizedStateResponse)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed crystallized state p2p message")
				continue
			}
			if err := ss.SetFinalizedEpochFromCrystallizedState(data); err != nil {
				log.Errorf("Could not set epoch for crystallised state: %v", err)
			}

			if err := ss.findAndSaveLatestFinalizedBlock(); err != nil {
				log.Errorf("unable to retrive last finalized block: %v", err)
			}

		}
	}
}

func (ss *Service) run(done <-chan struct{}) {
	announceBlockHashSub := ss.p2p.Subscribe(pb.BeaconBlockHashAnnounce{}, ss.announceBlockHashBuf)
	blockSub := ss.p2p.Subscribe(pb.BeaconBlockResponse{}, ss.blockBuf)
	announceCrystallizedHashSub := ss.p2p.Subscribe(pb.CrystallizedStateHashAnnounce{}, ss.announceCrystallizedHashBuf)
	crystallizedStateSub := ss.p2p.Subscribe(pb.CrystallizedStateResponse{}, ss.crystallizedStateBuf)
	announceActiveHashSub := ss.p2p.Subscribe(pb.ActiveStateHashAnnounce{}, ss.announceActiveHashBuf)
	activeStateSub := ss.p2p.Subscribe(pb.ActiveStateResponse{}, ss.activeStateBuf)

	defer announceBlockHashSub.Unsubscribe()
	defer blockSub.Unsubscribe()
	defer announceCrystallizedHashSub.Unsubscribe()
	defer crystallizedStateSub.Unsubscribe()
	defer announceActiveHashSub.Unsubscribe()
	defer activeStateSub.Unsubscribe()

	for {
		select {
		case <-done:
			log.Infof("Exiting goroutine")
			return
		case msg := <-ss.announceBlockHashBuf:
			data, ok := msg.Data.(*pb.BeaconBlockHashAnnounce)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Error("Received malformed beacon block hash announcement p2p message")
				continue
			}
			ss.ReceiveBlockHash(data, msg.Peer)
		case msg := <-ss.blockBuf:
			data, ok := msg.Data.(*pb.BeaconBlockResponse)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed beacon block p2p message")
				continue
			}
			if err := ss.ReceiveBlock(data); err != nil {
				log.Errorf("Could not receive block: %v", err)
			}
		case msg := <-ss.announceCrystallizedHashBuf:
			data, ok := msg.Data.(*pb.CrystallizedStateHashAnnounce)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Error("Received malformed crystallized state hash announcement p2p message")
				continue
			}
			ss.ReceiveCrystallizedStateHash(data, msg.Peer)
		case msg := <-ss.crystallizedStateBuf:
			data, ok := msg.Data.(*pb.CrystallizedStateResponse)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed crystallized state p2p message")
				continue
			}
			if err := ss.ReceiveCrystallizedState(data); err != nil {
				log.Errorf("Could not receive crystallized state: %v", err)
			}
		case msg := <-ss.announceActiveHashBuf:
			data, ok := msg.Data.(*pb.ActiveStateHashAnnounce)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Error("Received malformed active state hash announcement p2p message")
				continue
			}
			ss.ReceiveActiveStateHash(data, msg.Peer)
		case msg := <-ss.activeStateBuf:
			data, ok := msg.Data.(*pb.ActiveStateResponse)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed active state p2p message")
				continue
			}
			if err := ss.ReceiveActiveState(data); err != nil {
				log.Errorf("Could not receive active state: %v", err)
			}
		}
	}
}
