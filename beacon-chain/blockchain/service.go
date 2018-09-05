// Package blockchain defines the life-cycle and status of the beacon chain.
package blockchain

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "blockchain")
var nilBlock = &types.Block{}
var nilActiveState = &types.ActiveState{}
var nilCrystallizedState = &types.CrystallizedState{}

// ChainService represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type ChainService struct {
	ctx                            context.Context
	cancel                         context.CancelFunc
	beaconDB                       ethdb.Database
	chain                          *BeaconChain
	web3Service                    *powchain.Web3Service
	isValidator                    bool
	incomingBlockFeed              *event.Feed
	incomingBlockChan              chan *types.Block
	canonicalBlockFeed             *event.Feed
	canonicalCrystallizedStateFeed *event.Feed
	latestProcessedBlock           chan *types.Block
	candidateBlock                 *types.Block
	candidateActiveState           *types.ActiveState
	candidateCrystallizedState     *types.CrystallizedState
}

// Config options for the service.
type Config struct {
	BeaconBlockBuf   int
	IncomingBlockBuf int
	Chain            *BeaconChain
	Web3Service      *powchain.Web3Service
	BeaconDB         ethdb.Database
}

// NewChainService instantiates a new service instance that will
// be registered into a running beacon node.
func NewChainService(ctx context.Context, cfg *Config) (*ChainService, error) {
	ctx, cancel := context.WithCancel(ctx)
	var isValidator bool
	if cfg.Web3Service == nil {
		isValidator = false
	} else {
		isValidator = true
	}
	return &ChainService{
		ctx:                            ctx,
		chain:                          cfg.Chain,
		cancel:                         cancel,
		beaconDB:                       cfg.BeaconDB,
		web3Service:                    cfg.Web3Service,
		isValidator:                    isValidator,
		latestProcessedBlock:           make(chan *types.Block, cfg.BeaconBlockBuf),
		incomingBlockChan:              make(chan *types.Block, cfg.IncomingBlockBuf),
		incomingBlockFeed:              new(event.Feed),
		canonicalBlockFeed:             new(event.Feed),
		canonicalCrystallizedStateFeed: new(event.Feed),
		candidateBlock:                 nilBlock,
		candidateActiveState:           nilActiveState,
		candidateCrystallizedState:     nilCrystallizedState,
	}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	if c.isValidator {
		log.Infof("Starting service as validator")
	} else {
		log.Infof("Starting service as observer")
	}
	// TODO: Fetch the slot: (block, state) DAGs from persistent storage
	// to truly continue across sessions.
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

// CanonicalHead of the current beacon chain.
func (c *ChainService) CanonicalHead() (*types.Block, error) {
	return c.chain.CanonicalHead()
}

// CanonicalCrystallizedState of the current beacon chain's head.
func (c *ChainService) CanonicalCrystallizedState() *types.CrystallizedState {
	return c.chain.CrystallizedState()
}

// IncomingBlockFeed returns a feed that a sync service can send incoming p2p blocks into.
// The chain service will subscribe to this feed in order to process incoming blocks.
func (c *ChainService) IncomingBlockFeed() *event.Feed {
	return c.incomingBlockFeed
}

// HasStoredState checks if there is any Crystallized/Active State or blocks(not implemented) are
// persisted to the db.
func (c *ChainService) HasStoredState() (bool, error) {
	hasCrystallized, err := c.beaconDB.Has(crystallizedStateLookupKey)
	if err != nil {
		return false, err
	}
	return hasCrystallized, nil
}

// SaveBlock is a mock which saves a block to the local db using the
// blockhash as the key.
func (c *ChainService) SaveBlock(block *types.Block) error {
	return c.chain.saveBlock(block)
}

// ContainsBlock checks if a block for the hash exists in the chain.
// This method must be safe to call from a goroutine.
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

// CheckForCanonicalBlockBySlot checks if the canonical block for that slot exists
// in the db.
func (c *ChainService) CheckForCanonicalBlockBySlot(slotnumber uint64) (bool, error) {
	return c.chain.hasCanonicalBlockForSlot(slotnumber)
}

// GetCanonicalBlockBySlotNumber retrieves the canonical block for that slot which
// has been saved in the db.
func (c *ChainService) GetCanonicalBlockBySlotNumber(slotnumber uint64) (*types.Block, error) {
	return c.chain.getCanonicalBlockForSlot(slotnumber)
}

// updateHead applies the fork choice rule to the last received
// slot.
func (c *ChainService) updateHead(slot uint64) {
	// Super naive fork choice rule: pick the first element at each slot
	// level as canonical.
	//
	// TODO: Implement real fork choice rule here.
	log.WithField("slotNumber", c.candidateBlock.SlotNumber()).Info("Applying fork choice rule")
	if err := c.chain.SetActiveState(c.candidateActiveState); err != nil {
		log.Errorf("Write active state to disk failed: %v", err)
	}

	if err := c.chain.SetCrystallizedState(c.candidateCrystallizedState); err != nil {
		log.Errorf("Write crystallized state to disk failed: %v", err)
	}

	// TODO: Utilize this value in the fork choice rule.
	vals, err := casper.ShuffleValidatorsToCommittees(
		c.candidateCrystallizedState.DynastySeed(),
		c.candidateCrystallizedState.Validators(),
		c.candidateCrystallizedState.CurrentDynasty(),
		c.candidateCrystallizedState.CrosslinkingStartShard(),
	)

	if err != nil {
		log.Errorf("Unable to get validators by slot and by shard: %v", err)
		return
	}
	log.Debugf("Received %d validators by slot", len(vals))

	h, err := c.candidateBlock.Hash()
	if err != nil {
		log.Errorf("Unable to hash canonical block: %v", err)
		return
	}

	// Save canonical slotnumber to DB.
	if err := c.chain.saveCanonicalSlotNumber(c.candidateBlock.SlotNumber(), h); err != nil {
		log.Errorf("Unable to save slot number to db: %v", err)
	}

	// Save canonical block to DB.
	if err := c.chain.saveCanonicalBlock(c.candidateBlock); err != nil {
		log.Errorf("Unable to save block to db: %v", err)
	}
	log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Info("Canonical block determined")

	// We fire events that notify listeners of a new block (or crystallized state in
	// the case of a state transition). This is useful for the beacon node's gRPC
	// server to stream these events to beacon clients.
	transition := c.chain.IsCycleTransition(slot)
	if transition {
		c.canonicalCrystallizedStateFeed.Send(c.candidateCrystallizedState)
	}
	c.canonicalBlockFeed.Send(c.candidateBlock)

	c.candidateBlock = nilBlock
	c.candidateActiveState = nilActiveState
	c.candidateCrystallizedState = nilCrystallizedState
}

func (c *ChainService) blockProcessing(done <-chan struct{}) {
	sub := c.incomingBlockFeed.Subscribe(c.incomingBlockChan)
	defer sub.Unsubscribe()
	for {
		select {
		case <-done:
			log.Debug("Chain service context closed, exiting goroutine")
			return
		case block := <-c.incomingBlockChan:
			var canProcessBlock bool
			var canProcessAttestations bool
			var err error
			var blockVoteCache map[[32]byte]*types.VoteCache

			h, err := block.Hash()
			if err != nil {
				log.Debugf("Could not hash incoming block: %v", err)
			}

			receivedSlotNumber := block.SlotNumber()

			log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Info("Received full block, processing validity conditions")

			parentExists, err := c.chain.hasBlock(block.ParentHash())
			if err != nil {
				log.Debugf("Could not check existence of parent hash: %v", err)
			}

			// If parentHash does not exist, received block fails validity conditions.
			if !parentExists && receivedSlotNumber > 1 {
				continue
			}

			// Process block as a validator if beacon node has registered, else process block as an observer.
			canProcessBlock, err = c.chain.CanProcessBlock(c.web3Service.Client(), block, c.isValidator)
			if err != nil {
				// We might receive a lot of blocks that fail validity conditions,
				// so we create a debug level log instead of an error log.
				log.Debugf("Incoming block failed validity conditions: %v", err)
			}

			// Process attestations as a beacon chain node.
			var processedAttestations []*pb.AttestationRecord
			for index, attestation := range block.Attestations() {
				if err := c.chain.processAttestation(index, block); err != nil {
					canProcessAttestations = false
					log.Errorf("could not process attestation for block %d because %v", block.SlotNumber(), err)
				} else {
					canProcessAttestations = true
					processedAttestations = append(processedAttestations, attestation)
				}
			}

			// If we cannot process an attestation in this block, we keep listening.
			if !canProcessAttestations {
				continue
			}
			// If we cannot process this block, we keep listening.
			if !canProcessBlock {
				continue
			}

			// With a valid beacon block, we can compute its attestations and store its votes/deposits in cache.
			for index := range block.Attestations() {
				blockVoteCache, err = c.chain.calculateBlockVoteCache(index, block)
				if err != nil {
					log.Debugf("could not calculate new block vote cache: %v", nil)
				}
			}

			if c.candidateBlock != nilBlock && receivedSlotNumber > c.candidateBlock.SlotNumber() && receivedSlotNumber > 1 {
				c.updateHead(receivedSlotNumber)
			}

			if err := c.chain.saveBlock(block); err != nil {
				log.Errorf("Failed to save block: %v", err)
			}

			log.Info("Finished processing received block")

			// Do not proceed further, because a candidate has already been chosen.
			if c.candidateBlock != nilBlock {
				continue
			}

			// 3 steps:
			// - Compute the active state for the block.
			// - Compute the crystallized state for the block if cycle transition.
			// - Store both states and the block into a data structure used for fork choice
			//
			// This data structure will be used by the updateHead function to determine
			// canonical blocks and states.
			// TODO: Using latest block hash for seed, this will eventually be replaced by randao.

			// Entering cycle transitions.
			isTransition := c.chain.IsCycleTransition(receivedSlotNumber)
			activeState := c.chain.ActiveState()
			crystallizedState := c.chain.CrystallizedState()
			if isTransition {
				crystallizedState, activeState = c.chain.initCycle(crystallizedState, activeState)
			}

			activeState, err = c.chain.computeNewActiveState(processedAttestations, activeState, blockVoteCache, h)
			if err != nil {
				log.Errorf("Compute active state failed: %v", err)
			}

			c.candidateBlock = block
			c.candidateActiveState = activeState
			c.candidateCrystallizedState = crystallizedState

			log.Info("Finished processing state for candidate block")
		}
	}
}
