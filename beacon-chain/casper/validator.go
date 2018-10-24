package casper

import (
	"bytes"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

const bitsInByte = 8

// InitialValidators creates a new validator set that is used to
// generate a new crystallized state.
func InitialValidators() []*pb.ValidatorRecord {
	config := params.GetConfig()
	randaoPreCommit := [32]byte{}
	randaoReveal := hashutil.Hash(randaoPreCommit[:])
	validators := make([]*pb.ValidatorRecord, config.BootstrappedValidatorsCount)
	for i := uint64(0); i < config.BootstrappedValidatorsCount; i++ {
		validators[i] = &pb.ValidatorRecord{
			Status:            uint64(params.Active),
			Balance:           config.DepositSize * config.Gwei,
			WithdrawalAddress: []byte{},
			Pubkey:            []byte{},
			RandaoCommitment:  randaoReveal[:],
		}
	}
	return validators
}

// ActiveValidatorIndices filters out active validators based on validator status
// and returns their indices in a list.
func ActiveValidatorIndices(validators []*pb.ValidatorRecord) []uint32 {
	indices := make([]uint32, 0, len(validators))
	for i, v := range validators {
		if v.Status == uint64(params.Active) {
			indices = append(indices, uint32(i))
		}

	}
	return indices
}

// GetShardAndCommitteesForSlot returns the attester set of a given slot.
func GetShardAndCommitteesForSlot(shardCommittees []*pb.ShardAndCommitteeArray, lastStateRecalc uint64, slot uint64) (*pb.ShardAndCommitteeArray, error) {
	cycleLength := params.GetConfig().CycleLength

	var lowerBound uint64
	if lastStateRecalc >= cycleLength {
		lowerBound = lastStateRecalc - cycleLength
	}
	upperBound := lastStateRecalc + 2*cycleLength
	if slot < lowerBound || slot >= upperBound {
		return nil, fmt.Errorf("slot %d out of bounds: %d <= slot < %d",
			slot,
			lowerBound,
			upperBound,
		)
	}

	// If in the previous or current cycle, simply calculate offset
	if slot < lastStateRecalc+2*cycleLength {
		return shardCommittees[slot-lowerBound], nil
	}

	// Otherwise, use the 3rd cycle
	index := lowerBound + 2*cycleLength + slot%cycleLength
	return shardCommittees[index], nil
}

// AreAttesterBitfieldsValid validates that the length of the attester bitfield matches the attester indices
// defined in the Crystallized State.
func AreAttesterBitfieldsValid(attestation *pb.AggregatedAttestation, attesterIndices []uint32) bool {
	// Validate attester bit field has the correct length.
	if bitutil.BitLength(len(attesterIndices)) != len(attestation.AttesterBitfield) {
		log.Errorf("attestation has incorrect bitfield length. Found %v, expected %v",
			len(attestation.AttesterBitfield), bitutil.BitLength(len(attesterIndices)))
		return false
	}

	// Valid attestation can not have non-zero trailing bits.
	lastBit := len(attesterIndices)
	remainingBits := lastBit % bitsInByte
	if lastBit < int(params.GetConfig().MinCommiteeSize) || remainingBits == 0 {
		return true
	}

	for i := 0; i < bitsInByte-remainingBits; i++ {
		isBitSet, err := bitutil.CheckBit(attestation.AttesterBitfield, lastBit+i)
		if err != nil {
			log.Errorf("Bitfield check failed for attestation at index: %d with: %v", lastBit+i, err)
			return false
		}

		if isBitSet {
			log.Error("attestation has non-zero trailing bits")
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

	return 0, fmt.Errorf("can't find validator index for public key %#x", pubKey)
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

	return 0, fmt.Errorf("can't find shard ID for validator with public key %#x", pubKey)
}

// ValidatorSlotAndRole returns a validator's assingned slot number
// and whether it should act as an attester or proposer.
func ValidatorSlotAndRole(pubKey []byte, validators []*pb.ValidatorRecord, shardCommittees []*pb.ShardAndCommitteeArray) (uint64, pbrpc.ValidatorRole, error) {
	index, err := ValidatorIndex(pubKey, validators)
	if err != nil {
		return 0, pbrpc.ValidatorRole_UNKNOWN, err
	}

	for slot, slotCommittee := range shardCommittees {
		for i, committee := range slotCommittee.ArrayShardAndCommittee {
			for v, validator := range committee.Committee {
				if validator != index {
					continue
				}
				if i == 0 && v == slot%len(committee.Committee) {
					return uint64(slot), pbrpc.ValidatorRole_PROPOSER, nil
				}

				return uint64(slot), pbrpc.ValidatorRole_ATTESTER, nil
			}
		}
	}
	return 0, pbrpc.ValidatorRole_UNKNOWN, fmt.Errorf("can't find slot number for validator with public key %#x", pubKey)
}

// TotalActiveValidatorDeposit returns the total deposited amount in Gwei for all active validators.
func TotalActiveValidatorDeposit(validators []*pb.ValidatorRecord) uint64 {
	var totalDeposit uint64
	indices := ActiveValidatorIndices(validators)

	for _, index := range indices {
		totalDeposit += validators[index].GetBalance()
	}
	return totalDeposit
}

// TotalActiveValidatorDepositInEth returns the total deposited amount in ETH for all active validators.
func TotalActiveValidatorDepositInEth(validators []*pb.ValidatorRecord) uint64 {
	totalDeposit := TotalActiveValidatorDeposit(validators)
	depositInEth := totalDeposit / params.GetConfig().Gwei

	return depositInEth
}

// VotedBalanceInAttestation checks for the total balance in the validator set and the balances of the voters in the
// attestation.
func VotedBalanceInAttestation(validators []*pb.ValidatorRecord, indices []uint32,
	attestation *pb.AggregatedAttestation) (uint64, uint64, error) {

	// find the total and vote balance of the shard committee.
	var totalBalance uint64
	var voteBalance uint64
	for _, attesterIndex := range indices {
		// find balance of validators who voted.
		bitCheck, err := bitutil.CheckBit(attestation.AttesterBitfield, int(attesterIndex))
		if err != nil {
			return 0, 0, err
		}
		if bitCheck {
			voteBalance += validators[attesterIndex].Balance
		}
		// add to total balance of the committee.
		totalBalance += validators[attesterIndex].Balance
	}

	return totalBalance, voteBalance, nil
}

// AddPendingValidator runs for every validator that is inducted as part of a log created on the PoW chain.
func AddPendingValidator(
	validators []*pb.ValidatorRecord,
	pubKey []byte,
	withdrawalShard uint64,
	withdrawalAddr []byte,
	randaoCommitment []byte) []*pb.ValidatorRecord {

	// TODO(#633): Use BLS to verify signature proof of possession and pubkey and hash of pubkey.

	newValidatorRecord := &pb.ValidatorRecord{
		Pubkey:            pubKey,
		WithdrawalShard:   withdrawalShard,
		WithdrawalAddress: withdrawalAddr,
		RandaoCommitment:  randaoCommitment,
		Balance:           params.GetConfig().DepositSize * params.GetConfig().Gwei,
		Status:            uint64(params.PendingActivation),
		ExitSlot:          0,
	}

	index := minEmptyValidator(validators)
	if index > 0 {
		validators[index] = newValidatorRecord
		return validators
	}

	validators = append(validators, newValidatorRecord)
	return validators
}

// ExitValidator exits validator from the active list. It returns
// updated validator record with an appropriate status of each validator.
func ExitValidator(
	validator *pb.ValidatorRecord,
	currentSlot uint64,
	panalize bool) *pb.ValidatorRecord {
	// TODO(#614): Add validator set change
	validator.ExitSlot = currentSlot
	if panalize {
		validator.Status = uint64(params.Penalized)
	} else {
		validator.Status = uint64(params.PendingExit)
	}
	return validator
}

// ChangeValidators updates the validator set during state transition.
func ChangeValidators(currentSlot uint64, totalPenalties uint64, validators []*pb.ValidatorRecord) []*pb.ValidatorRecord {
	maxAllowableChange := 2 * params.GetConfig().DepositSize * params.GetConfig().Gwei

	totalBalance := TotalActiveValidatorDeposit(validators)

	// Determine the max total wei that can deposit and withdraw.
	if totalBalance > maxAllowableChange {
		maxAllowableChange = totalBalance
	}

	var totalChanged uint64
	for i := 0; i < len(validators); i++ {
		if validators[i].Status == uint64(params.PendingActivation) {
			validators[i].Status = uint64(params.Active)
			totalChanged += params.GetConfig().DepositSize * params.GetConfig().Gwei

			// TODO(#614): Add validator set change.
		}
		if validators[i].Status == uint64(params.PendingExit) {
			validators[i].Status = uint64(params.PendingWithdraw)
			validators[i].ExitSlot = currentSlot
			totalChanged += validators[i].Balance

			// TODO(#614): Add validator set change.
		}
		if totalChanged > maxAllowableChange {
			break
		}
	}

	// Calculate withdraw validators that have been logged out long enough,
	// apply their penalties if they were slashed.
	for i := 0; i < len(validators); i++ {
		isPendingWithdraw := validators[i].Status == uint64(params.PendingWithdraw)
		isPenalized := validators[i].Status == uint64(params.Penalized)
		withdrawalSlot := validators[i].ExitSlot + params.GetConfig().WithdrawalPeriod

		if (isPendingWithdraw || isPenalized) && currentSlot >= withdrawalSlot {
			penaltyFactor := totalPenalties * 3
			if penaltyFactor > totalBalance {
				penaltyFactor = totalBalance
			}

			if validators[i].Status == uint64(params.Penalized) {
				validators[i].Balance -= validators[i].Balance * totalBalance / validators[i].Balance
			}
			validators[i].Status = uint64(params.Withdrawn)
		}
	}
	return validators
}

// CopyValidators creates a fresh new validator set by copying all the validator information
// from the old validator set. This is used in calculating the new state of the crystallized
// state, where the changes to the validator balances are applied to the new validator set.
func CopyValidators(validatorSet []*pb.ValidatorRecord) []*pb.ValidatorRecord {
	newValidatorSet := make([]*pb.ValidatorRecord, len(validatorSet))

	for i, validator := range validatorSet {
		newValidatorSet[i] = &pb.ValidatorRecord{
			Pubkey:            validator.Pubkey,
			WithdrawalShard:   validator.WithdrawalShard,
			WithdrawalAddress: validator.WithdrawalAddress,
			RandaoCommitment:  validator.RandaoCommitment,
			Balance:           validator.Balance,
			Status:            validator.Status,
			ExitSlot:          validator.ExitSlot,
		}
	}
	return newValidatorSet
}

// CheckValidatorMinDeposit checks if a validator deposit has fallen below min online deposit size,
// it exits the validator if it's below.
func CheckValidatorMinDeposit(validatorSet []*pb.ValidatorRecord, currentSlot uint64) []*pb.ValidatorRecord {
	for index, validator := range validatorSet {
		MinDepositInGWei := params.GetConfig().MinDeposit * params.GetConfig().Gwei
		isValidatorActive := validator.Status == uint64(params.Active)
		if validator.Balance < MinDepositInGWei && isValidatorActive {
			validatorSet[index] = ExitValidator(validator, currentSlot, false)
		}
	}
	return validatorSet
}

// minEmptyValidator returns the lowest validator index which the status is withdrawn.
func minEmptyValidator(validators []*pb.ValidatorRecord) int {
	for i := 0; i < len(validators); i++ {
		if validators[i].Status == uint64(params.Withdrawn) {
			return i
		}
	}
	return -1
}
