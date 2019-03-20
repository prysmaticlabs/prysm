package blockchain

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// BlockProcessor interface defines the methods in the blockchain service which
// handle new block operations.
type BlockProcessor interface {
	CanonicalBlockFeed() *event.Feed
	ReceiveBlock(ctx context.Context, block *pb.BeaconBlock) (*pb.BeaconState, error)
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
func (c *ChainService) ReceiveBlock(ctx context.Context, block *pb.BeaconBlock) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveBlock")
	defer span.End()
	beaconState, err := c.beaconDB.State(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve beacon state: %v", err)
	}
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

	// if there exists a block for the slot being processed.
	if err := c.beaconDB.SaveBlock(block); err != nil {
		return nil, fmt.Errorf("failed to save block: %v", err)
	}

	// Announce the new block to the network.
	c.p2p.Broadcast(ctx, &pb.BeaconBlockAnnounce{
		Hash:       blockRoot[:],
		SlotNumber: block.Slot,
	})

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
		beaconState, err = c.runStateTransition(headRoot, nil, beaconState)
		if err != nil {
			return nil, fmt.Errorf("could not execute state transition without block %v", err)
		}
		numSkippedSlots++
	}
	if numSkippedSlots > 0 {
		log.Warnf("Processed %d skipped slots", numSkippedSlots)
	}

	beaconState, err = c.runStateTransition(headRoot, block, beaconState)
	if err != nil {
		return nil, fmt.Errorf("could not execute state transition with block %v", err)
	}

	// Forward processed block to operation pool to remove individual operation from DB.
	if c.opsPoolService.IncomingProcessedBlockFeed().Send(block) == 0 {
		log.Error("Sent processed block to no subscribers")
	}

	// Remove pending deposits from the deposit queue.
	for _, dep := range block.Body.Deposits {
		c.beaconDB.RemovePendingDeposit(ctx, dep)
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

func (c *ChainService) runStateTransition(
	headRoot [32]byte, block *pb.BeaconBlock, beaconState *pb.BeaconState,
) (*pb.BeaconState, error) {
	beaconState, err := state.ExecuteStateTransition(
		c.ctx,
		beaconState,
		block,
		headRoot,
		&state.TransitionConfig{
			VerifySignatures: true, // We activate signature verification in this state transition.
			Logging:          true, // We enable logging in this state transition call.
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not execute state transition %v", err)
	}
	log.WithField(
		"slotsSinceGenesis", beaconState.Slot-params.BeaconConfig().GenesisSlot,
	).Info("Slot transition successfully processed")

	if block != nil {
		log.WithField(
			"slotsSinceGenesis", beaconState.Slot-params.BeaconConfig().GenesisSlot,
		).Info("Block transition successfully processed")
	}

	if helpers.IsEpochEnd(beaconState.Slot) {
		// Save activated validators of this epoch to public key -> index DB.
		if err := c.saveValidatorIdx(beaconState); err != nil {
			return nil, fmt.Errorf("could not save validator index: %v", err)
		}
		// Delete exited validators of this epoch to public key -> index DB.
		if err := c.deleteValidatorIdx(beaconState); err != nil {
			return nil, fmt.Errorf("could not delete validator index: %v", err)
		}
		// Update FFG checkpoints in DB.
		if err := c.updateFFGCheckPts(beaconState); err != nil {
			return nil, fmt.Errorf("could not update FFG checkpts: %v", err)
		}
		log.WithField(
			"SlotsSinceGenesis", beaconState.Slot-params.BeaconConfig().GenesisSlot,
		).Info("Epoch transition successfully processed")
	}
	return beaconState, nil
}

func (c *ChainService) saveHistoricalState(beaconState *pb.BeaconState) error {
	return c.beaconDB.SaveHistoricalState(beaconState)
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
