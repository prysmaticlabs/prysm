package casper

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

const bitsInByte = 8

// RotateValidatorSet is called every dynasty transition. The primary functions are:
// 1.) Go through queued validator indices and induct them to be active by setting start
// dynasty to current cycle.
// 2.) Remove bad active validator whose balance is below threshold to the exit set by
// setting end dynasty to current cycle.
func RotateValidatorSet(validators []*pb.ValidatorRecord, dynasty uint64) []*pb.ValidatorRecord {
	upperbound := len(ActiveValidatorIndices(validators, dynasty))/30 + 1

	// Loop through active validator set, remove validator whose balance is below 50%.
	for _, index := range ActiveValidatorIndices(validators, dynasty) {
		if validators[index].Balance < params.DefaultBalance/2 {
			validators[index].EndDynasty = dynasty
		}
	}
	// Get the total number of validator we can induct.
	inductNum := upperbound
	if len(QueuedValidatorIndices(validators, dynasty)) < inductNum {
		inductNum = len(QueuedValidatorIndices(validators, dynasty))
	}

	// Induct queued validator to active validator set until the switch dynasty is greater than current number.
	for _, index := range QueuedValidatorIndices(validators, dynasty) {
		validators[index].StartDynasty = dynasty
		inductNum--
		if inductNum == 0 {
			break
		}
	}
	return validators
}

// ActiveValidatorIndices filters out active validators based on start and end dynasty
// and returns their indices in a list.
func ActiveValidatorIndices(validators []*pb.ValidatorRecord, dynasty uint64) []uint32 {
	var indices []uint32
	for i := 0; i < len(validators); i++ {
		if validators[i].StartDynasty <= dynasty && dynasty < validators[i].EndDynasty {
			indices = append(indices, uint32(i))
		}
	}
	return indices
}

// ExitedValidatorIndices filters out exited validators based on start and end dynasty
// and returns their indices in a list.
func ExitedValidatorIndices(validators []*pb.ValidatorRecord, dynasty uint64) []uint32 {
	var indices []uint32
	for i := 0; i < len(validators); i++ {
		if validators[i].StartDynasty < dynasty && validators[i].EndDynasty <= dynasty {
			indices = append(indices, uint32(i))
		}
	}
	return indices
}

// QueuedValidatorIndices filters out queued validators based on start and end dynasty
// and returns their indices in a list.
func QueuedValidatorIndices(validators []*pb.ValidatorRecord, dynasty uint64) []uint32 {
	var indices []uint32
	for i := 0; i < len(validators); i++ {
		if validators[i].StartDynasty > dynasty {
			indices = append(indices, uint32(i))
		}
	}
	return indices
}

// SampleAttestersAndProposers returns lists of random sampled attesters and proposer indices.
func SampleAttestersAndProposers(seed common.Hash, validators []*pb.ValidatorRecord, dynasty uint64) ([]uint32, uint32, error) {
	attesterCount := params.MinCommiteeSize
	if len(validators) < params.MinCommiteeSize {
		attesterCount = len(validators)
	}
	indices, err := utils.ShuffleIndices(seed, ActiveValidatorIndices(validators, dynasty))
	if err != nil {
		return nil, 0, err
	}
	return indices[:int(attesterCount)], indices[len(indices)-1], nil
}

// GetAttestersTotalDeposit from the pending attestations.
func GetAttestersTotalDeposit(attestations []*pb.AttestationRecord) uint64 {
	var numOfBits int
	for _, attestation := range attestations {
		for _, byte := range attestation.AttesterBitfield {
			numOfBits += int(utils.BitSetCount(byte))
		}
	}
	// Assume there's no slashing condition, the following logic will change later phase.
	return uint64(numOfBits) * params.DefaultBalance
}

// GetShardAndCommitteesForSlot returns the attester set of a given slot.
func GetShardAndCommitteesForSlot(shardCommittees []*pb.ShardAndCommitteeArray, lcs uint64, slot uint64) (*pb.ShardAndCommitteeArray, error) {
	if !(lcs <= slot && slot < lcs+params.CycleLength*2) {
		return nil, fmt.Errorf("can not return attester set of given slot, input slot %v has to be in between %v and %v", slot, lcs, lcs+params.CycleLength*2)
	}
	return shardCommittees[slot-lcs], nil
}

// AreAttesterBitfieldsValid validates that the length of the attester bitfield matches the attester indices
// defined in the Crystallized State.
func AreAttesterBitfieldsValid(attestation *pb.AttestationRecord, attesterIndices []uint32) bool {
	// Validate attester bit field has the correct length.
	if utils.BitLength(len(attesterIndices)) != len(attestation.AttesterBitfield) {
		log.Debugf("attestation has incorrect bitfield length. Found %v, expected %v",
			len(attestation.AttesterBitfield), utils.BitLength(len(attesterIndices)))
		return false
	}

	// Valid attestation can not have non-zero trailing bits.
	lastBit := len(attesterIndices)
	remainingBits := lastBit % bitsInByte
	if remainingBits == 0 {
		return true
	}

	for i := 0; i < bitsInByte-remainingBits; i++ {
		if utils.CheckBit(attestation.AttesterBitfield, lastBit+i) {
			log.Debugf("attestation has non-zero trailing bits")
			return false
		}
	}

	return true
}
