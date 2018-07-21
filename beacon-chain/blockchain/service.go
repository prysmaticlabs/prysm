package blockchain

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/database"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "blockchain")

// ChainService represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type ChainService struct {
	ctx      context.Context
	cancel   context.CancelFunc
	beaconDB *database.BeaconDB
	chain    *BeaconChain
}

// NewChainService instantiates a new service instance that will
// be registered into a running beacon node.
func NewChainService(ctx context.Context, beaconDB *database.BeaconDB) (*ChainService, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &ChainService{ctx, cancel, beaconDB, nil}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	log.Infof("Starting blockchain service")
	if _, err := NewBeaconChain(c.beaconDB.DB()); err != nil {
		log.Errorf("Unable to setup blockchain: %v", err)
	}
}

// Stop the blockchain service's main event loop and associated goroutines.
func (c *ChainService) Stop() error {
	defer c.cancel()
	log.Info("Stopping blockchain service")
	return nil
}

// processBlocks runs an event loop handling incoming blocks via p2p.
func (c *ChainService) processBlocks() {
	return
}
