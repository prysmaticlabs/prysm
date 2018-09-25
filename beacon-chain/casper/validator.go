package casper

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
)

const bitsInByte = 8

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

// GetShardAndCommitteesForSlot returns the attester set of a given slot.
func GetShardAndCommitteesForSlot(shardCommittees []*pb.ShardAndCommitteeArray, lastStateRecalc uint64, slot uint64) (*pb.ShardAndCommitteeArray, error) {
	if lastStateRecalc < params.CycleLength {
		lastStateRecalc = 0
	} else {
		lastStateRecalc = lastStateRecalc - params.CycleLength
	}

	lowerBound := lastStateRecalc
	upperBound := lastStateRecalc + params.CycleLength*2
	if !(slot >= lowerBound && slot < upperBound) {
		return nil, fmt.Errorf("cannot return attester set of given slot, input slot %v has to be in between %v and %v",
			slot,
			lowerBound,
			upperBound,
		)
	}

	return shardCommittees[slot-lastStateRecalc], nil
}

// AreAttesterBitfieldsValid validates that the length of the attester bitfield matches the attester indices
// defined in the Crystallized State.
func AreAttesterBitfieldsValid(attestation *pb.AggregatedAttestation, attesterIndices []uint32) bool {
	// Validate attester bit field has the correct length.
	if shared.BitLength(len(attesterIndices)) != len(attestation.AttesterBitfield) {
		log.Debugf("attestation has incorrect bitfield length. Found %v, expected %v",
			len(attestation.AttesterBitfield), shared.BitLength(len(attesterIndices)))
		return false
	}

	// Valid attestation can not have non-zero trailing bits.
	lastBit := len(attesterIndices)
	remainingBits := lastBit % bitsInByte
	if remainingBits == 0 {
		return true
	}

	for i := 0; i < bitsInByte-remainingBits; i++ {
		if shared.CheckBit(attestation.AttesterBitfield, lastBit+i) {
			log.Debugf("attestation has non-zero trailing bits")
			return false
		}
	}

	return true
}

// ProposerShardAndIndex returns the index and the shardID of a proposer from a given slot.
func ProposerShardAndIndex(shardCommittees []*pb.ShardAndCommitteeArray, lastStateRecalc uint64, slot uint64) (uint64, uint32, error) {
	slotCommittees, err := GetShardAndCommitteesForSlot(
		shardCommittees,
		lastStateRecalc,
		slot)
	if err != nil {
		return 0, 0, err
	}

	proposerShardID := slotCommittees.ArrayShardAndCommittee[0].ShardId
	fmt.Println(slotCommittees.ArrayShardAndCommittee[0].Committee)
	index := slot % uint64(len(slotCommittees.ArrayShardAndCommittee[0].Committee))
	proposerIndex := slotCommittees.ArrayShardAndCommittee[0].Committee[index]
	return proposerShardID, proposerIndex, nil
}

// ValidatorIndex returns the index of the validator given an input public key.
func ValidatorIndex(pubKey uint64, dynasty uint64, validators []*pb.ValidatorRecord) (uint32, error) {
	activeValidators := ActiveValidatorIndices(validators, dynasty)

	for _, index := range activeValidators {
		if validators[index].PublicKey == pubKey {
			return index, nil
		}
	}

	return 0, fmt.Errorf("can't find validator index for public key %d", pubKey)
}

// ValidatorShardID returns the shard ID of the validator currently participates in.
func ValidatorShardID(pubKey uint64, dynasty uint64, validators []*pb.ValidatorRecord, shardCommittees []*pb.ShardAndCommitteeArray) (uint64, error) {
	index, err := ValidatorIndex(pubKey, dynasty, validators)
	if err != nil {
		return 0, err
	}

	for _, slotCommittee := range shardCommittees {
		for _, committee := range slotCommittee.ArrayShardAndCommittee {
			for _, validator := range committee.Committee {
				if validator == index {
					return committee.ShardId, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("can't find shard ID for validator with public key %d", pubKey)
}

// ValidatorSlotAndResponsibility returns a validator's assingned slot number
// and whether it should act as an attester or proposer.
func ValidatorSlotAndResponsibility(pubKey uint64, dynasty uint64, validators []*pb.ValidatorRecord, shardCommittees []*pb.ShardAndCommitteeArray) (uint64, string, error) {
	index, err := ValidatorIndex(pubKey, dynasty, validators)
	if err != nil {
		return 0, "", err
	}

	for slot, slotCommittee := range shardCommittees {
		for i, committee := range slotCommittee.ArrayShardAndCommittee {
			for v, validator := range committee.Committee {
				if i == 0 && v == slot%len(committee.Committee) && validator == index {
					return uint64(slot), "proposer", nil
				}
				if validator == index {
					return uint64(slot), "attester", nil
				}
			}
		}
	}
	return 0, "", fmt.Errorf("can't find slot number for validator with public key %d", pubKey)
}

// TotalActiveValidatorDeposit returns the total deposited amount in wei for all active validators.
func TotalActiveValidatorDeposit(dynasty uint64, validators []*pb.ValidatorRecord) uint64 {
	var totalDeposit uint64
	activeValidators := ActiveValidatorIndices(validators, dynasty)

	for _, index := range activeValidators {
		totalDeposit += validators[index].GetBalance()
	}
	return totalDeposit
}

// TotalActiveValidatorDepositInEth returns the total deposited amount in ETH for all active validators.
func TotalActiveValidatorDepositInEth(dynasty uint64, validators []*pb.ValidatorRecord) uint64 {
	totalDeposit := TotalActiveValidatorDeposit(dynasty, validators)
	depositInEth := totalDeposit / uint64(params.EtherDenomination)

	return depositInEth
}
