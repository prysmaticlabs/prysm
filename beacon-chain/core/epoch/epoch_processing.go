// Package epoch contains epoch processing libraries. These libraries
// process new balance for the validators, justify and finalize new
// check points, shuffle and reassign validators to different slots and
// shards.
package epoch

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
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

		// If the block root at slot matches attestation's block root at slot,
		// then we know this attestation has correctly voted for head.
		slot, err := helpers.AttestationDataSlot(state, srcAtt.Data)
		if err != nil {
			return nil, errors.Wrap(err, "could not get attestation slot")
		}
		headRoot, err := helpers.BlockRootAtSlot(state, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get block root for slot %d", slot)
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

// ProcessJustificationAndFinalization processes justification and finalization during
// epoch processing. This is where a beacon node can justify and finalize a new epoch.
//
// Spec pseudocode definition:
//   def process_justification_and_finalization(state: BeaconState) -> None:
//      if get_current_epoch(state) <= GENESIS_EPOCH + 1:
//          return
//
//      previous_epoch = get_previous_epoch(state)
//      current_epoch = get_current_epoch(state)
//      old_previous_justified_checkpoint = state.previous_justified_checkpoint
//      old_current_justified_checkpoint = state.current_justified_checkpoint
//
//      # Process justifications
//      state.previous_justified_checkpoint = state.current_justified_checkpoint
//      state.justification_bits[1:] = state.justification_bits[:-1]
//      state.justification_bits[0] = 0b0
//      matching_target_attestations = get_matching_target_attestations(state, previous_epoch)  # Previous epoch
//      if get_attesting_balance(state, matching_target_attestations) * 3 >= get_total_active_balance(state) * 2:
//          state.current_justified_checkpoint = Checkpoint(epoch=previous_epoch,
//                                                          root=get_block_root(state, previous_epoch))
//          state.justification_bits[1] = 0b1
//      matching_target_attestations = get_matching_target_attestations(state, current_epoch)  # Current epoch
//      if get_attesting_balance(state, matching_target_attestations) * 3 >= get_total_active_balance(state) * 2:
//          state.current_justified_checkpoint = Checkpoint(epoch=current_epoch,
//                                                          root=get_block_root(state, current_epoch))
//          state.justification_bits[0] = 0b1
//
//      # Process finalizations
//      bits = state.justification_bits
//      # The 2nd/3rd/4th most recent epochs are justified, the 2nd using the 4th as source
//      if all(bits[1:4]) and old_previous_justified_checkpoint.epoch + 3 == current_epoch:
//          state.finalized_checkpoint = old_previous_justified_checkpoint
//      # The 2nd/3rd most recent epochs are justified, the 2nd using the 3rd as source
//      if all(bits[1:3]) and old_previous_justified_checkpoint.epoch + 2 == current_epoch:
//          state.finalized_checkpoint = old_previous_justified_checkpoint
//      # The 1st/2nd/3rd most recent epochs are justified, the 1st using the 3rd as source
//      if all(bits[0:3]) and old_current_justified_checkpoint.epoch + 2 == current_epoch:
//          state.finalized_checkpoint = old_current_justified_checkpoint
//      # The 1st/2nd most recent epochs are justified, the 1st using the 2nd as source
//      if all(bits[0:2]) and old_current_justified_checkpoint.epoch + 1 == current_epoch:
//          state.finalized_checkpoint = old_current_justified_checkpoint
func ProcessJustificationAndFinalization(state *pb.BeaconState, prevAttestedBal uint64, currAttestedBal uint64) (*pb.BeaconState, error) {
	if state.Slot <= helpers.StartSlot(2) {
		return state, nil
	}

	prevEpoch := helpers.PrevEpoch(state)
	currentEpoch := helpers.CurrentEpoch(state)
	oldPrevJustifiedCheckpoint := state.PreviousJustifiedCheckpoint
	oldCurrJustifiedCheckpoint := state.CurrentJustifiedCheckpoint

	totalBal, err := helpers.TotalActiveBalance(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not get total balance")
	}

	// Process justifications
	state.PreviousJustifiedCheckpoint = state.CurrentJustifiedCheckpoint
	state.JustificationBits.Shift(1)

	// Note: the spec refers to the bit index position starting at 1 instead of starting at zero.
	// We will use that paradigm here for consistency with the godoc spec definition.

	// If 2/3 or more of total balance attested in the previous epoch.
	if 3*prevAttestedBal >= 2*totalBal {
		blockRoot, err := helpers.BlockRoot(state, prevEpoch)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get block root for previous epoch %d", prevEpoch)
		}
		state.CurrentJustifiedCheckpoint = &ethpb.Checkpoint{Epoch: prevEpoch, Root: blockRoot}
		state.JustificationBits.SetBitAt(1, true)
	}

	// If 2/3 or more of the total balance attested in the current epoch.
	if 3*currAttestedBal >= 2*totalBal {
		blockRoot, err := helpers.BlockRoot(state, currentEpoch)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get block root for current epoch %d", prevEpoch)
		}
		state.CurrentJustifiedCheckpoint = &ethpb.Checkpoint{Epoch: currentEpoch, Root: blockRoot}
		state.JustificationBits.SetBitAt(0, true)
	}

	// Process finalization according to ETH2.0 specifications.
	justification := state.JustificationBits.Bytes()[0]

	// 2nd/3rd/4th (0b1110) most recent epochs are justified, the 2nd using the 4th as source.
	if justification&0x0E == 0x0E && (oldPrevJustifiedCheckpoint.Epoch+3) == currentEpoch {
		state.FinalizedCheckpoint = oldPrevJustifiedCheckpoint
	}

	// 2nd/3rd (0b0110) most recent epochs are justified, the 2nd using the 3rd as source.
	if justification&0x06 == 0x06 && (oldPrevJustifiedCheckpoint.Epoch+2) == currentEpoch {
		state.FinalizedCheckpoint = oldPrevJustifiedCheckpoint
	}

	// 1st/2nd/3rd (0b0111) most recent epochs are justified, the 1st using the 3rd as source.
	if justification&0x07 == 0x07 && (oldCurrJustifiedCheckpoint.Epoch+2) == currentEpoch {
		state.FinalizedCheckpoint = oldCurrJustifiedCheckpoint
	}

	// The 1st/2nd (0b0011) most recent epochs are justified, the 1st using the 2nd as source
	if justification&0x03 == 0x03 && (oldCurrJustifiedCheckpoint.Epoch+1) == currentEpoch {
		state.FinalizedCheckpoint = oldCurrJustifiedCheckpoint
	}

	return state, nil
}

// ProcessCrosslinks processes crosslink and finds the crosslink
// with enough state to make it canonical in state.
//
// Spec pseudocode definition:
//  def process_crosslinks(state: BeaconState) -> None:
//    state.previous_crosslinks = [c for c in state.current_crosslinks]
//    for epoch in (get_previous_epoch(state), get_current_epoch(state)):
//        for offset in range(get_epoch_committee_count(state, epoch)):
//            shard = (get_epoch_start_shard(state, epoch) + offset) % SHARD_COUNT
//            crosslink_committee = get_crosslink_committee(state, epoch, shard)
//            winning_crosslink, attesting_indices = get_winning_crosslink_and_attesting_indices(state, epoch, shard)
//            if 3 * get_total_balance(state, attesting_indices) >= 2 * get_total_balance(state, crosslink_committee):
//                state.current_crosslinks[shard] = winning_crosslink
func ProcessCrosslinks(state *pb.BeaconState) (*pb.BeaconState, error) {
	copy(state.PreviousCrosslinks, state.CurrentCrosslinks)
	epochs := []uint64{helpers.PrevEpoch(state), helpers.CurrentEpoch(state)}
	for _, e := range epochs {
		count, err := helpers.CommitteeCount(state, e)
		if err != nil {
			return nil, errors.Wrap(err, "could not get epoch committee count")
		}
		startShard, err := helpers.StartShard(state, e)
		if err != nil {
			return nil, errors.Wrap(err, "could not get epoch start shards")
		}
		for offset := uint64(0); offset < count; offset++ {
			shard := (startShard + offset) % params.BeaconConfig().ShardCount
			committee, err := helpers.CrosslinkCommittee(state, e, shard)
			if err != nil {
				return nil, errors.Wrap(err, "could not get crosslink committee")
			}
			crosslink, indices, err := winningCrosslink(state, shard, e)
			if err != nil {
				return nil, errors.Wrap(err, "could not get winning crosslink")
			}
			attestedBalance := helpers.TotalBalance(state, indices)
			totalBalance := helpers.TotalBalance(state, committee)

			// In order for a crosslink to get included in state, the attesting balance needs to
			// be greater than 2/3 of the total balance.
			if 3*attestedBalance >= 2*totalBalance {
				state.CurrentCrosslinks[shard] = crosslink
			}
		}
	}
	return state, nil
}

// ProcessRewardsAndPenalties processes the rewards and penalties of individual validator.
//
// Spec pseudocode definition:
//  def process_rewards_and_penalties(state: BeaconState) -> None:
//    if get_current_epoch(state) == GENESIS_EPOCH:
//        return
//
//    rewards1, penalties1 = get_attestation_deltas(state)
//    rewards2, penalties2 = get_crosslink_deltas(state)
//    for i in range(len(state.validator_registry)):
//        increase_balance(state, i, rewards1[i] + rewards2[i])
//        decrease_balance(state, i, penalties1[i] + penalties2[i])
func ProcessRewardsAndPenalties(state *pb.BeaconState) (*pb.BeaconState, error) {
	// Can't process rewards and penalties in genesis epoch.
	if helpers.CurrentEpoch(state) == 0 {
		return state, nil
	}
	attsRewards, attsPenalties, err := attestationDelta(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attestation delta")
	}
	clRewards, clPenalties, err := crosslinkDelta(state)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get crosslink delta")
	}

	for i := 0; i < len(state.Validators); i++ {
		state = helpers.IncreaseBalance(state, uint64(i), attsRewards[i]+clRewards[i])
		state = helpers.DecreaseBalance(state, uint64(i), attsPenalties[i]+clPenalties[i])
	}
	return state, nil
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
	churnLimit, err := helpers.ValidatorChurnLimit(state)
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
		balance := state.Balances[i]
		halfInc := params.BeaconConfig().EffectiveBalanceIncrement / 2
		if balance < v.EffectiveBalance || v.EffectiveBalance+3*halfInc < balance {
			v.EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance
			if v.EffectiveBalance > balance-balance%params.BeaconConfig().EffectiveBalanceIncrement {
				v.EffectiveBalance = balance - balance%params.BeaconConfig().EffectiveBalanceIncrement
			}
		}
	}

	// Set active index root.
	//    index_epoch = Epoch(next_epoch + ACTIVATION_EXIT_DELAY)
	//    index_root_position = index_epoch % EPOCHS_PER_HISTORICAL_VECTOR
	//    indices_list = List[ValidatorIndex, VALIDATOR_REGISTRY_LIMIT](get_active_validator_indices(state, index_epoch))
	//    state.active_index_roots[index_root_position] = hash_tree_root(indices_list)
	activationDelay := params.BeaconConfig().ActivationExitDelay
	idxRootPosition := (nextEpoch + activationDelay) % params.BeaconConfig().EpochsPerHistoricalVector
	activeIndices, err := helpers.ActiveValidatorIndices(state, nextEpoch+activationDelay)
	if err != nil {
		return nil, errors.Wrap(err, "could not get active indices")
	}
	idxRoot, err := ssz.HashTreeRootWithCapacity(activeIndices, uint64(1099511627776))
	if err != nil {
		return nil, errors.Wrap(err, "could not tree hash active indices")
	}
	state.ActiveIndexRoots[idxRootPosition] = idxRoot[:]

	commRootPosition := nextEpoch % params.BeaconConfig().EpochsPerHistoricalVector
	comRoot, err := helpers.CompactCommitteesRoot(state, nextEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get compact committee root")
	}
	state.CompactCommitteesRoots[commRootPosition] = comRoot[:]

	// Set total slashed balances.
	slashedExitLength := params.BeaconConfig().EpochsPerSlashingsVector
	state.Slashings[nextEpoch%slashedExitLength] = 0

	// Set RANDAO mix.
	randaoMixLength := params.BeaconConfig().EpochsPerHistoricalVector
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

	// Update start shard.
	delta, err := helpers.ShardDelta(state, currentEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get shard delta")
	}
	state.StartShard = (state.StartShard + delta) % params.BeaconConfig().ShardCount

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
		attestingIndices, err := helpers.AttestingIndices(state, att.Data, att.AggregationBits)
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

// winningCrosslink returns the most staked balance-wise crosslink of a given shard and epoch.
// It also returns the attesting inaidces of the winning cross link.
//
// Spec pseudocode definition:
//  def get_winning_crosslink_and_attesting_indices(state: BeaconState,
//                                                epoch: Epoch,
//                                                shard: Shard) -> Tuple[Crosslink, List[ValidatorIndex]]:
//    attestations = [a for a in get_matching_source_attestations(state, epoch) if a.data.crosslink.shard == shard]
//    crosslinks = list(filter(
//        lambda c: hash_tree_root(state.current_crosslinks[shard]) in (c.parent_root, hash_tree_root(c)),
//        [a.data.crosslink for a in attestations]
//    ))
//    # Winning crosslink has the crosslink data root with the most balance voting for it (ties broken lexicographically)
//    winning_crosslink = max(crosslinks, key=lambda c: (
//        get_attesting_balance(state, [a for a in attestations if a.data.crosslink == c]), c.data_root
//    ), default=Crosslink())
//    winning_attestations = [a for a in attestations if a.data.crosslink == winning_crosslink]
//    return winning_crosslink, get_unslashed_attesting_indices(state, winning_attestations)
func winningCrosslink(state *pb.BeaconState, shard uint64, epoch uint64) (*ethpb.Crosslink, []uint64, error) {
	var shardAtts []*pb.PendingAttestation
	matchedAtts, err := MatchAttestations(state, epoch)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get matching attestations")
	}

	// Filter out source attestations by shard.
	for _, att := range matchedAtts.source {
		if att.Data.Crosslink.Shard == shard {
			shardAtts = append(shardAtts, att)
		}
	}
	var candidateCrosslinks []*ethpb.Crosslink
	// Filter out shard crosslinks with correct current or previous crosslink data.
	for _, a := range shardAtts {
		stateCrosslink := state.CurrentCrosslinks[shard]
		stateCrosslinkRoot, err := ssz.HashTreeRoot(stateCrosslink)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not hash tree root crosslink from state")
		}
		attCrosslinkRoot, err := ssz.HashTreeRoot(a.Data.Crosslink)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not hash tree root crosslink from attestation")
		}
		currCrosslinkMatches := bytes.Equal(stateCrosslinkRoot[:], attCrosslinkRoot[:])
		prevCrosslinkMatches := bytes.Equal(stateCrosslinkRoot[:], a.Data.Crosslink.ParentRoot)
		if currCrosslinkMatches || prevCrosslinkMatches {
			candidateCrosslinks = append(candidateCrosslinks, a.Data.Crosslink)
		}
	}

	if len(candidateCrosslinks) == 0 {
		return &ethpb.Crosslink{
			DataRoot:   params.BeaconConfig().ZeroHash[:],
			ParentRoot: params.BeaconConfig().ZeroHash[:],
		}, nil, nil
	}
	var crosslinkAtts []*pb.PendingAttestation
	var winnerBalance uint64
	var winnerCrosslink *ethpb.Crosslink
	// Out of the existing shard crosslinks, pick the one that has the
	// most balance staked.
	crosslinkAtts = attsForCrosslink(candidateCrosslinks[0], shardAtts)
	winnerBalance, err = AttestingBalance(state, crosslinkAtts)
	if err != nil {
		return nil, nil, err
	}

	winnerCrosslink = candidateCrosslinks[0]
	for _, c := range candidateCrosslinks {
		crosslinkAtts = attsForCrosslink(c, shardAtts)
		attestingBalance, err := AttestingBalance(state, crosslinkAtts)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not get crosslink's attesting balance")
		}
		if attestingBalance > winnerBalance {
			winnerCrosslink = c
		}
	}

	crosslinkIndices, err := unslashedAttestingIndices(state, attsForCrosslink(winnerCrosslink, shardAtts))
	if err != nil {
		return nil, nil, errors.New("could not get crosslink indices")
	}

	return winnerCrosslink, crosslinkIndices, nil
}

// baseReward takes state and validator index and calculate
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
func baseReward(state *pb.BeaconState, index uint64) (uint64, error) {
	totalBalance, err := helpers.TotalActiveBalance(state)
	if err != nil {
		return 0, errors.Wrap(err, "could not calculate active balance")
	}
	effectiveBalance := state.Validators[index].EffectiveBalance
	baseReward := effectiveBalance * params.BeaconConfig().BaseRewardFactor /
		mathutil.IntegerSquareRoot(totalBalance) / params.BeaconConfig().BaseRewardsPerEpoch
	return baseReward, nil
}

// attestationDelta calculates the rewards and penalties of individual
// validator for voting the correct FFG source, FFG target, and head. It
// also calculates proposer delay inclusion and inactivity rewards
// and penalties. Individual rewards and penalties are returned in list.
//
// Note: we calculated adjusted quotient outside of base reward because it's too inefficient
// to repeat the same calculation for every validator versus just doing it once.
//
// Spec pseudocode definition:
//  def get_attestation_deltas(state: BeaconState) -> Tuple[Sequence[Gwei], Sequence[Gwei]]:
//    previous_epoch = get_previous_epoch(state)
//    total_balance = get_total_active_balance(state)
//    rewards = [Gwei(0) for _ in range(len(state.validators))]
//    penalties = [Gwei(0) for _ in range(len(state.validators))]
//    eligible_validator_indices = [
//        ValidatorIndex(index) for index, v in enumerate(state.validators)
//        if is_active_validator(v, previous_epoch) or (v.slashed and previous_epoch + 1 < v.withdrawable_epoch)
//    ]
//
//    # Micro-incentives for matching FFG source, FFG target, and head
//    matching_source_attestations = get_matching_source_attestations(state, previous_epoch)
//    matching_target_attestations = get_matching_target_attestations(state, previous_epoch)
//    matching_head_attestations = get_matching_head_attestations(state, previous_epoch)
//    for attestations in (matching_source_attestations, matching_target_attestations, matching_head_attestations):
//        unslashed_attesting_indices = get_unslashed_attesting_indices(state, attestations)
//        attesting_balance = get_total_balance(state, unslashed_attesting_indices)
//        for index in eligible_validator_indices:
//            if index in unslashed_attesting_indices:
//                rewards[index] += get_base_reward(state, index) * attesting_balance // total_balance
//            else:
//                penalties[index] += get_base_reward(state, index)
//
//    # Proposer and inclusion delay micro-rewards
//    for index in get_unslashed_attesting_indices(state, matching_source_attestations):
//        index = ValidatorIndex(index)
//        attestation = min([
//            a for a in matching_source_attestations
//            if index in get_attesting_indices(state, a.data, a.aggregation_bits)
//        ], key=lambda a: a.inclusion_delay)
//        proposer_reward = Gwei(get_base_reward(state, index) // PROPOSER_REWARD_QUOTIENT)
//        rewards[attestation.proposer_index] += proposer_reward
//        max_attester_reward = get_base_reward(state, index) - proposer_reward
//        rewards[index] += Gwei(
//            max_attester_reward
//            * (SLOTS_PER_EPOCH + MIN_ATTESTATION_INCLUSION_DELAY - attestation.inclusion_delay)
//            // SLOTS_PER_EPOCH
//        )
//
//    # Inactivity penalty
//    finality_delay = previous_epoch - state.finalized_checkpoint.epoch
//    if finality_delay > MIN_EPOCHS_TO_INACTIVITY_PENALTY:
//        matching_target_attesting_indices = get_unslashed_attesting_indices(state, matching_target_attestations)
//        for index in eligible_validator_indices:
//            index = ValidatorIndex(index)
//            penalties[index] += Gwei(BASE_REWARDS_PER_EPOCH * get_base_reward(state, index))
//            if index not in matching_target_attesting_indices:
//                penalties[index] += Gwei(
//                    state.validators[index].effective_balance * finality_delay // INACTIVITY_PENALTY_QUOTIENT
//                )
//
//    return rewards, penalties
func attestationDelta(state *pb.BeaconState) ([]uint64, []uint64, error) {
	prevEpoch := helpers.PrevEpoch(state)
	totalBalance, err := helpers.TotalActiveBalance(state)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get total active balance")
	}

	rewards := make([]uint64, len(state.Validators))
	penalties := make([]uint64, len(state.Validators))

	// Filter out the list of eligible validator indices. The eligible validator
	// has to be active or slashed but before withdrawn.
	var eligible []uint64
	for i, v := range state.Validators {
		isActive := helpers.IsActiveValidator(v, prevEpoch)
		isSlashed := v.Slashed && (prevEpoch+1 < v.WithdrawableEpoch)
		if isActive || isSlashed {
			eligible = append(eligible, uint64(i))
		}
	}

	// Apply rewards and penalties for voting correct source target and head.
	// Construct a attestations list contains source, target and head attestations.
	atts, err := MatchAttestations(state, prevEpoch)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get source, target and head attestations")
	}
	var attsPackage [][]*pb.PendingAttestation
	attsPackage = append(attsPackage, atts.source)
	attsPackage = append(attsPackage, atts.Target)
	attsPackage = append(attsPackage, atts.head)

	// Cache the validators who voted correctly for source in a map
	// to calculate earliest attestation rewards later.
	attestersVotedSource := make(map[uint64]*pb.PendingAttestation)
	// Compute rewards / penalties for each attestation in the list and update
	// the rewards and penalties lists.
	for i, matchAtt := range attsPackage {
		indices, err := unslashedAttestingIndices(state, matchAtt)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not get attestation indices")
		}

		attested := make(map[uint64]bool)
		// Construct a map to look up validators that voted for source, target or head.
		for _, index := range indices {
			if i == 0 {
				attestersVotedSource[index] = &pb.PendingAttestation{InclusionDelay: params.BeaconConfig().FarFutureEpoch}
			}
			attested[index] = true
		}
		attestedBalance := helpers.TotalBalance(state, indices)

		// Update rewards and penalties to each eligible validator index.
		for _, index := range eligible {
			base, err := baseReward(state, index)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not get base reward")
			}
			if _, ok := attested[index]; ok {
				rewards[index] += base * attestedBalance / totalBalance
			} else {
				penalties[index] += base
			}
		}
	}

	// For every index, filter the matching source attestation that correspond to the index,
	// sort by inclusion delay and get the one that was included on chain first.
	for _, att := range atts.source {
		indices, err := helpers.AttestingIndices(state, att.Data, att.AggregationBits)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not get attester indices")
		}
		for _, i := range indices {
			if _, ok := attestersVotedSource[i]; ok {
				if attestersVotedSource[i].InclusionDelay > att.InclusionDelay {
					attestersVotedSource[i] = att
				}
			}
		}
	}

	for i, a := range attestersVotedSource {
		slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch

		baseReward, err := baseReward(state, i)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not get proposer reward")
		}
		proposerReward := baseReward / params.BeaconConfig().ProposerRewardQuotient
		rewards[a.ProposerIndex] += proposerReward
		attesterReward := baseReward - proposerReward
		rewards[i] += attesterReward * (slotsPerEpoch + params.BeaconConfig().MinAttestationInclusionDelay - a.InclusionDelay) / slotsPerEpoch
	}

	// Apply penalties for quadratic leaks.
	// When epoch since finality exceeds inactivity penalty constant, the penalty gets increased
	// based on the finality delay.
	finalityDelay := prevEpoch - state.FinalizedCheckpoint.Epoch
	if finalityDelay > params.BeaconConfig().MinEpochsToInactivityPenalty {
		targetIndices, err := unslashedAttestingIndices(state, atts.Target)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not get attestation indices")
		}
		attestedTarget := make(map[uint64]bool)
		for _, index := range targetIndices {
			attestedTarget[index] = true
		}
		for _, index := range eligible {
			base, err := baseReward(state, index)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not get base reward")
			}
			penalties[index] += params.BeaconConfig().BaseRewardsPerEpoch * base
			if _, ok := attestedTarget[index]; !ok {
				penalties[index] += state.Validators[index].EffectiveBalance * finalityDelay /
					params.BeaconConfig().InactivityPenaltyQuotient
			}
		}
	}
	return rewards, penalties, nil
}

// crosslinkDelta calculates the rewards and penalties of individual
// validator for submitting the correct crosslink.
// Individual rewards and penalties are returned in list.
//
// Note: we calculated adjusted quotient outside of base reward because it's too inefficient
// to repeat the same calculation for every validator versus just doing it once.
//
// Spec pseudocode definition:
//  def get_crosslink_deltas(state: BeaconState) -> Tuple[List[Gwei], List[Gwei]]:
//    rewards = [0 for index in range(len(state.validator_registry))]
//    penalties = [0 for index in range(len(state.validator_registry))]
//    epoch = get_previous_epoch(state)
//    for offset in range(get_epoch_committee_count(state, epoch)):
//        shard = (get_epoch_start_shard(state, epoch) + offset) % SHARD_COUNT
//        crosslink_committee = get_crosslink_committee(state, epoch, shard)
//        winning_crosslink, attesting_indices = get_winning_crosslink_and_attesting_indices(state, epoch, shard)
//        attesting_balance = get_total_balance(state, attesting_indices)
//        committee_balance = get_total_balance(state, crosslink_committee)
//        for index in crosslink_committee:
//            base_reward = get_base_reward(state, index)
//            if index in attesting_indices:
//                rewards[index] += base_reward * attesting_balance // committee_balance
//            else:
//                penalties[index] += base_reward
//    return rewards, penalties
func crosslinkDelta(state *pb.BeaconState) ([]uint64, []uint64, error) {
	rewards := make([]uint64, len(state.Validators))
	penalties := make([]uint64, len(state.Validators))
	epoch := helpers.PrevEpoch(state)
	count, err := helpers.CommitteeCount(state, epoch)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get epoch committee count")
	}
	startShard, err := helpers.StartShard(state, epoch)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get epoch start shard")
	}
	for i := uint64(0); i < count; i++ {
		shard := (startShard + i) % params.BeaconConfig().ShardCount
		committee, err := helpers.CrosslinkCommittee(state, epoch, shard)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not get crosslink's committee")
		}
		_, attestingIndices, err := winningCrosslink(state, shard, epoch)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not get winning crosslink")
		}

		attested := make(map[uint64]bool)
		// Construct a map to look up validators that voted for crosslink.
		for _, index := range attestingIndices {
			attested[index] = true
		}
		committeeBalance := helpers.TotalBalance(state, committee)
		attestingBalance := helpers.TotalBalance(state, attestingIndices)

		for _, index := range committee {
			base, err := baseReward(state, index)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not get base reward")
			}
			if _, ok := attested[index]; ok {
				rewards[index] += base * attestingBalance / committeeBalance
			} else {
				penalties[index] += base
			}
		}
	}

	return rewards, penalties, nil
}

// attsForCrosslink returns the attestations of the input crosslink.
func attsForCrosslink(crosslink *ethpb.Crosslink, atts []*pb.PendingAttestation) []*pb.PendingAttestation {
	var crosslinkAtts []*pb.PendingAttestation
	for _, a := range atts {
		if proto.Equal(a.Data.Crosslink, crosslink) {
			crosslinkAtts = append(crosslinkAtts, a)
		}
	}
	return crosslinkAtts
}
