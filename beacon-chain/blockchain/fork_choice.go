package blockchain

import (
	"fmt"

	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// LMDGhost applies the Latest Message Driven, Greediest Heaviest Observed Sub-Tree
// fork-choice rule defined in the Ethereum Serenity specification for the beacon chain.
//
// Spec pseudocode definition:
//    head = start_block
//    while 1:
//        children = get_children(store, head)
//        if len(children) == 0:
//            return head
//        head = max(children, key=get_vote_count)
func LMDGhost(
	block *pb.BeaconBlock,
	voteTargets map[[32]byte]*pb.BeaconBlock,
	observedBlocks []*pb.BeaconBlock,
	beaconDB *db.BeaconDB,
) (*pb.BeaconBlock, error) {
	head := block
	for {
		children, err := b.BlockChildren(head, observedBlocks)
		if err != nil {
			return nil, fmt.Errorf("could not fetch block children: %v", err)
		}
		if len(children) == 0 {
			return head, nil
		}
		maxChild := children[0]
		maxChildVotes, err := VoteCount(maxChild, voteTargets, beaconDB)
		if err != nil {
			return nil, fmt.Errorf("unable to determine vote count for block: %v", err)
		}
		for i := 0; i < len(children); i++ {
			candidateChildVotes, err := VoteCount(children[i], voteTargets, beaconDB)
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

// VoteCount determines the number of votes on a beacon block by counting the number
// of target blocks that have such beacon block as a common ancestor.
//
// Spec pseudocode definition:
//	def get_vote_count(block: BeaconBlock) -> int:
//		return len([target for target in attestation_targets
//			if get_ancestor(store, target, block.slot) == block])
func VoteCount(block *pb.BeaconBlock, targets map[[32]byte]*pb.BeaconBlock, beaconDB *db.BeaconDB) (int, error) {
	votes := 0
	for k := range targets {
		ancestor, err := BlockAncestor(targets[k], block.Slot, beaconDB)
		if err != nil {
			return 0, err
		}
		ancestorHash, err := hashutil.HashBeaconBlock(ancestor)
		if err != nil {
			return 0, err
		}
		blockHash, err := hashutil.HashBeaconBlock(block)
		if err != nil {
			return 0, err
		}
		if blockHash == ancestorHash {
			votes++
		}
	}
	return votes, nil
}

// BlockAncestor obtains the ancestor at of a block at a certain slot.
//
// Spec pseudocode definition:
//	Let get_ancestor(store: Store, block: BeaconBlock, slot: SlotNumber) ->
//	BeaconBlock be the ancestor of block with slot number slot.
//	The get_ancestor function can be defined recursively as
//		def get_ancestor(store: Store, block: BeaconBlock, slot: SlotNumber) ->
//		BeaconBlock: return block if block.slot ==
//		slot else get_ancestor(store, store.get_parent(block), slot)
func BlockAncestor(block *pb.BeaconBlock, slot uint64, beaconDB *db.BeaconDB) (*pb.BeaconBlock, error) {
	if block.Slot == slot {
		return block, nil
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
