// Package validators contains libraries to shuffle validators
// and retrieve active validator indices from a given slot
// or an attestation. It also provides helper functions to locate
// validator based on pubic key.
package validators

import (
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

var config = params.BeaconConfig()

// InitialValidatorRegistry creates a new validator set that is used to
// generate a new bootstrapped state.
func InitialValidatorRegistry() []*pb.ValidatorRecord {
	randaoPreCommit := [32]byte{}
	randaoReveal := hashutil.Hash(randaoPreCommit[:])
	validators := make([]*pb.ValidatorRecord, config.DepositsForChainStart)
	for i := uint64(0); i < config.DepositsForChainStart; i++ {
		pubkey := hashutil.Hash([]byte{byte(i)})
		validators[i] = &pb.ValidatorRecord{
			ExitEpoch:              config.FarFutureEpoch,
			Balance:                config.MaxDeposit,
			Pubkey:                 pubkey[:],
			RandaoCommitmentHash32: randaoReveal[:],
			RandaoLayers:           1,
		}
	}
	return validators
}

// ActiveValidators returns the active validator records in a list.
//
// Spec pseudocode definition:
//   [state.validator_registry[i] for i in get_active_validator_indices(state.validator_registry)]
func ActiveValidators(state *pb.BeaconState, validatorIndices []uint32) []*pb.ValidatorRecord {
	activeValidators := make([]*pb.ValidatorRecord, 0, len(validatorIndices))
	for _, validatorIndex := range validatorIndices {
		activeValidators = append(activeValidators, state.ValidatorRegistry[validatorIndex])
	}
	return activeValidators
}

// BeaconProposerIdx returns the index of the proposer of the block at a
// given slot.
//
// Spec pseudocode definition:
//  def get_beacon_proposer_index(state: BeaconState,slot: int) -> int:
//    """
//    Returns the beacon proposer index for the ``slot``.
//    """
//    first_committee, _ = get_crosslink_committees_at_slot(state, slot)[0]
//    return first_committee[slot % len(first_committee)]
func BeaconProposerIdx(state *pb.BeaconState, slot uint64) (uint64, error) {
	committeeArray, err := CrosslinkCommitteesAtSlot(state, slot)
	if err != nil {
		return 0, err
	}
	firstCommittee := committeeArray[0].Committee

	return firstCommittee[slot%uint64(len(firstCommittee))], nil
}

// ValidatorIdx returns the idx of the validator given an input public key.
func ValidatorIdx(pubKey []byte, validators []*pb.ValidatorRecord) (uint64, error) {

	for idx := range validators {
		if bytes.Equal(validators[idx].Pubkey, pubKey) {
			return uint64(idx), nil
		}
	}

	return 0, fmt.Errorf("can't find validator index for public key %#x", pubKey)
}

// TotalEffectiveBalance returns the total deposited amount at stake in Gwei
// of all active validators.
//
// Spec pseudocode definition:
//   sum([get_effective_balance(state, i) for i in active_validator_indices])
func TotalEffectiveBalance(state *pb.BeaconState, validatorIndices []uint64) uint64 {
	var totalDeposit uint64

	for _, idx := range validatorIndices {
		totalDeposit += EffectiveBalance(state, idx)
	}
	return totalDeposit
}

// NewRegistryDeltaChainTip returns the new validator registry delta chain tip.
//
// Spec pseudocode definition:
//   def get_new_validator_registry_delta_chain_tip(current_validator_registry_delta_chain_tip: Hash32,
//                                               validator_index: int,
//                                               pubkey: int,
//                                               flag: int) -> Hash32:
// 	  """
//    Compute the next root in the validator registry delta chain.
//    """
//    return hash_tree_root(
//        ValidatorRegistryDeltaBlock(
//            latest_registry_delta_root=current_validator_registry_delta_chain_tip,
//            validator_index=validator_index,
//            pubkey=pubkey,
//            flag=flag,
//        )
//    )
func NewRegistryDeltaChainTip(
	flag pb.ValidatorRegistryDeltaBlock_ValidatorRegistryDeltaFlags,
	idx uint64,
	slot uint64,
	pubKey []byte,
	currentValidatorRegistryDeltaChainTip []byte) ([32]byte, error) {

	newDeltaChainTip := &pb.ValidatorRegistryDeltaBlock{
		LatestRegistryDeltaRootHash32: currentValidatorRegistryDeltaChainTip,
		ValidatorIndex:                idx,
		Pubkey:                        pubKey,
		Flag:                          flag,
		Slot:                          slot,
	}

	// TODO(716): Replace serialization with tree hash function.
	serializedChainTip, err := proto.Marshal(newDeltaChainTip)
	if err != nil {
		return [32]byte{}, fmt.Errorf("could not marshal new chain tip: %v", err)
	}
	return hashutil.Hash(serializedChainTip), nil
}

// EffectiveBalance returns the balance at stake for the validator.
// Beacon chain allows validators to top off their balance above MAX_DEPOSIT,
// but they can be slashed at most MAX_DEPOSIT at any time.
//
// Spec pseudocode definition:
//   def get_effective_balance(state: State, index: int) -> int:
//     """
//     Returns the effective balance (also known as "balance at stake") for a ``validator`` with the given ``index``.
//     """
//     return min(state.validator_balances[idx], MAX_DEPOSIT)
func EffectiveBalance(state *pb.BeaconState, idx uint64) uint64 {
	if state.ValidatorBalances[idx] > config.MaxDeposit {
		return config.MaxDeposit
	}
	return state.ValidatorBalances[idx]
}

// Attesters returns the validator records using validator indices.
//
// Spec pseudocode definition:
//   Let this_epoch_boundary_attesters = [state.validator_registry[i]
//   for indices in this_epoch_boundary_attester_indices for i in indices].
func Attesters(state *pb.BeaconState, attesterIndices []uint64) []*pb.ValidatorRecord {

	var boundaryAttesters []*pb.ValidatorRecord
	for _, attesterIdx := range attesterIndices {
		boundaryAttesters = append(boundaryAttesters, state.ValidatorRegistry[attesterIdx])
	}

	return boundaryAttesters
}

// ValidatorIndices returns all the validator indices from the input attestations
// and state.
//
// Spec pseudocode definition:
//   Let this_epoch_boundary_attester_indices be the union of the validator
//   index sets given by [get_attestation_participants(state, a.data, a.participation_bitfield)
//   for a in attestations]
func ValidatorIndices(
	state *pb.BeaconState,
	attestations []*pb.PendingAttestationRecord,
) ([]uint64, error) {

	var attesterIndicesIntersection []uint64
	for _, attestation := range attestations {
		attesterIndices, err := AttestationParticipants(
			state,
			attestation.Data,
			attestation.ParticipationBitfield)
		if err != nil {
			return nil, err
		}

		attesterIndicesIntersection = sliceutil.Union(attesterIndicesIntersection, attesterIndices)
	}

	return attesterIndicesIntersection, nil
}

// AttestingValidatorIndices returns the shard committee validator indices
// if the validator shard committee matches the input attestations.
//
// Spec pseudocode definition:
// Let attesting_validator_indices(crosslink_committee, shard_block_root)
// be the union of the validator index sets given by
// [get_attestation_participants(state, a.data, a.participation_bitfield)
// for a in this_epoch_attestations + previous_epoch_attestations
// if a.shard == shard_committee.shard and a.shard_block_root == shard_block_root]
func AttestingValidatorIndices(
	state *pb.BeaconState,
	shard uint64,
	shardBlockRoot []byte,
	thisEpochAttestations []*pb.PendingAttestationRecord,
	prevEpochAttestations []*pb.PendingAttestationRecord) ([]uint64, error) {

	var validatorIndicesCommittees []uint64
	attestations := append(thisEpochAttestations, prevEpochAttestations...)

	for _, attestation := range attestations {
		if attestation.Data.Shard == shard &&
			bytes.Equal(attestation.Data.ShardBlockRootHash32, shardBlockRoot) {

			validatorIndicesCommittee, err := AttestationParticipants(state, attestation.Data, attestation.ParticipationBitfield)
			if err != nil {
				return nil, fmt.Errorf("could not get attester indices: %v", err)
			}
			validatorIndicesCommittees = sliceutil.Union(validatorIndicesCommittees, validatorIndicesCommittee)
		}
	}
	return validatorIndicesCommittees, nil
}

// AttestingBalance returns the combined balances from the input validator
// records.
//
// Spec pseudocode definition:
//   Let this_epoch_boundary_attesting_balance =
//   sum([get_effective_balance(state, i) for i in this_epoch_boundary_attester_indices])
func AttestingBalance(state *pb.BeaconState, boundaryAttesterIndices []uint64) uint64 {

	var boundaryAttestingBalance uint64
	for _, idx := range boundaryAttesterIndices {
		boundaryAttestingBalance += EffectiveBalance(state, idx)
	}

	return boundaryAttestingBalance
}

// AllValidatorsIndices returns all validator indices from 0 to
// the last validator.
func AllValidatorsIndices(state *pb.BeaconState) []uint64 {
	validatorIndices := make([]uint64, len(state.ValidatorRegistry))
	for i := 0; i < len(validatorIndices); i++ {
		validatorIndices[i] = uint64(i)
	}
	return validatorIndices
}

// ProcessDeposit mutates a corresponding index in the beacon state for
// a validator depositing ETH into the beacon chain. Specifically, this function
// adds a validator balance or tops up an existing validator's balance
// by some deposit amount. This function returns a mutated beacon state and
// the validator index corresponding to the validator in the processed
// deposit.
func ProcessDeposit(
	state *pb.BeaconState,
	validatorIdxMap map[[32]byte]int,
	pubkey []byte,
	amount uint64,
	_ /*proofOfPossession*/ []byte,
	withdrawalCredentials []byte,
	randaoCommitment []byte,
) (*pb.BeaconState, error) {
	// TODO(#258): Validate proof of possession using BLS.
	var publicKeyExists bool
	var existingValidatorIdx int

	existingValidatorIdx, publicKeyExists = validatorIdxMap[bytesutil.ToBytes32(pubkey)]
	if !publicKeyExists {
		// If public key does not exist in the registry, we add a new validator
		// to the beacon state.
		newValidator := &pb.ValidatorRecord{
			Pubkey:                 pubkey,
			RandaoCommitmentHash32: randaoCommitment,
			RandaoLayers:           0,
			ActivationEpoch:        config.FarFutureEpoch,
			ExitEpoch:              config.FarFutureEpoch,
			WithdrawalEpoch:        config.FarFutureEpoch,
			PenalizedEpoch:         config.FarFutureEpoch,
			StatusFlags:            0,
		}
		state.ValidatorRegistry = append(state.ValidatorRegistry, newValidator)
		state.ValidatorBalances = append(state.ValidatorBalances, amount)

	} else {
		if !bytes.Equal(
			state.ValidatorRegistry[existingValidatorIdx].WithdrawalCredentialsHash32,
			withdrawalCredentials,
		) {
			return nil, fmt.Errorf(
				"expected withdrawal credentials to match, received %#x == %#x",
				state.ValidatorRegistry[existingValidatorIdx].WithdrawalCredentialsHash32,
				withdrawalCredentials,
			)
		}
		state.ValidatorBalances[existingValidatorIdx] += amount
	}
	return state, nil
}

// ActivateValidator takes in validator index and updates
// validator's activation slot.
//
// Spec pseudocode definition:
// def activate_validator(state: BeaconState, index: ValidatorIndex, is_genesis: bool) -> None:
//    """
//    Activate the validator of the given ``index``.
//    Note that this function mutates ``state``.
//    """
//    validator = state.validator_registry[index]
//
//    validator.activation_epoch = GENESIS_EPOCH if is_genesis else get_entry_exit_effect_epoch(get_current_epoch(state))
func ActivateValidator(state *pb.BeaconState, idx uint64, genesis bool) (*pb.BeaconState, error) {
	validator := state.ValidatorRegistry[idx]
	if genesis {
		validator.ActivationEpoch = config.GenesisEpoch
	} else {
		validator.ActivationEpoch = helpers.EntryExitEffectEpoch(helpers.CurrentEpoch(state))
	}

	state.ValidatorRegistry[idx] = validator
	return state, nil
}

// InitiateValidatorExit takes in validator index and updates
// validator with INITIATED_EXIT status flag.
//
// Spec pseudocode definition:
// def initiate_validator_exit(state: BeaconState, index: int) -> None:
//    validator = state.validator_registry[index]
//    validator.status_flags |= INITIATED_EXIT
func InitiateValidatorExit(state *pb.BeaconState, idx uint64) *pb.BeaconState {
	state.ValidatorRegistry[idx].StatusFlags |=
		pb.ValidatorRecord_INITIATED_EXIT
	return state
}

// ExitValidator takes in validator index and does house
// keeping work to exit validator with entry exit delay.
//
// Spec pseudocode definition:
// def exit_validator(state: BeaconState, index: ValidatorIndex) -> None:
//    """
//    Exit the validator of the given ``index``.
//    Note that this function mutates ``state``.
//    """
//    validator = state.validator_registry[index]
//
//    # The following updates only occur if not previous exited
//    if validator.exit_epoch <= get_entry_exit_effect_epoch(get_current_epoch(state)):
//        return
//
//    validator.exit_epoch = get_entry_exit_effect_epoch(get_current_epoch(state))
func ExitValidator(state *pb.BeaconState, idx uint64) (*pb.BeaconState, error) {
	validator := state.ValidatorRegistry[idx]

	exitEpoch := helpers.EntryExitEffectEpoch(helpers.CurrentEpoch(state))
	if validator.ExitEpoch <= exitEpoch {
		return nil, fmt.Errorf("validator %d could not exit until epoch %d",
			idx, exitEpoch)
	}

	validator.ExitEpoch = exitEpoch
	return state, nil
}

// PenalizeValidator slashes the malicious validator's balance and awards
// the whistleblower's balance.
//
// Spec pseudocode definition:
// def penalize_validator(state: BeaconState, index: ValidatorIndex) -> None:
//    """
//    Penalize the validator of the given ``index``.
//    Note that this function mutates ``state``.
//    """
//    exit_validator(state, index)
//    validator = state.validator_registry[index]
//    state.latest_penalized_balances[get_current_epoch(state) % LATEST_PENALIZED_EXIT_LENGTH] += get_effective_balance(state, index)
//
//    whistleblower_index = get_beacon_proposer_index(state, state.slot)
//    whistleblower_reward = get_effective_balance(state, index) // WHISTLEBLOWER_REWARD_QUOTIENT
//    state.validator_balances[whistleblower_index] += whistleblower_reward
//    state.validator_balances[index] -= whistleblower_reward
//    validator.penalized_epoch = get_current_epoch(state)
func PenalizeValidator(state *pb.BeaconState, idx uint64) (*pb.BeaconState, error) {
	state, err := ExitValidator(state, idx)
	if err != nil {
		return nil, fmt.Errorf("could not exit penalized validator: %v", err)
	}

	penalizedDuration := helpers.CurrentEpoch(state) % config.LatestPenalizedExitLength
	state.LatestPenalizedBalances[penalizedDuration] += EffectiveBalance(state, idx)

	whistleblowerIdx, err := BeaconProposerIdx(state, state.Slot)
	if err != nil {
		return nil, fmt.Errorf("could not get proposer idx: %v", err)
	}
	whistleblowerReward := EffectiveBalance(state, idx) /
		config.WhistlerBlowerRewardQuotient

	state.ValidatorBalances[whistleblowerIdx] += whistleblowerReward
	state.ValidatorBalances[idx] -= whistleblowerReward

	state.ValidatorRegistry[idx].PenalizedEpoch = helpers.CurrentEpoch(state)
	return state, nil
}

// PrepareValidatorForWithdrawal sets validator's status flag to
// WITHDRAWABLE.
//
// Spec pseudocode definition:
// def prepare_validator_for_withdrawal(state: BeaconState, index: int) -> None:
//    validator = state.validator_registry[index]
//    validator.status_flags |= WITHDRAWABLE
func PrepareValidatorForWithdrawal(state *pb.BeaconState, idx uint64) *pb.BeaconState {
	state.ValidatorRegistry[idx].StatusFlags |=
		pb.ValidatorRecord_WITHDRAWABLE
	return state
}

// UpdateRegistry rotates validators in and out of active pool.
// the amount to rotate is determined by max validator balance churn.
//
// Spec pseudocode definition:
// def update_validator_registry(state: BeaconState) -> None:
//    """
//    Update validator registry.
//    Note that this function mutates ``state``.
//    """
//    current_epoch = get_current_epoch(state)
//    # The active validators
//    active_validator_indices = get_active_validator_indices(state.validator_registry, current_epoch)
//    # The total effective balance of active validators
//    total_balance = sum([get_effective_balance(state, i) for i in active_validator_indices])
//
//    # The maximum balance churn in Gwei (for deposits and exits separately)
//    max_balance_churn = max(
//        MAX_DEPOSIT_AMOUNT,
//        total_balance // (2 * MAX_BALANCE_CHURN_QUOTIENT)
//    )
//
//    # Activate validators within the allowable balance churn
//    balance_churn = 0
//    for index, validator in enumerate(state.validator_registry):
//        if validator.activation_epoch > get_entry_exit_effect_epoch(current_epoch) and state.validator_balances[index] >= MAX_DEPOSIT_AMOUNT:
//            # Check the balance churn would be within the allowance
//            balance_churn += get_effective_balance(state, index)
//            if balance_churn > max_balance_churn:
//                break
//
//            # Activate validator
//            activate_validator(state, index, is_genesis=False)
//
//    # Exit validators within the allowable balance churn
//    balance_churn = 0
//    for index, validator in enumerate(state.validator_registry):
//        if validator.exit_epoch > get_entry_exit_effect_epoch(current_epoch) and validator.status_flags & INITIATED_EXIT:
//            # Check the balance churn would be within the allowance
//            balance_churn += get_effective_balance(state, index)
//            if balance_churn > max_balance_churn:
//                break
//
//            # Exit validator
//            exit_validator(state, index)
//
//    state.validator_registry_update_epoch = current_epoch
func UpdateRegistry(state *pb.BeaconState) (*pb.BeaconState, error) {
	currentEpoch := helpers.CurrentEpoch(state)
	activeValidatorIndices := helpers.ActiveValidatorIndices(
		state.ValidatorRegistry, currentEpoch)

	totalBalance := TotalEffectiveBalance(state, activeValidatorIndices)

	// The maximum balance churn in Gwei (for deposits and exits separately).
	maxBalChurn := maxBalanceChurn(totalBalance)

	var balChurn uint64
	var err error
	for idx, validator := range state.ValidatorRegistry {
		// Activate validators within the allowable balance churn.
		if validator.ActivationEpoch > helpers.EntryExitEffectEpoch(currentEpoch) &&
			state.ValidatorBalances[idx] >= config.MaxDeposit {
			balChurn += EffectiveBalance(state, uint64(idx))
			if balChurn > maxBalChurn {
				break
			}
			state, err = ActivateValidator(state, uint64(idx), false)
			if err != nil {
				return nil, fmt.Errorf("could not activate validator %d: %v", idx, err)
			}
		}
	}

	balChurn = 0
	for idx, validator := range state.ValidatorRegistry {
		// Exit validators within the allowable balance churn.
		if validator.ExitEpoch > helpers.EntryExitEffectEpoch(currentEpoch) &&
			validator.StatusFlags == pb.ValidatorRecord_INITIATED_EXIT {
			balChurn += EffectiveBalance(state, uint64(idx))
			if balChurn > maxBalChurn {
				break
			}
			state, err = ExitValidator(state, uint64(idx))
			if err != nil {
				return nil, fmt.Errorf("could not exit validator %d: %v", idx, err)
			}
		}
	}
	state.ValidatorRegistryUpdateSlot = state.Slot
	return state, nil
}

// ProcessPenaltiesAndExits prepares the validators and the penalized validators
// for withdrawal.
//
// Spec pseudocode definition:
// def process_penalties_and_exits(state: BeaconState) -> None:
//    """
//    Process the penalties and prepare the validators who are eligible to withdrawal.
//    Note that this function mutates ``state``.
//    """
//    current_epoch = get_current_epoch(state)
//    # The active validators
//    active_validator_indices = get_active_validator_indices(state.validator_registry, current_epoch)
//    # The total effective balance of active validators
//    total_balance = sum(get_effective_balance(state, i) for i in active_validator_indices)
//
//    for index, validator in enumerate(state.validator_registry):
//        if current_epoch == validator.penalized_epoch + LATEST_PENALIZED_EXIT_LENGTH // 2:
//            epoch_index = current_epoch % LATEST_PENALIZED_EXIT_LENGTH
//            total_at_start = state.latest_penalized_balances[(epoch_index + 1) % LATEST_PENALIZED_EXIT_LENGTH]
//            total_at_end = state.latest_penalized_balances[epoch_index]
//            total_penalties = total_at_end - total_at_start
//            penalty = get_effective_balance(state, index) * min(total_penalties * 3, total_balance) // total_balance
//            state.validator_balances[index] -= penalty
//
//    def eligible(index):
//        validator = state.validator_registry[index]
//        if validator.penalized_epoch <= current_epoch:
//            penalized_withdrawal_epochs = LATEST_PENALIZED_EXIT_LENGTH // 2
//            return current_epoch >= validator.penalized_epoch + penalized_withdrawal_epochs
//        else:
//            return current_epoch >= validator.exit_epoch + MIN_VALIDATOR_WITHDRAWAL_EPOCHS
//
//    all_indices = list(range(len(state.validator_registry)))
//    eligible_indices = filter(eligible, all_indices)
//    # Sort in order of exit epoch, and validators that exit within the same epoch exit in order of validator index
//    sorted_indices = sorted(eligible_indices, key=lambda index: state.validator_registry[index].exit_epoch)
//    withdrawn_so_far = 0
//    for index in sorted_indices:
//        prepare_validator_for_withdrawal(state, index)
//        withdrawn_so_far += 1
//        if withdrawn_so_far >= MAX_WITHDRAWALS_PER_EPOCH:
//            break
func ProcessPenaltiesAndExits(state *pb.BeaconState) *pb.BeaconState {
	currentEpoch := helpers.CurrentEpoch(state)
	activeValidatorIndices := helpers.ActiveValidatorIndices(
		state.ValidatorRegistry, currentEpoch)
	totalBalance := TotalEffectiveBalance(state, activeValidatorIndices)

	for idx, validator := range state.ValidatorRegistry {
		penalized := validator.PenalizedEpoch +
			config.LatestPenalizedExitLength/2
		if currentEpoch == penalized {
			penalizedEpoch := currentEpoch % config.LatestPenalizedExitLength
			penalizedEpochStart := (penalizedEpoch + 1) % config.LatestPenalizedExitLength
			totalAtStart := state.LatestPenalizedBalances[penalizedEpochStart]
			totalAtEnd := state.LatestPenalizedBalances[penalizedEpoch]
			totalPenalties := totalAtStart - totalAtEnd

			penaltyMultiplier := totalPenalties * 3
			if totalBalance < penaltyMultiplier {
				penaltyMultiplier = totalBalance
			}
			penalty := EffectiveBalance(state, uint64(idx)) *
				penaltyMultiplier / totalBalance
			state.ValidatorBalances[idx] -= penalty
		}
	}
	allIndices := AllValidatorsIndices(state)
	var eligibleIndices []uint64
	for _, idx := range allIndices {
		if eligibleToExit(state, idx) {
			eligibleIndices = append(eligibleIndices, idx)
		}
	}
	var withdrawnSoFar uint64
	for _, idx := range eligibleIndices {
		state = PrepareValidatorForWithdrawal(state, idx)
		withdrawnSoFar++
		if withdrawnSoFar >= config.MaxWithdrawalsPerEpoch {
			break
		}
	}
	return state
}

// maxBalanceChurn returns the maximum balance churn in Gwei,
// this determines how many validators can be rotated
// in and out of the validator pool.
// Spec pseudocode definition:
//     max_balance_churn = max(
//        MAX_DEPOSIT * GWEI_PER_ETH,
//        total_balance // (2 * MAX_BALANCE_CHURN_QUOTIENT))
func maxBalanceChurn(totalBalance uint64) uint64 {
	maxBalanceChurn := totalBalance / 2 * config.MaxBalanceChurnQuotient
	if maxBalanceChurn > config.MaxDeposit {
		return maxBalanceChurn
	}
	return config.MaxDeposit
}

// eligibleToExit checks if a validator is eligible to exit whether it was
// penalized or not.
//
// Spec pseudocode definition:
// def eligible(index):
//    validator = state.validator_registry[index]
//    if validator.penalized_epoch <= current_epoch:
//         penalized_withdrawal_epochs = LATEST_PENALIZED_EXIT_LENGTH // 2
//        return current_epoch >= validator.penalized_epoch + penalized_withdrawal_epochs
//    else:
//        return current_epoch >= validator.exit_epoch + MIN_VALIDATOR_WITHDRAWAL_EPOCHS
func eligibleToExit(state *pb.BeaconState, idx uint64) bool {
	currentEpoch := helpers.CurrentEpoch(state)
	validator := state.ValidatorRegistry[idx]

	if validator.PenalizedEpoch <= currentEpoch {
		penalizedWithdrawalEpochs := config.LatestPenalizedExitLength / 2
		return currentEpoch >= validator.PenalizedEpoch+penalizedWithdrawalEpochs
	}
	return currentEpoch >= validator.ExitEpoch+config.MinValidatorWithdrawalEpochs
}
