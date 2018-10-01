// Package simulator defines the simulation utility to test the beacon-chain.
package simulator

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "simulator")

// Simulator struct.
type Simulator struct {
	ctx                    context.Context
	cancel                 context.CancelFunc
	p2p                    shared.P2P
	web3Service            types.POWChainService
	chainService           types.StateFetcher
	beaconDB               ethdb.Database
	enablePOWChain         bool
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
	P2P             shared.P2P
	Web3Service     types.POWChainService
	ChainService    types.StateFetcher
	BeaconDB        ethdb.Database
	EnablePOWChain  bool
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
		chainService:           cfg.ChainService,
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
		encoded, err := lastBlock.Marshal()
		if err != nil {
			return err
		}
		return sim.beaconDB.Put([]byte("last-simulated-block"), encoded)
	}
	return nil
}

func (sim *Simulator) lastSimulatedSessionBlock() (*types.Block, error) {
	hasSimulated, err := sim.beaconDB.Has([]byte("last-simulated-block"))
	if err != nil {
		return nil, fmt.Errorf("Could not determine if a previous simulation occurred: %v", err)
	}
	if !hasSimulated {
		return nil, nil
	}
	enc, err := sim.beaconDB.Get([]byte("last-simulated-block"))
	if err != nil {
		return nil, fmt.Errorf("Could not fetch simulated block from db: %v", err)
	}
	lastSimulatedBlockProto := &pb.BeaconBlock{}
	if err = proto.Unmarshal(enc, lastSimulatedBlockProto); err != nil {
		return nil, fmt.Errorf("Could not unmarshal simulated block from db: %v", err)
	}
	return types.NewBlock(lastSimulatedBlockProto), nil
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
			var hash [32]byte
			if sim.slotNum == 1 {
				genesisBlock, err := sim.chainService.GenesisBlock()
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

			block := types.NewBlock(&pb.BeaconBlock{
				SlotNumber:            sim.slotNum,
				Timestamp:             ptypes.TimestampNow(),
				PowChainRef:           powChainRef,
				ActiveStateHash:       activeStateHash[:],
				CrystallizedStateHash: crystallizedStateHash[:],
				ParentHash:            parentHash,
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
