// Package simulator defines the simulation utility to test the beacon-chain.
package simulator

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/shared/p2p"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "simulator")

// Simulator struct.
type Simulator struct {
	ctx                           context.Context
	cancel                        context.CancelFunc
	p2p                           types.P2P
	web3Service                   types.POWChainService
	chainService                  types.StateFetcher
	beaconDB                      ethdb.Database
	delay                         time.Duration
	slotNum                       uint64
	broadcastedBlocks             map[[32]byte]*types.Block
	broadcastedBlockHashes        [][32]byte
	blockRequestChan              chan p2p.Message
	broadcastedCrystallizedStates map[[32]byte]*types.CrystallizedState
	crystallizedStateRequestChan  chan p2p.Message
}

// Config options for the simulator service.
type Config struct {
	Delay                       time.Duration
	BlockRequestBuf             int
	CrystallizedStateRequestBuf int
}

// DefaultConfig options for the simulator.
func DefaultConfig() *Config {
	return &Config{
		Delay:                       time.Second * 5,
		BlockRequestBuf:             100,
		CrystallizedStateRequestBuf: 100,
	}
}

// NewSimulator creates a simulator instance for a syncer to consume fake, generated blocks.
func NewSimulator(ctx context.Context, cfg *Config, beaconDB ethdb.Database, beaconp2p types.P2P, web3Service types.POWChainService, chainService types.StateFetcher) *Simulator {
	ctx, cancel := context.WithCancel(ctx)
	return &Simulator{
		ctx:                           ctx,
		cancel:                        cancel,
		p2p:                           beaconp2p,
		web3Service:                   web3Service,
		chainService:                  chainService,
		beaconDB:                      beaconDB,
		delay:                         cfg.Delay,
		slotNum:                       0,
		broadcastedBlocks:             make(map[[32]byte]*types.Block),
		broadcastedBlockHashes:        [][32]byte{},
		blockRequestChan:              make(chan p2p.Message, cfg.BlockRequestBuf),
		broadcastedCrystallizedStates: make(map[[32]byte]*types.CrystallizedState),
		crystallizedStateRequestChan:  make(chan p2p.Message, cfg.CrystallizedStateRequestBuf),
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
	blockReqSub := sim.p2p.Subscribe(pb.BeaconBlockRequest{}, sim.blockRequestChan)
	crystallizedStateReqSub := sim.p2p.Subscribe(pb.CrystallizedStateRequest{}, sim.crystallizedStateRequestChan)
	defer blockReqSub.Unsubscribe()
	defer crystallizedStateReqSub.Unsubscribe()

	crystallizedState := sim.chainService.CurrentCrystallizedState()

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

			crystallizedStateHash, err := crystallizedState.Hash()
			if err != nil {
				log.Errorf("Could not fetch crystallized state hash: %v", err)
				continue
			}

			// Is it epoch transition time?
			if sim.slotNum >= crystallizedState.LastStateRecalc()+params.CycleLength {
				// We populate the validators in the crystallized state with some fake
				// set of validators for simulation purposes.
				var validators []*pb.ValidatorRecord
				for i := 0; i < params.BootstrappedValidatorsCount; i++ {
					validator := &pb.ValidatorRecord{StartDynasty: 0, EndDynasty: params.DefaultEndDynasty, Balance: params.DefaultBalance, WithdrawalAddress: []byte{}, PublicKey: 0}
					validators = append(validators, validator)
				}

				crystallizedState.SetValidators(validators)
				crystallizedState.SetStateRecalc(sim.slotNum)
				crystallizedState.SetLastJustifiedSlot(sim.slotNum)
				crystallizedState.UpdateJustifiedSlot(sim.slotNum)
				log.WithField("lastJustifiedEpoch", crystallizedState.LastJustifiedSlot()).Info("Last justified epoch")
				log.WithField("lastFinalizedEpoch", crystallizedState.LastFinalizedSlot()).Info("Last finalized epoch")

				cHash, err := crystallizedState.Hash()
				if err != nil {
					log.Errorf("Could not hash simulated crystallized state: %v", err)
					continue
				}
				crystallizedStateHash = cHash
				log.WithField("announcedStateHash", fmt.Sprintf("0x%x", crystallizedStateHash)).Info("Announcing crystallized state hash")
				sim.p2p.Broadcast(&pb.CrystallizedStateHashAnnounce{
					Hash: crystallizedStateHash[:],
				})
				sim.broadcastedCrystallizedStates[crystallizedStateHash] = crystallizedState
			}

			// If we have not broadcast a simulated block yet, we set parent hash
			// to the genesis block.
			if sim.slotNum == 0 {
				parentHash = []byte("genesis")
			} else {
				parentHash = sim.broadcastedBlockHashes[len(sim.broadcastedBlockHashes)-1][:]
			}

			log.WithField("currentSlot", sim.slotNum).Info("Current slot")

			block := types.NewBlock(&pb.BeaconBlock{
				SlotNumber:            sim.slotNum,
				Timestamp:             ptypes.TimestampNow(),
				PowChainRef:           sim.web3Service.LatestBlockHash().Bytes(),
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

			log.WithField("announcedBlockHash", fmt.Sprintf("0x%x", h)).Info("Announcing block hash")
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
			log.Infof("Responding to full block request for hash: 0x%x", h)
			// Sends the full block body to the requester.
			res := &pb.BeaconBlockResponse{Block: block.Proto()}
			sim.p2p.Send(res, msg.Peer)

		case msg := <-sim.crystallizedStateRequestChan:
			data, ok := msg.Data.(*pb.CrystallizedStateRequest)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Error("Received malformed crystallized state request p2p message")
				continue
			}
			var h [32]byte
			copy(h[:], data.Hash[:32])

			state := sim.broadcastedCrystallizedStates[h]
			h, err := state.Hash()
			if err != nil {
				log.Errorf("Could not hash state: %v", err)
				continue
			}
			log.Infof("Responding to crystallized state request for hash: 0x%x", h)
			res := &pb.CrystallizedStateResponse{CrystallizedState: state.Proto()}
			sim.p2p.Send(res, msg.Peer)
		}
	}
}
