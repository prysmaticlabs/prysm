package blockchain

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/stategenerator"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"go.opencensus.io/trace"
)

// ForkChoice interface defines the methods for applying fork choice rule
// operations to the blockchain.
type ForkChoice interface {
	ApplyForkChoiceRule(ctx context.Context, block *pb.BeaconBlock, computedState *pb.BeaconState) error
}

// updateFFGCheckPts checks whether the existing FFG check points saved in DB
// are not older than the ones just processed in state. If it's older, we update
// the db with the latest FFG check points, both justification and finalization.
func (c *ChainService) updateFFGCheckPts(state *pb.BeaconState) error {
	lastJustifiedSlot := helpers.StartSlot(state.JustifiedEpoch)
	savedJustifiedBlock, err := c.beaconDB.JustifiedBlock()
	if err != nil {
		return err
	}
	// If the last processed justification slot in state is greater than
	// the slot of justified block saved in DB.
	if lastJustifiedSlot > savedJustifiedBlock.Slot {
		// Retrieve the new justified block from DB using the new justified slot and save it.
		newJustifiedBlock, err := c.beaconDB.BlockBySlot(lastJustifiedSlot)
		if err != nil {
			return err
		}
		// If the new justified slot is a skip slot in db then we keep getting it's ancestors
		// until we can get a block.
		lastAvailBlkSlot := lastJustifiedSlot
		for newJustifiedBlock == nil {
			log.Debugf("Saving new justified block, no block with slot %d in db, trying slot %d",
				lastAvailBlkSlot, lastAvailBlkSlot-1)
			lastAvailBlkSlot--
			newJustifiedBlock, err = c.beaconDB.BlockBySlot(lastAvailBlkSlot)
			if err != nil {
				return err
			}
		}
		// Generate the new justified state with using new justified block and save it.
		newJustifiedState, err := stategenerator.GenerateStateFromBlock(c.ctx, c.beaconDB, lastJustifiedSlot)
		if err != nil {
			return err
		}

		if err := c.beaconDB.SaveJustifiedBlock(newJustifiedBlock); err != nil {
			return err
		}
		if err := c.beaconDB.SaveJustifiedState(newJustifiedState); err != nil {
			return err
		}
	}

	lastFinalizedSlot := helpers.StartSlot(state.FinalizedEpoch)
	savedFinalizedBlock, err := c.beaconDB.FinalizedBlock()
	// If the last processed finalized slot in state is greater than
	// the slot of finalized block saved in DB.
	if err != nil {
		return err
	}
	if lastFinalizedSlot > savedFinalizedBlock.Slot {
		// Retrieve the new finalized block from DB using the new finalized slot and save it.
		newFinalizedBlock, err := c.beaconDB.BlockBySlot(lastFinalizedSlot)
		if err != nil {
			return err
		}
		// If the new finalized slot is a skip slot in db then we keep getting it's ancestors
		// until we can get a block.
		lastAvailBlkSlot := lastFinalizedSlot
		for newFinalizedBlock == nil {
			log.Debugf("Saving new finalized block, no block with slot %d in db, trying slot %d",
				lastAvailBlkSlot, lastAvailBlkSlot-1)
			lastAvailBlkSlot--
			newFinalizedBlock, err = c.beaconDB.BlockBySlot(lastAvailBlkSlot)
			if err != nil {
				return err
			}
		}

		// Generate the new finalized state with using new finalized block and save it.
		newFinalizedState, err := stategenerator.GenerateStateFromBlock(c.ctx, c.beaconDB, lastFinalizedSlot)
		if err != nil {
			return err
		}
		if err := c.beaconDB.SaveFinalizedBlock(newFinalizedBlock); err != nil {
			return err
		}
		if err := c.beaconDB.SaveFinalizedState(newFinalizedState); err != nil {
			return err
		}
	}
	return nil
}

// ApplyForkChoiceRule determines the current beacon chain head using LMD GHOST as a block-vote
// weighted function to select a canonical head in Ethereum Serenity.
func (c *ChainService) ApplyForkChoiceRule(ctx context.Context, block *pb.BeaconBlock, computedState *pb.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ApplyForkChoiceRule")
	defer span.End()
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

	if err := c.saveHistoricalState(computedState); err != nil {
		log.Errorf("Could not save new historical state: %v", err)
	}

	return nil
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
	startSlot := block.Slot + 1
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

// VoteCount determines the number of votes on a beacon block by counting the number
// of target blocks that have such beacon block as a common ancestor.
//
// Spec pseudocode definition:
//  def get_vote_count(block: BeaconBlock) -> int:
//        return sum(
//            get_effective_balance(start_state.validator_balances[validator_index]) // FORK_CHOICE_BALANCE_INCREMENT
//            for validator_index, target in attestation_targets
//            if get_ancestor(store, target, block.slot) == block
//        )
func VoteCount(block *pb.BeaconBlock, state *pb.BeaconState, targets map[uint64]*pb.BeaconBlock, beaconDB *db.BeaconDB) (int, error) {
	balances := 0
	for validatorIndex, targetBlock := range targets {
		ancestor, err := BlockAncestor(targetBlock, block.Slot, beaconDB)
		if err != nil {
			return 0, err
		}
		// This covers the following case, we start at B5, and want to process B6 and B7
		// B6 can be processed, B7 can not be processed because it's pointed to the
		// block older than current block 5.
		// B4 - B5 - B6
		//   \ - - - - - B7
		if ancestor == nil {
			continue
		}
		ancestorRoot, err := hashutil.HashBeaconBlock(ancestor)
		if err != nil {
			return 0, err
		}
		blockRoot, err := hashutil.HashBeaconBlock(block)
		if err != nil {
			return 0, err
		}
		if blockRoot == ancestorRoot {
			balances += int(helpers.EffectiveBalance(state, validatorIndex))
		}
	}
	return balances, nil
}

// BlockAncestor obtains the ancestor at of a block at a certain slot.
//
// Spec pseudocode definition:
//  def get_ancestor(store: Store, block: BeaconBlock, slot: Slot) -> BeaconBlock:
//    """
//    Get the ancestor of ``block`` with slot number ``slot``; return ``None`` if not found.
//    """
//    if block.slot == slot:
//        return block
//    elif block.slot < slot:
//        return None
//    else:
//        return get_ancestor(store, store.get_parent(block), slot)
func BlockAncestor(block *pb.BeaconBlock, slot uint64, beaconDB *db.BeaconDB) (*pb.BeaconBlock, error) {
	if block.Slot == slot {
		return block, nil
	}
	if block.Slot < slot {
		return nil, nil
	}
	parentHash := bytesutil.ToBytes32(block.ParentRootHash32)
	parent, err := beaconDB.Block(parentHash)
	if err != nil {
		return nil, fmt.Errorf("could not get parent block: %v", err)
	}
	if parent == nil {
		return nil, fmt.Errorf("parent block does not exist: %v", err)
	}
	return BlockAncestor(parent, slot, beaconDB)
}
