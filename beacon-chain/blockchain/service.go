// Package blockchain defines the life-cycle and status of the beacon chain.
package blockchain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "blockchain")

// ChainService represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type ChainService struct {
	ctx                context.Context
	cancel             context.CancelFunc
	beaconDB           *db.BeaconDB
	web3Service        *powchain.Web3Service
	incomingBlockFeed  *event.Feed
	incomingBlockChan  chan *types.Block
	processedBlockChan chan *types.Block
	canonicalBlockFeed *event.Feed
	canonicalStateFeed *event.Feed
	genesisTime        time.Time
	unProcessedBlocks  map[uint64]*types.Block
	unfinalizedBlocks  map[[32]byte]*types.BeaconState
	enablePOWChain     bool
}

// Config options for the service.
type Config struct {
	BeaconBlockBuf   int
	IncomingBlockBuf int
	Web3Service      *powchain.Web3Service
	BeaconDB         *db.BeaconDB
	DevMode          bool
	EnablePOWChain   bool
}

// NewChainService instantiates a new service instance that will
// be registered into a running beacon node.
func NewChainService(ctx context.Context, cfg *Config) (*ChainService, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &ChainService{
		ctx:                ctx,
		cancel:             cancel,
		beaconDB:           cfg.BeaconDB,
		web3Service:        cfg.Web3Service,
		incomingBlockChan:  make(chan *types.Block, cfg.IncomingBlockBuf),
		processedBlockChan: make(chan *types.Block),
		incomingBlockFeed:  new(event.Feed),
		canonicalBlockFeed: new(event.Feed),
		canonicalStateFeed: new(event.Feed),
		unProcessedBlocks:  make(map[uint64]*types.Block),
		unfinalizedBlocks:  make(map[[32]byte]*types.BeaconState),
		enablePOWChain:     cfg.EnablePOWChain,
	}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	log.Info("Starting service")

	var err error
	c.genesisTime, err = c.beaconDB.GetGenesisTime()
	if err != nil {
		log.Fatalf("Unable to retrieve genesis time, therefore blockchain service cannot be started %v", err)
		return
	}

	// TODO(#675): Initialize unfinalizedBlocks map from disk in case this
	// is a beacon node restarting.
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

// CanonicalStateFeed returns a feed that is written to
// whenever a new state is determined to be canonical in the chain.
func (c *ChainService) CanonicalStateFeed() *event.Feed {
	return c.canonicalStateFeed
}

// doesPoWBlockExist checks if the referenced PoW block exists.
func (c *ChainService) doesPoWBlockExist(hash [32]byte) bool {
	powBlock, err := c.web3Service.Client().BlockByHash(c.ctx, hash)
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
			if block == nil {
				continue
			}

			h, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash incoming block: %v", err)
				continue
			}

			log.Info("Updating chain head...")
			currentHead, err := c.beaconDB.GetChainHead()
			if err != nil {
				log.Errorf("Could not get current chain head: %v", err)
				continue
			}
			currentState, err := c.beaconDB.GetState()
			if err != nil {
				log.Errorf("Could not get current beacon state: %v", err)
				continue
			}

			blockState := c.unfinalizedBlocks[h]

			var headUpdated bool
			newHead := currentHead
			// If both blocks have the same crystallized state root, we favor one which has
			// the higher slot.
			if currentHead.StateRootHash32() == block.StateRootHash32() {
				if block.SlotNumber() > currentHead.SlotNumber() {
					newHead = block
					headUpdated = true
				}
				// 2a. Pick the block with the higher last_finalized_slot.
				// 2b. If same, pick the block with the higher last_justified_slot.
			} else if blockState.LastFinalizedSlot() > currentState.LastFinalizedSlot() {
				newHead = block
				headUpdated = true
			} else if blockState.LastFinalizedSlot() == currentState.LastFinalizedSlot() {
				if blockState.LastJustifiedSlot() > currentState.LastJustifiedSlot() {
					newHead = block
					headUpdated = true
				} else if blockState.LastJustifiedSlot() == currentState.LastJustifiedSlot() {
					if block.SlotNumber() > currentHead.SlotNumber() {
						newHead = block
						headUpdated = true
					}
				}
			}

			// If no new head was found, we do not update the chain.
			if !headUpdated {
				log.Info("Chain head not updated")
				continue
			}

			// TODO(#674): Handle chain reorgs.
			newState := blockState
			if err := c.beaconDB.UpdateChainHead(newHead, newState); err != nil {
				log.Errorf("Failed to update chain: %v", err)
				continue
			}
			log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Info("Chain head block and state updated")
			// We fire events that notify listeners of a new block in
			// the case of a state transition. This is useful for the beacon node's gRPC
			// server to stream these events to beacon clients.
			// When the transition is a cycle transition, we stream the state containing the new validator
			// assignments to clients.
			if block.SlotNumber()%params.BeaconConfig().CycleLength == 0 {
				c.canonicalStateFeed.Send(newState)
			}
			c.canonicalBlockFeed.Send(newHead)
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

		// Listen for a newly received incoming block from the feed. Blocks
		// can be received either from the sync service, the RPC service,
		// or via p2p.
		case block := <-c.incomingBlockChan:

			// Before sending the blocks for processing we check to see if the blocks
			// are valid to continue being processed. If the slot number in the block
			// has already been processed by the beacon node, we throw it away. If the
			// slot number is too high to be processed in the current slot, we store
			// it in a cache.

			beaconState, err := c.beaconDB.GetState()
			if err != nil {
				log.Errorf("Unable to retrieve beacon state %v", err)
				continue
			}

			currentSlot := beaconState.Slot()

			if currentSlot+1 < block.SlotNumber() {
				c.unProcessedBlocks[block.SlotNumber()] = block
				continue
			}

			if currentSlot+1 == block.SlotNumber() {

				if err := c.receiveBlock(block); err != nil {
					log.Error(err)
					processedBlock <- nil
					continue
				}

				// Push the block to trigger the fork choice rule.
				processedBlock <- block
			} else {
				log.Debugf(
					"Block slot number is lower than the current slot in the beacon state %d",
					block.SlotNumber())
				c.sendAndDeleteCachedBlocks(currentSlot)
			}
		}
	}
}

// receiveBlock is a function that defines the operations that are preformed on
// any block that is received from p2p layer or rpc. It checks the block to see
// if it passes the pre-processing conditions, if it does then the per slot
// state transition function is carried out on the block.
// spec:
//  def process_block(block):
//      if not block_pre_processing_conditions(block):
//          return False
//
//  	# process skipped slots
//
// 		while (state.slot < block.slot - 1):
//      	state = slot_state_transition(state, block=None)
//
//		# process slot with block
//		state = slot_state_transition(state, block)
//
//		# check state root
//      if block.state_root == hash(state):
//			return state
//		else:
//			return False  # or throw or whatever
//
func (c *ChainService) receiveBlock(block *types.Block) error {

	blockhash, err := block.Hash()
	if err != nil {
		return fmt.Errorf("could not hash incoming block: %v", err)
	}

	beaconState, err := c.beaconDB.GetState()
	if err != nil {
		return fmt.Errorf("failed to get beacon state: %v", err)
	}

	if block.SlotNumber() == 0 {
		return errors.New("cannot process a genesis block: received block with slot 0")
	}

	// Save blocks with higher slot numbers in cache.
	if !c.isBlockReadyForProcessing(block) && block.SlotNumber() > beaconState.Slot() {
		c.unProcessedBlocks[block.SlotNumber()] = block
		log.Debugf("block with hash %#x is not ready for processing", blockhash)
		return nil
	}

	log.WithField("slotNumber", block.SlotNumber()).Info("Executing state transition")

	// Check for skipped slots and update the corresponding proposers
	// randao layer.
	for beaconState.Slot() < block.SlotNumber()-1 {
		beaconState, err = state.ExecuteStateTransition(beaconState, nil)
		if err != nil {
			return fmt.Errorf("unable to execute state transition %v", err)
		}
	}

	beaconState, err = state.ExecuteStateTransition(beaconState, block)
	if err != nil {
		return errors.New("unable to execute state transition")
	}

	if beaconState.IsValidatorSetChange(block.SlotNumber()) {
		log.WithField("slotNumber", block.SlotNumber()).Info("Validator set rotation occurred")
	}

	// TODO(#1074): Verify block.state_root == hash_tree_root(state)
	// if there exists a block for the slot being processed.

	if err := c.beaconDB.SaveBlock(block); err != nil {
		return fmt.Errorf("failed to save block: %v", err)
	}
	if err := c.beaconDB.SaveUnfinalizedBlockState(beaconState); err != nil {
		return fmt.Errorf("error persisting unfinalized block's state: %v", err)
	}

	log.WithField("hash", fmt.Sprintf("%#x", blockhash)).Debug("Processed beacon block")

	// We keep a map of unfinalized blocks in memory along with their state
	// pair to apply the fork choice rule.
	c.unfinalizedBlocks[blockhash] = beaconState

	return nil
}

func (c *ChainService) isBlockReadyForProcessing(block *types.Block) bool {

	beaconState, err := c.beaconDB.GetState()
	if err != nil {
		log.Debugf("failed to get beacon state: %v", err)
		return false
	}

	if err := state.IsValidBlock(c.ctx, beaconState, block, c.enablePOWChain,
		c.beaconDB.HasBlock, c.web3Service.Client().BlockByHash, c.genesisTime); err != nil {
		log.Debugf("block does not fulfill pre-processing conditions %v", err)
		return false
	}

	return true
}

// sendAndDeleteCachedBlocks checks if there is any block saved in the cache with a
// slot number equivalent to the current slot. If there is then the block is
// sent to the incoming block channel and deleted from the cache.
func (c *ChainService) sendAndDeleteCachedBlocks(currentSlot uint64) {
	if block, ok := c.unProcessedBlocks[currentSlot+1]; ok && c.isBlockReadyForProcessing(block) {
		c.incomingBlockChan <- block
		delete(c.unProcessedBlocks, currentSlot)
	}
}

// DEPRECATED: Will be replaced by new block processing method
func (c *ChainService) processBlockOld(block *types.Block) error {
	blockHash, err := block.Hash()
	if err != nil {
		return fmt.Errorf("failed to get hash of block: %v", err)
	}

	parent, err := c.beaconDB.GetBlock(block.ParentHash())
	if err != nil {
		return fmt.Errorf("could not get parent block: %v", err)
	}
	if parent == nil {
		return fmt.Errorf("block points to nil parent: %#x", block.ParentHash())
	}

	beaconState, err := c.beaconDB.GetState()
	if err != nil {
		return fmt.Errorf("failed to get beacon state: %v", err)
	}

	if c.enablePOWChain && !c.doesPoWBlockExist(beaconState.ProcessedPowReceiptRootHash32()) {
		return errors.New("proof-of-Work chain reference in block does not exist")
	}

	// Verifies the block against the validity conditions specifies as part of the
	// Ethereum 2.0 specification.
	if err := state.IsValidBlockOld(
		block,
		beaconState,
		parent.SlotNumber(),
		c.genesisTime,
		c.beaconDB.HasBlock,
	); err != nil {
		return fmt.Errorf("block failed validity conditions: %v", err)
	}

	if err := c.calculateNewBlockVotes(block, beaconState); err != nil {
		return fmt.Errorf("failed to calculate block vote cache: %v", err)
	}

	// First, include new attestations to the active state
	// so that they're accounted for during cycle transitions.
	beaconState.SetPendingAttestations(block.Attestations())

	// If the block is valid, we compute its associated state tuple (active, crystallized)
	beaconState, err = c.executeStateTransitionOld(beaconState, block, parent.SlotNumber())
	if err != nil {
		return fmt.Errorf("initialize new cycle transition failed: %v", err)
	}

	if err := c.beaconDB.SaveBlock(block); err != nil {
		return fmt.Errorf("failed to save block: %v", err)
	}
	if err := c.beaconDB.SaveUnfinalizedBlockState(beaconState); err != nil {
		return fmt.Errorf("error persisting unfinalized block's state: %v", err)
	}

	log.WithField("hash", fmt.Sprintf("%#x", blockHash)).Info("Processed beacon block")

	// We keep a map of unfinalized blocks in memory along with their state
	// pair to apply the fork choice rule.
	c.unfinalizedBlocks[blockHash] = beaconState

	return nil
}

// DEPRECATED: Will be removed soon
func (c *ChainService) executeStateTransitionOld(
	beaconState *types.BeaconState,
	block *types.Block,
	parentSlot uint64,
) (*types.BeaconState, error) {
	log.WithField("slotNumber", block.SlotNumber()).Info("Executing state transition")
	blockVoteCache, err := c.beaconDB.ReadBlockVoteCache(beaconState.LatestBlockRootHashes32())
	if err != nil {
		return nil, err
	}
	newState, err := state.NewStateTransition(beaconState, block, parentSlot, blockVoteCache)
	if err != nil {
		return nil, err
	}
	if newState.IsValidatorSetChange(block.SlotNumber()) {
		log.WithField("slotNumber", block.SlotNumber()).Info("Validator set rotation occurred")
	}
	return newState, nil
}

func (c *ChainService) calculateNewBlockVotes(block *types.Block, beaconState *types.BeaconState) error {
	for _, attestation := range block.Attestations() {
		parentHashes, err := beaconState.SignedParentHashes(block, attestation)
		if err != nil {
			return err
		}
		shardCommittees, err := v.GetShardAndCommitteesForSlot(
			beaconState.ShardAndCommitteesForSlots(),
			beaconState.LastStateRecalculationSlot(),
			attestation.GetSlot(),
		)
		if err != nil {
			return fmt.Errorf("unable to fetch ShardAndCommittees for slot %d: %v", attestation.Slot, err)
		}
		attesterIndices, err := v.AttesterIndices(shardCommittees, attestation)
		if err != nil {
			return err
		}

		// Read block vote cache from DB.
		var blockVoteCache utils.BlockVoteCache
		if blockVoteCache, err = c.beaconDB.ReadBlockVoteCache(parentHashes); err != nil {
			return err
		}

		// Update block vote cache.
		for _, h := range parentHashes {
			// Skip calculating for this hash if the hash is part of oblique parent hashes.
			var skip bool
			for _, oblique := range attestation.ObliqueParentHashes {
				if bytes.Equal(h[:], oblique) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}

			// Initialize vote cache of a given block hash if it doesn't exist already.
			if !blockVoteCache.IsVoteCacheExist(h) {
				blockVoteCache[h] = utils.NewBlockVote()
			}

			// Loop through attester indices, if the attester has voted but was not accounted for
			// in the cache, then we add attester's index and balance to the block cache.
			for i, attesterIndex := range attesterIndices {
				var attesterExists bool
				isBitSet, err := bitutil.CheckBit(attestation.AttesterBitfield, i)
				if err != nil {
					log.Errorf("Bitfield check for cache adding failed at index: %d with: %v", i, err)
				}

				if !isBitSet {
					continue
				}
				for _, indexInCache := range blockVoteCache[h].VoterIndices {
					if attesterIndex == indexInCache {
						attesterExists = true
						break
					}
				}
				if !attesterExists {
					blockVoteCache[h].VoterIndices = append(blockVoteCache[h].VoterIndices, attesterIndex)
					blockVoteCache[h].VoteTotalDeposit += beaconState.ValidatorRegistry()[attesterIndex].Balance
				}
			}
		}

		// Write updated block vote cache back to DB.
		if err = c.beaconDB.WriteBlockVoteCache(blockVoteCache); err != nil {
			return err
		}
	}

	return nil
}
