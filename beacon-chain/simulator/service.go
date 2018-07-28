package simulator

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/shared/p2p"

	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
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
	delay                  time.Duration
	broadcastedBlockHashes map[[32]byte]*types.Block
	blockRequestChan       chan p2p.Message
}

// Config options for the simulator service.
type Config struct {
	Delay           time.Duration
	BlockRequestBuf int
}

// DefaultConfig options for the simulator.
func DefaultConfig() *Config {
	return &Config{Delay: time.Second * 7, BlockRequestBuf: 100}
}

// NewSimulator creates a simulator instance for a syncer to consume fake, generated blocks.
func NewSimulator(ctx context.Context, cfg *Config, beaconp2p types.P2P, web3Service types.POWChainService, chainService types.StateFetcher) *Simulator {
	ctx, cancel := context.WithCancel(ctx)
	return &Simulator{
		ctx:          ctx,
		cancel:       cancel,
		p2p:          beaconp2p,
		web3Service:  web3Service,
		chainService: chainService,
		delay:        cfg.Delay,
		broadcastedBlockHashes: make(map[[32]byte]*types.Block),
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
	return nil
}

func (sim *Simulator) run(delayChan <-chan time.Time, done <-chan struct{}) {
	blockReqSub := sim.p2p.Subscribe(pb.BeaconBlockRequest{}, sim.blockRequestChan)
	defer blockReqSub.Unsubscribe()
	for {
		select {
		case <-done:
			log.Debug("Simulator context closed, exiting goroutine")
			return
		case <-delayChan:
			activeStateHash, err := sim.chainService.CurrentActiveState().Hash()
			if err != nil {
				log.Errorf("Could not fetch active state hash: %v", err)
			}

			var validators []*pb.ValidatorRecord
			for i := 0; i < 100; i++ {
				validator := &pb.ValidatorRecord{Balance: 1000, WithdrawalAddress: []byte{'A'}, PublicKey: 0}
				validators = append(validators, validator)
			}

			crystallizedState := sim.chainService.CurrentCrystallizedState()
			crystallizedState.UpdateActiveValidators(validators)

			crystallizedStateHash, err := crystallizedState.Hash()
			if err != nil {
				log.Errorf("Could not fetch crystallized state hash: %v", err)
			}

			block, err := types.NewBlock(&pb.BeaconBlockResponse{
				Timestamp:             ptypes.TimestampNow(),
				MainChainRef:          sim.web3Service.LatestBlockHash().Bytes(),
				ActiveStateHash:       activeStateHash[:],
				CrystallizedStateHash: crystallizedStateHash[:],
				ParentHash:            make([]byte, 32),
			})
			if err != nil {
				log.Errorf("Could not create simulated block: %v", err)
			}

			h, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash simulated block: %v", err)
			}

			log.WithField("announcedBlockHash", fmt.Sprintf("0x%x", h)).Info("Announcing block hash")
			sim.p2p.Broadcast(&pb.BeaconBlockHashAnnounce{
				Hash: h[:],
			})
			// We then store the block in a map for later retrieval upon a request for its full
			// data being sent back.
			sim.broadcastedBlockHashes[h] = block

		case msg := <-sim.blockRequestChan:
			data, ok := msg.Data.(*pb.BeaconBlockRequest)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Error("Received malformed beacon block request p2p message")
				continue
			}
			var h [32]byte
			copy(h[:], data.Hash[:32])

			block := sim.broadcastedBlockHashes[h]
			h, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash block: %v", err)
			}
			log.Infof("Responding to full block request for hash: 0x%x", h)
			// Sends the full block body to the requester.
			sim.p2p.Send(block.Proto(), msg.Peer)
		}
	}
}
