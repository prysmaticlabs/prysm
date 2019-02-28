// Package blockchain defines the life-cycle and status of the beacon chain
// as well as the Ethereum Serenity beacon chain fork-choice rule based on
// Casper Proof of Stake finality.
package blockchain

import (
	"context"
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
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "blockchain")

type operationService interface {
	IncomingProcessedBlockFeed() *event.Feed
}

// ChainService represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type ChainService struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	beaconDB             *db.BeaconDB
	web3Service          *powchain.Web3Service
	opsPoolService       operationService
	incomingBlockFeed    *event.Feed
	incomingBlockChan    chan *pb.BeaconBlock
	chainStartChan       chan time.Time
	canonicalBlockFeed   *event.Feed
	canonicalStateFeed   *event.Feed
	genesisTime          time.Time
	enablePOWChain       bool
	stateInitializedFeed *event.Feed
}

// Config options for the service.
type Config struct {
	BeaconBlockBuf   int
	IncomingBlockBuf int
	Web3Service      *powchain.Web3Service
	BeaconDB         *db.BeaconDB
	OpsPoolService   operationService
	DevMode          bool
	EnablePOWChain   bool
}

// NewChainService instantiates a new service instance that will
// be registered into a running beacon node.
func NewChainService(ctx context.Context, cfg *Config) (*ChainService, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &ChainService{
		ctx:                  ctx,
		cancel:               cancel,
		beaconDB:             cfg.BeaconDB,
		web3Service:          cfg.Web3Service,
		opsPoolService:       cfg.OpsPoolService,
		incomingBlockChan:    make(chan *pb.BeaconBlock, cfg.IncomingBlockBuf),
		chainStartChan:       make(chan time.Time),
		incomingBlockFeed:    new(event.Feed),
		canonicalBlockFeed:   new(event.Feed),
		canonicalStateFeed:   new(event.Feed),
		stateInitializedFeed: new(event.Feed),
		enablePOWChain:       cfg.EnablePOWChain,
	}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	beaconState, err := c.beaconDB.State(context.TODO())
	if err != nil {
		log.Fatalf("Could not fetch beacon state: %v", err)
	}
	// If the chain has already been initialized, simply start the block processing routine.
	if beaconState != nil {
		log.Info("Beacon chain data already exists, starting service")
		c.genesisTime = time.Unix(int64(beaconState.GenesisTime), 0)
		go c.blockProcessing()
	} else {
		log.Info("Waiting for ChainStart log from the Validator Deposit Contract to start the beacon chain...")
		if c.web3Service == nil {
			log.Fatal("Not configured web3Service for POW chain")
			return // return need for TestStartUninitializedChainWithoutConfigPOWChain.
		}
		subChainStart := c.web3Service.ChainStartFeed().Subscribe(c.chainStartChan)
		go func() {
			genesisTime := <-c.chainStartChan
			initialDeposits := c.web3Service.ChainStartDeposits()
			if err := c.initializeBeaconChain(genesisTime, initialDeposits); err != nil {
				log.Fatalf("Could not initialize beacon chain: %v", err)
			}
			c.stateInitializedFeed.Send(genesisTime)
			go c.blockProcessing()
			subChainStart.Unsubscribe()
		}()
	}
}

// initializes the state and genesis block of the beacon chain to persistent storage
// based on a genesis timestamp value obtained from the ChainStart event emitted
// by the ETH1.0 Deposit Contract and the POWChain service of the node.
func (c *ChainService) initializeBeaconChain(genesisTime time.Time, deposits []*pb.Deposit) error {
	log.Info("ChainStart time reached, starting the beacon chain!")
	c.genesisTime = genesisTime
	unixTime := uint64(genesisTime.Unix())
	if err := c.beaconDB.InitializeState(unixTime, deposits); err != nil {
		return fmt.Errorf("could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := c.beaconDB.State(context.TODO())
	if err != nil {
		return fmt.Errorf("could not attempt fetch beacon state: %v", err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		return fmt.Errorf("could not hash beacon state: %v", err)
	}
	genBlock := b.NewGenesisBlock(stateRoot[:])
	if err := c.beaconDB.SaveBlock(genBlock); err != nil {
		return fmt.Errorf("could not save genesis block to disk: %v", err)
	}
	if err := c.beaconDB.UpdateChainHead(genBlock, beaconState); err != nil {
		return fmt.Errorf("could not set chain head, %v", err)
	}
	return nil
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

// StateInitializedFeed returns a feed that is written to
// when the beacon state is first initialized.
func (c *ChainService) StateInitializedFeed() *event.Feed {
	return c.stateInitializedFeed
}

// ChainHeadRoot returns the hash root of the last beacon block processed by the
// block chain service.
func (c *ChainService) ChainHeadRoot() ([32]byte, error) {
	head, err := c.beaconDB.ChainHead()
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not retrieve chain head: %v", err)
	}

	root, err := hashutil.HashBeaconBlock(head)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not tree hash parent block: %v", err)
	}
	return root, nil
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

// blockProcessing subscribes to incoming blocks, processes them if possible, and then applies
// the fork-choice rule to update the beacon chain's head.
func (c *ChainService) blockProcessing() {
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
			beaconState, err := c.beaconDB.State(context.TODO())
			if err != nil {
				log.Errorf("Unable to retrieve beacon state %v", err)
				continue
			}

			if block.Slot > beaconState.Slot {
				computedState, err := c.ReceiveBlock(block, beaconState)
				if err != nil {
					log.Errorf("Could not process received block: %v", err)
					continue
				}
				if err := c.ApplyForkChoiceRule(block, computedState); err != nil {
					log.Errorf("Could not update chain head: %v", err)
					continue
				}
			}
		}
	}
}

// ApplyForkChoiceRule determines the current beacon chain head using LMD GHOST as a block-vote
// weighted function to select a canonical head in Ethereum Serenity.
func (c *ChainService) ApplyForkChoiceRule(block *pb.BeaconBlock, computedState *pb.BeaconState) error {
	h, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return fmt.Errorf("could not tree hash incoming block: %v", err)
	}
	// TODO(#1307): Use LMD GHOST as the fork-choice rule for Ethereum Serenity.
	// TODO(#674): Handle chain reorgs.
	if err := c.beaconDB.UpdateChainHead(block, computedState); err != nil {
		return fmt.Errorf("failed to update chain: %v", err)
	}
	log.WithField("blockRoot", fmt.Sprintf("0x%x", h)).Info("Chain head block and state updated")
	// We fire events that notify listeners of a new block in
	// the case of a state transition. This is useful for the beacon node's gRPC
	// server to stream these events to beacon clients.
	// When the transition is a cycle transition, we stream the state containing the new validator
	// assignments to clients.
	if block.Slot%params.BeaconConfig().SlotsPerEpoch == 0 {
		c.canonicalStateFeed.Send(computedState)
	}
	c.canonicalBlockFeed.Send(block)
	return nil
}

// ReceiveBlock is a function that defines the operations that are preformed on
// any block that is received from p2p layer or rpc. It checks the block to see
// if it passes the pre-processing conditions, if it does then the per slot
// state transition function is carried out on the block.
// spec:
//  def process_block(block):
//      if not block_pre_processing_conditions(block):
//          return nil, error
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
//			return state, error
//		else:
//			return nil, error  # or throw or whatever
//
func (c *ChainService) ReceiveBlock(block *pb.BeaconBlock, beaconState *pb.BeaconState) (*pb.BeaconState, error) {
	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash incoming block: %v", err)
	}

	if block.Slot == params.BeaconConfig().GenesisSlot {
		return nil, fmt.Errorf("cannot process a genesis block: received block with slot %d",
			block.Slot-params.BeaconConfig().GenesisSlot)
	}

	// Save blocks with higher slot numbers in cache.
	if err := c.isBlockReadyForProcessing(block, beaconState); err != nil {
		return nil, fmt.Errorf("block with root %#x is not ready for processing: %v", blockRoot, err)
	}

	// Retrieve the last processed beacon block's hash root.
	headRoot, err := c.ChainHeadRoot()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve chain head root: %v", err)
	}

	log.WithField("slotNumber", block.Slot-params.BeaconConfig().GenesisSlot).Info(
		"Executing state transition")

	// Check for skipped slots.
	for beaconState.Slot < block.Slot-1 {
		beaconState, err = state.ExecuteStateTransition(
			beaconState,
			nil,
			headRoot,
			true, /* sig verify */
		)
		if err != nil {
			return nil, fmt.Errorf("could not execute state transition without block %v", err)
		}
		log.WithField(
			"slotsSinceGenesis", beaconState.Slot-params.BeaconConfig().GenesisSlot,
		).Info("Slot transition successfully processed")
	}

	beaconState, err = state.ExecuteStateTransition(
		beaconState,
		block,
		headRoot,
		true, /* no sig verify */
	)
	if err != nil {
		return nil, fmt.Errorf("could not execute state transition with block %v", err)
	}
	log.WithField(
		"slotsSinceGenesis", beaconState.Slot-params.BeaconConfig().GenesisSlot,
	).Info("Slot transition successfully processed")
	log.WithField(
		"slotsSinceGenesis", beaconState.Slot-params.BeaconConfig().GenesisSlot,
	).Info("Block transition successfully processed")
	if (beaconState.Slot+1)%params.BeaconConfig().SlotsPerEpoch == 0 {
		log.WithField(
			"SlotsSinceGenesis", beaconState.Slot-params.BeaconConfig().GenesisSlot,
		).Info("Epoch transition successfully processed")
	}

	// if there exists a block for the slot being processed.
	if err := c.beaconDB.SaveBlock(block); err != nil {
		return nil, fmt.Errorf("failed to save block: %v", err)
	}

	// Forward processed block to operation pool to remove individual operation from DB.
	c.opsPoolService.IncomingProcessedBlockFeed().Send(block)

	// Remove pending deposits from the deposit queue.
	for _, dep := range block.Body.Deposits {
		c.beaconDB.RemovePendingDeposit(c.ctx, dep)
	}

	log.WithField("hash", fmt.Sprintf("%#x", blockRoot)).Debug("Processed beacon block")
	return beaconState, nil
}

func (c *ChainService) isBlockReadyForProcessing(block *pb.BeaconBlock, beaconState *pb.BeaconState) error {
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
