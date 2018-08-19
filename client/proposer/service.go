// Package proposer defines all relevant functionality for a Proposer actor
// within Ethereum 2.0.
package proposer

import (
	"context"

	"github.com/prysmaticlabs/prysm/client/types"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "proposer")

// Proposer holds functionality required to run a block proposer
// in Ethereum 2.0. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	ctx           context.Context
	cancel        context.CancelFunc
	beaconService types.BeaconClient
}

// NewProposer creates a new attester instance.
func NewProposer(ctx context.Context, beaconService types.BeaconClient) *Proposer {
	ctx, cancel := context.WithCancel(ctx)
	return &Proposer{
		ctx:           ctx,
		cancel:        cancel,
		beaconService: beaconService,
	}
}

// Start the main routine for a proposer.
func (p *Proposer) Start() {
	log.Info("Starting service")
	go p.run(p.ctx.Done())
}

// Stop the main loop.
func (p *Proposer) Stop() error {
	defer p.cancel()
	log.Info("Stopping service")
	return nil
}

// run the main event loop that listens for a proposer assignment.
func (p *Proposer) run(done <-chan struct{}) {
	for {
		select {
		case <-done:
			log.Debug("Proposer context closed, exiting goroutine")
			return
		case <-p.beaconService.ProposerAssignment():
			log.Info("Performing proposer responsibility")
			continue
		}
	}
}
