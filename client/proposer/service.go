// Package proposer defines all relevant functionality for a Proposer actor
// within the minimal sharding protocol.
package proposer

import (
	"context"

	"github.com/prysmaticlabs/prysm/client/types"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "proposer")

// Proposer holds functionality required to run a proposer actor defined
// in the beacon chain spec v2.1.
type Proposer struct {
	ctx           context.Context
	cancel        context.CancelFunc
	clientService types.RPCClient
}

// NewProposer creates a struct instance of a proposer service.
func NewProposer(ctx context.Context, clientService types.RPCClient) *Proposer {
	ctx, cancel := context.WithCancel(ctx)
	return &Proposer{
		ctx:           ctx,
		cancel:        cancel,
		clientService: clientService,
	}
}

// Start the main loop for proposing.
func (p *Proposer) Start() {
	log.Info("Starting service")
}

// Stop the main loop for proposing.
func (p *Proposer) Stop() error {
	defer p.cancel()
	log.Info("Stopping service")
	return nil
}
