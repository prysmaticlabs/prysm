package blockchain

import (
	"context"
	"github.com/prysmaticlabs/prysm/beacon-chain/database"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	log "github.com/sirupsen/logrus"
)

// ChainService represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type ChainService struct {
	ctx         context.Context
	cancel      context.CancelFunc
	beaconDB    *database.BeaconDB
	chain       *BeaconChain
	web3Service *powchain.Web3Service
	beaconBlock chan *types.Block
}

// NewChainService instantiates a new service instance that will
// be registered into a running beacon node.
func NewChainService(ctx context.Context, beaconDB *database.BeaconDB, web3Service *powchain.Web3Service) (*ChainService, error) {
	ctx, cancel := context.WithCancel(ctx)
	beaconChain, err := NewBeaconChain(beaconDB.DB())
	if err != nil {
		return nil, err
	}
	return &ChainService{ctx, cancel, beaconDB, beaconChain, web3Service, nil}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	log.Infof("Starting blockchain service")
	if _, err := NewBeaconChain(c.beaconDB.DB()); err != nil {
		log.Errorf("Unable to setup blockchain: %v", err)
	}
	go c.updateActiveState()
}

// Stop the blockchain service's main event loop and associated goroutines.
func (c *ChainService) Stop() error {
	defer c.cancel()
	log.Info("Stopping blockchain service")
	return nil
}

// updateActiveState receives a beacon block, computes a new active state and writes it to db
func (c *ChainService) updateActiveState() {
	for {
		select {
		case block := <-c.beaconBlock:
			log.Debugf("Received beacon block, last active state hash: %v", block.ActiveStateHash)

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
