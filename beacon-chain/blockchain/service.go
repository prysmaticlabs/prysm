// Package blockchain defines the life-cycle and status of the beacon chain.
package blockchain

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
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
	currentSlot                    uint64
	incomingBlockFeed              *event.Feed
	incomingBlockChan              chan *types.Block
	incomingAttestationFeed        *event.Feed
	incomingAttestationChan        chan *types.Attestation
	processedAttestationFeed       *event.Feed
	canonicalBlockFeed             *event.Feed
	canonicalCrystallizedStateFeed *event.Feed
	blocksPendingProcessing        [][32]byte
	lock                           sync.Mutex
}

// Config options for the service.
type Config struct {
	BeaconBlockBuf         int
	IncomingBlockBuf       int
	Chain                  *BeaconChain
	Web3Service            *powchain.Web3Service
	BeaconDB               ethdb.Database
	IncomingAttestationBuf int
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
		incomingAttestationChan:        make(chan *types.Attestation, cfg.IncomingAttestationBuf),
		incomingAttestationFeed:        new(event.Feed),
		processedAttestationFeed:       new(event.Feed),
		canonicalBlockFeed:             new(event.Feed),
		canonicalCrystallizedStateFeed: new(event.Feed),
		blocksPendingProcessing:        [][32]byte{},
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
	// TODO(#511): This is faulty, the ticker should start from a very
	// precise timestamp instead of rounding down to begin from a
	// certain slot. We need to ensure validators and the beacon chain
	// are properly synced at the correct timestamps for beginning
	// slot intervals.
	c.currentSlot = uint64(math.Floor(secondsSinceGenesis / params.SlotDuration))

	go c.updateHead(time.NewTicker(time.Second * params.SlotDuration).C)
	go c.blockProcessing()
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

// IncomingBlockFeed returns a feed that any service can send incoming p2p blocks into.
// The chain service will subscribe to this feed in order to process incoming blocks.
func (c *ChainService) IncomingBlockFeed() *event.Feed {
	return c.incomingBlockFeed
}

// IncomingAttestationFeed returns a feed that any service can send incoming p2p attestations into.
// The chain service will subscribe to this feed in order to relay incoming attestations.
func (c *ChainService) IncomingAttestationFeed() *event.Feed {
	return c.incomingAttestationFeed
}

// ProcessedAttestationFeed returns a feed that will be used to stream attestations that have been
// processed by the beacon node to its rpc clients.
func (c *ChainService) ProcessedAttestationFeed() *event.Feed {
	return c.processedAttestationFeed
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
func (c *ChainService) ContainsBlock(h [32]byte) (bool, error) {
	return c.chain.hasBlock(h)
}

// BlockSlotNumberByHash returns the slot number of a block.
func (c *ChainService) BlockSlotNumberByHash(h [32]byte) (uint64, error) {
	block, err := c.chain.getBlock(h)
	if err != nil {
		return 0, fmt.Errorf("could not get block from DB: %v", err)
	}
	return block.SlotNumber(), nil
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
func (c *ChainService) CheckForCanonicalBlockBySlot(slotNumber uint64) (bool, error) {
	return c.chain.hasCanonicalBlockForSlot(slotNumber)
}

// CanonicalBlockBySlotNumber retrieves the canonical block for that slot which
// has been saved in the db.
func (c *ChainService) CanonicalBlockBySlotNumber(slotNumber uint64) (*types.Block, error) {
	return c.chain.canonicalBlockForSlot(slotNumber)
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
func (c *ChainService) updateHead(slotInterval <-chan time.Time) {
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-slotInterval:
			c.currentSlot++

			log.WithField("slotNumber", c.currentSlot).Info("New beacon slot")

			// First, we check if there were any blocks processed in the previous slot.
			// If there is, we fetch the first one from the DB.
			if len(c.blocksPendingProcessing) == 0 {
				continue
			}

			// Naive fork choice rule: we pick the first block we processed for the previous slot
			// as canonical.
			block, err := c.chain.getBlock(c.blocksPendingProcessing[0])
			if err != nil {
				log.Errorf("Could not get block: %v", err)
				continue
			}

			h, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash incoming block: %v", err)
				continue
			}

			log.Info("Applying fork choice rule")
			aState := c.chain.ActiveState()
			cState := c.chain.CrystallizedState()
			isTransition := cState.IsCycleTransition(c.currentSlot - 1)

			if isTransition {
				cState, err = cState.NewStateRecalculations(aState, block)
				if err != nil {
					log.Errorf("Initialize new cycle transition failed: %v", err)
					continue
				}
			}

			parentBlock, err := c.chain.getBlock(block.ParentHash())
			if err != nil {
				log.Errorf("Failed to get parent of block 0x%x", h)
				continue
			}
			aState, err = aState.CalculateNewActiveState(block, cState, parentBlock.SlotNumber())
			if err != nil {
				log.Errorf("Compute active state failed: %v", err)
				continue
			}

			if err := c.chain.SetActiveState(aState); err != nil {
				log.Errorf("Write active state to disk failed: %v", err)
				continue
			}

			if err := c.chain.SetCrystallizedState(cState); err != nil {
				log.Errorf("Write crystallized state to disk failed: %v", err)
				continue
			}

			// Save canonical block hash with slot number to DB.
			if err := c.chain.saveCanonicalSlotNumber(block.SlotNumber(), h); err != nil {
				log.Errorf("Unable to save slot number to db: %v", err)
				continue
			}

			// Save canonical block to DB.
			if err := c.chain.saveCanonicalBlock(block); err != nil {
				log.Errorf("Unable to save block to db: %v", err)
				continue
			}

			log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Info("Canonical block determined")

			// We fire events that notify listeners of a new block (or crystallized state in
			// the case of a state transition). This is useful for the beacon node's gRPC
			// server to stream these events to beacon clients.
			if isTransition {
				c.canonicalCrystallizedStateFeed.Send(cState)
			}
			c.canonicalBlockFeed.Send(block)

			// Clear the blocks pending processing, mutex lock for thread safety
			// in updating this slice.
			c.lock.Lock()
			c.blocksPendingProcessing = [][32]byte{}
			c.lock.Unlock()
		}
	}
}

func (c *ChainService) blockProcessing() {
	sub := c.incomingBlockFeed.Subscribe(c.incomingBlockChan)
	subAttestation := c.incomingAttestationFeed.Subscribe(c.incomingAttestationChan)
	defer subAttestation.Unsubscribe()
	defer sub.Unsubscribe()
	for {
		select {
		case <-c.ctx.Done():
			log.Debug("Chain service context closed, exiting goroutine")
			return
		// Listen for a newly received incoming attestation from the sync service.
		case attestation := <-c.incomingAttestationChan:
			h, err := attestation.Hash()
			if err != nil {
				log.Debugf("Could not hash incoming attestation: %v", err)
			}
			if err := c.chain.saveAttestation(attestation); err != nil {
				log.Errorf("Could not save attestation: %v", err)
				continue
			}

			c.processedAttestationFeed.Send(attestation.Proto)
			log.Info("Relaying attestation 0x%v to proposers through grpc", h)

		// Listen for a newly received incoming block from the sync service.
		case block := <-c.incomingBlockChan:
			blockHash, err := block.Hash()
			if err != nil {
				log.Errorf("Failed to get hash of block: %v", err)
				continue
			}

			if !c.doesPoWBlockExist(block) {
				log.Debugf("Proof-of-Work chain reference in block does not exist")
				continue
			}

			// Process block as a validator if beacon node has registered, else process block as an observer.
			parentExists, err := c.chain.hasBlock(block.ParentHash())
			if err != nil {
				log.Errorf("Could not check existence of parent: %v", err)
				continue
			}
			if !parentExists {
				log.Debugf("Block points to nil parent", err)
				continue
			}

			aState := c.chain.ActiveState()
			cState := c.chain.CrystallizedState()
			if valid := block.IsValid(aState, cState, c.currentSlot-1); !valid {
				log.Debugf("Block failed validity conditions: %v", err)
				continue
			}

			if err := c.chain.saveBlockAndAttestations(block); err != nil {
				log.Errorf("Failed to save block: %v", err)
				continue
			}

			log.Infof("Finished processing received block: %x", blockHash)

			// We push the hash of the block we just stored to a pending processing
			// slice the fork choice rule will utilize.
			c.lock.Lock()
			c.blocksPendingProcessing = append(c.blocksPendingProcessing, blockHash)
			c.lock.Unlock()
			log.Info("Finished processing received block")
		}
	}
}
