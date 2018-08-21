// Package blockchain defines the life-cycle and status of the beacon chain.
package blockchain

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "blockchain")

// ChainService represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type ChainService struct {
	ctx                             context.Context
	cancel                          context.CancelFunc
	beaconDB                        *database.DB
	chain                           *BeaconChain
	web3Service                     *powchain.Web3Service
	canonicalBlockEvent             chan *types.Block
	canonicalCrystallizedStateEvent chan *types.CrystallizedState
	latestProcessedBlock            chan *types.Block
	lastFinalizedSlot               uint64
	latestSlotEvent                 chan uint64
	processedBlockHashes            [][32]byte
	// These are the data structures used by the fork choice rule.
	// We store processed blocks and states into a slice by SlotNumber.
	// For example, at slot 5, we might have received 10 different blocks,
	// and a canonical chain must be derived from this DAG.
	processedBlocksBySlot             map[uint64][]*types.Block
	processedCrystallizedStatesBySlot map[uint64][]*types.CrystallizedState
	processedActiveStatesBySlot       map[uint64][]*types.ActiveState
}

// Config options for the service.
type Config struct {
	BeaconBlockBuf  int
	AnnouncementBuf int
	SlotBuf         int
}

// DefaultConfig options.
func DefaultConfig() *Config {
	return &Config{
		BeaconBlockBuf:  10,
		AnnouncementBuf: 10,
		SlotBuf:         10,
	}
}

// NewChainService instantiates a new service instance that will
// be registered into a running beacon node.
func NewChainService(ctx context.Context, cfg *Config, beaconChain *BeaconChain, beaconDB *database.DB, web3Service *powchain.Web3Service) (*ChainService, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &ChainService{
		ctx:                               ctx,
		chain:                             beaconChain,
		cancel:                            cancel,
		beaconDB:                          beaconDB,
		web3Service:                       web3Service,
		latestProcessedBlock:              make(chan *types.Block, cfg.BeaconBlockBuf),
		lastFinalizedSlot:                 1, // TODO: Initialize from the db.
		latestSlotEvent:                   make(chan uint64, cfg.SlotBuf),
		canonicalBlockEvent:               make(chan *types.Block, cfg.AnnouncementBuf),
		canonicalCrystallizedStateEvent:   make(chan *types.CrystallizedState, cfg.AnnouncementBuf),
		processedBlockHashes:              [][32]byte{},
		processedBlocksBySlot:             make(map[uint64][]*types.Block),
		processedCrystallizedStatesBySlot: make(map[uint64][]*types.CrystallizedState),
		processedActiveStatesBySlot:       make(map[uint64][]*types.ActiveState),
	}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	log.Infof("Starting service")
	go c.updateHead(c.ctx.Done())
	go c.blockProcessing(c.ctx.Done())
}

// Stop the blockchain service's main event loop and associated goroutines.
func (c *ChainService) Stop() error {
	defer c.cancel()
	log.Info("Stopping service")
	log.Infof("Persisting current active and crystallized states before closing")
	if err := c.chain.PersistActiveState(); err != nil {
		return fmt.Errorf("Error persisting active state: %v", err)
	}
	if err := c.chain.PersistCrystallizedState(); err != nil {
		return fmt.Errorf("Error persisting crystallized state: %v", err)
	}
	return nil
}

// HasStoredState checks if there is any Crystallized/Active State or blocks(not implemented) are
// persisted to the db.
func (c *ChainService) HasStoredState() (bool, error) {

	hasActive, err := c.beaconDB.DB().Has([]byte(activeStateLookupKey))
	if err != nil {
		return false, err
	}
	hasCrystallized, err := c.beaconDB.DB().Has([]byte(crystallizedStateLookupKey))
	if err != nil {
		return false, err
	}
	if !hasActive || !hasCrystallized {
		return false, nil
	}

	return true, nil
}

// ProcessedBlockHashes exposes a getter for the processed block hashes of the chain.
func (c *ChainService) ProcessedBlockHashes() [][32]byte {
	return c.processedBlockHashes
}

// ProcessBlock accepts a new block for inclusion in the chain.
func (c *ChainService) ProcessBlock(block *types.Block) error {
	if block.SlotNumber() > c.lastFinalizedSlot && block.SlotNumber() > 1 {
		c.latestSlotEvent <- block.SlotNumber()
	}
	h, err := block.Hash()
	if err != nil {
		return fmt.Errorf("could not hash incoming block: %v", err)
	}
	log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Info("Received full block, processing validity conditions")
	canProcess, err := c.chain.CanProcessBlock(c.web3Service.Client(), block)
	if err != nil {
		// We might receive a lot of blocks that fail validity conditions,
		// so we create a debug level log instead of an error log.
		return fmt.Errorf("Incoming block failed validity conditions: %v", err)
	}
	if canProcess {
		// If the block can be processed, we derive its state and store it in an in-memory
		// data structure for our fork choice rule. This block is NOT yet canonical.
		c.latestProcessedBlock <- block
	}
	return nil
}

// SaveBlock is a mock which saves a block to the local db using the
// blockhash as the key.
func (c *ChainService) SaveBlock(block *types.Block) error {
	return c.chain.saveBlock(block)
}

// ContainsBlock checks if a block for the hash exists in the chain.
// This method must be safe to call from a goroutine.
//
// TODO: implement function.
func (c *ChainService) ContainsBlock(h [32]byte) bool {
	return false
}

// CurrentCrystallizedState of the canonical chain.
func (c *ChainService) CurrentCrystallizedState() *types.CrystallizedState {
	return c.chain.CrystallizedState()
}

// CurrentActiveState of the canonical chain.
func (c *ChainService) CurrentActiveState() *types.ActiveState {
	return c.chain.ActiveState()
}

// CanonicalBlockEvent returns a channel that is written to
// whenever a new block is determined to be canonical in the chain.
func (c *ChainService) CanonicalBlockEvent() <-chan *types.Block {
	return c.canonicalBlockEvent
}

// CanonicalCrystallizedStateEvent returns a channel that is written to
// whenever a new crystallized state is determined to be canonical in the chain.
func (c *ChainService) CanonicalCrystallizedStateEvent() <-chan *types.CrystallizedState {
	return c.canonicalCrystallizedStateEvent
}

// run processes the changes needed every beacon chain block,
// including epoch transition if needed.
func (c *ChainService) updateHead(done <-chan struct{}) {
	for {
		select {
		case <-done:
			log.Debug("Chain service context closed, exiting goroutine")
			return
		// Listens for a newly received slot interval to apply the fork choice rule on the
		// last slot before it.
		//
		// For example, if we are currently in slot 1 and have received 5 processed blocks in between slot 1
		// and 2, once a block of slot 2 is received, we update the head of the blockchain by applying
		// a fork choice rule on slot 1.
		case slot := <-c.latestSlotEvent:
			log.WithField("slotNumber", c.lastFinalizedSlot).Info("Applying fork choice rule")
			// Super naive fork choice rule: pick the first element at each slot
			// level as canonical.
			//
			// TODO: Implement real fork choice rule here.
			canonicalActiveState := c.processedActiveStatesBySlot[c.lastFinalizedSlot][0]
			if err := c.chain.SetActiveState(canonicalActiveState); err != nil {
				log.Errorf("Write active state to disk failed: %v", err)
			}

			canonicalCrystallizedState := c.processedCrystallizedStatesBySlot[c.lastFinalizedSlot][0]
			if err := c.chain.SetCrystallizedState(canonicalCrystallizedState); err != nil {
				log.Errorf("Write crystallized state to disk failed: %v", err)
			}

			// TODO: Utilize this value in the fork choice rule.
			vals, err := c.chain.validatorsByHeightShard(canonicalCrystallizedState)
			if err != nil {
				log.Errorf("Unable to get validators by height and by shard: %v", err)
				continue
			}
			log.Debugf("Received %d validators by height", len(vals))

			canonicalBlock := c.processedBlocksBySlot[c.lastFinalizedSlot][0]
			h, err := canonicalBlock.Hash()
			if err != nil {
				log.Errorf("Unable to hash canonical block: %v", err)
				continue
			}
			log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Info("Canonical block determined")
			// Save canonical block to DB.
			// TODO: Implement a SaveCanonical method to differentiate between saving any other
			// regular block.
			if err := c.SaveBlock(canonicalBlock); err != nil {
				log.Errorf("Unable to save block to db: %v", err)
			}
			// We fire events that notify listeners of a new block (or crystallized state in
			// the case of a state transition). This is useful for the beacon node's gRPC
			// server to stream these events to beacon clients.
			transition := c.chain.IsEpochTransition(slot)
			if transition {
				c.canonicalCrystallizedStateEvent <- canonicalCrystallizedState
			}
			c.canonicalBlockEvent <- canonicalBlock
			// Update the last finalized slot.
			c.lastFinalizedSlot = slot
		}
	}
}

func (c *ChainService) blockProcessing(done <-chan struct{}) {
	for {
		select {
		case <-done:
			log.Debug("Chain service context closed, exiting goroutine")
			return
		// Listen for the latestProcessedBlock which has
		// passed validity conditions but has not yet been determined as
		// canonical by the fork choice rule.
		case block := <-c.latestProcessedBlock:
			// 3 steps:
			// - Compute the active state for the block.
			// - Compute the crystallized state for the block if epoch transition.
			// - Store both states and the block into a data structure used for fork choice.
			//
			// Another routine will run that will continually compute
			// the canonical block and states from this data structure using the
			// fork choice rule
			slot := block.SlotNumber()

			// TODO: Using latest block hash for seed, this will eventually be replaced by randao.
			activeState, err := c.chain.computeNewActiveState(block.PowChainRef())
			if err != nil {
				log.Errorf("Compute active state failed: %v", err)
			}

			// Entering cycle transitions.
			transition := c.chain.IsEpochTransition(block.SlotNumber())
			if transition {
				crystallized, err := c.chain.computeNewCrystallizedState(activeState, block)
				if err != nil {
					log.Errorf("Compute crystallized state failed: %v", err)
				}
				c.processedCrystallizedStatesBySlot[slot] = append(c.processedCrystallizedStatesBySlot[slot], crystallized)
			} else {
				c.processedCrystallizedStatesBySlot[slot] = append(c.processedCrystallizedStatesBySlot[slot], c.chain.CrystallizedState())
			}

			// We store a slice of received states and blocks.
			// perceived slot number for forks.
			c.processedBlocksBySlot[slot] = append(c.processedBlocksBySlot[slot], block)
			c.processedActiveStatesBySlot[slot] = append(c.processedActiveStatesBySlot[slot], activeState)
			log.Info("Finished processing received block into DAG")
		}
	}
}
