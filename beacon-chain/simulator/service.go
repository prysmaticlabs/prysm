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
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/slotticker"
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
	ctx              context.Context
	cancel           context.CancelFunc
	p2p              p2pAPI
	web3Service      types.POWChainService
	beaconDB         beaconDB
	enablePOWChain   bool
	blockRequestChan chan p2p.Message
}

// Config options for the simulator service.
type Config struct {
	BlockRequestBuf int
	P2P             p2pAPI
	Web3Service     types.POWChainService
	BeaconDB        beaconDB
	EnablePOWChain  bool
}

type beaconDB interface {
	GetActiveState() *types.ActiveState
	GetCrystallizedState() *types.CrystallizedState
	GetCanonicalBlock() (*types.Block, error)
	GetGenesisTime() (time.Time, error)
}

// DefaultConfig options for the simulator.
func DefaultConfig() *Config {
	return &Config{
		BlockRequestBuf: 100,
	}
}

// NewSimulator creates a simulator instance for a syncer to consume fake, generated blocks.
func NewSimulator(ctx context.Context, cfg *Config) *Simulator {
	ctx, cancel := context.WithCancel(ctx)
	return &Simulator{
		ctx:              ctx,
		cancel:           cancel,
		p2p:              cfg.P2P,
		web3Service:      cfg.Web3Service,
		beaconDB:         cfg.BeaconDB,
		enablePOWChain:   cfg.EnablePOWChain,
		blockRequestChan: make(chan p2p.Message, cfg.BlockRequestBuf),
	}
}

// Start the sim.
func (sim *Simulator) Start() {
	log.Info("Starting service")
	genesisTime, err := sim.beaconDB.GetGenesisTime()
	if err != nil {
		log.Fatal(err)
		return
	}

	slotTicker := slotticker.GetSlotTicker(genesisTime, params.GetConfig().SlotDuration)
	go func() {
		sim.run(slotTicker.C(), sim.blockRequestChan)
		close(sim.blockRequestChan)
		slotTicker.Done()
	}()
}

// Stop the sim.
func (sim *Simulator) Stop() error {
	defer sim.cancel()
	log.Info("Stopping service")
	return nil
}

func (sim *Simulator) run(slotInterval <-chan uint64, requestChan <-chan p2p.Message) {
	blockReqSub := sim.p2p.Subscribe(&pb.BeaconBlockRequest{}, sim.blockRequestChan)
	defer blockReqSub.Unsubscribe()

	lastBlock, err := sim.beaconDB.GetCanonicalBlock()
	if err != nil {
		log.Errorf("Could not fetch latest block: %v", err)
		return
	}

	lastHash, err := lastBlock.Hash()
	if err != nil {
		log.Errorf("Could not get hash of the latest block: %v", err)
	}

	broadcastedBlocks := map[[32]byte]*types.Block{}

	for {
		select {
		case <-sim.ctx.Done():
			log.Debug("Simulator context closed, exiting goroutine")
			return
		case slot := <-slotInterval:
			aStateHash, err := sim.beaconDB.GetActiveState().Hash()
			if err != nil {
				log.Errorf("Could not fetch active state hash: %v", err)
				continue
			}
			cStateHash, err := sim.beaconDB.GetCrystallizedState().Hash()
			if err != nil {
				log.Errorf("Failed to fetch crystallized state hash: %v", err)
				continue
			}

			var powChainRef []byte
			if sim.enablePOWChain {
				powChainRef = sim.web3Service.LatestBlockHash().Bytes()
			} else {
				powChainRef = []byte{byte(slot)}
			}

			parentHash := make([]byte, 32)
			copy(parentHash, lastHash[:])
			block := types.NewBlock(&pb.BeaconBlock{
				Slot:                  slot,
				Timestamp:             ptypes.TimestampNow(),
				PowChainRef:           powChainRef,
				ActiveStateRoot:       aStateHash[:],
				CrystallizedStateRoot: cStateHash[:],
				AncestorHashes:        [][]byte{parentHash},
				Attestations: []*pb.AggregatedAttestation{
					{Slot: slot - 1, AttesterBitfield: []byte{byte(255)}},
				},
			})

			hash, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash simulated block: %v", err)
				continue
			}
			sim.p2p.Broadcast(&pb.BeaconBlockHashAnnounce{
				Hash: hash[:],
			})

			log.WithFields(logrus.Fields{
				"hash": fmt.Sprintf("%#x", hash),
				"slot": slot,
			}).Debug("Broadcast block hash")

			broadcastedBlocks[hash] = block

			lastHash = hash
		case msg := <-requestChan:
			data := msg.Data.(*pb.BeaconBlockRequest)
			var hash [32]byte
			copy(hash[:], data.Hash)

			block := broadcastedBlocks[hash]
			if block == nil {
				log.Errorf("Requested block not found: %#x", hash)
				continue
			}

			log.WithFields(logrus.Fields{
				"hash": fmt.Sprintf("%#x", hash),
			}).Debug("Responding to full block request")

			// Sends the full block body to the requester.
			res := &pb.BeaconBlockResponse{Block: block.Proto(), Attestation: &pb.AggregatedAttestation{
				Slot:             block.SlotNumber(),
				AttesterBitfield: []byte{byte(255)},
			}}
			sim.p2p.Send(res, msg.Peer)

			delete(broadcastedBlocks, hash)
		}
	}
}
