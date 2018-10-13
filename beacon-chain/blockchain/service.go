// Package blockchain defines the life-cycle and status of the beacon chain.
package blockchain

import (
	"context"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "blockchain")

// ChainService represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type ChainService struct {
	ctx                            context.Context
	cancel                         context.CancelFunc
	beaconDB                       *db.BeaconDB
	web3Service                    *powchain.Web3Service
	incomingBlockFeed              *event.Feed
	incomingBlockChan              chan *types.Block
	processedBlockChan             chan *types.Block
	canonicalBlockFeed             *event.Feed
	canonicalCrystallizedStateFeed *event.Feed
	genesisTime                    time.Time
	enableCrossLinks               bool
	enableRewardChecking           bool
	enableAttestationValidity      bool
	enablePOWChain                 bool
}

// Config options for the service.
type Config struct {
	BeaconBlockBuf            int
	IncomingBlockBuf          int
	Web3Service               *powchain.Web3Service
	BeaconDB                  *db.BeaconDB
	DevMode                   bool
	EnableCrossLinks          bool
	EnableRewardChecking      bool
	EnableAttestationValidity bool
	EnablePOWChain            bool
}

// NewChainService instantiates a new service instance that will
// be registered into a running beacon node.
func NewChainService(ctx context.Context, cfg *Config) (*ChainService, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &ChainService{
		ctx:                            ctx,
		cancel:                         cancel,
		beaconDB:                       cfg.BeaconDB,
		web3Service:                    cfg.Web3Service,
		incomingBlockChan:              make(chan *types.Block, cfg.IncomingBlockBuf),
		processedBlockChan:             make(chan *types.Block),
		incomingBlockFeed:              new(event.Feed),
		canonicalBlockFeed:             new(event.Feed),
		canonicalCrystallizedStateFeed: new(event.Feed),
		enablePOWChain:                 cfg.EnablePOWChain,
		enableCrossLinks:               cfg.EnableCrossLinks,
		enableRewardChecking:           cfg.EnableRewardChecking,
		enableAttestationValidity:      cfg.EnableAttestationValidity,
	}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	log.Info("Starting service")

	var err error
	c.genesisTime, err = c.beaconDB.GetGenesisTime()
	if err != nil {
		log.Fatal(err)
		return
	}

	go c.updateHead(c.processedBlockChan)
	go c.blockProcessing(c.processedBlockChan)
}

// Stop the blockchain service's main event loop and associated goroutines.
func (c *ChainService) Stop() error {
	defer c.cancel()

	log.Info("Stopping service")
	return nil
}

// IncomingBlockFeed returns a feed that any service can send incoming p2p blocks into.
// The chain service will subscribe to this feed in order to process incoming blocks.
func (c *ChainService) IncomingBlockFeed() *event.Feed {
	return c.incomingBlockFeed
}

// CanonicalBlockFeed returns a channel that is written to
// whenever a new block is determined to be canonical in the chain.
func (c *ChainService) CanonicalBlockFeed() *event.Feed {
	return c.canonicalBlockFeed
}

// CanonicalCrystallizedStateFeed returns a feed that is written to
// whenever a new crystallized state is determined to be canonical in the chain.
func (c *ChainService) CanonicalCrystallizedStateFeed() *event.Feed {
	return c.canonicalCrystallizedStateFeed
}

// doesPoWBlockExist checks if the referenced PoW block exists.
func (c *ChainService) doesPoWBlockExist(block *types.Block) bool {
	powBlock, err := c.web3Service.Client().BlockByHash(context.Background(), block.PowChainRef())
	if err != nil {
		log.Debugf("fetching PoW block corresponding to mainchain reference failed: %v", err)
		return false
	}

	return powBlock != nil
}

// updateHead applies the fork choice rule to the beacon chain
// at the start of each new slot interval. The function looks
// at an in-memory slice of block hashes pending processing and
// selects the best block according to the in-protocol fork choice
// rule as canonical. This block is then persisted to storage.
func (c *ChainService) updateHead(processedBlock <-chan *types.Block) {
	for {
		select {
		case <-c.ctx.Done():
			return
		case block := <-processedBlock:
			log.WithField("slot", block.SlotNumber()).Info("New beacon slot")

			h, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash incoming block: %v", err)
				continue
			}

			log.Info("Applying fork choice rule")

			parentBlock, err := c.beaconDB.GetBlock(block.ParentHash())
			if err != nil {
				log.Errorf("Failed to get parent of block %#x", h)
				continue
			}

			cState := c.beaconDB.GetCrystallizedState()
			aState := c.beaconDB.GetActiveState()
			var stateTransitioned bool

			for cState.IsCycleTransition(block.SlotNumber()) {
				log.Infof("Recalculating active state")
				cState, aState, err = cState.NewStateRecalculations(
					aState,
					block,
					c.enableCrossLinks,
					c.enableRewardChecking,
				)
				if err != nil {
					log.Errorf("Initialize new cycle transition failed: %v", err)
					continue
				}
				stateTransitioned = true
			}

			aState, err = aState.CalculateNewActiveState(
				block,
				cState,
				parentBlock.SlotNumber(),
				c.enableAttestationValidity,
			)
			if err != nil {
				log.Errorf("Compute active state failed: %v", err)
				continue
			}

			if err := c.beaconDB.SaveActiveState(aState); err != nil {
				log.Errorf("Write active state to disk failed: %v", err)
				continue
			}

			if stateTransitioned {
				if err := c.beaconDB.SaveCrystallizedState(cState); err != nil {
					log.Errorf("Write crystallized state to disk failed: %v", err)
					continue
				}
			}

			// Save canonical block hash with slot number to DB.
			if err := c.beaconDB.SaveCanonicalSlotNumber(block.SlotNumber(), h); err != nil {
				log.Errorf("Unable to save slot number to db: %v", err)
				continue
			}

			// Save canonical block to DB.
			if err := c.beaconDB.SaveCanonicalBlock(block); err != nil {
				log.Errorf("Unable to save block to db: %v", err)
				continue
			}

			log.WithField("blockHash", fmt.Sprintf("%#x", h)).Info("Canonical block determined")

			// We fire events that notify listeners of a new block (or crystallized state in
			// the case of a state transition). This is useful for the beacon node's gRPC
			// server to stream these events to beacon clients.
			if stateTransitioned {
				c.canonicalCrystallizedStateFeed.Send(cState)
			}
			c.canonicalBlockFeed.Send(block)
		}
	}
}

func (c *ChainService) blockProcessing(processedBlock chan<- *types.Block) {
	subBlock := c.incomingBlockFeed.Subscribe(c.incomingBlockChan)
	defer subBlock.Unsubscribe()
	for {
		select {
		case <-c.ctx.Done():
			log.Debug("Chain service context closed, exiting goroutine")
			return

		// Listen for a newly received incoming block from the sync service.
		case block := <-c.incomingBlockChan:
			blockHash, err := block.Hash()
			if err != nil {
				log.Errorf("Failed to get hash of block: %v", err)
				continue
			}

			if c.enablePOWChain && !c.doesPoWBlockExist(block) {
				log.Debugf("Proof-of-Work chain reference in block does not exist")
				continue
			}

			// Check if we have received the parent block.
			parentExists, err := c.beaconDB.HasBlock(block.ParentHash())
			if err != nil {
				log.Errorf("Could not check existence of parent: %v", err)
				continue
			}
			if !parentExists {
				log.Debugf("Block points to nil parent: %#x", block.ParentHash())
				continue
			}
			parent, err := c.beaconDB.GetBlock(block.ParentHash())
			if err != nil {
				log.Debugf("Could not get parent block: %v", err)
				continue
			}

			aState := c.beaconDB.GetActiveState()
			cState := c.beaconDB.GetCrystallizedState()

			if valid := block.IsValid(
				c.beaconDB,
				aState,
				cState,
				parent.SlotNumber(),
				c.enableAttestationValidity,
				c.genesisTime,
			); !valid {
				log.Debugf("Block failed validity conditions: %v", err)
				continue
			}

			if err := c.beaconDB.SaveBlock(block); err != nil {
				log.Errorf("Failed to save block: %v", err)
				continue
			}

			log.Infof("Finished processing received block: %#x", blockHash)

			// Push the block to trigger the fork choice rule
			processedBlock <- block
		}
	}
}
