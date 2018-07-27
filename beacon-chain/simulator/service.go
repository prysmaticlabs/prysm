package simulator

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "simulator")

// Simulator struct.
type Simulator struct {
	ctx    context.Context
	cancel context.CancelFunc
	delay  time.Duration
}

// NewSimulator hi.
func NewSimulator(ctx context.Context, delay time.Duration) *Simulator {
	ctx, cancel := context.WithCancel(ctx)
	return &Simulator{
		ctx:    ctx,
		cancel: cancel,
		delay:  delay,
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
	return
}
