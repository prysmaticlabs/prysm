package simulator

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"golang.org/x/crypto/blake2b"

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
	web3Service            *powchain.Web3Service
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

// NewSimulator hi.
func NewSimulator(ctx context.Context, cfg *Config, beaconp2p types.P2P, web3Service *powchain.Web3Service) *Simulator {
	ctx, cancel := context.WithCancel(ctx)
	return &Simulator{
		ctx:         ctx,
		cancel:      cancel,
		p2p:         beaconp2p,
		web3Service: web3Service,
		delay:       cfg.Delay,
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
	blockReqSub := sim.p2p.Feed(pb.BeaconBlockRequest{}).Subscribe(sim.blockRequestChan)
	defer blockReqSub.Unsubscribe()
	for {
		select {
		case <-done:
			log.Debug("Simulator context closed, exiting goroutine")
			return
		case <-delayChan:
			stateHash := blake2b.Sum256([]byte{1, 2, 3, 4, 5})
			block, err := types.NewBlockWithData(&pb.BeaconBlockResponse{
				Timestamp:             ptypes.TimestampNow(),
				MainChainRef:          sim.web3Service.LatestBlockHash().Bytes(),
				ActiveStateHash:       stateHash[:],
				CrystallizedStateHash: stateHash[:],
			})
			if err != nil {
				log.Errorf("Could not create simulated block: %v", err)
			}

			h, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash simulated block: %v", err)
			}

			log.WithField("announcedBlockHash", fmt.Sprintf("%x", h)).Info("Announcing block hash")
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
			log.WithField("blockRequestHash", fmt.Sprintf("%x", h)).Info("Responding to full block request for hash")
			// Sends the full block body to the requester.
			sim.p2p.Send(block.Proto(), msg.Peer)
		}
	}
}
