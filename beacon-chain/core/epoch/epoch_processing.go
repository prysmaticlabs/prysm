// Package epoch contains epoch processing libraries. These libraries
// process new balance for the validators, justify and finalize new
// check points, shuffle and reassign validators to different slots and
// shards.
package epoch

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// MatchedAttestations is an object that contains the correctly
// voted attestations based on source, target and head criteria.
type MatchedAttestations struct {
	source []*pb.PendingAttestation
	Target []*pb.PendingAttestation
	head   []*pb.PendingAttestation
}

// MatchAttestations matches the attestations gathered in a span of an epoch
// and categorize them whether they correctly voted for source, target and head.
// We combined the individual helpers from spec for efficiency and to achieve O(N) run time.
//
// Spec pseudocode definition:
//  def get_matching_source_attestations(state: BeaconState, epoch: Epoch) -> List[PendingAttestation]:
//    assert epoch in (get_current_epoch(state), get_previous_epoch(state))
//    return state.current_epoch_attestations if epoch == get_current_epoch(state) else state.previous_epoch_attestations
//
//  def get_matching_target_attestations(state: BeaconState, epoch: Epoch) -> List[PendingAttestation]:
//    return [
//        a for a in get_matching_source_attestations(state, epoch)
//        if a.data.target_root == get_block_root(state, epoch)
//    ]
//
//  def get_matching_head_attestations(state: BeaconState, epoch: Epoch) -> List[PendingAttestation]:
//    return [
//        a for a in get_matching_source_attestations(state, epoch)
//        if a.data.beacon_block_root == get_block_root_at_slot(state, get_attestation_data_slot(state, a.data))
//    ]
func MatchAttestations(state *pb.BeaconState, epoch uint64) (*MatchedAttestations, error) {
	currentEpoch := helpers.CurrentEpoch(state)
	previousEpoch := helpers.PrevEpoch(state)

	// Input epoch for matching the source attestations has to be within range
	// of current epoch & previous epoch.
	if epoch != currentEpoch && epoch != previousEpoch {
		return nil, fmt.Errorf("input epoch: %d != current epoch: %d or previous epoch: %d",
			epoch, currentEpoch, previousEpoch)
	}

	// Decide if the source attestations are coming from current or previous epoch.
	var srcAtts []*pb.PendingAttestation
	if epoch == currentEpoch {
		srcAtts = state.CurrentEpochAttestations
	} else {
		srcAtts = state.PreviousEpochAttestations
	}
	targetRoot, err := helpers.BlockRoot(state, epoch)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get block root for epoch %d", epoch)
	}

	tgtAtts := make([]*pb.PendingAttestation, 0, len(srcAtts))
	headAtts := make([]*pb.PendingAttestation, 0, len(srcAtts))
	for _, srcAtt := range srcAtts {
		// If the target root matches attestation's target root,
		// then we know this attestation has correctly voted for target.
		if bytes.Equal(srcAtt.Data.Target.Root, targetRoot) {
			tgtAtts = append(tgtAtts, srcAtt)
		}
		headRoot, err := helpers.BlockRootAtSlot(state, srcAtt.Data.Slot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get block root for slot %d", srcAtt.Data.Slot)
		}
		if bytes.Equal(srcAtt.Data.BeaconBlockRoot, headRoot) {
			headAtts = append(headAtts, srcAtt)
		}
	}

	return &MatchedAttestations{
		source: srcAtts,
		Target: tgtAtts,
		head:   headAtts,
	}, nil
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
func AttestingBalance(state *pb.BeaconState, atts []*pb.PendingAttestation) (uint64, error) {
	indices, err := unslashedAttestingIndices(state, atts)
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
//    for index, validator in enumerate(state.validator_registry):
//        if (
//            validator.activation_eligibility_epoch == FAR_FUTURE_EPOCH and
//            validator.effective_balance >= MAX_EFFECTIVE_BALANCE
//        ):
//            validator.activation_eligibility_epoch = get_current_epoch(state)
//
//        if is_active_validator(validator, get_current_epoch(state)) and validator.effective_balance <= EJECTION_BALANCE:
//            initiate_validator_exit(state, index)
//
//    # Queue validators eligible for activation and not dequeued for activation prior to finalized epoch
//    activation_queue = sorted([
//        index for index, validator in enumerate(state.validator_registry) if
//        validator.activation_eligibility_epoch != FAR_FUTURE_EPOCH and
//        validator.activation_epoch >= get_delayed_activation_exit_epoch(state.finalized_epoch)
//    ], key=lambda index: state.validator_registry[index].activation_eligibility_epoch)
//    # Dequeued validators for activation up to churn limit (without resetting activation epoch)
//    for index in activation_queue[:get_churn_limit(state)]:
//        validator = state.validator_registry[index]
//        if validator.activation_epoch == FAR_FUTURE_EPOCH:
//            validator.activation_epoch = get_delayed_activation_exit_epoch(get_current_epoch(state))
func ProcessRegistryUpdates(state *pb.BeaconState) (*pb.BeaconState, error) {
	currentEpoch := helpers.CurrentEpoch(state)

	var err error
	for idx, validator := range state.Validators {
		// Process the validators for activation eligibility.
		eligibleToActivate := validator.ActivationEligibilityEpoch == params.BeaconConfig().FarFutureEpoch
		properBalance := validator.EffectiveBalance >= params.BeaconConfig().MaxEffectiveBalance
		if eligibleToActivate && properBalance {
			validator.ActivationEligibilityEpoch = currentEpoch
		}
		// Process the validators for ejection.
		isActive := helpers.IsActiveValidator(validator, currentEpoch)
		belowEjectionBalance := validator.EffectiveBalance <= params.BeaconConfig().EjectionBalance
		if isActive && belowEjectionBalance {
			state, err = validators.InitiateValidatorExit(state, uint64(idx))
			if err != nil {
				return nil, errors.Wrapf(err, "could not initiate exit for validator %d", idx)
			}
		}
	}

	// Queue the validators whose eligible to activate and sort them by activation eligibility epoch number
	var activationQ []uint64
	for idx, validator := range state.Validators {
		eligibleActivated := validator.ActivationEligibilityEpoch != params.BeaconConfig().FarFutureEpoch
		canBeActive := validator.ActivationEpoch >= helpers.DelayedActivationExitEpoch(state.FinalizedCheckpoint.Epoch)
		if eligibleActivated && canBeActive {
			activationQ = append(activationQ, uint64(idx))
		}
	}
	sort.Slice(activationQ, func(i, j int) bool {
		return state.Validators[i].ActivationEligibilityEpoch < state.Validators[j].ActivationEligibilityEpoch
	})

	// Only activate just enough validators according to the activation churn limit.
	limit := len(activationQ)
	activeValidatorCount, err := helpers.ActiveValidatorCount(state, currentEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get active validator count")
	}

	churnLimit, err := helpers.ValidatorChurnLimit(activeValidatorCount)
	if err != nil {
		return nil, errors.Wrap(err, "could not get churn limit")
	}

	// Prevent churn limit cause index out of bound.
	if int(churnLimit) < limit {
		limit = int(churnLimit)
	}
	for _, index := range activationQ[:limit] {
		validator := state.Validators[index]
		if validator.ActivationEpoch == params.BeaconConfig().FarFutureEpoch {
			validator.ActivationEpoch = helpers.DelayedActivationExitEpoch(currentEpoch)
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
func ProcessSlashings(state *pb.BeaconState) (*pb.BeaconState, error) {
	currentEpoch := helpers.CurrentEpoch(state)
	totalBalance, err := helpers.TotalActiveBalance(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not get total active balance")
	}

	// Compute slashed balances in the current epoch
	exitLength := params.BeaconConfig().EpochsPerSlashingsVector

	// Compute the sum of state slashings
	totalSlashing := uint64(0)
	for _, slashing := range state.Slashings {
		totalSlashing += slashing
	}

	// Compute slashing for each validator.
	for index, validator := range state.Validators {
		correctEpoch := (currentEpoch + exitLength/2) == validator.WithdrawableEpoch
		if validator.Slashed && correctEpoch {
			minSlashing := mathutil.Min(totalSlashing*3, totalBalance)
			increment := params.BeaconConfig().EffectiveBalanceIncrement
			penaltyNumerator := validator.EffectiveBalance / increment * minSlashing
			penalty := penaltyNumerator / totalBalance * increment
			state = helpers.DecreaseBalance(state, uint64(index), penalty)
		}
	}
	return state, err
}

// ProcessFinalUpdates processes the final updates during epoch processing.
//
// Spec pseudocode definition:
//  def process_final_updates(state: BeaconState) -> None:
//    current_epoch = get_current_epoch(state)
//    next_epoch = Epoch(current_epoch + 1)
//    # Reset eth1 data votes
//    if (state.slot + 1) % SLOTS_PER_ETH1_VOTING_PERIOD == 0:
//        state.eth1_data_votes = []
//    # Update effective balances with hysteresis
//    for index, validator in enumerate(state.validators):
//        balance = state.balances[index]
//        HALF_INCREMENT = EFFECTIVE_BALANCE_INCREMENT // 2
//        if balance < validator.effective_balance or validator.effective_balance + 3 * HALF_INCREMENT < balance:
//            validator.effective_balance = min(balance - balance % EFFECTIVE_BALANCE_INCREMENT, MAX_EFFECTIVE_BALANCE)
//    # Set active index root
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
func ProcessFinalUpdates(state *pb.BeaconState) (*pb.BeaconState, error) {
	currentEpoch := helpers.CurrentEpoch(state)
	nextEpoch := currentEpoch + 1

	// Reset ETH1 data votes.
	if (state.Slot+1)%params.BeaconConfig().SlotsPerEth1VotingPeriod == 0 {
		state.Eth1DataVotes = []*ethpb.Eth1Data{}
	}

	// Update effective balances with hysteresis.
	for i, v := range state.Validators {
		if v == nil {
			return nil, fmt.Errorf("validator %d is nil in state", i)
		}
		if i >= len(state.Balances) {
			return nil, fmt.Errorf("validator index exceeds validator length in state %d >= %d", i, len(state.Balances))
		}
		balance := state.Balances[i]
		halfInc := params.BeaconConfig().EffectiveBalanceIncrement / 2
		if balance < v.EffectiveBalance || v.EffectiveBalance+3*halfInc < balance {
			v.EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance
			if v.EffectiveBalance > balance-balance%params.BeaconConfig().EffectiveBalanceIncrement {
				v.EffectiveBalance = balance - balance%params.BeaconConfig().EffectiveBalanceIncrement
			}
		}
	}

	// Set total slashed balances.
	slashedExitLength := params.BeaconConfig().EpochsPerSlashingsVector
	slashedEpoch := int(nextEpoch % slashedExitLength)
	if slashedEpoch >= len(state.Slashings) {
		return nil, fmt.Errorf("slashing epoch exceeds slashing length in state %d >= %d", slashedEpoch, len(state.Slashings))
	}
	state.Slashings[slashedEpoch] = 0

	// Set RANDAO mix.
	randaoMixLength := params.BeaconConfig().EpochsPerHistoricalVector
	if int(currentEpoch) >= len(state.RandaoMixes) {
		return nil, fmt.Errorf("current epoch exceeds randao length in state %d >= %d", currentEpoch, len(state.RandaoMixes))
	}
	mix := helpers.RandaoMix(state, currentEpoch)
	state.RandaoMixes[nextEpoch%randaoMixLength] = mix

	// Set historical root accumulator.
	epochsPerHistoricalRoot := params.BeaconConfig().SlotsPerHistoricalRoot / params.BeaconConfig().SlotsPerEpoch
	if nextEpoch%epochsPerHistoricalRoot == 0 {
		historicalBatch := &pb.HistoricalBatch{
			BlockRoots: state.BlockRoots,
			StateRoots: state.StateRoots,
		}
		batchRoot, err := ssz.HashTreeRoot(historicalBatch)
		if err != nil {
			return nil, errors.Wrap(err, "could not hash historical batch")
		}
		state.HistoricalRoots = append(state.HistoricalRoots, batchRoot[:])
	}

	// Rotate current and previous epoch attestations.
	state.PreviousEpochAttestations = state.CurrentEpochAttestations
	state.CurrentEpochAttestations = []*pb.PendingAttestation{}

	return state, nil
}

// unslashedAttestingIndices returns all the attesting indices from a list of attestations,
// it sorts the indices and filters out the slashed ones.
//
// Spec pseudocode definition:
//  def get_unslashed_attesting_indices(state: BeaconState,
//                                    attestations: Sequence[PendingAttestation]) -> Set[ValidatorIndex]:
//    output = set()  # type: Set[ValidatorIndex]
//    for a in attestations:
//        output = output.union(get_attesting_indices(state, a.data, a.aggregation_bits))
//    return set(filter(lambda index: not state.validators[index].slashed, output))
func unslashedAttestingIndices(state *pb.BeaconState, atts []*pb.PendingAttestation) ([]uint64, error) {
	var setIndices []uint64
	seen := make(map[uint64]bool)
	for _, att := range atts {
		committee, err := helpers.BeaconCommittee(state, att.Data.Slot, att.Data.CommitteeIndex)
		if err != nil {
			return nil, err
		}
		attestingIndices, err := helpers.AttestingIndices(att.AggregationBits, committee)
		if err != nil {
			return nil, errors.Wrap(err, "could not get attester indices")
		}
		// Create a set for attesting indices
		set := make([]uint64, 0, len(attestingIndices))
		for _, index := range attestingIndices {
			if !seen[index] {
				set = append(set, index)
			}
			seen[index] = true
		}
		setIndices = append(setIndices, set...)
	}
	// Sort the attesting set indices by increasing order.
	sort.Slice(setIndices, func(i, j int) bool { return setIndices[i] < setIndices[j] })
	// Remove the slashed validator indices.
	for i := 0; i < len(setIndices); i++ {
		if state.Validators[setIndices[i]].Slashed {
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
func BaseReward(state *pb.BeaconState, index uint64) (uint64, error) {
	totalBalance, err := helpers.TotalActiveBalance(state)
	if err != nil {
		return 0, errors.Wrap(err, "could not calculate active balance")
	}
	effectiveBalance := state.Validators[index].EffectiveBalance
	baseReward := effectiveBalance * params.BeaconConfig().BaseRewardFactor /
		mathutil.IntegerSquareRoot(totalBalance) / params.BeaconConfig().BaseRewardsPerEpoch
	return baseReward, nil
}
