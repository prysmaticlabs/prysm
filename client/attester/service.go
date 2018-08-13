// Package attester defines all relevant functionality for a Attester actor
// within Ethereum 2.0.
package attester

import (
	"context"

	"github.com/prysmaticlabs/prysm/client/types"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "attester")

// Attester holds functionality required to run an attester
// as defined in Ethereum 2.0. Must satisfy the Service interface defined in
// sharding/service.go.
type Attester struct {
	ctx           context.Context
	cancel        context.CancelFunc
	beaconService types.BeaconClient
}

// NewAttester creates a new attester instance.
func NewAttester(ctx context.Context, beaconService types.BeaconClient) *Attester {
	ctx, cancel := context.WithCancel(ctx)
	return &Attester{
		ctx:           ctx,
		cancel:        cancel,
		beaconService: beaconService,
	}
}

// Start the main routine for a attester.
func (at *Attester) Start() {
	log.Info("Starting service")
	go at.run(at.ctx.Done())
}

// Stop the main loop.
func (at *Attester) Stop() error {
	defer at.cancel()
	log.Info("Stopping service")
	return nil
}

// run the main event loop that listens for an attestation assignment.
func (at *Attester) run(done <-chan struct{}) {
	for {
		select {
		case <-done:
			log.Debug("Attester context closed, exiting goroutine")
			return
		case <-at.beaconService.AttesterAssignment():
			log.Info("Performing attestation responsibility")
			continue
		}
	}
}
