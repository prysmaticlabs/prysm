package casper

import (
	"bytes"
	"fmt"
	"os"

	"github.com/golang/protobuf/jsonpb"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
)

const bitsInByte = 8

// InitialValidators creates a new validator set that is used to
// generate a new crystallized state.
func InitialValidators() []*pb.ValidatorRecord {
	var validators []*pb.ValidatorRecord
	for i := 0; i < params.GetConfig().BootstrappedValidatorsCount; i++ {
		validator := &pb.ValidatorRecord{
			Status:            uint64(params.Active),
			Balance:           uint64(params.GetConfig().DepositSize),
			WithdrawalAddress: []byte{},
			Pubkey:            []byte{},
		}
		validators = append(validators, validator)
	}
	return validators
}

// InitialValidatorsFromJSON retrieves the validator set that is stored in
// genesis.json.
func InitialValidatorsFromJSON(genesisJSONPath string) ([]*pb.ValidatorRecord, error) {
	// #nosec G304
	// genesisJSONPath is a user input for the path of genesis.json.
	// Ex: /path/to/my/genesis.json.
	f, err := os.Open(genesisJSONPath)
	if err != nil {
		return nil, err
	}

	cState := &pb.CrystallizedState{}
	if err := jsonpb.Unmarshal(f, cState); err != nil {
		return nil, fmt.Errorf("error converting JSON to proto: %v", err)
	}

	return cState.Validators, nil
}

// ActiveValidatorIndices filters out active validators based on start and end dynasty
// and returns their indices in a list.
func ActiveValidatorIndices(validators []*pb.ValidatorRecord) []uint32 {
	var indices = make([]uint32, len(validators))
	for i := 0; i < len(validators); i++ {
		if validators[i].Status == uint64(params.Active) {
			indices = append(indices, uint32(i))
		}
	}
	return indices[len(validators):]
}

// ExitedValidatorIndices filters out exited validators based on start and end dynasty
// and returns their indices in a list.
func ExitedValidatorIndices(validators []*pb.ValidatorRecord) []uint32 {
	var indices = make([]uint32, len(validators))
	for i := 0; i < len(validators); i++ {
		if validators[i].Status == uint64(params.PendingExit) {
			indices = append(indices, uint32(i))
		}
	}
	return indices[len(validators):]
}

// QueuedValidatorIndices filters out queued validators based on start and end dynasty
// and returns their indices in a list.
func QueuedValidatorIndices(validators []*pb.ValidatorRecord) []uint32 {
	var indices = make([]uint32, len(validators))
	for i := 0; i < len(validators); i++ {
		if validators[i].Status == uint64(params.PendingActivation) {
			indices = append(indices, uint32(i))
		}
	}
	return indices[len(validators):]
}

// GetShardAndCommitteesForSlot returns the attester set of a given slot.
func GetShardAndCommitteesForSlot(shardCommittees []*pb.ShardAndCommitteeArray, lastStateRecalc uint64, slot uint64) (*pb.ShardAndCommitteeArray, error) {
	if lastStateRecalc < params.GetConfig().CycleLength {
		lastStateRecalc = 0
	} else {
		lastStateRecalc = lastStateRecalc - params.GetConfig().CycleLength
	}

	lowerBound := lastStateRecalc
	upperBound := lastStateRecalc + params.GetConfig().CycleLength*2
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
	if bitutil.BitLength(len(attesterIndices)) != len(attestation.AttesterBitfield) {
		log.Debugf("attestation has incorrect bitfield length. Found %v, expected %v",
			len(attestation.AttesterBitfield), bitutil.BitLength(len(attesterIndices)))
		return false
	}

	// Valid attestation can not have non-zero trailing bits.
	lastBit := len(attesterIndices)
	remainingBits := lastBit % bitsInByte
	if remainingBits == 0 {
		return true
	}

	for i := 0; i < bitsInByte-remainingBits; i++ {
		if bitutil.CheckBit(attestation.AttesterBitfield, lastBit+i) {
			log.Debugf("attestation has non-zero trailing bits")
			return false
		}
	}

	return true
}

// ProposerShardAndIndex returns the index and the shardID of a proposer from a given slot.
func ProposerShardAndIndex(shardCommittees []*pb.ShardAndCommitteeArray, lastStateRecalc uint64, slot uint64) (uint64, uint64, error) {
	slotCommittees, err := GetShardAndCommitteesForSlot(
		shardCommittees,
		lastStateRecalc,
		slot)
	if err != nil {
		return 0, 0, err
	}

	proposerShardID := slotCommittees.ArrayShardAndCommittee[0].Shard
	proposerIndex := slot % uint64(len(slotCommittees.ArrayShardAndCommittee[0].Committee))
	return proposerShardID, proposerIndex, nil
}

// ValidatorIndex returns the index of the validator given an input public key.
func ValidatorIndex(pubKey []byte, validators []*pb.ValidatorRecord) (uint32, error) {
	activeValidators := ActiveValidatorIndices(validators)

	for _, index := range activeValidators {
		if bytes.Equal(validators[index].Pubkey, pubKey) {
			return index, nil
		}
	}

	return 0, fmt.Errorf("can't find validator index for public key %x", pubKey)
}

// ValidatorShardID returns the shard ID of the validator currently participates in.
func ValidatorShardID(pubKey []byte, validators []*pb.ValidatorRecord, shardCommittees []*pb.ShardAndCommitteeArray) (uint64, error) {
	index, err := ValidatorIndex(pubKey, validators)
	if err != nil {
		return 0, err
	}

	for _, slotCommittee := range shardCommittees {
		for _, committee := range slotCommittee.ArrayShardAndCommittee {
			for _, validator := range committee.Committee {
				if validator == index {
					return committee.Shard, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("can't find shard ID for validator with public key %x", pubKey)
}

// ValidatorSlotAndResponsibility returns a validator's assingned slot number
// and whether it should act as an attester or proposer.
func ValidatorSlotAndResponsibility(pubKey []byte, validators []*pb.ValidatorRecord, shardCommittees []*pb.ShardAndCommitteeArray) (uint64, string, error) {
	index, err := ValidatorIndex(pubKey, validators)
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
	return 0, "", fmt.Errorf("can't find slot number for validator with public key %x", pubKey)
}

// TotalActiveValidatorDeposit returns the total deposited amount in wei for all active validators.
func TotalActiveValidatorDeposit(validators []*pb.ValidatorRecord) uint64 {
	var totalDeposit uint64
	activeValidators := ActiveValidatorIndices(validators)

	for _, index := range activeValidators {
		totalDeposit += validators[index].GetBalance()
	}
	return totalDeposit
}

// TotalActiveValidatorDepositInEth returns the total deposited amount in ETH for all active validators.
func TotalActiveValidatorDepositInEth(validators []*pb.ValidatorRecord) uint64 {
	totalDeposit := TotalActiveValidatorDeposit(validators)
	depositInEth := totalDeposit / uint64(params.GetConfig().Gwei)

	return depositInEth
}

// CommitteeinShardAndSlot returns the shard committee for a a particular slot index and shard.
func CommitteeInShardAndSlot(slotIndex uint64, shardID uint64, shardCommitteeArray []*pb.ShardAndCommitteeArray) ([]uint32, error) {
	shardCommittee := shardCommitteeArray[slotIndex].ArrayShardAndCommittee

	for i := 0; i < len(shardCommittee); i++ {
		if shardID == shardCommittee[i].Shard {
			return shardCommittee[i].Committee, nil
		}
	}

	return nil, fmt.Errorf("unable to find committee based on slot index: %v, and Shard: %v", slotIndex, shardID)
}

func VotedBalanceInAttestation(validators []*pb.ValidatorRecord, indices []uint32, attestation *pb.AggregatedAttestation) (uint64, uint64) {
	// find the total and vote balance of the shard committee.
	var totalBalance uint64
	var voteBalance uint64
	for _, attesterIndex := range indices {
		// find balance of validators who voted.
		if bitutil.CheckBit(attestation.AttesterBitfield, int(attesterIndex)) {
			voteBalance += validators[attesterIndex].Balance
		}
		// add to total balance of the committee.
		totalBalance += validators[attesterIndex].Balance
	}

	return totalBalance, voteBalance
}
