package blockchain

import (
	"context"

	log "github.com/sirupsen/logrus"
)

// ChainService represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type ChainService struct {
	ctx    context.Context
	cancel context.CancelFunc
	chain  *BeaconChain
}

// NewChainService instantiates a new service instance that will
// be registered into a running beacon node.
func NewChainService(ctx context.Context) (*ChainService, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &ChainService{ctx, cancel, nil}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	log.Infof("Starting blockchain service")
	chain, err := NewBeaconChain()
	if err != nil {
		log.Errorf("Unable to setup blockchain: %v", err)
	}
}

// Stop the blockchain service's main event loop and associated goroutines.
func (c *ChainService) Stop() error {
	defer c.cancel()
	log.Info("Stopping blockchain service")
	return nil
}
