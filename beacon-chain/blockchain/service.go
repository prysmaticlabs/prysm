// Package blockchain defines the life-cycle and status of the beacon chain.
package blockchain

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "blockchain")

// ChainService represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type ChainService struct {
	ctx                            context.Context
	cancel                         context.CancelFunc
	beaconDB                       ethdb.Database
	chain                          *BeaconChain
	web3Service                    *powchain.Web3Service
	validator                      bool
	incomingBlockFeed              *event.Feed
	incomingBlockChan              chan *types.Block
	canonicalBlockFeed             *event.Feed
	canonicalCrystallizedStateFeed *event.Feed
	latestProcessedBlock           chan *types.Block
	lastSlot                       uint64
	// These are the data structures used by the fork choice rule.
	// We store processed blocks and states into a slice by SlotNumber.
	// For example, at slot 5, we might have received 10 different blocks,
	// and a canonical chain must be derived from this DAG.
	//
	// NOTE: These are temporary and will be replaced by a structure
	// that can support light-client proofs, such as a Sparse Merkle Trie.
	processedBlockHashesBySlot        map[uint64][][]byte
	processedBlocksBySlot             map[uint64][]*types.Block
	processedCrystallizedStatesBySlot map[uint64][]*types.CrystallizedState
	processedActiveStatesBySlot       map[uint64][]*types.ActiveState
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
		ctx:                               ctx,
		chain:                             cfg.Chain,
		cancel:                            cancel,
		beaconDB:                          cfg.BeaconDB,
		web3Service:                       cfg.Web3Service,
		validator:                         isValidator,
		latestProcessedBlock:              make(chan *types.Block, cfg.BeaconBlockBuf),
		incomingBlockChan:                 make(chan *types.Block, cfg.IncomingBlockBuf),
		lastSlot:                          1, // TODO: Initialize from the db.
		incomingBlockFeed:                 new(event.Feed),
		canonicalBlockFeed:                new(event.Feed),
		canonicalCrystallizedStateFeed:    new(event.Feed),
		processedBlockHashesBySlot:        make(map[uint64][][]byte),
		processedBlocksBySlot:             make(map[uint64][]*types.Block),
		processedCrystallizedStatesBySlot: make(map[uint64][]*types.CrystallizedState),
		processedActiveStatesBySlot:       make(map[uint64][]*types.ActiveState),
	}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	if c.validator {
		log.Infof("Starting service as validator")
	} else {
		log.Infof("Starting service as observer")
	}
	head, err := c.chain.CanonicalHead()
	if err != nil {
		log.Fatalf("Could not fetch latest canonical head from DB: %v", err)
	}
	// If there was a canonical head stored in persistent storage,
	// the fork choice rule proceed where it left off.
	if head != nil {
		c.lastSlot = head.SlotNumber() + 1
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

// IncomingBlockFeed returns a feed that a sync service can send incoming p2p blocks into.
// The chain service will subscribe to this feed in order to process incoming blocks.
func (c *ChainService) IncomingBlockFeed() *event.Feed {
	return c.incomingBlockFeed
}

// HasStoredState checks if there is any Crystallized/Active State or blocks(not implemented) are
// persisted to the db.
func (c *ChainService) HasStoredState() (bool, error) {

	hasActive, err := c.beaconDB.Has([]byte(activeStateLookupKey))
	if err != nil {
		return false, err
	}
	hasCrystallized, err := c.beaconDB.Has([]byte(crystallizedStateLookupKey))
	if err != nil {
		return false, err
	}
	if !hasActive || !hasCrystallized {
		return false, nil
	}

	return true, nil
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

// updateHead applies the fork choice rule to the last received
// slot.
func (c *ChainService) updateHead(slot uint64) {
	// Super naive fork choice rule: pick the first element at each slot
	// level as canonical.
	//
	// TODO: Implement real fork choice rule here.

	// If no blocks were stored at the last slot, we simply skip the fork choice
	// rule.
	if len(c.processedBlocksBySlot[c.lastSlot]) == 0 {
		return
	}
	log.WithField("slotNumber", c.lastSlot).Info("Applying fork choice rule")
	canonicalActiveState := c.processedActiveStatesBySlot[c.lastSlot][0]
	if err := c.chain.SetActiveState(canonicalActiveState); err != nil {
		log.Errorf("Write active state to disk failed: %v", err)
	}

	canonicalCrystallizedState := c.processedCrystallizedStatesBySlot[c.lastSlot][0]
	if err := c.chain.SetCrystallizedState(canonicalCrystallizedState); err != nil {
		log.Errorf("Write crystallized state to disk failed: %v", err)
	}

	// TODO: Utilize this value in the fork choice rule.
	vals, err := casper.ShuffleValidatorsToCommittees(
		canonicalCrystallizedState.DynastySeed(),
		canonicalCrystallizedState.Validators(),
		canonicalCrystallizedState.CurrentDynasty(),
		canonicalCrystallizedState.CrosslinkingStartShard())

	if err != nil {
		log.Errorf("Unable to get validators by height and by shard: %v", err)
		return
	}
	log.Debugf("Received %d validators by height", len(vals))

	canonicalBlock := c.processedBlocksBySlot[c.lastSlot][0]
	h, err := canonicalBlock.Hash()
	if err != nil {
		log.Errorf("Unable to hash canonical block: %v", err)
		return
	}
	// Save canonical block to DB.
	if err := c.chain.saveCanonical(canonicalBlock); err != nil {
		log.Errorf("Unable to save block to db: %v", err)
	}
	log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Info("Canonical block determined")

	// Update the last received slot.
	c.lastSlot = slot

	// We fire events that notify listeners of a new block (or crystallized state in
	// the case of a state transition). This is useful for the beacon node's gRPC
	// server to stream these events to beacon clients.
	transition := c.chain.IsCycleTransition(slot)
	if transition {
		c.canonicalCrystallizedStateFeed.Send(canonicalCrystallizedState)
	}
	c.canonicalBlockFeed.Send(canonicalBlock)
}

func (c *ChainService) blockProcessing(done <-chan struct{}) {
	sub := c.incomingBlockFeed.Subscribe(c.incomingBlockChan)
	defer sub.Unsubscribe()
	for {
		select {
		case <-done:
			log.Debug("Chain service context closed, exiting goroutine")
			return
		// Listen for a newly received incoming block from the sync service.
		case block := <-c.incomingBlockChan:
			// 3 steps:
			// - Compute the active state for the block.
			// - Compute the crystallized state for the block if cycle transition.
			// - Store both states and the block into a data structure used for fork choice.
			//
			// Another routine will run that will continually compute
			// the canonical block and states from this data structure using the
			// fork choice rule.
			var canProcess bool
			var err error
			var blockVoteCache map[*common.Hash]*types.VoteCache

			h, err := block.Hash()
			if err != nil {
				log.Debugf("Could not hash incoming block: %v", err)
			}

			receivedSlotNumber := block.SlotNumber()

			log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Info("Received full block, processing validity conditions")

			// Check if parentHash is in previous slot's processed blockHash list.
			// TODO: This is messy. Instead, we should implement c.chain.CanProcessBlock
			// to take in the block and the DAG of previously processed blocks
			// and determine all validity conditions from those two parameters.
			isParentHashExistent := false
			for i := 0; i < len(c.processedBlockHashesBySlot[receivedSlotNumber-1]); i++ {
				p := block.ParentHash()
				if bytes.Equal(c.processedBlockHashesBySlot[receivedSlotNumber-1][i], p[:]) {
					isParentHashExistent = true
				}
			}

			// If parentHash does not exist, received block fails validity conditions.
			if !isParentHashExistent && receivedSlotNumber > 1 {
				continue
			}

			// Process block as a validator if beacon node has registered, else process block as an observer.
			if c.validator {
				canProcess, err = c.chain.CanProcessBlock(c.web3Service.Client(), block, true)
			} else {
				canProcess, err = c.chain.CanProcessBlock(nil, block, false)
			}
			if err != nil {
				// We might receive a lot of blocks that fail validity conditions,
				// so we create a debug level log instead of an error log.
				log.Debugf("Incoming block failed validity conditions: %v", err)
			}

			// If we cannot process this block, we keep listening.
			if !canProcess {
				continue
			}

			// Process attestations as a beacon chain node.
			var processedAttestations []*pb.AttestationRecord
			for index, attestation := range block.Attestations() {
				// Don't add invalid attestation to block vote cache.
				if err := c.chain.processAttestation(index, block); err == nil {
					processedAttestations = append(processedAttestations, attestation)
					blockVoteCache, err = c.chain.calculateBlockVoteCache(index, block)
					if err != nil {
						log.Debugf("could not calculate new block vote cache: %v", nil)
					}
				}
			}

			if receivedSlotNumber > c.lastSlot && receivedSlotNumber > 1 {
				c.updateHead(receivedSlotNumber)
			}

			// 3 steps:
			// - Compute the active state for the block.
			// - Compute the crystallized state for the block if cycle transition.
			// - Store both states and the block into a data structure used for fork choice
			//
			// This data structure will be used by the updateHead function to determine
			// canonical blocks and states.
			// TODO: Using latest block hash for seed, this will eventually be replaced by randao.
			activeState, err := c.chain.computeNewActiveState(processedAttestations, c.chain.ActiveState(), blockVoteCache)
			if err != nil {
				log.Errorf("Compute active state failed: %v", err)
			}
			if err := c.chain.SetActiveState(activeState); err != nil {
				log.Errorf("Set active state failed: %v", err)
			}
			// Entering cycle transitions.
			transition := c.chain.IsCycleTransition(receivedSlotNumber)
			if transition {
				crystallized, err := c.chain.computeNewCrystallizedState(activeState, block)
				if err != nil {
					log.Errorf("Compute crystallized state failed: %v", err)
				}
				c.processedCrystallizedStatesBySlot[receivedSlotNumber] = append(
					c.processedCrystallizedStatesBySlot[receivedSlotNumber],
					crystallized,
				)
			} else {
				c.processedCrystallizedStatesBySlot[receivedSlotNumber] = append(
					c.processedCrystallizedStatesBySlot[receivedSlotNumber],
					c.chain.CrystallizedState(),
				)
			}

			// We store a slice of received states and blocks.
			// perceived slot number for forks.
			c.processedBlockHashesBySlot[receivedSlotNumber] = append(
				c.processedBlockHashesBySlot[receivedSlotNumber],
				h[:],
			)

			c.processedBlocksBySlot[receivedSlotNumber] = append(
				c.processedBlocksBySlot[receivedSlotNumber],
				block,
			)
			c.processedActiveStatesBySlot[receivedSlotNumber] = append(
				c.processedActiveStatesBySlot[receivedSlotNumber],
				activeState,
			)
			log.Info("Finished processing received block and states into DAG")
		}
	}
}
