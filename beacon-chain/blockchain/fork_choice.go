package blockchain

import (
	"bytes"
	"fmt"
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
) (*pb.BeaconBlock, error) {
	targets, err := AttestationTargets(beaconState, block, beaconDB)
	if err != nil {
		return nil, fmt.Errorf("could not fetch attestation targets for active validators: %v", err)
	}
	head := block
	for {
		children, err := BlockChildren(head, observedBlocks)
		if err != nil {
			return nil, fmt.Errorf("could not fetch block children: %v", err)
		}
		if len(children) == 0 {
			fmt.Println("NOCHILD")
			return head, nil
		}
		fmt.Println("CHILD")
		maxChild := children[0] // so far max child will be potentialHead [potentialHead]
		for i := 1; i < len(children); i++ {
			candidateChildVotes, err := VoteCount(children[i], targets, beaconDB)
			if err != nil {
				return nil, fmt.Errorf("unable to determine vote count for block: %v", err)
			}
			maxChildVotes, err := VoteCount(maxChild, targets, beaconDB)
			if err != nil {
				return nil, fmt.Errorf("unable to determine vote count for block: %v", err)
			}
			fmt.Printf("candidate: %d, max: %d\n", candidateChildVotes, maxChildVotes)
			if candidateChildVotes > maxChildVotes {
				maxChild = children[i]
			}
		}
		head = maxChild
	}
}

// VoteCount determines the number of votes on a beacon block by counting the number
// of target blocks that have such beacon block as a common ancestor.
func VoteCount(block *pb.BeaconBlock, targets []*pb.BeaconBlock, beaconDB *db.BeaconDB) (int, error) {
	votes := 0
	for _, target := range targets {
		ancestor, err := BlockAncestor(target, block.Slot, beaconDB)
		if err != nil {
			return 0, err
		}
		if reflect.DeepEqual(ancestor, block) {
			votes++
		}
	}
	fmt.Println(votes)
	return votes, nil
}

// BlockAncestor obtains the ancestor at of a block at a certain slot.
func BlockAncestor(block *pb.BeaconBlock, slot uint64, beaconDB *db.BeaconDB) (*pb.BeaconBlock, error) {
	fmt.Println("WORKSSS")
	if block.Slot == slot {
		fmt.Println("YAY")
		return block, nil
	}
	parentHash := bytesutil.ToBytes32(block.ParentRootHash32)
	parent, err := beaconDB.GetBlock(parentHash)
	if err != nil {
		return nil, fmt.Errorf("could not get parent blocK: %v", err)
	}
	return BlockAncestor(parent, slot, beaconDB)
}

// BlockChildren obtains the blocks in a list of observed blocks which have the current
// beacon block's hash as their parent root hash.
func BlockChildren(block *pb.BeaconBlock, observedBlocks []*pb.BeaconBlock) ([]*pb.BeaconBlock, error) {
	var children []*pb.BeaconBlock
	encoded, err := proto.Marshal(block)
	if err != nil {
		return nil, fmt.Errorf("could not marshal block: %v", err)
	}
	hash := hashutil.Hash(encoded)
	for _, observed := range observedBlocks {
		if bytes.Equal(observed.ParentRootHash32, hash[:]) {
			children = append(children, observed)
		}
	}
	return children, nil
}

// AttestationTargets fetches the blocks corresponding to the latest observed attestation
// for each active validator in the state's registry.
func AttestationTargets(
	beaconState *pb.BeaconState, block *pb.BeaconBlock, beaconDB *db.BeaconDB,
) ([]*pb.BeaconBlock, error) {
	activeValidators := validators.ActiveValidatorIndices(beaconState.ValidatorRegistry, beaconState.Slot)
	var attestationTargets []*pb.BeaconBlock
	for _, validatorIndex := range activeValidators {
		target, err:= LatestAttestationTarget(validatorIndex, beaconDB)
		if err != nil {
            return nil, err
		}
		attestationTargets = append(attestationTargets, target)
	}
	return attestationTargets, nil
}

// LatestAttestationTarget obtains the block target corresponding to the latest
// attestation seen by the validator. It is the attestation with the highest slot number
// in store from the validator. In case of a tie, pick the one observed first.
func LatestAttestationTarget(validatorIndex uint32, beaconDB *db.BeaconDB) (*pb.BeaconBlock, error) {
	latestAttsProto, err := beaconDB.GetLatestAttestationsForValidator(validatorIndex)
	if err != nil {
		return nil, fmt.Errorf(
			"could not fetch latest attestations for validator at index %d: %v",
			validatorIndex,
			err,
		)
	}
	latestAtts := latestAttsProto.Attestations
	if len(latestAtts) == 0 {
		return nil, nil
	}
	highestSlotAtt := latestAtts[0]
	for i := 1; i < len(latestAtts); i++ {
		if latestAtts[i].Data.Slot > highestSlotAtt.Data.Slot {
			highestSlotAtt = latestAtts[i]
		}
	}
	blockHash := bytesutil.ToBytes32(highestSlotAtt.Data.BeaconBlockRootHash32)
	return beaconDB.GetBlock(blockHash)
}
