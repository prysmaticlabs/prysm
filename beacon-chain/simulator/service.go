package simulator

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "simulator")

// Simulator serves as a runtime utlity in development for broadcasting beacon blocks
// to the network.
type Simulator struct {
	ctx    context.Context
	cancel context.CancelFunc
	delay  time.Duration
}

// NewSimulator kicks off a novel instance.
func NewSimulator(ctx context.Context, delay time.Duration) *Simulator {
	ctx, cancel := context.WithCancel(ctx)
	return &Simulator{
		ctx:    ctx,
		cancel: cancel,
		delay:  delay,
	}
}

// Start kicks off the simulator's main routine.
func (sim *Simulator) Start() {
	log.Info("Starting service")
	go sim.run(time.NewTicker(sim.delay).C, sim.ctx.Done())
}

// Stop closes the contexts gracefully.
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
			log.Info("Received a tick")
		}
	}
}
