// Package validators defines helper functions to locate validator
// based on pubic key. Each validator is associated with a given index,
// shard ID and slot number to propose or attest. This package also defines
// functions to initialize validators, verify validator bit fields,
// and rotate validator in and out of committees.
package validators

import (
	"bytes"
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const bitsInByte = 8

// InitialValidatorRegistry creates a new validator set that is used to
// generate a new crystallized state.
func InitialValidatorRegistry() []*pb.ValidatorRecord {
	config := params.BeaconConfig()
	randaoPreCommit := [32]byte{}
	randaoReveal := hashutil.Hash(randaoPreCommit[:])
	validators := make([]*pb.ValidatorRecord, config.BootstrappedValidatorsCount)
	for i := uint64(0); i < config.BootstrappedValidatorsCount; i++ {
		validators[i] = &pb.ValidatorRecord{
			Status:                 pb.ValidatorRecord_ACTIVE,
			Balance:                config.MaxDeposit * config.Gwei,
			Pubkey:                 []byte{},
			RandaoCommitmentHash32: randaoReveal[:],
		}
	}
	return validators
}

// ActiveValidatorIndices filters out active validators based on validator status
// and returns their indices in a list.
//
// Spec pseudocode definition:
//   def get_active_validator_indices(validators: [ValidatorRecord]) -> List[int]:
//     """
//     Gets indices of active validators from ``validators``.
//     """
//     return [i for i, v in enumerate(validators) if is_active_validator(v)]
func ActiveValidatorIndices(validators []*pb.ValidatorRecord) []uint32 {
	indices := make([]uint32, 0, len(validators))
	for i, v := range validators {
		if isActiveValidator(v) {
			indices = append(indices, uint32(i))
		}

	}
	return indices
}

// ActiveValidator returns the active validator records in a list.
//
// Spec pseudocode definition:
//   [state.validator_registry[i] for i in get_active_validator_indices(state.validator_registry)]
func ActiveValidator(state *pb.BeaconState, validatorIndices []uint32) []*pb.ValidatorRecord {
	activeValidators := make([]*pb.ValidatorRecord, 0, len(validatorIndices))
	for _, validatorIndex := range validatorIndices {
		activeValidators = append(activeValidators, state.ValidatorRegistry[validatorIndex])
	}
	return activeValidators
}

// ShardAndCommitteesAtSlot returns the shard and committee list for a given
// slot within the range of 2 * epoch length within the same 2 epoch slot
// window as the state slot.
//
// Spec pseudocode definition:
//   def get_shard_committees_at_slot(state: BeaconState, slot: int) -> List[ShardCommittee]:
//     """
//     Returns the ``ShardCommittee`` for the ``slot``.
//     """
//     earliest_slot_in_array = state.Slot - (state.Slot % EPOCH_LENGTH) - EPOCH_LENGTH
//     assert earliest_slot_in_array <= slot < earliest_slot_in_array + EPOCH_LENGTH * 2
//     return state.shard_committees_at_slots[slot - earliest_slot_in_array]
func ShardAndCommitteesAtSlot(state *pb.BeaconState, slot uint64) (*pb.ShardAndCommitteeArray, error) {
	epochLength := params.BeaconConfig().EpochLength
	var earliestSlot uint64

	// If the state slot is less than epochLength, then the earliestSlot would
	// result in a negative number. Therefore we should default to
	// earliestSlot = 0 in this case.
	if state.Slot > epochLength {
		earliestSlot = state.Slot - (state.Slot % epochLength) - epochLength
	}

	if slot < earliestSlot || slot >= earliestSlot+(epochLength*2) {
		return nil, fmt.Errorf("slot %d out of bounds: %d <= slot < %d",
			slot,
			earliestSlot,
			earliestSlot+(epochLength*2),
		)
	}

	return state.ShardAndCommitteesAtSlots[slot-earliestSlot], nil
}

// GetShardAndCommitteesForSlot returns the attester set of a given slot.
// Deprecated: Use ShardAndCommitteesAtSlot instead.
func GetShardAndCommitteesForSlot(shardCommittees []*pb.ShardAndCommitteeArray, lastStateRecalc uint64, slot uint64) (*pb.ShardAndCommitteeArray, error) {
	cycleLength := params.BeaconConfig().CycleLength

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
		return false
	}

	// Valid attestation can not have non-zero trailing bits.
	lastBit := len(attesterIndices)
	remainingBits := lastBit % bitsInByte
	if remainingBits == 0 {
		return true
	}

	for i := 0; i < bitsInByte-remainingBits; i++ {
		isBitSet, err := bitutil.CheckBit(attestation.AttesterBitfield, lastBit+i)
		if err != nil {
			return false
		}

		if isBitSet {
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
	activeValidatorRegistry := ActiveValidatorIndices(validators)

	for _, index := range activeValidatorRegistry {
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

// TotalActiveValidatorBalance returns the total deposited amount in Gwei for all active validators.
//
// Spec pseudocode definition:
//   sum([get_effective_balance(v) for v in active_validators])
func TotalActiveValidatorBalance(activeValidators []*pb.ValidatorRecord) uint64 {
	var totalDeposit uint64

	for _, v := range activeValidators {
		totalDeposit += EffectiveBalance(v)
	}
	return totalDeposit
}

// TotalActiveValidatorDepositInEth returns the total deposited amount in ETH for all active validators.
func TotalActiveValidatorDepositInEth(validators []*pb.ValidatorRecord) uint64 {
	totalDeposit := TotalActiveValidatorBalance(validators)
	depositInEth := totalDeposit / params.BeaconConfig().Gwei

	return depositInEth
}

// VotedBalanceInAttestation checks for the total balance in the validator set and the balances of the voters in the
// attestation.
func VotedBalanceInAttestation(validators []*pb.ValidatorRecord, indices []uint32,
	attestation *pb.AggregatedAttestation) (uint64, uint64, error) {

	// find the total and vote balance of the shard committee.
	var totalBalance uint64
	var voteBalance uint64
	for index, attesterIndex := range indices {
		// find balance of validators who voted.
		bitCheck, err := bitutil.CheckBit(attestation.AttesterBitfield, index)
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
	randaoCommitment []byte,
	status pb.ValidatorRecord_StatusCodes) []*pb.ValidatorRecord {

	// TODO(#633): Use BLS to verify signature proof of possession and pubkey and hash of pubkey.

	newValidatorRecord := &pb.ValidatorRecord{
		Pubkey:                 pubKey,
		RandaoCommitmentHash32: randaoCommitment,
		Balance:                params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei,
		Status:                 status,
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
	penalize bool) *pb.ValidatorRecord {
	// TODO(#614): Add validator set change
	validator.LatestStatusChangeSlot = currentSlot
	if penalize {
		validator.Status = pb.ValidatorRecord_EXITED_WITH_PENALTY
	} else {
		validator.Status = pb.ValidatorRecord_ACTIVE_PENDING_EXIT
	}
	return validator
}

// ChangeValidatorRegistry updates the validator set during state transition.
func ChangeValidatorRegistry(currentSlot uint64, totalPenalties uint64, validators []*pb.ValidatorRecord) []*pb.ValidatorRecord {
	maxAllowableChange := 2 * params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei

	totalBalance := TotalActiveValidatorBalance(validators)

	// Determine the max total wei that can deposit and withdraw.
	if totalBalance > maxAllowableChange {
		maxAllowableChange = totalBalance
	}

	var totalChanged uint64
	for i := 0; i < len(validators); i++ {
		if validators[i].Status == pb.ValidatorRecord_PENDING_ACTIVATION {
			validators[i].Status = pb.ValidatorRecord_ACTIVE
			totalChanged += params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei

			// TODO(#614): Add validator set change.
		}
		if validators[i].Status == pb.ValidatorRecord_ACTIVE_PENDING_EXIT {
			validators[i].Status = pb.ValidatorRecord_ACTIVE_PENDING_EXIT
			validators[i].LatestStatusChangeSlot = currentSlot
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
		isPendingWithdraw := validators[i].Status == pb.ValidatorRecord_ACTIVE_PENDING_EXIT
		isPenalized := validators[i].Status == pb.ValidatorRecord_EXITED_WITH_PENALTY
		withdrawalSlot := validators[i].LatestStatusChangeSlot + params.BeaconConfig().MinWithdrawalPeriod

		if (isPendingWithdraw || isPenalized) && currentSlot >= withdrawalSlot {
			penaltyFactor := totalPenalties * 3
			if penaltyFactor > totalBalance {
				penaltyFactor = totalBalance
			}

			if validators[i].Status == pb.ValidatorRecord_EXITED_WITH_PENALTY {
				validators[i].Balance -= validators[i].Balance * totalBalance / validators[i].Balance
			}
			validators[i].Status = pb.ValidatorRecord_EXITED_WITHOUT_PENALTY
		}
	}
	return validators
}

// CopyValidatorRegistry creates a fresh new validator set by copying all the validator information
// from the old validator set. This is used in calculating the new state of the crystallized
// state, where the changes to the validator balances are applied to the new validator set.
func CopyValidatorRegistry(validatorSet []*pb.ValidatorRecord) []*pb.ValidatorRecord {
	newValidatorSet := make([]*pb.ValidatorRecord, len(validatorSet))

	for i, validator := range validatorSet {
		newValidatorSet[i] = &pb.ValidatorRecord{
			Pubkey:                 validator.Pubkey,
			RandaoCommitmentHash32: validator.RandaoCommitmentHash32,
			Balance:                validator.Balance,
			Status:                 validator.Status,
			LatestStatusChangeSlot: validator.LatestStatusChangeSlot,
		}
	}
	return newValidatorSet
}

// CheckValidatorMinDeposit checks if a validator deposit has fallen below min online deposit size,
// it exits the validator if it's below.
func CheckValidatorMinDeposit(validatorSet []*pb.ValidatorRecord, currentSlot uint64) []*pb.ValidatorRecord {
	for index, validator := range validatorSet {
		MinDepositInGWei := params.BeaconConfig().MinOnlineDepositSize * params.BeaconConfig().Gwei
		isValidatorActive := validator.Status == pb.ValidatorRecord_ACTIVE
		if validator.Balance < MinDepositInGWei && isValidatorActive {
			validatorSet[index] = ExitValidator(validator, currentSlot, false)
		}
	}
	return validatorSet
}

// EffectiveBalance returns the balance at stake for the validator.
//
// Spec pseudocode definition:
//   def get_effective_balance(validator: ValidatorRecord) -> int:
//     """
//     Returns the effective balance (also known as "balance at stake") for the ``validator``.
//     """
//     return min(validator.balance, MAX_DEPOSIT)
func EffectiveBalance(validator *pb.ValidatorRecord) uint64 {
	if validator.Balance > params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei {
		return params.BeaconConfig().MaxDeposit
	}
	return validator.Balance
}

// minEmptyValidator returns the lowest validator index which the status is withdrawn.
func minEmptyValidator(validators []*pb.ValidatorRecord) int {
	for i := 0; i < len(validators); i++ {
		if validators[i].Status == pb.ValidatorRecord_EXITED_WITHOUT_PENALTY {
			return i
		}
	}
	return -1
}

// isActiveValidator returns the boolean value on whether the validator
// is active or not.
//
// Spec pseudocode definition:
//   def is_active_validator(validator: ValidatorRecord) -> bool:
//     """
//     Returns the ``ShardCommittee`` for the ``slot``.
//     """
//     return validator.status in [ACTIVE, ACTIVE_PENDING_EXIT]
func isActiveValidator(validator *pb.ValidatorRecord) bool {
	return validator.Status == pb.ValidatorRecord_ACTIVE_PENDING_EXIT ||
		validator.Status == pb.ValidatorRecord_ACTIVE
}
