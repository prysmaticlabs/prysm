// Package blockchain defines the life-cycle and status of the beacon chain.
package blockchain

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/gogo/protobuf/proto"
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
	currentSlot                    uint64
	incomingBlockFeed              *event.Feed
	incomingBlockChan              chan *types.Block
	canonicalBlockFeed             *event.Feed
	canonicalCrystallizedStateFeed *event.Feed
	blocksPendingProcessing        [][]byte
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
	return &ChainService{
		ctx:                            ctx,
		chain:                          cfg.Chain,
		cancel:                         cancel,
		beaconDB:                       cfg.BeaconDB,
		web3Service:                    cfg.Web3Service,
		incomingBlockChan:              make(chan *types.Block, cfg.IncomingBlockBuf),
		incomingBlockFeed:              new(event.Feed),
		canonicalBlockFeed:             new(event.Feed),
		canonicalCrystallizedStateFeed: new(event.Feed),
		blocksPendingProcessing:        [][]byte{},
	}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	// TODO(#474): Fetch the slot: (block, state) DAGs from persistent storage
	// to truly continue across sessions.
	log.Infof("Starting service")
	genesisTimestamp := time.Unix(0, 0)
	secondsSinceGenesis := time.Since(genesisTimestamp).Seconds()
	// Set the current slot.
	// TODO: This is faulty, the ticker should start from a very
	// precise timestamp instead of rounding down to begin from a
	// certain slot. We need to ensure validators and the beacon chain
	// are properly synced at the correct timestamps for begininng
	// slot intervals.
	c.currentSlot = uint64(math.Floor(secondsSinceGenesis / 8.0))

	go c.updateHead(time.NewTicker(time.Second*8).C, c.ctx.Done())
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

// doesPoWBlockExist checks if the referenced PoW block exists.
func (c *ChainService) doesPoWBlockExist(block *types.Block) bool {
	powBlock, err := c.web3Service.Client().BlockByHash(context.Background(), block.PowChainRef())
	if err != nil {
		log.Debugf("fetching PoW block corresponding to mainchain reference failed: %v", err)
		return false
	}

	return powBlock != nil
}

func (c *ChainService) updateHead(slotInterval <-chan time.Time, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case <-slotInterval:
			c.currentSlot++

			// First, we check if there were any blocks processed in the previous slot.
			// If there is, we fetch the first one from the DB.
			if len(c.blocksPendingProcessing) == 0 {
				continue
			}

			// Naive fork choice rule: we pick the first block we processed for the previous slot
			// as canonical.
			// TODO: Use a better lookup key or abstract into a GetBlock method from DB.
			hasBlock, err := c.beaconDB.Has(c.blocksPendingProcessing[0])
			if err != nil {
				continue
			}
			if !hasBlock {
				continue
			}
			encoded, err := c.beaconDB.Get(c.blocksPendingProcessing[0])
			if err != nil {
				continue
			}
			blockProto := &pb.BeaconBlock{}
			if err := proto.Unmarshal(encoded, blockProto); err != nil {
				continue
			}
			block := types.NewBlock(blockProto)
			h, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash incoming block: %v", err)
			}

			log.WithField("slotNumber", block.SlotNumber()).Info("Applying fork choice rule")

			aState := c.chain.ActiveState()
			cState := c.chain.CrystallizedState()
			isTransition := cState.IsCycleTransition(c.currentSlot - 1)

			if isTransition {
				cState, err = cState.CalculateNewCrystallizedState(aState, block.SlotNumber())
				if err != nil {
					log.Errorf("Initialize new cycle transition failed: %v", err)
				}
			}

			parentBlock, err := c.chain.getBlock(block.ParentHash())
			if err != nil {
				log.Errorf("Failed to get parent slot of block 0x%x", h)
				continue
			}
			aState, err = aState.CalculateNewActiveState(block, cState, parentBlock.SlotNumber())
			if err != nil {
				log.Errorf("Compute active state failed: %v", err)
				continue
			}

			if err := c.chain.SetActiveState(aState); err != nil {
				log.Errorf("Write active state to disk failed: %v", err)
			}

			if err := c.chain.SetCrystallizedState(cState); err != nil {
				log.Errorf("Write crystallized state to disk failed: %v", err)
			}

			vals, err := casper.ShuffleValidatorsToCommittees(
				cState.DynastySeed(),
				cState.Validators(),
				cState.CurrentDynasty(),
				cState.CrosslinkingStartShard(),
			)

			if err != nil {
				log.Errorf("Unable to get validators by slot and by shard: %v", err)
				return
			}
			log.Debugf("Received %d validators by slot", len(vals))

			// Save canonical block hash with slot number to DB.
			if err := c.chain.saveCanonicalSlotNumber(block.SlotNumber(), h); err != nil {
				log.Errorf("Unable to save slot number to db: %v", err)
			}

			// Save canonical block to DB.
			if err := c.chain.saveCanonicalBlock(block); err != nil {
				log.Errorf("Unable to save block to db: %v", err)
			}

			log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Info("Canonical block determined")

			// We fire events that notify listeners of a new block (or crystallized state in
			// the case of a state transition). This is useful for the beacon node's gRPC
			// server to stream these events to beacon clients.
			if isTransition {
				c.canonicalCrystallizedStateFeed.Send(cState)
			}
			c.canonicalBlockFeed.Send(block)

			// Clear the blocks pending processing.
			c.blocksPendingProcessing = [][]byte{}
		}
	}
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
			aState := c.chain.ActiveState()
			cState := c.chain.CrystallizedState()
			blockHash, err := block.Hash()
			if err != nil {
				log.Errorf("Failed to get hash of block: %v", err)
				continue
			}

			// Process block as a validator if beacon node has registered, else process block as an observer.
			parentExists, err := c.chain.hasBlock(block.ParentHash())
			if err != nil {
				log.Errorf("Could not check existence of parent: %v", err)
				continue
			}

			if !parentExists || !c.doesPoWBlockExist(block) || !block.IsValid(aState, cState) {
				continue
			}

			if err := c.chain.saveBlockAndAttestations(block); err != nil {
				log.Errorf("Failed to save block: %v", err)
				continue
			}

			log.Infof("Finished processing received block: %x", blockHash)

			// We push the hash of the block we just stored to a pending processing slice the fork choice rule
			// will utilize.
			c.blocksPendingProcessing = append(c.blocksPendingProcessing, blockHash[:])
			log.Info("Finished processing received block")
		}
	}
}
