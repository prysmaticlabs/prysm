// Package blockchain defines the life-cycle and status of the beacon chain
// as well as the Ethereum Serenity beacon chain fork-choice rule based on
// Casper Proof of Stake finality.
package blockchain

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/attestation"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	handler "github.com/prysmaticlabs/prysm/shared/messagehandler"
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
	attsService          *attestation.Service
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
	AttsService      *attestation.Service
	BeaconDB         *db.BeaconDB
	OpsPoolService   operationService
	DevMode          bool
	EnablePOWChain   bool
}

// attestationTarget consists of validator index and block, it's
// used to represent which validator index has voted which block.
type attestationTarget struct {
	validatorIndex uint64
	block          *pb.BeaconBlock
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
		attsService:          cfg.AttsService,
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
	beaconState, err := c.beaconDB.State(c.ctx)
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
			depositRoot := c.web3Service.DepositRoot()
			latestBlockHash := c.web3Service.LatestBlockHash()
			eth1Data := &pb.Eth1Data{
				DepositRootHash32: depositRoot[:],
				BlockHash32:       latestBlockHash[:],
			}
			beaconState, err := c.initializeBeaconChain(genesisTime, initialDeposits, eth1Data)
			if err != nil {
				log.Fatalf("Could not initialize beacon chain: %v", err)
			}
			c.stateInitializedFeed.Send(genesisTime)
			c.canonicalStateFeed.Send(beaconState)
			go c.blockProcessing()
			subChainStart.Unsubscribe()
		}()
	}
}

// initializes the state and genesis block of the beacon chain to persistent storage
// based on a genesis timestamp value obtained from the ChainStart event emitted
// by the ETH1.0 Deposit Contract and the POWChain service of the node.
func (c *ChainService) initializeBeaconChain(genesisTime time.Time, deposits []*pb.Deposit,
	eth1data *pb.Eth1Data) (*pb.BeaconState, error) {
	log.Info("ChainStart time reached, starting the beacon chain!")
	c.genesisTime = genesisTime
	unixTime := uint64(genesisTime.Unix())
	if err := c.beaconDB.InitializeState(unixTime, deposits, eth1data); err != nil {
		return nil, fmt.Errorf("could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := c.beaconDB.State(c.ctx)
	if err != nil {
		return nil, fmt.Errorf("could not attempt fetch beacon state: %v", err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		return nil, fmt.Errorf("could not hash beacon state: %v", err)
	}
	genBlock := b.NewGenesisBlock(stateRoot[:])
	if err := c.beaconDB.SaveBlock(genBlock); err != nil {
		return nil, fmt.Errorf("could not save genesis block to disk: %v", err)
	}
	if err := c.beaconDB.UpdateChainHead(genBlock, beaconState); err != nil {
		return nil, fmt.Errorf("could not set chain head, %v", err)
	}
	return beaconState, nil
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
			handler.SafelyHandleMessage(c.ctx, c.processBlock, block)
		}
	}
}

func (c *ChainService) processBlock(message proto.Message) {
	block := message.(*pb.BeaconBlock)
	beaconState, err := c.beaconDB.State(c.ctx)
	if err != nil {
		log.Errorf("Unable to retrieve beacon state %v", err)
		return
	}
	if block.Slot > beaconState.Slot {
		computedState, err := c.ReceiveBlock(block, beaconState)
		if err != nil {
			log.Errorf("Could not process received block: %v", err)
			return
		}
		if err := c.ApplyForkChoiceRule(block, computedState); err != nil {
			log.Errorf("Could not update chain head: %v", err)
			return
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
	if helpers.IsEpochStart(block.Slot) {
		if c.canonicalStateFeed.Send(computedState) == 0 {
			log.Error("Sent canonical state to no subscribers")
		}
	}
	if c.canonicalBlockFeed.Send(&pb.BeaconBlockAnnounce{
		Hash:       h[:],
		SlotNumber: block.Slot,
	}) == 0 {
		log.Error("Sent canonical block to no subscribers")
	}
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
	numSkippedSlots := 0
	for beaconState.Slot < block.Slot-1 {
		beaconState, err = state.ExecuteStateTransition(
			c.ctx,
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
		numSkippedSlots++
	}
	if numSkippedSlots > 0 {
		log.Warnf("Processed %d skipped slots", numSkippedSlots)
	}

	beaconState, err = state.ExecuteStateTransition(
		c.ctx,
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
	if helpers.IsEpochEnd(beaconState.Slot) {
		// Save activated validators of this epoch to public key -> index DB.
		if err := c.saveValidatorIdx(beaconState); err != nil {
			return nil, fmt.Errorf("could not save validator index: %v", err)
		}
		// Delete exited validators of this epoch to public key -> index DB.
		if err := c.deleteValidatorIdx(beaconState); err != nil {
			return nil, fmt.Errorf("could not delete validator index: %v", err)
		}
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

	// Check state root
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		return nil, fmt.Errorf("could not hash beacon state: %v", err)
	}
	if !bytes.Equal(block.StateRootHash32, stateRoot[:]) {
		return nil, fmt.Errorf("beacon state root is not equal to block state root: %#x != %#x", stateRoot, block.StateRootHash32)
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

// saveValidatorIdx saves the validators public key to index mapping in DB, these
// validators were activated from current epoch. After it saves, current epoch key
// is deleted from ActivatedValidators mapping.
func (c *ChainService) saveValidatorIdx(state *pb.BeaconState) error {
	for _, idx := range validators.ActivatedValidators[helpers.CurrentEpoch(state)] {
		pubKey := state.ValidatorRegistry[idx].Pubkey
		if err := c.beaconDB.SaveValidatorIndex(pubKey, int(idx)); err != nil {
			return fmt.Errorf("could not save validator index: %v", err)
		}
	}
	delete(validators.ActivatedValidators, helpers.CurrentEpoch(state))
	return nil
}

// deleteValidatorIdx deletes the validators public key to index mapping in DB, the
// validators were exited from current epoch. After it deletes, current epoch key
// is deleted from ExitedValidators mapping.
func (c *ChainService) deleteValidatorIdx(state *pb.BeaconState) error {
	for _, idx := range validators.ExitedValidators[helpers.CurrentEpoch(state)] {
		pubKey := state.ValidatorRegistry[idx].Pubkey
		if err := c.beaconDB.DeleteValidatorIndex(pubKey); err != nil {
			return fmt.Errorf("could not delete validator index: %v", err)
		}
	}
	delete(validators.ExitedValidators, helpers.CurrentEpoch(state))
	return nil
}

// attestationTargets retrieves the list of attestation targets since last finalized epoch,
// each attestation target consists of validator index and its attestation target (i.e. the block
// which the validator attested to)
func (c *ChainService) attestationTargets(state *pb.BeaconState) ([]*attestationTarget, error) {
	indices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, state.FinalizedEpoch)
	attestationTargets := make([]*attestationTarget, len(indices))
	for i, index := range indices {
		block, err := c.attsService.LatestAttestationTarget(c.ctx, index)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve attestation target: %v", err)
		}
		attestationTargets[i] = &attestationTarget{
			validatorIndex: index,
			block:          block,
		}
	}
	return attestationTargets, nil
}

// blockChildren returns the child blocks of the given block.
// ex:
//       /- C - E
// A - B - D - F
//       \- G
// Input: B. Output: [C, D, G]
//
// Spec pseudocode definition:
//	get_children(store: Store, block: BeaconBlock) -> List[BeaconBlock]
//		returns the child blocks of the given block.
func (c *ChainService) blockChildren(block *pb.BeaconBlock, state *pb.BeaconState) ([]*pb.BeaconBlock, error) {
	var children []*pb.BeaconBlock

	currentRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash incoming block: %v", err)
	}
	startSlot := block.Slot
	currentSlot := state.Slot
	for i := startSlot; i <= currentSlot; i++ {
		block, err := c.beaconDB.BlockBySlot(i)
		if err != nil {
			return nil, fmt.Errorf("could not get block by slot: %v", err)
		}
		// Continue if there's a skip block.
		if block == nil {
			continue
		}

		parentRoot := bytesutil.ToBytes32(block.ParentRootHash32)
		if currentRoot == parentRoot {
			children = append(children, block)
		}
	}
	return children, nil
}

// lmdGhost applies the Latest Message Driven, Greediest Heaviest Observed Sub-Tree
// fork-choice rule defined in the Ethereum Serenity specification for the beacon chain.
//
// Spec pseudocode definition:
//	def lmd_ghost(store: Store, start_state: BeaconState, start_block: BeaconBlock) -> BeaconBlock:
//    """
//    Execute the LMD-GHOST algorithm to find the head ``BeaconBlock``.
//    """
//    validators = start_state.validator_registry
//    active_validator_indices = get_active_validator_indices(validators, slot_to_epoch(start_state.slot))
//    attestation_targets = [
//        (validator_index, get_latest_attestation_target(store, validator_index))
//        for validator_index in active_validator_indices
//    ]
//
//    def get_vote_count(block: BeaconBlock) -> int:
//        return sum(
//            get_effective_balance(start_state.validator_balances[validator_index]) // FORK_CHOICE_BALANCE_INCREMENT
//            for validator_index, target in attestation_targets
//            if get_ancestor(store, target, block.slot) == block
//        )
//
//    head = start_block
//    while 1:
//        children = get_children(store, head)
//        if len(children) == 0:
//            return head
//        head = max(children, key=get_vote_count)
func (c *ChainService) lmdGhost(
	block *pb.BeaconBlock,
	state *pb.BeaconState,
	voteTargets map[uint64]*pb.BeaconBlock,
) (*pb.BeaconBlock, error) {
	head := block
	for {
		children, err := c.blockChildren(head, state)
		if err != nil {
			return nil, fmt.Errorf("could not fetch block children: %v", err)
		}
		if len(children) == 0 {
			return head, nil
		}
		maxChild := children[0]

		maxChildVotes, err := VoteCount(maxChild, state, voteTargets, c.beaconDB)
		if err != nil {
			return nil, fmt.Errorf("unable to determine vote count for block: %v", err)
		}
		for i := 0; i < len(children); i++ {
			candidateChildVotes, err := VoteCount(children[i], state, voteTargets, c.beaconDB)
			if err != nil {
				return nil, fmt.Errorf("unable to determine vote count for block: %v", err)
			}
			if candidateChildVotes > maxChildVotes {
				maxChild = children[i]
			}
		}
		head = maxChild
	}
}
