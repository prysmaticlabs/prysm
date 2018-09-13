// Package simulator defines the simulation utility to test the beacon-chain.
package simulator

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/shared/p2p"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "simulator")

// Simulator struct.
type Simulator struct {
	ctx                    context.Context
	cancel                 context.CancelFunc
	p2p                    types.P2P
	web3Service            types.POWChainService
	chainService           types.StateFetcher
	beaconDB               *db.DB
	validator              bool
	delay                  time.Duration
	slotNum                uint64
	broadcastedBlocks      map[[32]byte]*types.Block
	broadcastedBlockHashes [][32]byte
	blockRequestChan       chan p2p.Message
}

// Config options for the simulator service.
type Config struct {
	Delay           time.Duration
	BlockRequestBuf int
	P2P             types.P2P
	Validator       bool
	Web3Service     types.POWChainService
	ChainService    types.StateFetcher
	BeaconDB        *db.DB
}

// DefaultConfig options for the simulator.
func DefaultConfig() *Config {
	return &Config{
		Delay:           time.Second * 5,
		BlockRequestBuf: 100,
	}
}

// NewSimulator creates a simulator instance for a syncer to consume fake, generated blocks.
func NewSimulator(ctx context.Context, cfg *Config) *Simulator {
	ctx, cancel := context.WithCancel(ctx)
	return &Simulator{
		ctx:                    ctx,
		cancel:                 cancel,
		p2p:                    cfg.P2P,
		web3Service:            cfg.Web3Service,
		chainService:           cfg.ChainService,
		beaconDB:               cfg.BeaconDB,
		delay:                  cfg.Delay,
		validator:              cfg.Validator,
		slotNum:                1,
		broadcastedBlocks:      make(map[[32]byte]*types.Block),
		broadcastedBlockHashes: [][32]byte{},
		blockRequestChan:       make(chan p2p.Message, cfg.BlockRequestBuf),
	}
}

// Start the sim.
func (sim *Simulator) Start() {
	log.Info("Starting service")
	go sim.run(time.NewTicker(sim.delay).C, sim.ctx.Done())
}

// Stop the sim.
func (sim *Simulator) Stop() error {
	defer sim.cancel()
	log.Info("Stopping service")
	// Persist the last simulated block in the DB for future sessions
	// to continue from the last simulated slot number.
	if len(sim.broadcastedBlockHashes) > 0 {
		lastBlockHash := sim.broadcastedBlockHashes[len(sim.broadcastedBlockHashes)-1]
		lastBlock := sim.broadcastedBlocks[lastBlockHash]
		return sim.beaconDB.TrackSimulatedBlock(lastBlock)
	}
	return nil
}

func (sim *Simulator) lastSimulatedSessionBlock() (*types.Block, error) {
	return sim.beaconDB.GetLastSimulatedBlock()
}

func (sim *Simulator) run(delayChan <-chan time.Time, done <-chan struct{}) {
	blockReqSub := sim.p2p.Subscribe(&pb.BeaconBlockRequest{}, sim.blockRequestChan)
	defer blockReqSub.Unsubscribe()

	// Check if we saved a simulated block in the DB from a previous session.
	// If that is the case, simulator will start from there.
	var parentHash []byte
	lastSimulatedBlock, err := sim.lastSimulatedSessionBlock()
	if err != nil {
		log.Errorf("Could not fetch last simulated session's block: %v", err)
	}
	if lastSimulatedBlock != nil {
		h, err := lastSimulatedBlock.Hash()
		if err != nil {
			log.Errorf("Could not hash last simulated session's block: %v", err)
		}
		sim.slotNum = lastSimulatedBlock.SlotNumber()
		sim.broadcastedBlockHashes = append(sim.broadcastedBlockHashes, h)
	}

	for {
		select {
		case <-done:
			log.Debug("Simulator context closed, exiting goroutine")
			return
		case <-delayChan:
			activeStateHash, err := sim.chainService.CurrentActiveState().Hash()
			if err != nil {
				log.Errorf("Could not fetch active state hash: %v", err)
				continue
			}
			crystallizedStateHash, err := sim.chainService.CurrentCrystallizedState().Hash()
			if err != nil {
				log.Errorf("Could not fetch crystallized state hash: %v", err)
				continue
			}

			// If we have not broadcast a simulated block yet, we set parent hash
			// to the genesis block.
			if sim.slotNum == 1 {
				parentHash = []byte("genesis")
			} else {
				parentHash = sim.broadcastedBlockHashes[len(sim.broadcastedBlockHashes)-1][:]
			}

			log.WithField("currentSlot", sim.slotNum).Debug("Current slot")

			var powChainRef []byte
			if sim.validator {
				powChainRef = sim.web3Service.LatestBlockHash().Bytes()
			} else {
				powChainRef = []byte{'N', '/', 'A'}
			}

			block := types.NewBlock(&pb.BeaconBlock{
				SlotNumber:            sim.slotNum,
				Timestamp:             ptypes.TimestampNow(),
				PowChainRef:           powChainRef,
				ActiveStateHash:       activeStateHash[:],
				CrystallizedStateHash: crystallizedStateHash[:],
				ParentHash:            parentHash,
			})

			sim.slotNum++

			h, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash simulated block: %v", err)
				continue
			}

			log.WithField("announcedBlockHash", fmt.Sprintf("0x%x", h)).Debug("Announcing block hash")
			sim.p2p.Broadcast(&pb.BeaconBlockHashAnnounce{
				Hash: h[:],
			})
			// We then store the block in a map for later retrieval upon a request for its full
			// data being sent back.
			sim.broadcastedBlocks[h] = block
			sim.broadcastedBlockHashes = append(sim.broadcastedBlockHashes, h)

		case msg := <-sim.blockRequestChan:
			data, ok := msg.Data.(*pb.BeaconBlockRequest)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Error("Received malformed beacon block request p2p message")
				continue
			}
			var h [32]byte
			copy(h[:], data.Hash[:32])

			block := sim.broadcastedBlocks[h]
			h, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash block: %v", err)
				continue
			}
			log.Debugf("Responding to full block request for hash: 0x%x", h)
			// Sends the full block body to the requester.
			res := &pb.BeaconBlockResponse{Block: block.Proto()}
			sim.p2p.Send(res, msg.Peer)
		}
	}
}
