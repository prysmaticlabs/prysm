package simulator

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"

	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "simulator")

// Simulator struct.
type Simulator struct {
	ctx         context.Context
	cancel      context.CancelFunc
	p2p         types.P2P
	web3Service *powchain.Web3Service
	delay       time.Duration
}

// NewSimulator hi.
func NewSimulator(ctx context.Context, beaconp2p types.P2P, web3Service *powchain.Web3Service, delay time.Duration) *Simulator {
	ctx, cancel := context.WithCancel(ctx)
	return &Simulator{
		ctx:         ctx,
		cancel:      cancel,
		p2p:         beaconp2p,
		web3Service: web3Service,
		delay:       delay,
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
			announce := &pb.BeaconBlockHashAnnounce{
				Hash: []byte("foobar"),
			}
			sim.p2p.Broadcast(announce)
		}
	}
}
