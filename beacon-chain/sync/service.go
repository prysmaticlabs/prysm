// Package sync defines the utilities for the beacon-chain to sync with the network.
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
	synced                      bool
	announceBlockHashBuf        chan p2p.Message
	blockBuf                    chan p2p.Message
	announceCrystallizedHashBuf chan p2p.Message
	crystallizedStateBuf        chan p2p.Message
	announceActiveHashBuf       chan p2p.Message
	activeStateBuf              chan p2p.Message
}

// Config allows the channel's buffer sizes to be changed.
type Config struct {
	BlockHashBufferSize             int
	BlockBufferSize                 int
	ActiveStateHashBufferSize       int
	ActiveStateBufferSize           int
	CrystallizedStateHashBufferSize int
	CrystallizedStateBufferSize     int
}

// DefaultConfig provides the default configuration for a sync service.
func DefaultConfig() Config {
	return Config{
		BlockHashBufferSize:             100,
		BlockBufferSize:                 100,
		ActiveStateHashBufferSize:       100,
		ActiveStateBufferSize:           100,
		CrystallizedStateHashBufferSize: 100,
		CrystallizedStateBufferSize:     100,
	}
}

// NewSyncService accepts a context and returns a new Service.
func NewSyncService(ctx context.Context, cfg Config, beaconp2p types.P2P, cs types.ChainService) *Service {
	ctx, cancel := context.WithCancel(ctx)
	stored, err := cs.HasStoredState()
	if err != nil {
		log.Errorf("error retrieving stored state: %v", err)
	}

	return &Service{
		ctx:                         ctx,
		cancel:                      cancel,
		p2p:                         beaconp2p,
		chainService:                cs,
		synced:                      !stored,
		announceBlockHashBuf:        make(chan p2p.Message, cfg.BlockHashBufferSize),
		blockBuf:                    make(chan p2p.Message, cfg.BlockBufferSize),
		announceCrystallizedHashBuf: make(chan p2p.Message, cfg.ActiveStateHashBufferSize),
		crystallizedStateBuf:        make(chan p2p.Message, cfg.ActiveStateBufferSize),
		announceActiveHashBuf:       make(chan p2p.Message, cfg.CrystallizedStateHashBufferSize),
		activeStateBuf:              make(chan p2p.Message, cfg.CrystallizedStateBufferSize),
	}
}

// Start begins the block processing goroutine.
func (ss *Service) Start() {
	stored, err := ss.chainService.HasStoredState()
	if err != nil {
		log.Errorf("error retrieving stored state: %v", err)
		return
	}

	if !stored {
		// TODO: Support initial sync when the chain is partially synced with the network.
		return
	}

	go ss.run(ss.ctx.Done())
}

// Stop kills the block processing goroutine, but does not wait until the goroutine exits.
func (ss *Service) Stop() error {
	log.Info("Stopping service")
	ss.cancel()
	return nil
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
func (ss *Service) ReceiveBlock(data *pb.BeaconBlock) error {
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
func (ss *Service) ReceiveCrystallizedState(data *pb.CrystallizedState) error {
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
//
// TODO: implement hash forwarding
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
//
// TODO: implement filter for non requested state objects.
func (ss *Service) ReceiveActiveState(data *pb.ActiveState) error {
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
			response, ok := msg.Data.(*pb.BeaconBlockResponse)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed beacon block p2p message")
				continue
			}
			if err := ss.ReceiveBlock(response.Block); err != nil {
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
			response, ok := msg.Data.(*pb.CrystallizedStateResponse)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed crystallized state p2p message")
				continue
			}
			if err := ss.ReceiveCrystallizedState(response.CrystallizedState); err != nil {
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
			response, ok := msg.Data.(*pb.ActiveStateResponse)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Errorf("Received malformed active state p2p message")
				continue
			}
			if err := ss.ReceiveActiveState(response.ActiveState); err != nil {
				log.Errorf("Could not receive active state: %v", err)
			}
		}
	}
}
