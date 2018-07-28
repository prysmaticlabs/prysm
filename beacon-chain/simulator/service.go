package simulator

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "simulator")

// Simulator struct.
type Simulator struct {
	ctx               context.Context
	cancel            context.CancelFunc
	p2p               types.P2P
	web3Service       *powchain.Web3Service
	delay             time.Duration
	broadcastedBlocks map[[32]byte]*types.Block
}

// Config options for the simulator service.
type Config struct {
	Delay time.Duration
}

// DefaultConfig options for the simulator.
func DefaultConfig() *Config {
	return &Config{Delay: time.Second * 10}
}

// NewSimulator hi.
func NewSimulator(ctx context.Context, cfg *Config, beaconp2p types.P2P, web3Service *powchain.Web3Service) *Simulator {
	ctx, cancel := context.WithCancel(ctx)
	return &Simulator{
		ctx:               ctx,
		cancel:            cancel,
		p2p:               beaconp2p,
		web3Service:       web3Service,
		delay:             cfg.Delay,
		broadcastedBlocks: make(map[[32]byte]*types.Block),
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
	for {
		select {
		case <-done:
			log.Debug("Simulator context closed, exiting goroutine")
			return
		case <-delayChan:
			block := types.NewBlock(0)
			h, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash simulated block: %v", err)
			}
			sim.p2p.Broadcast(block.Proto())
			// We then store the block in a map for later retrieval upon a request for its full
			// data being sent back.
			sim.broadcastedBlocks[h] = block
		}
	}
}
