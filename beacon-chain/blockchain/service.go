// Package blockchain defines the life-cycle and status of the beacon chain.
package blockchain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
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
	canonicalBlockFeed             *event.Feed
	canonicalCrystallizedStateFeed *event.Feed
	blocksPendingProcessing        [][32]byte
	lock                           sync.Mutex
	genesisTime                    time.Time
	slotTicker                     utils.SlotTicker
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
		incomingBlockFeed:              new(event.Feed),
		canonicalBlockFeed:             new(event.Feed),
		canonicalCrystallizedStateFeed: new(event.Feed),
		blocksPendingProcessing:        [][32]byte{},
		enablePOWChain:                 cfg.EnablePOWChain,
		enableCrossLinks:               cfg.EnableCrossLinks,
		enableRewardChecking:           cfg.EnableRewardChecking,
		enableAttestationValidity:      cfg.EnableAttestationValidity,
	}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	// TODO(#474): Fetch the slot: (block, state) DAGs from persistent storage
	// to truly continue across sessions.
	log.Info("Starting service")

	var err error
	c.genesisTime, err = c.beaconDB.GetGenesisTime()
	if err != nil {
		log.Fatal(err)
		return
	}

	c.slotTicker = utils.GetSlotTicker(c.genesisTime)
	go c.updateHead(c.slotTicker.C())
	go c.blockProcessing()
}

// Stop the blockchain service's main event loop and associated goroutines.
func (c *ChainService) Stop() error {
	defer c.cancel()
	c.slotTicker.Done()

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
func (c *ChainService) updateHead(slotInterval <-chan uint64) {
	for {
		select {
		case <-c.ctx.Done():
			return
		case slot := <-slotInterval:
			log.WithField("slotNumber", slot).Info("New beacon slot")

			// First, we check if there were any blocks processed in the previous slot.
			if len(c.blocksPendingProcessing) == 0 {
				continue
			}

			// We keep track of the highest scoring received block and its associated
			// states.
			var highestScoringBlock *types.Block
			var highestScoringCrystallizedState *types.CrystallizedState
			var highestScoringActiveState *types.ActiveState
			var highestScore uint64

			// We detect if this there is a cycle transition.
			var cycleTransitioned bool

			log.Info("Applying fork choice rule")

			currentCanonicalCrystallizedState := c.beaconDB.GetCrystallizedState()
			currentCanonicalActiveState := c.beaconDB.GetActiveState()

			// We loop over every block pending processing in order to determine
			// the highest scoring one.
			for i := 0; i < len(c.blocksPendingProcessing); i++ {
				block, err := c.beaconDB.GetBlock(c.blocksPendingProcessing[i])
				if err != nil {
					log.Errorf("Could not get block: %v", err)
					continue
				}

				h, err := block.Hash()
				if err != nil {
					log.Errorf("Could not hash incoming block: %v", err)
					continue
				}

				parentBlock, err := c.beaconDB.GetBlock(block.ParentHash())
				if err != nil {
					log.Errorf("Failed to get parent of block %x", h)
					continue
				}

				cState := currentCanonicalCrystallizedState
				aState := currentCanonicalActiveState

				for cState.IsCycleTransition(parentBlock.SlotNumber()) {
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
					cycleTransitioned = true
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

				// Initially, we set the highest scoring block to the first value in the
				// processed blocks list.
				if i == 0 {
					highestScoringBlock = block
					highestScoringCrystallizedState = cState
					highestScoringActiveState = aState
					continue
				}
				// Score the block and determine if its score is greater than the previously computed one.
				if block.Score(cState.LastFinalizedSlot(), cState.LastJustifiedSlot()) > highestScore {
					highestScoringBlock = block
					highestScoringCrystallizedState = cState
					highestScoringActiveState = aState
				}
			}

			// If no highest scoring block was determined, we do not update the head of the chain.
			if highestScoringBlock == nil {
				continue
			}

			if err := c.beaconDB.SaveActiveState(highestScoringActiveState); err != nil {
				log.Errorf("Write active state to disk failed: %v", err)
				continue
			}

			if cycleTransitioned {
				if err := c.beaconDB.SaveCrystallizedState(highestScoringCrystallizedState); err != nil {
					log.Errorf("Write crystallized state to disk failed: %v", err)
					continue
				}
			}

			h, err := highestScoringBlock.Hash()
			if err != nil {
				log.Errorf("Could not hash highest scoring block: %v", err)
				continue
			}

			// Save canonical block hash with slot number to DB.
			if err := c.beaconDB.SaveCanonicalSlotNumber(highestScoringBlock.SlotNumber(), h); err != nil {
				log.Errorf("Unable to save slot number to db: %v", err)
				continue
			}

			// Save canonical block to DB.
			if err := c.beaconDB.SaveCanonicalBlock(highestScoringBlock); err != nil {
				log.Errorf("Unable to save block to db: %v", err)
				continue
			}

			log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Info("Canonical block determined")

			// We fire events that notify listeners of a new block (or crystallized state in
			// the case of a state transition). This is useful for the beacon node's gRPC
			// server to stream these events to beacon clients.
			if cycleTransitioned {
				c.canonicalCrystallizedStateFeed.Send(highestScoringCrystallizedState)
			}
			c.canonicalBlockFeed.Send(highestScoringBlock)

			// Clear the blocks pending processing, mutex lock for thread safety
			// in updating this slice.
			c.lock.Lock()
			c.blocksPendingProcessing = [][32]byte{}
			c.lock.Unlock()
		}
	}
}

func (c *ChainService) blockProcessing() {
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
				log.Debugf("Block points to nil parent: %v", err)
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

			log.Infof("Finished processing received block: 0x%x", blockHash)

			// We push the hash of the block we just stored to a pending processing
			// slice the fork choice rule will utilize.
			c.lock.Lock()
			c.blocksPendingProcessing = append(c.blocksPendingProcessing, blockHash)
			c.lock.Unlock()
		}
	}
}
