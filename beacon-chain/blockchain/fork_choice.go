package blockchain

import (
	"bytes"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func lmdGHOST(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
	observedBlocks []*pb.BeaconBlock,
	beaconDB *db.BeaconDB,
) *pb.BeaconBlock {
	targets := attestationTargets(beaconState, block, beaconDB)
	head := block
	for {
		children := blockChildren(head, observedBlocks)
		if len(children) == 0 {
			return head
		}
		maxChild := children[0]
		for i := 1; i < len(children); i++ {
			if voteCount(children[i], targets, beaconDB) > voteCount(maxChild, targets, beaconDB) {
				maxChild = children[i]
			}
		}
		head = maxChild
	}
}

func voteCount(block *pb.BeaconBlock, targets []*pb.BeaconBlock, beaconDB *db.BeaconDB) int {
	votes := 0
	for _, target := range targets {
		ancestor := blockAncestor(target, block.Slot, beaconDB)
		if reflect.DeepEqual(ancestor, block) {
			votes++
		}
	}
	return votes
}

func blockAncestor(block *pb.BeaconBlock, slot uint64, beaconDB *db.BeaconDB) *pb.BeaconBlock {
	if block.Slot == slot {
		return block
	}
	// TODO(#1307): This should instead be the parent of the block in DB.
	parent, _ := beaconDB.GetChainHead()
	return blockAncestor(parent, slot, beaconDB)
}

func blockChildren(block *pb.BeaconBlock, observedBlocks []*pb.BeaconBlock) []*pb.BeaconBlock {
	var children []*pb.BeaconBlock
	encoded, _ := proto.Marshal(block)
	hash := hashutil.Hash(encoded)
	for _, observed := range observedBlocks {
		if bytes.Equal(observed.ParentRootHash32, hash[:]) {
			children = append(children, observed)
		}
	}
}

func attestationTargets(beaconState *pb.BeaconState, block *pb.BeaconBlock, beaconDB *db.BeaconDB) []*pb.BeaconBlock {
	activeValidators := validators.ActiveValidatorIndices(beaconState.ValidatorRegistry, beaconState.Slot)
	var attestationTargets []*pb.BeaconBlock
	for _, validatorIndex := range activeValidators {
		target := latestAttestationTarget(beaconState.ValidatorRegistry[validatorIndex], beaconDB)
		attestationTargets = append(attestationTargets, target)
	}
	return attestationTargets
}

func latestAttestationTarget(validator *pb.ValidatorRecord, beaconDB *db.BeaconDB) *pb.BeaconBlock {
	// TODO(#1307) Fetch the block target corresponding to the latest attestation by the validator.
	return nil
}
