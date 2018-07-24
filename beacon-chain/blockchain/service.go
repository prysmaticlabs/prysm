package blockchain

import (
	"context"

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
	log.Infof("Starting blockchain service")

	beaconChain, err := NewBeaconChain(c.beaconDB.DB())
	if err != nil {
		log.Errorf("Unable to setup blockchain: %v", err)
	}
	c.chain = beaconChain
	go c.updateActiveState()
	go c.processIncomingBlock()
}

// Stop the blockchain service's main event loop and associated goroutines.
func (c *ChainService) Stop() error {
	defer c.cancel()
	log.Info("Stopping blockchain service")
	return nil
}

func (c *ChainService) processIncomingBlock() {
	// TODO: select on receiving a block via p2p.
	// Run CanProcessBlock
	// Calculate both states for the processed block, insert into a trie structure
	// Persist this trie structure
	// Repeat. A separate goroutine will be updating the head upon this trie structure being modified.
	for {
		select {
		case block := <-c.latestBeaconBlock:
			canProcess, err := c.chain.CanProcessBlock(c.web3Service.Client(), block)
			if err != nil {
				log.Errorf("Could not check if incoming block could be processed: %v", err)
			}
			if !canProcess {
				log.WithFields(logrus.Fields{
					"incomingBlockHash": block.Hash(),
				}).Debug("Could not process incoming block")
				continue
			}
		case <-c.ctx.Done():
			log.Debug("Chain service context closed, exiting goroutine")
			return
		}
	}
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
