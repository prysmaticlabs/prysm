// Package validators contains libraries to shuffle validators
// and retrieve active validator indices from a given slot
// or an attestation. It also provides helper functions to locate
// validator based on pubic key.
package validators

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	bytesutil "github.com/prysmaticlabs/prysm/shared/bytes"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slices"
)

var config = params.BeaconConfig()

// InitialValidatorRegistry creates a new validator set that is used to
// generate a new crystallized state.
func InitialValidatorRegistry() []*pb.ValidatorRecord {
	randaoPreCommit := [32]byte{}
	randaoReveal := hashutil.Hash(randaoPreCommit[:])
	validators := make([]*pb.ValidatorRecord, config.DepositsForChainStart)
	for i := uint64(0); i < config.DepositsForChainStart; i++ {
		pubkey := hashutil.Hash([]byte{byte(i)})
		validators[i] = &pb.ValidatorRecord{
			ExitSlot:               config.FarFutureSlot,
			Balance:                config.MaxDeposit * config.Gwei,
			Pubkey:                 pubkey[:],
			RandaoCommitmentHash32: randaoReveal[:],
		}
	}
	return validators
}

// ActiveValidatorIndices filters out active validators based on validator status
// and returns their indices in a list.
//
// Spec pseudocode definition:
//   def get_active_validator_indices(validators: [ValidatorRecord], slot: int) -> List[int]:
//     """
//     Gets indices of active validators from ``validators``.
//     """
//     return [i for i, v in enumerate(validators) if is_active_validator(v, slot)]
func ActiveValidatorIndices(validators []*pb.ValidatorRecord, slot uint64) []uint32 {
	indices := make([]uint32, 0, len(validators))
	for i, v := range validators {
		if isActiveValidator(v, slot) {
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

// ShardCommitteesAtSlot returns the shard and committee list for a given
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
func ShardCommitteesAtSlot(state *pb.BeaconState, slot uint64) (*pb.ShardCommitteeArray, error) {
	epochLength := config.EpochLength
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
	return state.ShardCommitteesAtSlots[slot-earliestSlot], nil
}

// BeaconProposerIdx returns the index of the proposer of the block at a
// given slot.
//
// Spec pseudocode definition:
//  def get_beacon_proposer_index(state: BeaconState,slot: int) -> int:
//    """
//    Returns the beacon proposer index for the ``slot``.
//    """
//    first_committee = get_shard_committees_at_slot(state, slot)[0].committee
//    return first_committee[slot % len(first_committee)]
func BeaconProposerIdx(state *pb.BeaconState, slot uint64) (uint32, error) {
	committeeArray, err := ShardCommitteesAtSlot(state, slot)
	if err != nil {
		return 0, err
	}
	firstCommittee := committeeArray.ArrayShardCommittee[0].Committee

	return firstCommittee[slot%uint64(len(firstCommittee))], nil
}

// ProposerShardAndIdx returns the index and the shardID of a proposer from a given slot.
func ProposerShardAndIdx(state *pb.BeaconState, slot uint64) (uint64, uint64, error) {
	slotCommittees, err := ShardCommitteesAtSlot(
		state,
		slot)
	if err != nil {
		return 0, 0, err
	}

	proposerShardID := slotCommittees.ArrayShardCommittee[0].Shard
	proposerIdx := slot % uint64(len(slotCommittees.ArrayShardCommittee[0].Committee))
	return proposerShardID, proposerIdx, nil
}

// ValidatorIdx returns the idx of the validator given an input public key.
func ValidatorIdx(pubKey []byte, validators []*pb.ValidatorRecord) (uint32, error) {

	for idx := range validators {
		if bytes.Equal(validators[idx].Pubkey, pubKey) {
			return uint32(idx), nil
		}
	}

	return 0, fmt.Errorf("can't find validator index for public key %#x", pubKey)
}

// ValidatorShardID returns the shard ID of the validator currently participates in.
func ValidatorShardID(pubKey []byte, validators []*pb.ValidatorRecord, shardCommittees []*pb.ShardCommitteeArray) (uint64, error) {
	idx, err := ValidatorIdx(pubKey, validators)
	if err != nil {
		return 0, err
	}

	for _, slotCommittee := range shardCommittees {
		for _, committee := range slotCommittee.ArrayShardCommittee {
			for _, validator := range committee.Committee {
				if validator == idx {
					return committee.Shard, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("can't find shard ID for validator with public key %#x", pubKey)
}

// ValidatorSlotAndRole returns a validator's assingned slot number
// and whether it should act as an attester or proposer.
func ValidatorSlotAndRole(pubKey []byte, validators []*pb.ValidatorRecord, shardCommittees []*pb.ShardCommitteeArray) (uint64, pbrpc.ValidatorRole, error) {
	idx, err := ValidatorIdx(pubKey, validators)
	if err != nil {
		return 0, pbrpc.ValidatorRole_UNKNOWN, err
	}

	for slot, slotCommittee := range shardCommittees {
		for i, committee := range slotCommittee.ArrayShardCommittee {
			for v, validator := range committee.Committee {
				if validator != idx {
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

// TotalEffectiveBalance returns the total deposited amount at stake in Gwei
// of all active validators.
//
// Spec pseudocode definition:
//   sum([get_effective_balance(state, i) for i in active_validator_indices])
func TotalEffectiveBalance(state *pb.BeaconState, validatorIndices []uint32) uint64 {
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
	idx uint32,
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
//     return min(state.validator_balances[idx], MAX_DEPOSIT * GWEI_PER_ETH)
func EffectiveBalance(state *pb.BeaconState, idx uint32) uint64 {
	if state.ValidatorBalances[idx] > config.MaxDeposit*config.Gwei {
		return config.MaxDeposit * config.Gwei
	}
	return state.ValidatorBalances[idx]
}

// Attesters returns the validator records using validator indices.
//
// Spec pseudocode definition:
//   Let this_epoch_boundary_attesters = [state.validator_registry[i]
//   for indices in this_epoch_boundary_attester_indices for i in indices].
func Attesters(state *pb.BeaconState, attesterIndices []uint32) []*pb.ValidatorRecord {

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
) ([]uint32, error) {

	var attesterIndicesIntersection []uint32
	for _, attestation := range attestations {
		attesterIndices, err := AttestationParticipants(
			state,
			attestation.Data,
			attestation.ParticipationBitfield)
		if err != nil {
			return nil, err
		}

		attesterIndicesIntersection = slices.Union(attesterIndicesIntersection, attesterIndices)
	}

	return attesterIndicesIntersection, nil
}

// AttestingValidatorIndices returns the shard committee validator indices
// if the validator shard committee matches the input attestations.
//
// Spec pseudocode definition:
// Let attesting_validator_indices(shard_committee, shard_block_root)
// be the union of the validator index sets given by
// [get_attestation_participants(state, a.data, a.participation_bitfield)
// for a in this_epoch_attestations + previous_epoch_attestations
// if a.shard == shard_committee.shard and a.shard_block_root == shard_block_root]
func AttestingValidatorIndices(
	state *pb.BeaconState,
	shardCommittee *pb.ShardCommittee,
	shardBlockRoot []byte,
	thisEpochAttestations []*pb.PendingAttestationRecord,
	prevEpochAttestations []*pb.PendingAttestationRecord) ([]uint32, error) {

	var validatorIndicesCommittees []uint32
	attestations := append(thisEpochAttestations, prevEpochAttestations...)

	for _, attestation := range attestations {
		if attestation.Data.Shard == shardCommittee.Shard &&
			bytes.Equal(attestation.Data.ShardBlockRootHash32, shardBlockRoot) {

			validatorIndicesCommittee, err := AttestationParticipants(state, attestation.Data, attestation.ParticipationBitfield)
			if err != nil {
				return nil, fmt.Errorf("could not get attester indices: %v", err)
			}
			validatorIndicesCommittees = slices.Union(validatorIndicesCommittees, validatorIndicesCommittee)
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
func AttestingBalance(state *pb.BeaconState, boundaryAttesterIndices []uint32) uint64 {

	var boundaryAttestingBalance uint64
	for _, idx := range boundaryAttesterIndices {
		boundaryAttestingBalance += EffectiveBalance(state, idx)
	}

	return boundaryAttestingBalance
}

// AllValidatorsIndices returns all validator indices from 0 to
// the last validator.
func AllValidatorsIndices(state *pb.BeaconState) []uint32 {
	validatorIndices := make([]uint32, len(state.ValidatorRegistry))
	for i := 0; i < len(validatorIndices); i++ {
		validatorIndices[i] = uint32(i)
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
	custodyCommitment []byte,
) (*pb.BeaconState, error) {
	// TODO(#258): Validate proof of possession using BLS.
	var publicKeyExists bool
	var existingValidatorIdx int

	existingValidatorIdx, publicKeyExists = validatorIdxMap[bytesutil.ToBytes32(pubkey)]
	if !publicKeyExists {
		// If public key does not exist in the registry, we add a new validator
		// to the beacon state.
		newValidator := &pb.ValidatorRecord{
			Pubkey:                       pubkey,
			RandaoCommitmentHash32:       randaoCommitment,
			RandaoLayers:                 0,
			ExitCount:                    0,
			CustodyCommitmentHash32:      custodyCommitment,
			LatestCustodyReseedSlot:      config.GenesisSlot,
			PenultimateCustodyReseedSlot: config.GenesisSlot,
			ActivationSlot:               config.FarFutureSlot,
			ExitSlot:                     config.FarFutureSlot,
			WithdrawalSlot:               config.FarFutureSlot,
			PenalizedSlot:                config.FarFutureSlot,
			StatusFlags:                  0,
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

// isActiveValidator returns the boolean value on whether the validator
// is active or not.
//
// Spec pseudocode definition:
//   def is_active_validator(validator: ValidatorRecord, slot: int) -> bool:
//     """
//     Checks if ``validator`` is active.
//     """
//     return validator.activation_slot <= slot < validator.exit_slot
func isActiveValidator(validator *pb.ValidatorRecord, slot uint64) bool {
	return validator.ActivationSlot <= slot &&
		slot < validator.ExitSlot
}

// ActivateValidator takes in validator index and updates
// validator's activation slot.
//
// Spec pseudocode definition:
// def activate_validator(state: BeaconState, index: int, genesis: bool) -> None:
//    validator = state.validator_registry[index]
//
//    validator.activation_slot = GENESIS_SLOT if genesis else (state.slot + ENTRY_EXIT_DELAY)
//    state.validator_registry_delta_chain_tip = hash_tree_root(
//        ValidatorRegistryDeltaBlock(
//            current_validator_registry_delta_chain_tip=state.validator_registry_delta_chain_tip,
//            validator_index=index,
//            pubkey=validator.pubkey,
//            slot=validator.activation_slot,
//            flag=ACTIVATION,
//        )
//    )
func ActivateValidator(state *pb.BeaconState, idx uint32, genesis bool) (*pb.BeaconState, error) {
	validator := state.ValidatorRegistry[idx]
	if genesis {
		validator.ActivationSlot = config.GenesisSlot
	} else {
		validator.ActivationSlot = state.Slot + config.EntryExitDelay
	}
	newChainTip, err := NewRegistryDeltaChainTip(
		pb.ValidatorRegistryDeltaBlock_ACTIVATION,
		idx,
		validator.ActivationSlot,
		validator.Pubkey,
		state.ValidatorRegistryDeltaChainTipHash32,
	)
	if err != nil {
		return nil, fmt.Errorf("could not get new chain tip %v", err)
	}
	state.ValidatorRegistryDeltaChainTipHash32 = newChainTip[:]
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
func InitiateValidatorExit(state *pb.BeaconState, idx uint32) *pb.BeaconState {
	state.ValidatorRegistry[idx].StatusFlags |=
		pb.ValidatorRecord_INITIATED_EXIT
	return state
}

// ExitValidator takes in validator index and does house
// keeping work to exit validator with entry exit delay.
//
// Spec pseudocode definition:
// def exit_validator(state: BeaconState, index: int) -> None:
//    validator = state.validator_registry[index]
//
//    if validator.exit_slot < state.slot + ENTRY_EXIT_DELAY:
//        return
//
//    validator.exit_slot = state.slot + ENTRY_EXIT_DELAY
//
//    state.validator_registry_exit_count += 1
//    validator.exit_count = state.validator_registry_exit_count
//    state.validator_registry_delta_chain_tip = hash_tree_root(
//        ValidatorRegistryDeltaBlock(
//            current_validator_registry_delta_chain_tip=state.validator_registry_delta_chain_tip,
//            validator_index=index,
//            pubkey=validator.pubkey,
//            slot=validator.exit_slot,
//            flag=EXIT,
//        )
//    )
func ExitValidator(state *pb.BeaconState, idx uint32) (*pb.BeaconState, error) {
	validator := state.ValidatorRegistry[idx]

	if validator.ExitSlot < state.Slot+config.EntryExitDelay {
		return nil, fmt.Errorf("validator %d could not exit until slot %d",
			idx, state.Slot+config.EntryExitDelay)
	}

	validator.ExitSlot = state.Slot + config.EntryExitDelay

	state.ValidatorRegistryExitCount++
	validator.ExitCount = state.ValidatorRegistryExitCount
	newChainTip, err := NewRegistryDeltaChainTip(
		pb.ValidatorRegistryDeltaBlock_EXIT,
		idx,
		validator.ExitSlot,
		validator.Pubkey,
		state.ValidatorRegistryDeltaChainTipHash32,
	)
	if err != nil {
		return nil, fmt.Errorf("could not get new chain tip %v", err)
	}
	state.ValidatorRegistryDeltaChainTipHash32 = newChainTip[:]

	return state, nil
}

// PenalizeValidator slashes the malicious validator's balance and awards
// the whistleblower's balance.
//
// Spec pseudocode definition:
// def penalize_validator(state: BeaconState, index: int) -> None:
//    exit_validator(state, index)
//    validator = state.validator_registry[index]
//    state.latest_penalized_exit_balances[(state.slot // EPOCH_LENGTH) % LATEST_PENALIZED_EXIT_LENGTH] += get_effective_balance(state, index)
//
//    whistleblower_index = get_beacon_proposer_index(state, state.slot)
//    whistleblower_reward = get_effective_balance(state, index) // WHISTLEBLOWER_REWARD_QUOTIENT
//    state.validator_balances[whistleblower_index] += whistleblower_reward
//    state.validator_balances[index] -= whistleblower_reward
//    validator.penalized_slot = state.slot
func PenalizeValidator(state *pb.BeaconState, idx uint32) (*pb.BeaconState, error) {
	state, err := ExitValidator(state, idx)
	if err != nil {
		return nil, fmt.Errorf("could not exit penalized validator: %v", err)
	}

	penalizedDuration := (state.Slot / config.EpochLength) %
		config.LatestPenalizedExitLength
	state.LatestPenalizedExitBalances[penalizedDuration] +=
		EffectiveBalance(state, idx)

	whistleblowerIdx, err := BeaconProposerIdx(state, state.Slot)
	if err != nil {
		return nil, fmt.Errorf("could not get proposer idx: %v", err)
	}
	whistleblowerReward := EffectiveBalance(state, idx) /
		config.WhistlerBlowerRewardQuotient

	state.ValidatorBalances[whistleblowerIdx] += whistleblowerReward
	state.ValidatorBalances[idx] -= whistleblowerReward

	state.ValidatorRegistry[idx].PenalizedSlot = state.Slot
	return state, nil
}

// PrepareValidatorForWithdrawal sets validator's status flag to
// WITHDRAWABLE.
//
// Spec pseudocode definition:
// def prepare_validator_for_withdrawal(state: BeaconState, index: int) -> None:
//    validator = state.validator_registry[index]
//    validator.status_flags |= WITHDRAWABLE
func PrepareValidatorForWithdrawal(state *pb.BeaconState, idx uint32) *pb.BeaconState {
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
//    # The active validators
//    active_validator_indices = get_active_validator_indices(state.validator_registry, state.slot)
//    # The total effective balance of active validators
//    total_balance = sum([get_effective_balance(state, i) for i in active_validator_indices])
//
//    # The maximum balance churn in Gwei (for deposits and exits separately)
//    max_balance_churn = max(
//        MAX_DEPOSIT * GWEI_PER_ETH,
//        total_balance // (2 * MAX_BALANCE_CHURN_QUOTIENT)
//    )
//
//    # Activate validators within the allowable balance churn
//    balance_churn = 0
//    for index, validator in enumerate(state.validator_registry):
//        if validator.activation_slot > state.slot + ENTRY_EXIT_DELAY and state.validator_balances[index] >= MAX_DEPOSIT * GWEI_PER_ETH:
//            # Check the balance churn would be within the allowance
//            balance_churn += get_effective_balance(state, index)
//            if balance_churn > max_balance_churn:
//                break
//
//            # Activate validator
//            activate_validator(state, index, False)
//
//    # Exit validators within the allowable balance churn
//    balance_churn = 0
//    for index, validator in enumerate(state.validator_registry):
//        if validator.exit_slot > state.slot + ENTRY_EXIT_DELAY and validator.status_flags & INITIATED_EXIT:
//            # Check the balance churn would be within the allowance
//            balance_churn += get_effective_balance(state, index)
//            if balance_churn > max_balance_churn:
//                break
//
//            # Exit validator
//            exit_validator(state, index)
//
//    state.validator_registry_latest_change_slot = state.slot
func UpdateRegistry(state *pb.BeaconState) (*pb.BeaconState, error) {
	activeValidatorIndices := ActiveValidatorIndices(
		state.ValidatorRegistry, state.Slot)

	totalBalance := TotalEffectiveBalance(state, activeValidatorIndices)

	// The maximum balance churn in Gwei (for deposits and exits separately).
	maxBalChurn := maxBalanceChurn(totalBalance)

	var balChurn uint64
	var err error
	for idx, validator := range state.ValidatorRegistry {
		// Activate validators within the allowable balance churn.
		if validator.ActivationSlot > state.Slot+config.EntryExitDelay &&
			state.ValidatorBalances[idx] >= config.MaxDepositInGwei {
			balChurn += EffectiveBalance(state, uint32(idx))
			if balChurn > maxBalChurn {
				break
			}
			state, err = ActivateValidator(state, uint32(idx), false)
			if err != nil {
				return nil, fmt.Errorf("could not activate validator %d: %v", idx, err)
			}
		}
	}

	balChurn = 0
	for idx, validator := range state.ValidatorRegistry {
		// Exit validators within the allowable balance churn.
		if validator.ExitSlot > state.Slot+config.EntryExitDelay &&
			validator.StatusFlags == pb.ValidatorRecord_INITIATED_EXIT {
			balChurn += EffectiveBalance(state, uint32(idx))
			if balChurn > maxBalChurn {
				break
			}
			state, err = ExitValidator(state, uint32(idx))
			if err != nil {
				return nil, fmt.Errorf("could not exit validator %d: %v", idx, err)
			}
		}
	}
	state.ValidatorRegistryLatestChangeSlot = state.Slot
	return state, nil
}

// ProcessPenaltiesAndExits prepares the validators and the penalized validators
// for withdrawal.
//
// Spec pseudocode definition:
// def process_penalties_and_exits(state: BeaconState) -> None:
//    # The active validators
//    active_validator_indices = get_active_validator_indices(state.validator_registry, state.slot)
//    # The total effective balance of active validators
//    total_balance = sum([get_effective_balance(state, i) for i in active_validator_indices])
//
//    for index, validator in enumerate(state.validator_registry):
//        if (state.slot // EPOCH_LENGTH) == (validator.penalized_slot // EPOCH_LENGTH) + LATEST_PENALIZED_EXIT_LENGTH // 2:
//            e = (state.slot // EPOCH_LENGTH) % LATEST_PENALIZED_EXIT_LENGTH
//            total_at_start = state.latest_penalized_exit_balances[(e + 1) % LATEST_PENALIZED_EXIT_LENGTH]
//            total_at_end = state.latest_penalized_exit_balances[e]
//            total_penalties = total_at_end - total_at_start
//            penalty = get_effective_balance(state, index) * min(total_penalties * 3, total_balance) // total_balance
//            state.validator_balances[index] -= penalty
//
//    all_indices = list(range(len(state.validator_registry)))
//    eligible_indices = filter(eligible, all_indices)
//    sorted_indices = sorted(eligible_indices, key=lambda index: state.validator_registry[index].exit_count)
//    withdrawn_so_far = 0
//    for index in sorted_indices:
//        prepare_validator_for_withdrawal(state, index)
//        withdrawn_so_far += 1
//        if withdrawn_so_far >= MAX_WITHDRAWALS_PER_EPOCH:
//            break
func ProcessPenaltiesAndExits(state *pb.BeaconState) *pb.BeaconState {
	activeValidatorIndices := ActiveValidatorIndices(state.ValidatorRegistry, state.Slot)
	totalBalance := TotalEffectiveBalance(state, activeValidatorIndices)

	for idx, validator := range state.ValidatorRegistry {
		penalized := validator.PenalizedSlot/config.EpochLength +
			config.LatestPenalizedExitLength/2
		if state.Slot/config.EpochLength == penalized {
			penalizedEpoch := (state.Slot / config.EpochLength) % config.LatestPenalizedExitLength
			penalizedEpochStart := (penalizedEpoch + 1) % config.LatestPenalizedExitLength
			totalAtStart := state.LatestPenalizedExitBalances[penalizedEpochStart]
			totalAtEnd := state.LatestPenalizedExitBalances[penalizedEpoch]
			totalPenalties := totalAtStart - totalAtEnd

			penaltyMultiplier := totalPenalties * 3
			if totalBalance < penaltyMultiplier {
				penaltyMultiplier = totalBalance
			}
			penalty := EffectiveBalance(state, uint32(idx)) *
				penaltyMultiplier / totalBalance
			state.ValidatorBalances[idx] -= penalty
		}
	}
	allIndices := AllValidatorsIndices(state)
	var eligibleIndices []uint32
	for _, idx := range allIndices {
		if eligibleToExit(state, idx) {
			eligibleIndices = append(eligibleIndices, idx)
		}
	}
	sort.Slice(eligibleIndices, func(i, j int) bool {
		return state.ValidatorRegistry[i].ExitCount < state.ValidatorRegistry[j].ExitCount
	})
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
	if maxBalanceChurn > config.MaxDepositInGwei {
		return maxBalanceChurn
	}
	return config.MaxDepositInGwei
}

// eligibleToExit checks if a validator is eligible to exit whether it was
// penalized or not.
//
// Spec pseudocode definition:
// def eligible(index):
//    validator = state.validator_registry[index]
//    if validator.penalized_slot <= state.slot:
//        PENALIZED_WITHDRAWAL_TIME = LATEST_PENALIZED_EXIT_LENGTH * EPOCH_LENGTH // 2
//        return state.slot >= validator.penalized_slot + PENALIZED_WITHDRAWAL_TIME
//    else:
//        return state.slot >= validator.exit_slot + MIN_VALIDATOR_WITHDRAWAL_TIME
func eligibleToExit(state *pb.BeaconState, idx uint32) bool {
	validator := state.ValidatorRegistry[idx]

	if validator.PenalizedSlot <= state.Slot {
		penalizedWithdrawalTime := config.LatestPenalizedExitLength * config.EpochLength / 2
		return state.Slot >= validator.PenalizedSlot+penalizedWithdrawalTime
	}
	return state.Slot >= validator.ExitSlot+config.MinValidatorWithdrawalTime
}
