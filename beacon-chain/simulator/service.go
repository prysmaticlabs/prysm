// Package simulator defines the simulation utility to test the beacon-chain.
package simulator

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotticker"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "simulator")

type p2pAPI interface {
	Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription
	Send(msg proto.Message, peer p2p.Peer)
	Broadcast(msg proto.Message)
}

type powChainService interface {
	LatestBlockHash() common.Hash
}

// Simulator struct.
type Simulator struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	p2p                  p2pAPI
	web3Service          powChainService
	beaconDB             *db.BeaconDB
	enablePOWChain       bool
	blockRequestChan     chan p2p.Message
	blockBySlotChan      chan p2p.Message
	stateReqChan         chan p2p.Message
	chainHeadRequestChan chan p2p.Message
}

// Config options for the simulator service.
type Config struct {
	BlockRequestBuf     int
	BlockSlotBuf        int
	ChainHeadRequestBuf int
	StateReqBuf         int
	P2P                 p2pAPI
	Web3Service         powChainService
	BeaconDB            *db.BeaconDB
	EnablePOWChain      bool
}

// DefaultConfig options for the simulator.
func DefaultConfig() *Config {
	return &Config{
		BlockRequestBuf:     100,
		BlockSlotBuf:        100,
		StateReqBuf:         100,
		ChainHeadRequestBuf: 100,
	}
}

// NewSimulator creates a simulator instance for a syncer to consume fake, generated blocks.
func NewSimulator(ctx context.Context, cfg *Config) *Simulator {
	ctx, cancel := context.WithCancel(ctx)
	return &Simulator{
		ctx:                  ctx,
		cancel:               cancel,
		p2p:                  cfg.P2P,
		web3Service:          cfg.Web3Service,
		beaconDB:             cfg.BeaconDB,
		enablePOWChain:       cfg.EnablePOWChain,
		blockRequestChan:     make(chan p2p.Message, cfg.BlockRequestBuf),
		blockBySlotChan:      make(chan p2p.Message, cfg.BlockSlotBuf),
		stateReqChan:         make(chan p2p.Message, cfg.StateReqBuf),
		chainHeadRequestChan: make(chan p2p.Message, cfg.ChainHeadRequestBuf),
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

	currentSlot, err := sim.beaconDB.GetSimulatorSlot()
	if err != nil {
		log.Fatal(err)
		return
	}

	slotTicker := slotticker.GetSimulatorTicker(genesisTime, params.BeaconConfig().SlotDuration, currentSlot)
	go func() {
		sim.run(slotTicker.C())
		close(sim.blockRequestChan)
		close(sim.blockBySlotChan)
		slotTicker.Done()
	}()
}

// Stop the sim.
func (sim *Simulator) Stop() error {
	defer sim.cancel()
	log.Info("Stopping service")
	return nil
}

func (sim *Simulator) run(slotInterval <-chan uint64) {
	chainHdReqSub := sim.p2p.Subscribe(&pb.ChainHeadRequest{}, sim.chainHeadRequestChan)
	blockReqSub := sim.p2p.Subscribe(&pb.BeaconBlockRequest{}, sim.blockRequestChan)
	blockBySlotSub := sim.p2p.Subscribe(&pb.BeaconBlockRequestBySlotNumber{}, sim.blockBySlotChan)
	stateReqSub := sim.p2p.Subscribe(&pb.BeaconStateRequest{}, sim.stateReqChan)
	defer blockReqSub.Unsubscribe()
	defer blockBySlotSub.Unsubscribe()
	defer stateReqSub.Unsubscribe()
	defer chainHdReqSub.Unsubscribe()

	lastBlock, err := sim.beaconDB.GetChainHead()
	if err != nil {
		log.Errorf("Could not fetch latest block: %v", err)
		return
	}

	lastHash, err := b.Hash(lastBlock)
	if err != nil {
		log.Errorf("Could not get hash of the latest block: %v", err)
	}
	broadcastedBlocksByHash := map[[32]byte]*pb.BeaconBlock{}
	broadcastedBlocksBySlot := map[uint64]*pb.BeaconBlock{}

	for {
		select {
		case <-sim.ctx.Done():
			log.Debug("Simulator context closed, exiting goroutine")
			return
		case msg := <-sim.chainHeadRequestChan:

			log.Debug("Received chain head request")
			if err := sim.SendChainHead(msg.Peer); err != nil {
				log.Errorf("Unable to send chain head response %v", err)
			}

		case slot := <-slotInterval:
			block, err := sim.generateBlock(slot, lastHash)
			if err != nil {
				log.Error(err)
				continue
			}

			hash, err := b.Hash(block)
			if err != nil {
				log.Errorf("Could not hash simulated block: %v", err)
				continue
			}
			sim.p2p.Broadcast(&pb.BeaconBlockAnnounce{
				Hash:       hash[:],
				SlotNumber: slot,
			})

			log.WithFields(logrus.Fields{
				"hash": fmt.Sprintf("%#x", hash),
				"slot": slot,
			}).Debug("Broadcast block hash and slot")

			sim.SaveSimulatorSlot(slot)
			broadcastedBlocksByHash[hash] = block
			broadcastedBlocksBySlot[slot] = block
			lastHash = hash
		case msg := <-sim.blockBySlotChan:
			data := msg.Data.(*pb.BeaconBlockRequestBySlotNumber)

			block := broadcastedBlocksBySlot[data.GetSlotNumber()]
			if block == nil {
				log.WithFields(logrus.Fields{
					"slot": fmt.Sprintf("%d", data.GetSlotNumber()),
				}).Debug("Requested block not found:")
				continue
			}

			log.WithFields(logrus.Fields{
				"slot": fmt.Sprintf("%d", data.GetSlotNumber()),
			}).Debug("Responding to full block request")

			// Sends the full block body to the requester.
			res := &pb.BeaconBlockResponse{Block: block, Attestation: &pb.Attestation{
				Data: &pb.AttestationData{
					Slot:             block.GetSlot(),
					AttesterBitfield: []byte{byte(255)},
				},
			}}
			sim.p2p.Send(res, msg.Peer)

		case msg := <-sim.blockRequestChan:
			data := msg.Data.(*pb.BeaconBlockRequest)
			var hash [32]byte
			copy(hash[:], data.Hash)

			block := broadcastedBlocksByHash[hash]
			if block == nil {
				log.WithFields(logrus.Fields{
					"hash": fmt.Sprintf("%#x", hash),
				}).Debug("Requested block not found:")
				continue
			}

			log.WithFields(logrus.Fields{
				"hash": fmt.Sprintf("%#x", hash),
			}).Debug("Responding to full block request")

			// Sends the full block body to the requester.
			res := &pb.BeaconBlockResponse{Block: block, Attestation: &pb.Attestation{
				Data: &pb.AttestationData{
					Slot:             block.GetSlot(),
					AttesterBitfield: []byte{byte(255)},
				},
			}}
			sim.p2p.Send(res, msg.Peer)
		case msg := <-sim.stateReqChan:
			data := msg.Data.(*pb.BeaconStateRequest)

			beaconState, err := sim.beaconDB.GetState()
			if err != nil {
				log.Errorf("Could not retrieve beacon state: %v", err)
				continue
			}

			hash, err := beaconState.Hash()
			if err != nil {
				log.Errorf("Could not hash beacon state: %v", err)
				continue
			}

			if !bytes.Equal(data.GetHash(), hash[:]) {
				log.WithFields(logrus.Fields{
					"hash": fmt.Sprintf("%#x", data.GetHash()),
				}).Debug("Requested beacon state is of a different hash")
				continue
			}

			log.WithFields(logrus.Fields{
				"hash": fmt.Sprintf("%#x", hash),
			}).Debug("Responding to full beacon state request")

			// Sends the full crystallized state to the requester.
			res := &pb.BeaconStateResponse{
				BeaconState: beaconState.Proto(),
			}
			sim.p2p.Send(res, msg.Peer)
		}
	}
}

// generateBlock generates fake blocks for the simulator.
func (sim *Simulator) generateBlock(slot uint64, lastHash [32]byte) (*pb.BeaconBlock, error) {
	beaconState, err := sim.beaconDB.GetState()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve beacon state: %v", err)
	}

	stateHash, err := beaconState.Hash()
	if err != nil {
		return nil, fmt.Errorf("could not hash beacon state: %v", err)
	}

	var powChainRef []byte
	if sim.enablePOWChain {
		powChainRef = sim.web3Service.LatestBlockHash().Bytes()
	} else {
		powChainRef = []byte{byte(slot)}
	}

	parentSlot := slot - 1
	committees, err := v.GetShardAndCommitteesForSlot(
		beaconState.ShardAndCommitteesForSlots(),
		beaconState.LastStateRecalculationSlot(),
		parentSlot,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get shard committee: %v", err)
	}

	parentHash := make([]byte, 32)
	copy(parentHash, lastHash[:])

	shardCommittees := committees.ArrayShardAndCommittee
	attestations := make([]*pb.Attestation, len(shardCommittees))

	// Create attestations for all committees of the previous block.
	// Ensure that all attesters have voted by calling FillBitfield.
	for i, shardCommittee := range shardCommittees {
		shardID := shardCommittee.Shard
		numAttesters := len(shardCommittee.Committee)
		attestations[i] = &pb.Attestation{
			Slot:               parentSlot,
			AttesterBitfield:   bitutil.FillBitfield(numAttesters),
			JustifiedBlockHash: parentHash,
			Shard:              shardID,
		}
	}

	block := &pb.BeaconBlock{
		Slot:                          slot,
		Timestamp:                     ptypes.TimestampNow(),
		CandidatePowReceiptRootHash32: powChainRef,
		StateRootHash32:               stateHash[:],
		ParentRootHash32:              parentHash,
		RandaoRevealHash32:            params.BeaconConfig().SimulatedBlockRandao[:],
		Attestations:                  attestations,
	}
	return block, nil
}

// SendChainHead sends the latest head of the local chain
// to the peer who requested it.
func (sim *Simulator) SendChainHead(peer p2p.Peer) error {
	block, err := sim.beaconDB.GetChainHead()
	if err != nil {
		return err
	}

	hash, err := b.Hash(block)
	if err != nil {
		return err
	}

	res := &pb.ChainHeadResponse{
		Hash:  hash[:],
		Slot:  block.GetSlot(),
		Block: block,
	}

	sim.p2p.Send(res, peer)

	log.WithFields(logrus.Fields{
		"hash": fmt.Sprintf("%#x", hash),
	}).Debug("Responding to chain head request")

	return nil
}

// SaveSimulatorSlot persists the current slot of the simulator.
func (sim *Simulator) SaveSimulatorSlot(slot uint64) {
	err := sim.beaconDB.SaveSimulatorSlot(slot)
	if err != nil {
		log.Error(err)
	}
}
