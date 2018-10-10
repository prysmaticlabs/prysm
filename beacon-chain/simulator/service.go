// Package simulator defines the simulation utility to test the beacon-chain.
package simulator

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "simulator")

type p2pAPI interface {
	Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription
	Send(msg proto.Message, peer p2p.Peer)
	Broadcast(msg proto.Message)
}

// Simulator struct.
type Simulator struct {
	ctx                    context.Context
	cancel                 context.CancelFunc
	p2p                    p2pAPI
	web3Service            types.POWChainService
	beaconDB               beaconDB
	enablePOWChain         bool
	delay                  time.Duration
	slotNum                uint64
	genesisTimestamp       time.Time
	broadcastedBlocks      map[[32]byte]*types.Block
	broadcastedBlockHashes [][32]byte
	blockRequestChan       chan p2p.Message
}

// Config options for the simulator service.
type Config struct {
	Delay           time.Duration
	BlockRequestBuf int
	P2P             p2pAPI
	Web3Service     types.POWChainService
	BeaconDB        beaconDB
	EnablePOWChain  bool
}

type beaconDB interface {
	HasSimulatedBlock() (bool, error)
	GetSimulatedBlock() (*types.Block, error)
	SaveSimulatedBlock(*types.Block) error
	GetActiveState() *types.ActiveState
	GetCrystallizedState() *types.CrystallizedState
	GetCanonicalBlockForSlot(uint64) (*types.Block, error)
	GetCanonicalBlock() (*types.Block, error)
}

// DefaultConfig options for the simulator.
func DefaultConfig() *Config {
	return &Config{
		Delay:           time.Second * time.Duration(params.GetConfig().SlotDuration),
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
		beaconDB:               cfg.BeaconDB,
		delay:                  cfg.Delay,
		enablePOWChain:         cfg.EnablePOWChain,
		slotNum:                1,
		broadcastedBlocks:      make(map[[32]byte]*types.Block),
		broadcastedBlockHashes: [][32]byte{},
		blockRequestChan:       make(chan p2p.Message, cfg.BlockRequestBuf),
	}
}

// Start the sim.
func (sim *Simulator) Start() {
	log.Info("Starting service")
	genesis, err := sim.beaconDB.GetCanonicalBlockForSlot(0)
	if err != nil {
		log.Fatalf("Could not get genesis block: %v", err)
	}
	sim.genesisTimestamp, err = genesis.Timestamp()
	if err != nil {
		log.Fatalf("Could not get genesis timestamp: %v", err)
	}

	go sim.run(time.NewTicker(sim.delay).C)
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
		return sim.beaconDB.SaveSimulatedBlock(lastBlock)
	}
	return nil
}

func (sim *Simulator) lastSimulatedSessionBlock() (*types.Block, error) {
	hasBlock, err := sim.beaconDB.HasSimulatedBlock()
	if err != nil {
		return nil, fmt.Errorf("Could not determine if a previous simulation occurred: %v", err)
	}
	if !hasBlock {
		return nil, nil
	}

	simulatedBlock, err := sim.beaconDB.GetSimulatedBlock()
	if err != nil {
		return nil, fmt.Errorf("Could not fetch simulated block from db: %v", err)
	}
	return simulatedBlock, nil
}

func (sim *Simulator) run(delayChan <-chan time.Time) {
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
		case <-sim.ctx.Done():
			log.Debug("Simulator context closed, exiting goroutine")
			return
		case <-delayChan:
			activeStateHash, err := sim.beaconDB.GetActiveState().Hash()
			if err != nil {
				log.Errorf("Could not fetch active state hash: %v", err)
				continue
			}
			crystallizedStateHash, err := sim.beaconDB.GetCrystallizedState().Hash()
			if err != nil {
				log.Errorf("Could not fetch crystallized state hash: %v", err)
				continue
			}

			// If we have not broadcast a simulated block yet, we set parent hash
			// to the genesis block.
			var hash [32]byte
			if sim.slotNum == 1 {
				genesisBlock, err := sim.beaconDB.GetCanonicalBlockForSlot(0)
				if err != nil {
					log.Errorf("Failed to retrieve genesis block: %v", err)
					continue
				}
				hash, err = genesisBlock.Hash()
				if err != nil {
					log.Errorf("Failed to hash genesis block: %v", err)
					continue
				}
				parentHash = hash[:]
			} else {
				parentHash = sim.broadcastedBlockHashes[len(sim.broadcastedBlockHashes)-1][:]
			}

			log.WithField("currentSlot", sim.slotNum).Debug("Current slot")

			var powChainRef []byte
			if sim.enablePOWChain {
				powChainRef = sim.web3Service.LatestBlockHash().Bytes()
			} else {
				powChainRef = []byte{byte(sim.slotNum)}
			}

			blockSlot := utils.CurrentSlot(sim.genesisTimestamp)
			if blockSlot == 0 {
				// cannot process a genesis block, so we start from 1
				blockSlot = 1
			}

			block := types.NewBlock(&pb.BeaconBlock{
				Slot:                  blockSlot,
				Timestamp:             ptypes.TimestampNow(),
				PowChainRef:           powChainRef,
				ActiveStateRoot:       activeStateHash[:],
				CrystallizedStateRoot: crystallizedStateHash[:],
				AncestorHashes:        [][]byte{parentHash},
				Attestations: []*pb.AggregatedAttestation{
					{Slot: sim.slotNum - 1, AttesterBitfield: []byte{byte(255)}},
				},
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
			data := msg.Data.(*pb.BeaconBlockRequest)
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
			res := &pb.BeaconBlockResponse{Block: block.Proto(), Attestation: &pb.AggregatedAttestation{
				Slot:             sim.slotNum - 1,
				AttesterBitfield: []byte{byte(255)},
			}}
			sim.p2p.Send(res, msg.Peer)
		}
	}
}
