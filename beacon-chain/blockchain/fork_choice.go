package blockchain

import (
	"bytes"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	bytesutil "github.com/prysmaticlabs/prysm/shared/bytes"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// LMDGhost applies the Latest Message Driven, Greediest Heaviest Observed Sub-Tree
// fork-choice rule defined in the Ethereum Serenity specification for the beacon chain.
func LMDGhost(
	beaconState *pb.BeaconState,
	block *pb.BeaconBlock,
	observedBlocks []*pb.BeaconBlock,
	beaconDB *db.BeaconDB,
) *pb.BeaconBlock {
	targets := AttestationTargets(beaconState, block, beaconDB)
	head := block
	for {
		children := BlockChildren(head, observedBlocks)
		if len(children) == 0 {
			return head
		}
		maxChild := children[0]
		for i := 1; i < len(children); i++ {
			if VoteCount(children[i], targets, beaconDB) > VoteCount(maxChild, targets, beaconDB) {
				maxChild = children[i]
			}
		}
		head = maxChild
	}
}

// VoteCount determines the number of votes on a beacon block by counting the number
// of observed blocks that have the current beacon block as a common ancestor.
func VoteCount(block *pb.BeaconBlock, targets []*pb.BeaconBlock, beaconDB *db.BeaconDB) int {
	votes := 0
	for _, target := range targets {
		ancestor := BlockAncestor(target, block.Slot, beaconDB)
		if reflect.DeepEqual(ancestor, block) {
			votes++
		}
	}
	return votes
}

// BlockAncestor obtains the ancestor at of a block at a certain slot.
func BlockAncestor(block *pb.BeaconBlock, slot uint64, beaconDB *db.BeaconDB) *pb.BeaconBlock {
	if block.Slot == slot {
		return block
	}
	parentHash := bytesutil.ToBytes32(block.ParentRootHash32)
	parent, _ := beaconDB.GetBlock(parentHash)
	return BlockAncestor(parent, slot, beaconDB)
}

// BlockChildren obtains the blocks in a list of observed blocks which have the current
// beacon block's hash as their parent root hash.
func BlockChildren(block *pb.BeaconBlock, observedBlocks []*pb.BeaconBlock) []*pb.BeaconBlock {
	var children []*pb.BeaconBlock
	encoded, _ := proto.Marshal(block)
	hash := hashutil.Hash(encoded)
	for _, observed := range observedBlocks {
		if bytes.Equal(observed.ParentRootHash32, hash[:]) {
			children = append(children, observed)
		}
	}
}

// AttestationTargets fetches the blocks corresponding to the latest observed attestation
// for each active validator in the state's registry.
func AttestationTargets(beaconState *pb.BeaconState, block *pb.BeaconBlock, beaconDB *db.BeaconDB) []*pb.BeaconBlock {
	activeValidators := validators.ActiveValidatorIndices(beaconState.ValidatorRegistry, beaconState.Slot)
	var attestationTargets []*pb.BeaconBlock
	for _, validatorIndex := range activeValidators {
		target := LatestAttestationTarget(validatorIndex, beaconDB)
		attestationTargets = append(attestationTargets, target)
	}
	return attestationTargets
}

// LatestAttestationTarget obtains the block target corresponding to the latest
// attestation seen by the validator. It is the attestation with the highest slot number
// in store from the validator. In case of a tie, pick the one observed first.
func LatestAttestationTarget(validatorIndex uint32, beaconDB *db.BeaconDB) *pb.BeaconBlock {
	latestAttsProto, _ := beaconDB.GetLatestAttestationsForValidator(validatorIndex)
	latestAtts := latestAttsProto.Attestations
	highestSlotAtt := latestAtts[0]
	for i := 1; i < len(latestAtts); i++ {
		if latestAtts[i].Data.Slot > highestSlotAtt.Data.Slot {
			highestSlotAtt = latestAtts[i]
		}
	}
	blockHash := bytesutil.ToBytes32(highestSlotAtt.Data.BeaconBlockRootHash32)
	target, _ := beaconDB.GetBlock(blockHash)
	return target
}
