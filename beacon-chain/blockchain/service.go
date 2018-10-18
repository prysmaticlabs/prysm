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
	unfinalizedBlocks              map[[32]byte]*statePair
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

// Struct used to represent an unfinalized block's state pair
// (active state, crystallized state) tuple.
type statePair struct {
	crystallizedState *types.CrystallizedState
	activeState       *types.ActiveState
	cycleTransition   bool
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
		unfinalizedBlocks:              make(map[[32]byte]*statePair),
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

	// TODO: Populate unfinalized blocks map from disk in case
	// of beacon node restarts.
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
			currentcState, err := c.beaconDB.GetCrystallizedState()
			if err != nil {
				log.Errorf("Could not get current crystallized state: %v", err)
				continue
			}
			blockcState := c.unfinalizedBlocks[h].crystallizedState

			var headUpdated bool
			newHead := currentHead
			// If both blocks have the same crystallized state root, we favor one which has
			// the higher slot.
			if currentHead.CrystallizedStateRoot() == block.CrystallizedStateRoot() {
				if block.SlotNumber() > currentHead.SlotNumber() {
					newHead = block
					headUpdated = true
				}
			} else {
				// 2a. Pick the block with the higher last_finalized_slot.
				// 2b. If same, pick the block with the higher last_justified_slot.
				if blockcState.LastFinalizedSlot() > currentcState.LastFinalizedSlot() {
					newHead = block
					headUpdated = true
				} else if blockcState.LastFinalizedSlot() == currentcState.LastFinalizedSlot() {
					if blockcState.LastJustifiedSlot() > currentcState.LastJustifiedSlot() {
						newHead = block
						headUpdated = true
					}
				}
			}

			// If no new head was found, we do not update the chain.
			if !headUpdated {
				continue
			}

			var newCState *types.CrystallizedState
			if c.unfinalizedBlocks[h].cycleTransition {
				newCState = blockcState
			}
			if err := c.beaconDB.UpdateChainHead(newHead, c.unfinalizedBlocks[h].activeState, newCState); err != nil {
				log.Errorf("Failed to update chain: %v", err)
				continue
			}
			log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Info("Chain head block and state updated")
			// We fire events that notify listeners of a new block (or crystallized state in
			// the case of a state transition). This is useful for the beacon node's gRPC
			// server to stream these events to beacon clients.
			if c.unfinalizedBlocks[h].cycleTransition {
				c.canonicalCrystallizedStateFeed.Send(blockcState)
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

		// Listen for a newly received incoming block from the sync service.
		case block := <-c.incomingBlockChan:
			blockHash, err := block.Hash()
			if err != nil {
				log.Errorf("Failed to get hash of block: %v", err)
				continue
			}

			if c.enablePOWChain && !c.doesPoWBlockExist(block) {
				log.Errorf("Proof-of-Work chain reference in block does not exist")
				continue
			}

			parent, err := c.beaconDB.GetBlock(block.ParentHash())
			if err != nil {
				log.Errorf("Could not get parent block: %v", err)
				continue
			}
			if parent == nil {
				log.Errorf("Block points to nil parent: %#x", block.ParentHash())
				continue
			}

			aState, err := c.beaconDB.GetActiveState()
			if err != nil {
				log.Errorf("Failed to get active state: %v", err)
			}
			cState, err := c.beaconDB.GetCrystallizedState()
			if err != nil {
				log.Errorf("Failed to get crystallized state: %v", err)
			}

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

			// If the block is valid, we compute its associated state tuple (active, crystallized)
			// and apply a block scoring function.
			var didCycleTransition bool
			for cState.IsCycleTransition(parent.SlotNumber()) {
				cState, err = cState.NewStateRecalculations(
					aState,
					block,
					c.enableCrossLinks,
					c.enableRewardChecking,
				)
				if err != nil {
					log.Errorf("Initialize new cycle transition failed: %v", err)
				}
				didCycleTransition = true
			}

			aState, err = aState.CalculateNewActiveState(
				block,
				cState,
				parent.SlotNumber(),
				c.enableAttestationValidity,
			)
			if err != nil {
				log.Errorf("Compute active state failed: %v", err)
			}

			if err := c.beaconDB.SaveBlock(block); err != nil {
				log.Errorf("Failed to save block: %v", err)
				continue
			}
			if err := c.beaconDB.SaveUnfinalizedBlockState(aState, cState); err != nil {
				log.Errorf("Error persisting unfinalized block's state: %v", err)
				continue
			}

			log.Infof("Finished processing received block: %#x", blockHash)

			c.unfinalizedBlocks[blockHash] = &statePair{
				crystallizedState: cState,
				activeState:       aState,
				cycleTransition:   didCycleTransition,
			}

			// Push the block to trigger the fork choice rule.
			processedBlock <- block
		}
	}
}
