package blockchain

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

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
