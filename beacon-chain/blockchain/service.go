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
	ctx                              context.Context
	cancel                           context.CancelFunc
	beaconDB                         *database.DB
	chain                            *BeaconChain
	web3Service                      *powchain.Web3Service
	latestBeaconBlock                chan *types.Block
	processedBlockHashes             [][32]byte
	processedActiveStateHashes       [][32]byte
	processedCrystallizedStateHashes [][32]byte
}

// NewChainService instantiates a new service instance that will
// be registered into a running beacon node.
func NewChainService(ctx context.Context, beaconDB *database.DB, web3Service *powchain.Web3Service) (*ChainService, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &ChainService{
		ctx:                              ctx,
		cancel:                           cancel,
		beaconDB:                         beaconDB,
		web3Service:                      web3Service,
		latestBeaconBlock:                make(chan *types.Block),
		processedBlockHashes:             [][32]byte{},
		processedActiveStateHashes:       [][32]byte{},
		processedCrystallizedStateHashes: [][32]byte{},
	}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	log.Infof("Starting service")

	beaconChain, err := NewBeaconChain(c.beaconDB.DB())
	if err != nil {
		log.Errorf("Unable to setup blockchain: %v", err)
	}
	c.chain = beaconChain
	go c.updateChainState()
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

// ProcessedBlockHashes by the chain service.
func (c *ChainService) ProcessedBlockHashes() [][32]byte {
	return c.processedBlockHashes
}

// ProcessedCrystallizedStateHashes by the chain service.
func (c *ChainService) ProcessedCrystallizedStateHashes() [][32]byte {
	return c.processedCrystallizedStateHashes
}

// ProcessedActiveStateHashes by the chain service.
func (c *ChainService) ProcessedActiveStateHashes() [][32]byte {
	return c.processedActiveStateHashes
}

// ProcessBlock accepts a new block for inclusion in the chain.
func (c *ChainService) ProcessBlock(block *types.Block) error {
	h, err := block.Hash()
	if err != nil {
		return fmt.Errorf("could not hash incoming block: %v", err)
	}
	log.WithField("blockHash", fmt.Sprintf("0x%x", h)).Info("Received full block, processing validity conditions")
	canProcess, err := c.chain.CanProcessBlock(c.web3Service.Client(), block)
	if err != nil {
		return err
	}
	if canProcess {
		c.latestBeaconBlock <- block
	}
	return nil
}

// ProcessCrystallizedState accepts a new crystallized state object for inclusion in the chain.
func (c *ChainService) ProcessCrystallizedState(state *types.CrystallizedState) error {
	h, err := state.Hash()
	if err != nil {
		return fmt.Errorf("could not hash incoming block: %v", err)
	}
	log.WithField("stateHash", fmt.Sprintf("0x%x", h)).Info("Received crystallized state, processing validity conditions")

	// TODO: Implement crystallized state verifier function and apply fork choice rules

	return nil
}

// ProcessActiveState accepts a new active state object for inclusion in the chain.
func (c *ChainService) ProcessActiveState(state *types.ActiveState) error {
	h, err := state.Hash()
	if err != nil {
		return fmt.Errorf("could not hash incoming block: %v", err)
	}
	log.WithField("stateHash", fmt.Sprintf("0x%x", h)).Info("Received active state, processing validity conditions")

	// TODO: Implement active state verifier function and apply fork choice rules

	return nil
}

// ContainsBlock checks if a block for the hash exists in the chain.
// This method must be safe to call from a goroutine
func (c *ChainService) ContainsBlock(h [32]byte) bool {
	// TODO
	return false
}

// ContainsCrystallizedState checks if a crystallized state for the hash exists in the chain.
func (c *ChainService) ContainsCrystallizedState(h [32]byte) bool {
	// TODO
	return false
}

// ContainsActiveState checks if a active state for the hash exists in the chain.
func (c *ChainService) ContainsActiveState(h [32]byte) bool {
	// TODO
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

// updateChainState receives a beacon block, computes a new active state and writes it to db. Also
// it checks for if there is an epoch transition. If there is one it computes the validator rewards
// and penalties.
func (c *ChainService) updateChainState() {
	for {
		select {
		case _ = <-c.latestBeaconBlock:
			// TODO: Using latest block hash for seed, this will eventually be replaced by randao
			//activeState, err := c.chain.computeNewActiveState(c.web3Service.LatestBlockHash())
			//if err != nil {
			//	log.Errorf("Compute active state failed: %v", err)
			//}
			//
			//err = c.chain.MutateActiveState(activeState)
			//if err != nil {
			//	log.Errorf("Write active state to disk failed: %v", err)
			//}
			//
			//// Entering epoch transitions.
			//transition := c.chain.IsEpochTransition(block.SlotNumber())
			//if transition {
			//	if err := c.chain.calculateRewardsFFG(); err != nil {
			//		log.Errorf("Error computing validator rewards and penalties %v", err)
			//	}
			//}

		case <-c.ctx.Done():
			log.Debug("Chain service context closed, exiting goroutine")
			return
		}
	}
}
