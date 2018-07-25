package blockchain

import (
	"context"
	"hash"

	"github.com/prysmaticlabs/prysm/beacon-chain/database"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "blockchain")

// ChainService represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type ChainService struct {
	ctx               context.Context
	cancel            context.CancelFunc
	beaconDB          *database.BeaconDB
	chain             *BeaconChain
	web3Service       *powchain.Web3Service
	latestBeaconBlock chan *types.Block
}

// NewChainService instantiates a new service instance that will
// be registered into a running beacon node.
func NewChainService(ctx context.Context, beaconDB *database.BeaconDB, web3Service *powchain.Web3Service) (*ChainService, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &ChainService{ctx, cancel, beaconDB, nil, web3Service, nil}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	log.Infof("Starting service")

	beaconChain, err := NewBeaconChain(c.beaconDB.DB())
	if err != nil {
		log.Errorf("Unable to setup blockchain: %v", err)
	}
	c.chain = beaconChain
	go c.updateActiveState()
}

// Stop the blockchain service's main event loop and associated goroutines.
func (c *ChainService) Stop() error {
	defer c.cancel()
	log.Info("Stopping service")
	return nil
}

// ProcessBlock accepts a new block for inclusion in the chain.
func (c *ChainService) ProcessBlock(b *types.Block) error {
	c.latestBeaconBlock <- b
	return nil
}

// ContainsBlock checks if a block for the hash exists in the chain.
// This method must be safe to call from a goroutine
func (c *ChainService) ContainsBlock(h hash.Hash) bool {
	// TODO
	return false
}

// updateActiveState receives a beacon block, computes a new active state and writes it to db.
func (c *ChainService) updateActiveState() {
	for {
		select {
		case block := <-c.latestBeaconBlock:
			log.WithFields(logrus.Fields{"activeStateHash": block.Data().ActiveStateHash}).Debug("Received beacon block")

			// TODO: Using latest block hash for seed, this will eventually be replaced by randao
			activeState, err := c.chain.computeNewActiveState(c.web3Service.LatestBlockHash())
			if err != nil {
				log.Errorf("Compute active state failed: %v", err)
			}

			err = c.chain.MutateActiveState(activeState)
			if err != nil {
				log.Errorf("Write active state to disk failed: %v", err)
			}

		case <-c.ctx.Done():
			log.Debug("Chain service context closed, exiting goroutine")
			return
		}
	}
}
