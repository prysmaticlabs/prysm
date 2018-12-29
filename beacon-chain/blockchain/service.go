// Package blockchain defines the life-cycle and status of the beacon chain.
package blockchain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
	incomingBlockChan  chan *pb.BeaconBlock
	processedBlockChan chan *pb.BeaconBlock
	canonicalBlockFeed *event.Feed
	canonicalStateFeed *event.Feed
	genesisTime        time.Time
	unProcessedBlocks  map[uint64]*pb.BeaconBlock
	unfinalizedBlocks  map[[32]byte]*pb.BeaconState
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
		incomingBlockChan:  make(chan *pb.BeaconBlock, cfg.IncomingBlockBuf),
		processedBlockChan: make(chan *pb.BeaconBlock),
		incomingBlockFeed:  new(event.Feed),
		canonicalBlockFeed: new(event.Feed),
		canonicalStateFeed: new(event.Feed),
		unProcessedBlocks:  make(map[uint64]*pb.BeaconBlock),
		unfinalizedBlocks:  make(map[[32]byte]*pb.BeaconState),
		enablePOWChain:     cfg.EnablePOWChain,
	}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	log.Info("Starting service")
	var err error
	c.genesisTime, err = c.beaconDB.GenesisTime()
	if err != nil {
		log.Fatalf("Unable to retrieve genesis time - blockchain service could not start: %v", err)
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

// Status always returns nil.
// TODO(1202): Add service health checks.
func (c *ChainService) Status() error {
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
func (c *ChainService) updateHead(processedBlock <-chan *pb.BeaconBlock) {
	for {
		select {
		case <-c.ctx.Done():
			return
		case block := <-processedBlock:
			if block == nil {
				continue
			}

			h, err := b.Hash(block)
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
			if bytes.Equal(currentHead.GetStateRootHash32(), block.GetStateRootHash32()) {
				if block.GetSlot() > currentHead.GetSlot() {
					newHead = block
					headUpdated = true
				}
				// 2a. Pick the block with the higher last_finalized_slot.
				// 2b. If same, pick the block with the higher last_justified_slot.
			} else if blockState.GetFinalizedSlot() > currentState.GetFinalizedSlot() {
				newHead = block
				headUpdated = true
			} else if blockState.GetFinalizedSlot() == currentState.GetFinalizedSlot() {
				if blockState.GetJustifiedSlot() > currentState.GetJustifiedSlot() {
					newHead = block
					headUpdated = true
				} else if blockState.GetJustifiedSlot() == currentState.GetJustifiedSlot() {
					if block.GetSlot() > currentHead.GetSlot() {
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
			if block.GetSlot()%params.BeaconConfig().CycleLength == 0 {
				c.canonicalStateFeed.Send(newState)
			}
			c.canonicalBlockFeed.Send(newHead)
		}
	}
}

func (c *ChainService) blockProcessing(processedBlock chan<- *pb.BeaconBlock) {
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

			currentSlot := beaconState.GetSlot()
			if currentSlot+1 < block.GetSlot() {
				c.unProcessedBlocks[block.GetSlot()] = block
				continue
			}

			if currentSlot+1 == block.GetSlot() {
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
					block.GetSlot())
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
func (c *ChainService) receiveBlock(block *pb.BeaconBlock) error {

	blockhash, err := b.Hash(block)
	if err != nil {
		return fmt.Errorf("could not hash incoming block: %v", err)
	}

	beaconState, err := c.beaconDB.GetState()
	if err != nil {
		return fmt.Errorf("failed to get beacon state: %v", err)
	}

	if block.GetSlot() == 0 {
		return errors.New("cannot process a genesis block: received block with slot 0")
	}

	// Save blocks with higher slot numbers in cache.
	if err := c.isBlockReadyForProcessing(block); err != nil {
		log.Debugf("block with hash %#x is not ready for processing: %v", blockhash, err)
		return nil
	}

	prevBlock, err := c.beaconDB.GetChainHead()
	if err != nil {
		return fmt.Errorf("could not retrieve chain head %v", err)
	}

	// TODO(#716):Replace with tree-hashing algorithm.
	blockRoot, err := b.Hash(prevBlock)
	if err != nil {
		return fmt.Errorf("could not hash block %v", err)
	}

	log.WithField("slotNumber", block.GetSlot()).Info("Executing state transition")

	// Check for skipped slots and update the corresponding proposers
	// randao layer.
	for beaconState.GetSlot() < block.GetSlot()-1 {
		beaconState, err = state.ExecuteStateTransition(beaconState, nil, blockRoot)
		if err != nil {
			return fmt.Errorf("could not execute state transition %v", err)
		}
	}

	beaconState, err = state.ExecuteStateTransition(beaconState, block, blockRoot)
	if err != nil {
		return errors.New("could not execute state transition")
	}

	if state.IsValidatorSetChange(beaconState, block.GetSlot()) {
		log.WithField("slotNumber", block.GetSlot()).Info("Validator set rotation occurred")
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

func (c *ChainService) isBlockReadyForProcessing(block *pb.BeaconBlock) error {
	beaconState, err := c.beaconDB.GetState()
	if err != nil {
		return fmt.Errorf("failed to get beacon state: %v", err)
	}

	var powBlockFetcher func(ctx context.Context, hash common.Hash) (*gethTypes.Block, error)
	if c.enablePOWChain {
		powBlockFetcher = c.web3Service.Client().BlockByHash
	}
	if err := b.IsValidBlock(c.ctx, beaconState, block, c.enablePOWChain,
		c.beaconDB.HasBlock, powBlockFetcher, c.genesisTime); err != nil {
		return fmt.Errorf("block does not fulfill pre-processing conditions %v", err)
	}
	return nil
}

// sendAndDeleteCachedBlocks checks if there is any block saved in the cache with a
// slot number equivalent to the current slot. If there is then the block is
// sent to the incoming block channel and deleted from the cache.
func (c *ChainService) sendAndDeleteCachedBlocks(currentSlot uint64) {
	if block, ok := c.unProcessedBlocks[currentSlot+1]; ok {
		if err := c.isBlockReadyForProcessing(block); err == nil {
			c.incomingBlockChan <- block
			delete(c.unProcessedBlocks, currentSlot)
		}
	}
}
