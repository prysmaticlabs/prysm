// Package epoch contains epoch processing libraries according to spec, able to
// process new balance for validators, justify and finalize new
// check points, and shuffle validators to different slots and
// shards.
package epoch

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// sortableIndices implements the Sort interface to sort newly activated validator indices
// by activation epoch and by index number.
type sortableIndices struct {
	indices    []uint64
	validators []*ethpb.Validator
}

func (s sortableIndices) Len() int      { return len(s.indices) }
func (s sortableIndices) Swap(i, j int) { s.indices[i], s.indices[j] = s.indices[j], s.indices[i] }
func (s sortableIndices) Less(i, j int) bool {
	if s.validators[s.indices[i]].ActivationEligibilityEpoch == s.validators[s.indices[j]].ActivationEligibilityEpoch {
		return s.indices[i] < s.indices[j]
	}
	return s.validators[s.indices[i]].ActivationEligibilityEpoch < s.validators[s.indices[j]].ActivationEligibilityEpoch
}

// AttestingBalance returns the total balance from all the attesting indices.
//
// WARNING: This method allocates a new copy of the attesting validator indices set and is
// considered to be very memory expensive. Avoid using this unless you really
// need to get attesting balance from attestations.
//
// Spec pseudocode definition:
//  def get_attesting_balance(state: BeaconState, attestations: List[PendingAttestation]) -> Gwei:
//    return get_total_balance(state, get_unslashed_attesting_indices(state, attestations))
func AttestingBalance(state *stateTrie.BeaconState, atts []*pb.PendingAttestation) (uint64, error) {
	indices, err := UnslashedAttestingIndices(state, atts)
	if err != nil {
		return 0, errors.Wrap(err, "could not get attesting indices")
	}
	return helpers.TotalBalance(state, indices), nil
}

// ProcessRegistryUpdates rotates validators in and out of active pool.
// the amount to rotate is determined churn limit.
//
// Spec pseudocode definition:
//   def process_registry_updates(state: BeaconState) -> None:
//    # Process activation eligibility and ejections
//    for index, validator in enumerate(state.validators):
//        if is_eligible_for_activation_queue(validator):
//            validator.activation_eligibility_epoch = get_current_epoch(state) + 1
//
//        if is_active_validator(validator, get_current_epoch(state)) and validator.effective_balance <= EJECTION_BALANCE:
//            initiate_validator_exit(state, ValidatorIndex(index))
//
//    # Queue validators eligible for activation and not yet dequeued for activation
//    activation_queue = sorted([
//        index for index, validator in enumerate(state.validators)
//        if is_eligible_for_activation(state, validator)
//        # Order by the sequence of activation_eligibility_epoch setting and then index
//    ], key=lambda index: (state.validators[index].activation_eligibility_epoch, index))
//    # Dequeued validators for activation up to churn limit
//    for index in activation_queue[:get_validator_churn_limit(state)]:
//        validator = state.validators[index]
//        validator.activation_epoch = compute_activation_exit_epoch(get_current_epoch(state))
func ProcessRegistryUpdates(state *stateTrie.BeaconState) (*stateTrie.BeaconState, error) {
	currentEpoch := helpers.CurrentEpoch(state)
	vals := state.Validators()
	var err error
	ejectionBal := params.BeaconConfig().EjectionBalance
	activationEligibilityEpoch := helpers.CurrentEpoch(state) + 1
	for idx, validator := range vals {
		// Process the validators for activation eligibility.
		if helpers.IsEligibleForActivationQueue(validator) {
			validator.ActivationEligibilityEpoch = activationEligibilityEpoch
			if err := state.UpdateValidatorAtIndex(uint64(idx), validator); err != nil {
				return nil, err
			}
		}

		// Process the validators for ejection.
		isActive := helpers.IsActiveValidator(validator, currentEpoch)
		belowEjectionBalance := validator.EffectiveBalance <= ejectionBal
		if isActive && belowEjectionBalance {
			state, err = validators.InitiateValidatorExit(state, uint64(idx))
			if err != nil {
				return nil, errors.Wrapf(err, "could not initiate exit for validator %d", idx)
			}
		}
	}

	// Queue validators eligible for activation and not yet dequeued for activation.
	var activationQ []uint64
	for idx, validator := range vals {
		if helpers.IsEligibleForActivation(state, validator) {
			activationQ = append(activationQ, uint64(idx))
		}
	}

	sort.Sort(sortableIndices{indices: activationQ, validators: vals})

	// Only activate just enough validators according to the activation churn limit.
	limit := uint64(len(activationQ))
	activeValidatorCount, err := helpers.ActiveValidatorCount(state, currentEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get active validator count")
	}

	churnLimit, err := helpers.ValidatorChurnLimit(activeValidatorCount)
	if err != nil {
		return nil, errors.Wrap(err, "could not get churn limit")
	}

	// Prevent churn limit cause index out of bound.
	if churnLimit < limit {
		limit = churnLimit
	}

	activationExitEpoch := helpers.ActivationExitEpoch(currentEpoch)
	for _, index := range activationQ[:limit] {
		validator, err := state.ValidatorAtIndex(index)
		if err != nil {
			return nil, err
		}
		validator.ActivationEpoch = activationExitEpoch
		if err := state.UpdateValidatorAtIndex(index, validator); err != nil {
			return nil, err
		}
	}
	return state, nil
}

// ProcessSlashings processes the slashed validators during epoch processing,
//
//  def process_slashings(state: BeaconState) -> None:
//    epoch = get_current_epoch(state)
//    total_balance = get_total_active_balance(state)
//    for index, validator in enumerate(state.validators):
//        if validator.slashed and epoch + EPOCHS_PER_SLASHINGS_VECTOR // 2 == validator.withdrawable_epoch:
//            increment = EFFECTIVE_BALANCE_INCREMENT  # Factored out from penalty numerator to avoid uint64 overflow
//			  penalty_numerator = validator.effective_balance // increment * min(sum(state.slashings) * 3, total_balance)
//            penalty = penalty_numerator // total_balance * increment
//            decrease_balance(state, ValidatorIndex(index), penalty)
func ProcessSlashings(state *stateTrie.BeaconState) (*stateTrie.BeaconState, error) {
	currentEpoch := helpers.CurrentEpoch(state)
	totalBalance, err := helpers.TotalActiveBalance(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not get total active balance")
	}

	// Compute slashed balances in the current epoch
	exitLength := params.BeaconConfig().EpochsPerSlashingsVector

	// Compute the sum of state slashings
	slashings := state.Slashings()
	totalSlashing := uint64(0)
	for _, slashing := range slashings {
		totalSlashing += slashing
	}

	// a callback is used here to apply the following actions  to all validators
	// below equally.
	increment := params.BeaconConfig().EffectiveBalanceIncrement
	err = state.ApplyToEveryValidator(func(idx int, val *ethpb.Validator) (bool, error) {
		correctEpoch := (currentEpoch + exitLength/2) == val.WithdrawableEpoch
		if val.Slashed && correctEpoch {
			minSlashing := mathutil.Min(totalSlashing*3, totalBalance)
			penaltyNumerator := val.EffectiveBalance / increment * minSlashing
			penalty := penaltyNumerator / totalBalance * increment
			if err := helpers.DecreaseBalance(state, uint64(idx), penalty); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, nil
	})
	return state, err
}

// ProcessFinalUpdates processes the final updates during epoch processing.
//
// Spec pseudocode definition:
//  def process_final_updates(state: BeaconState) -> None:
//    current_epoch = get_current_epoch(state)
//    next_epoch = Epoch(current_epoch + 1)
//    # Reset eth1 data votes
//    if next_epoch % EPOCHS_PER_ETH1_VOTING_PERIOD == 0:
//        state.eth1_data_votes = []
//    # Update effective balances with hysteresis
//    for index, validator in enumerate(state.validators):
//        balance = state.balances[index]
//        HYSTERESIS_INCREMENT = EFFECTIVE_BALANCE_INCREMENT // HYSTERESIS_QUOTIENT
//        DOWNWARD_THRESHOLD = HYSTERESIS_INCREMENT * HYSTERESIS_DOWNWARD_MULTIPLIER
//        UPWARD_THRESHOLD = HYSTERESIS_INCREMENT * HYSTERESIS_UPWARD_MULTIPLIER
//        if (
//            balance + DOWNWARD_THRESHOLD < validator.effective_balance
//            or validator.effective_balance + UPWARD_THRESHOLD < balance
//        ):
//        validator.effective_balance = min(balance - balance % EFFECTIVE_BALANCE_INCREMENT, MAX_EFFECTIVE_BALANCE)
//    index_epoch = Epoch(next_epoch + ACTIVATION_EXIT_DELAY)
//    index_root_position = index_epoch % EPOCHS_PER_HISTORICAL_VECTOR
//    indices_list = List[ValidatorIndex, VALIDATOR_REGISTRY_LIMIT](get_active_validator_indices(state, index_epoch))
//    state.active_index_roots[index_root_position] = hash_tree_root(indices_list)
//    # Set committees root
//    committee_root_position = next_epoch % EPOCHS_PER_HISTORICAL_VECTOR
//    state.compact_committees_roots[committee_root_position] = get_compact_committees_root(state, next_epoch)
//    # Reset slashings
//    state.slashings[next_epoch % EPOCHS_PER_SLASHINGS_VECTOR] = Gwei(0)
//    # Set randao mix
//    state.randao_mixes[next_epoch % EPOCHS_PER_HISTORICAL_VECTOR] = get_randao_mix(state, current_epoch)
//    # Set historical root accumulator
//    if next_epoch % (SLOTS_PER_HISTORICAL_ROOT // SLOTS_PER_EPOCH) == 0:
//        historical_batch = HistoricalBatch(block_roots=state.block_roots, state_roots=state.state_roots)
//        state.historical_roots.append(hash_tree_root(historical_batch))
//    # Update start shard
//    state.start_shard = Shard((state.start_shard + get_shard_delta(state, current_epoch)) % SHARD_COUNT)
//    # Rotate current/previous epoch attestations
//    state.previous_epoch_attestations = state.current_epoch_attestations
//    state.current_epoch_attestations = []
func ProcessFinalUpdates(state *stateTrie.BeaconState) (*stateTrie.BeaconState, error) {
	currentEpoch := helpers.CurrentEpoch(state)
	nextEpoch := currentEpoch + 1

	// Reset ETH1 data votes.
	if nextEpoch%params.BeaconConfig().EpochsPerEth1VotingPeriod == 0 {
		if err := state.SetEth1DataVotes([]*ethpb.Eth1Data{}); err != nil {
			return nil, err
		}
	}

	effBalanceInc := params.BeaconConfig().EffectiveBalanceIncrement
	maxEffBalance := params.BeaconConfig().MaxEffectiveBalance
	hysteresisInc := effBalanceInc / params.BeaconConfig().HysteresisQuotient
	downwardThreshold := hysteresisInc * params.BeaconConfig().HysteresisDownwardMultiplier
	upwardThreshold := hysteresisInc * params.BeaconConfig().HysteresisUpwardMultiplier

	bals := state.Balances()
	// Update effective balances with hysteresis.
	validatorFunc := func(idx int, val *ethpb.Validator) (bool, error) {
		if val == nil {
			return false, fmt.Errorf("validator %d is nil in state", idx)
		}
		if idx >= len(bals) {
			return false, fmt.Errorf("validator index exceeds validator length in state %d >= %d", idx, len(state.Balances()))
		}
		balance := bals[idx]

		if balance+downwardThreshold < val.EffectiveBalance || val.EffectiveBalance+upwardThreshold < balance {
			val.EffectiveBalance = maxEffBalance
			if val.EffectiveBalance > balance-balance%effBalanceInc {
				val.EffectiveBalance = balance - balance%effBalanceInc
			}
			return true, nil
		}
		return false, nil
	}

	if err := state.ApplyToEveryValidator(validatorFunc); err != nil {
		return nil, err
	}

	// Set total slashed balances.
	slashedExitLength := params.BeaconConfig().EpochsPerSlashingsVector
	slashedEpoch := nextEpoch % slashedExitLength
	slashings := state.Slashings()
	if uint64(len(slashings)) != slashedExitLength {
		return nil, fmt.Errorf(
			"state slashing length %d different than EpochsPerHistoricalVector %d",
			len(slashings),
			slashedExitLength,
		)
	}
	if err := state.UpdateSlashingsAtIndex(slashedEpoch /* index */, 0 /* value */); err != nil {
		return nil, err
	}

	// Set RANDAO mix.
	randaoMixLength := params.BeaconConfig().EpochsPerHistoricalVector
	if uint64(state.RandaoMixesLength()) != randaoMixLength {
		return nil, fmt.Errorf(
			"state randao length %d different than EpochsPerHistoricalVector %d",
			state.RandaoMixesLength(),
			randaoMixLength,
		)
	}
	mix, err := helpers.RandaoMix(state, currentEpoch)
	if err != nil {
		return nil, err
	}
	if err := state.UpdateRandaoMixesAtIndex(nextEpoch%randaoMixLength, mix); err != nil {
		return nil, err
	}

	// Set historical root accumulator.
	epochsPerHistoricalRoot := params.BeaconConfig().SlotsPerHistoricalRoot / params.BeaconConfig().SlotsPerEpoch
	if nextEpoch%epochsPerHistoricalRoot == 0 {
		historicalBatch := &pb.HistoricalBatch{
			BlockRoots: state.BlockRoots(),
			StateRoots: state.StateRoots(),
		}
		batchRoot, err := historicalBatch.HashTreeRoot()
		if err != nil {
			return nil, errors.Wrap(err, "could not hash historical batch")
		}
		if err := state.AppendHistoricalRoots(batchRoot); err != nil {
			return nil, err
		}
	}

	// Rotate current and previous epoch attestations.
	if err := state.SetPreviousEpochAttestations(state.CurrentEpochAttestations()); err != nil {
		return nil, err
	}
	if err := state.SetCurrentEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		return nil, err
	}
	return state, nil
}

// UnslashedAttestingIndices returns all the attesting indices from a list of attestations,
// it sorts the indices and filters out the slashed ones.
//
// Spec pseudocode definition:
//  def get_unslashed_attesting_indices(state: BeaconState,
//                                    attestations: Sequence[PendingAttestation]) -> Set[ValidatorIndex]:
//    output = set()  # type: Set[ValidatorIndex]
//    for a in attestations:
//        output = output.union(get_attesting_indices(state, a.data, a.aggregation_bits))
//    return set(filter(lambda index: not state.validators[index].slashed, output))
func UnslashedAttestingIndices(state *stateTrie.BeaconState, atts []*pb.PendingAttestation) ([]uint64, error) {
	var setIndices []uint64
	seen := make(map[uint64]bool)

	for _, att := range atts {
		committee, err := helpers.BeaconCommitteeFromState(state, att.Data.Slot, att.Data.CommitteeIndex)
		if err != nil {
			return nil, err
		}
		attestingIndices := attestationutil.AttestingIndices(att.AggregationBits, committee)
		// Create a set for attesting indices
		for _, index := range attestingIndices {
			if !seen[index] {
				setIndices = append(setIndices, index)
			}
			seen[index] = true
		}
	}
	// Sort the attesting set indices by increasing order.
	sort.Slice(setIndices, func(i, j int) bool { return setIndices[i] < setIndices[j] })
	// Remove the slashed validator indices.
	for i := 0; i < len(setIndices); i++ {
		v, err := state.ValidatorAtIndexReadOnly(setIndices[i])
		if err != nil {
			return nil, errors.Wrap(err, "failed to look up validator")
		}
		if v != nil && v.Slashed() {
			setIndices = append(setIndices[:i], setIndices[i+1:]...)
		}
	}

	return setIndices, nil
}

// BaseReward takes state and validator index and calculate
// individual validator's base reward quotient.
//
// Note: Adjusted quotient is calculated of base reward because it's too inefficient
// to repeat the same calculation for every validator versus just doing it once.
//
// Spec pseudocode definition:
//  def get_base_reward(state: BeaconState, index: ValidatorIndex) -> Gwei:
//      total_balance = get_total_active_balance(state)
//	    effective_balance = state.validator_registry[index].effective_balance
//	    return effective_balance * BASE_REWARD_FACTOR // integer_squareroot(total_balance) // BASE_REWARDS_PER_EPOCH
func BaseReward(state *stateTrie.BeaconState, index uint64) (uint64, error) {
	totalBalance, err := helpers.TotalActiveBalance(state)
	if err != nil {
		return 0, errors.Wrap(err, "could not calculate active balance")
	}
	val, err := state.ValidatorAtIndexReadOnly(index)
	if err != nil {
		return 0, err
	}
	effectiveBalance := val.EffectiveBalance()
	baseReward := effectiveBalance * params.BeaconConfig().BaseRewardFactor /
		mathutil.IntegerSquareRoot(totalBalance) / params.BeaconConfig().BaseRewardsPerEpoch
	return baseReward, nil
}
