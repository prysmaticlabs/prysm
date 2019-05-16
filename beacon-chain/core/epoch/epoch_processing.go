// Package epoch contains epoch processing libraries. These libraries
// process new balance for the validators, justify and finalize new
// check points, shuffle and reassign validators to different slots and
// shards.
package epoch

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "core/state")

type queueElement struct {
	idx                        int
	ActivationEligibilityEpoch uint64
}

// MatchedAttestations is an object that contains the correctly
// voted attestations based on source, target and head criteria.
type MatchedAttestations struct {
	source []*pb.PendingAttestation
	target []*pb.PendingAttestation
	head   []*pb.PendingAttestation
}

// CanProcessEpoch checks the eligibility to process epoch.
// The epoch can be processed at the end of the last slot of every epoch
//
// Spec pseudocode definition:
//    If (state.slot + 1) % SLOTS_PER_EPOCH == 0:
func CanProcessEpoch(state *pb.BeaconState) bool {
	return (state.Slot+1)%params.BeaconConfig().SlotsPerEpoch == 0
}

// CanProcessEth1Data checks the eligibility to process the eth1 data.
// The eth1 data can be processed every EPOCHS_PER_ETH1_VOTING_PERIOD.
//
// Spec pseudocode definition:
//    If next_epoch % EPOCHS_PER_ETH1_VOTING_PERIOD == 0
func CanProcessEth1Data(state *pb.BeaconState) bool {
	return helpers.NextEpoch(state)%
		params.BeaconConfig().EpochsPerEth1VotingPeriod == 0
}

// CanProcessValidatorRegistry checks the eligibility to process validator registry.
// It checks crosslink committees last changed slot and finalized slot against
// latest change slot.
//
// Spec pseudocode definition:
//    If the following are satisfied:
//		* state.finalized_epoch > state.validator_registry_latest_change_epoch
//		* state.latest_crosslinks[shard].epoch > state.validator_registry_update_epoch
// 			for every shard number shard in [(state.current_epoch_start_shard + i) %
//	 			SHARD_COUNT for i in range(get_current_epoch_committee_count(state) *
//	 			SLOTS_PER_EPOCH)] (that is, for every shard in the current committees)
func CanProcessValidatorRegistry(state *pb.BeaconState) bool {
	if state.FinalizedEpoch <= state.ValidatorRegistryUpdateEpoch {
		return false
	}
	if featureconfig.FeatureConfig().EnableCrosslinks {
		shardsProcessed := helpers.EpochCommitteeCount(state, helpers.CurrentEpoch(state)) * params.BeaconConfig().SlotsPerEpoch
		startShard := state.CurrentShufflingStartShard
		for i := startShard; i < shardsProcessed; i++ {
			if state.LatestCrosslinks[i%params.BeaconConfig().ShardCount].Epoch <=
				state.ValidatorRegistryUpdateEpoch {
				return false
			}
		}
	}
	return true
}

// ProcessEth1Data processes eth1 block deposit roots by checking how many times
// state.eth1_data_votes contains body.eth1_data.
// With sufficient number of times (>2*SLOTS_PER_ETH1_VOTING_PERIOD), it then
// marks the body Eth1 data as the latest data set.
//
// Spec pseudocode definition:
// def process_eth1_data(state: BeaconState, body: BeaconBlockBody) -> None:
//     state.eth1_data_votes.append(body.eth1_data)
//     if state.eth1_data_votes.count(body.eth1_data) * 2 > SLOTS_PER_ETH1_VOTING_PERIOD:
//         state.latest_eth1_data = body.eth1_data
func ProcessEth1Data(state *pb.BeaconState, body *pb.BeaconBlockBody) *pb.BeaconState {
	eth1DataBlockVote := &pb.Eth1DataVote{Eth1Data: body.Eth1Data}
	state.Eth1DataVotes = append(state.Eth1DataVotes, eth1DataBlockVote)
	var count uint64
	for _, eth1DataVote := range state.Eth1DataVotes {
		if eth1DataVote.Eth1Data == body.Eth1Data {
			count++
		}
	}
	if count*2 > params.BeaconConfig().SlotsPerEth1VotingPeriod {
		state.LatestEth1Data = body.Eth1Data
	}
	return state
}

// ProcessRegistryUpdates rotates validators in and out of active pool.
// the amount to rotate is determined churn limit.
//
// Spec pseudocode definition:
//   def process_registry_updates(state: BeaconState) -> None:
//     # Process activation eligibility and ejections
//     for index, validator in enumerate(state.validator_registry):
//         if validator.activation_eligibility_epoch == FAR_FUTURE_EPOCH and validator.effective_balance >= MAX_EFFECTIVE_BALANCE:
//             validator.activation_eligibility_epoch = get_current_epoch(state)
//         if is_active_validator(validator, get_current_epoch(state)) and validator.effective_balance <= EJECTION_BALANCE:
//             initiate_validator_exit(state, index)
//     # Queue validators eligible for activation and not dequeued for activation prior to finalized epoch
//     activation_queue = sorted([
//         index for index, validator in enumerate(state.validator_registry) if
//         validator.activation_eligibility_epoch != FAR_FUTURE_EPOCH and
//         validator.activation_epoch >= get_delayed_activation_exit_epoch(state.finalized_epoch)
//     ], key=lambda index: state.validator_registry[index].activation_eligibility_epoch)
//     # Dequeued validators for activation up to churn limit (without resetting activation epoch)
//     for index in activation_queue[:get_churn_limit(state)]:
//         validator = state.validator_registry[index]
//         if validator.activation_epoch == FAR_FUTURE_EPOCH:
//             validator.activation_epoch = get_delayed_activation_exit_epoch(get_current_epoch(state))
func ProcessRegistryUpdates(state *pb.BeaconState) *pb.BeaconState {
	currentEpoch := helpers.CurrentEpoch(state)
	validators.VStore.Lock()
	defer validators.VStore.Unlock()
	for idx, validator := range state.ValidatorRegistry {
		// Activate validators within the allowable balance churn.
		if validator.ActivationEligibilityEpoch == params.BeaconConfig().FarFutureEpoch &&
			validator.EffectiveBalance >= params.BeaconConfig().MaxEffectiveBalance {
			validator.ActivationEligibilityEpoch = currentEpoch
		}
		if helpers.IsActiveValidator(validator, currentEpoch) &&
			validator.EffectiveBalance <= params.BeaconConfig().EjectionBalance {
			state = validators.ExitValidator(state, uint64(idx))
		}
	}
	// queue validators eligible for activation and not dequeued for activation prior to finalized epoch

	activationQueue := []queueElement{}
	for idx, validator := range state.ValidatorRegistry {
		if validator.ActivationEligibilityEpoch != params.BeaconConfig().FarFutureEpoch &&
			validator.ActivationEpoch >= helpers.DelayedActivationExitEpoch(state.FinalizedEpoch) {
			qe := queueElement{idx: idx,
				ActivationEligibilityEpoch: validator.ActivationEligibilityEpoch}
			activationQueue = append(activationQueue, qe)
		}
	}
	sort.Slice(activationQueue, func(i, j int) bool {
		return activationQueue[i].ActivationEligibilityEpoch < activationQueue[j].ActivationEligibilityEpoch
	})
	// dequeued validators for activation up to churn limit (without resetting activation epoch)
	limit := uint64(len(activationQueue))
	cl := helpers.ChurnLimit(state)
	if cl < limit {
		limit = helpers.ChurnLimit(state)
	}
	for _, qe := range activationQueue[:limit] {
		validator := state.ValidatorRegistry[qe.idx]
		if validator.ActivationEpoch == params.BeaconConfig().FarFutureEpoch {
			validator.ActivationEpoch = helpers.DelayedActivationExitEpoch(currentEpoch)
			log.WithFields(logrus.Fields{
				"index":           qe.idx,
				"activationEpoch": validator.ActivationEpoch,
			}).Info("Validator activated")
		}
	}
	return state
}

// ProcessPrevSlotShardSeed computes and sets current epoch's calculation slot
// and start shard to previous epoch. Then it returns the updated state.
//
// Spec pseudocode definition:
//	Set state.previous_epoch_randao_mix = state.current_epoch_randao_mix
//	Set state.previous_shuffling_start_shard = state.current_shuffling_start_shard
//  Set state.previous_shuffling_seed = state.current_shuffling_seed.
func ProcessPrevSlotShardSeed(state *pb.BeaconState) *pb.BeaconState {
	state.PreviousShufflingEpoch = state.CurrentShufflingEpoch
	state.PreviousShufflingStartShard = state.CurrentShufflingStartShard
	state.PreviousShufflingSeedHash32 = state.CurrentShufflingSeedHash32
	return state
}

// ProcessPartialValidatorRegistry processes the portion of validator registry
// fields, it doesn't set registry latest change slot. This only gets called if
// validator registry update did not happen.
//
// Spec pseudocode definition:
//	Let epochs_since_last_registry_change = current_epoch -
//		state.validator_registry_update_epoch
//	If epochs_since_last_registry_update > 1 and
//		is_power_of_two(epochs_since_last_registry_update):
// 			set state.current_calculation_epoch = next_epoch
// 			set state.current_shuffling_seed = generate_seed(
// 				state, state.current_calculation_epoch)
func ProcessPartialValidatorRegistry(state *pb.BeaconState) (*pb.BeaconState, error) {
	epochsSinceLastRegistryChange := helpers.CurrentEpoch(state) -
		state.ValidatorRegistryUpdateEpoch
	if epochsSinceLastRegistryChange > 1 &&
		mathutil.IsPowerOf2(epochsSinceLastRegistryChange) {
		state.CurrentShufflingEpoch = helpers.NextEpoch(state)
		// TODO(#2072)we have removed the generation of a new seed for the timebeing to get it stable for the testnet.
		// this will be handled in Q2.
	}
	return state, nil
}

// CleanupAttestations removes any attestation in state's latest attestations
// such that the attestation slot is lower than state slot minus epoch length.
// Spec pseudocode definition:
// 		Remove any attestation in state.latest_attestations such
// 		that slot_to_epoch(att.data.slot) < slot_to_epoch(state) - 1
func CleanupAttestations(state *pb.BeaconState) *pb.BeaconState {
	currEpoch := helpers.CurrentEpoch(state)

	var latestAttestations []*pb.PendingAttestation
	for _, attestation := range state.LatestAttestations {
		if helpers.SlotToEpoch(attestation.Data.Slot) >= currEpoch {
			latestAttestations = append(latestAttestations, attestation)
		}
	}
	state.LatestAttestations = latestAttestations
	return state
}

// UpdateLatestActiveIndexRoots updates the latest index roots. Index root
// is computed by hashing validator indices of the next epoch + delay.
//
// Spec pseudocode definition:
// Let e = state.slot // SLOTS_PER_EPOCH.
// Set state.latest_index_roots[(next_epoch + ACTIVATION_EXIT_DELAY) %
// 	LATEST_INDEX_ROOTS_LENGTH] =
// 	hash_tree_root(get_active_validator_indices(state,
// 	next_epoch + ACTIVATION_EXIT_DELAY))
func UpdateLatestActiveIndexRoots(state *pb.BeaconState) (*pb.BeaconState, error) {
	nextEpoch := helpers.NextEpoch(state) + params.BeaconConfig().ActivationExitDelay
	validatorIndices := helpers.ActiveValidatorIndices(state, nextEpoch)
	indicesBytes := []byte{}
	for _, val := range validatorIndices {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, val)
		indicesBytes = append(indicesBytes, buf...)
	}
	indexRoot := hashutil.Hash(indicesBytes)
	state.LatestActiveIndexRoots[nextEpoch%params.BeaconConfig().LatestActiveIndexRootsLength] =
		indexRoot[:]
	return state, nil
}

// ProcessJustificationFinalization processes justification and finalization during
// epoch processing. This is where a beacon node can justify and finalize a new epoch.
//
// Spec pseudocode definition:
//	def process_justification_and_finalization(state: BeaconState) -> None:
//    if get_current_epoch(state) <= GENESIS_EPOCH + 1:
//        return
//
//    previous_epoch = get_previous_epoch(state)
//    current_epoch = get_current_epoch(state)
//    old_previous_justified_epoch = state.previous_justified_epoch
//    old_current_justified_epoch = state.current_justified_epoch
//
//    # Process justifications
//    state.previous_justified_epoch = state.current_justified_epoch
//    state.previous_justified_root = state.current_justified_root
//    state.justification_bitfield = (state.justification_bitfield << 1) % 2**64
//    previous_epoch_matching_target_balance = get_attesting_balance(state, get_matching_target_attestations(state, previous_epoch))
//    if previous_epoch_matching_target_balance * 3 >= get_total_active_balance(state) * 2:
//        state.current_justified_epoch = previous_epoch
//        state.current_justified_root = get_block_root(state, state.current_justified_epoch)
//        state.justification_bitfield |= (1 << 1)
//    current_epoch_matching_target_balance = get_attesting_balance(state, get_matching_target_attestations(state, current_epoch))
//    if current_epoch_matching_target_balance * 3 >= get_total_active_balance(state) * 2:
//        state.current_justified_epoch = current_epoch
//        state.current_justified_root = get_block_root(state, state.current_justified_epoch)
//        state.justification_bitfield |= (1 << 0)
//
//    # Process finalizations
//    bitfield = state.justification_bitfield
//    # The 2nd/3rd/4th most recent epochs are justified, the 2nd using the 4th as source
//    if (bitfield >> 1) % 8 == 0b111 and old_previous_justified_epoch == current_epoch - 3:
//        state.finalized_epoch = old_previous_justified_epoch
//        state.finalized_root = get_block_root(state, state.finalized_epoch)
//    # The 2nd/3rd most recent epochs are justified, the 2nd using the 3rd as source
//    if (bitfield >> 1) % 4 == 0b11 and old_previous_justified_epoch == current_epoch - 2:
//        state.finalized_epoch = old_previous_justified_epoch
//        state.finalized_root = get_block_root(state, state.finalized_epoch)
//    # The 1st/2nd/3rd most recent epochs are justified, the 1st using the 3rd as source
//    if (bitfield >> 0) % 8 == 0b111 and old_current_justified_epoch == current_epoch - 2:
//        state.finalized_epoch = old_current_justified_epoch
//        state.finalized_root = get_block_root(state, state.finalized_epoch)
//    # The 1st/2nd most recent epochs are justified, the 1st using the 2nd as source
//    if (bitfield >> 0) % 4 == 0b11 and old_current_justified_epoch == current_epoch - 1:
//        state.finalized_epoch = old_current_justified_epoch
//        state.finalized_root = get_block_root(state, state.finalized_epoch)
func ProcessJustificationFinalization(state *pb.BeaconState, prevAttestedBal uint64, currAttestedBal uint64) (
	*pb.BeaconState, error) {
	// There's no reason to process justification until the 3rd epoch.
	currentEpoch := helpers.CurrentEpoch(state)
	if currentEpoch <= params.BeaconConfig().GenesisEpoch+1 {
		return state, nil
	}

	prevEpoch := helpers.PrevEpoch(state)
	totalBal := totalActiveBalance(state)
	oldPrevJustifiedEpoch := state.PreviousJustifiedEpoch
	oldPrevJustifiedRoot := state.PreviousJustifiedRoot
	oldCurrJustifiedEpoch := state.CurrentJustifiedEpoch
	oldCurrJustifiedRoot := state.CurrentJustifiedRoot
	state.PreviousJustifiedEpoch = state.CurrentJustifiedEpoch
	state.PreviousJustifiedRoot = state.CurrentJustifiedRoot
	state.JustificationBitfield = (state.JustificationBitfield << 1) % (1 << 63)
	// Process justification.
	if 3*prevAttestedBal >= 2*totalBal {
		state.CurrentJustifiedEpoch = prevEpoch
		blockRoot, err := helpers.BlockRoot(state, prevEpoch)
		if err != nil {
			return nil, fmt.Errorf("could not get block root for previous epoch %d: %v",
				prevEpoch, err)
		}
		state.CurrentJustifiedRoot = blockRoot
		state.JustificationBitfield |= 2
	}
	if 3*currAttestedBal >= 2*totalBal {
		state.CurrentJustifiedEpoch = currentEpoch
		blockRoot, err := helpers.BlockRoot(state, currentEpoch)
		if err != nil {
			return nil, fmt.Errorf("could not get block root for current epoch %d: %v",
				prevEpoch, err)
		}
		state.CurrentJustifiedRoot = blockRoot
		state.JustificationBitfield |= 1
	}
	// Process finalization.
	bitfield := state.JustificationBitfield
	// When the 2nd, 3rd and 4th most recent epochs are all justified,
	// 2nd epoch can finalize the 4th epoch as a source.
	if oldPrevJustifiedEpoch == currentEpoch-3 && (bitfield>>1)%8 == 7 {
		state.FinalizedEpoch = oldPrevJustifiedEpoch
		state.FinalizedRoot = oldPrevJustifiedRoot
	}
	// when 2nd and 3rd most recent epochs are all justified,
	// 2nd epoch can finalize 3rd as a source.
	if oldPrevJustifiedEpoch == currentEpoch-2 && (bitfield>>1)%4 == 3 {
		state.FinalizedEpoch = oldPrevJustifiedEpoch
		state.FinalizedRoot = oldPrevJustifiedRoot
	}
	// when 1st, 2nd and 3rd most recent epochs are all justified,
	// 1st epoch can finalize 3rd as a source.
	if oldCurrJustifiedEpoch == currentEpoch-2 && (bitfield>>0)%8 == 7 {
		state.FinalizedEpoch = oldCurrJustifiedEpoch
		state.FinalizedRoot = oldCurrJustifiedRoot
	}
	// when 1st, 2nd most recent epochs are all justified,
	// 1st epoch can finalize 2nd as a source.
	if oldCurrJustifiedEpoch == currentEpoch-1 && (bitfield>>0)%4 == 3 {
		state.FinalizedEpoch = oldCurrJustifiedEpoch
		state.FinalizedRoot = oldCurrJustifiedRoot
	}
	return state, nil
}

// ProcessFinalUpdates processes the final updates during epoch processing.
//
// Spec pseudocode definition:
//	def process_final_updates(state: BeaconState) -> None:
//    current_epoch = get_current_epoch(state)
//    next_epoch = current_epoch + 1
//    # Reset eth1 data votes
//    if (state.slot + 1) % SLOTS_PER_ETH1_VOTING_PERIOD == 0:
//        state.eth1_data_votes = []
//    # Update effective balances with hysteresis
//    for index, validator in enumerate(state.validator_registry):
//        balance = state.balances[index]
//        HALF_INCREMENT = EFFECTIVE_BALANCE_INCREMENT // 2
//        if balance < validator.effective_balance or validator.effective_balance + 3 * HALF_INCREMENT < balance:
//            validator.effective_balance = min(balance - balance % EFFECTIVE_BALANCE_INCREMENT, MAX_EFFECTIVE_BALANCE)
//    # Update start shard
//    state.latest_start_shard = (state.latest_start_shard + get_shard_delta(state, current_epoch)) % SHARD_COUNT
//    # Set active index root
//    index_root_position = (next_epoch + ACTIVATION_EXIT_DELAY) % LATEST_ACTIVE_INDEX_ROOTS_LENGTH
//    state.latest_active_index_roots[index_root_position] = hash_tree_root(
//        get_active_validator_indices(state, next_epoch + ACTIVATION_EXIT_DELAY)
//    )
//    # Set total slashed balances
//    state.latest_slashed_balances[next_epoch % LATEST_SLASHED_EXIT_LENGTH] = (
//        state.latest_slashed_balances[current_epoch % LATEST_SLASHED_EXIT_LENGTH]
//    )
//    # Set randao mix
//    state.latest_randao_mixes[next_epoch % LATEST_RANDAO_MIXES_LENGTH] = get_randao_mix(state, current_epoch)
//    # Set historical root accumulator
//    if next_epoch % (SLOTS_PER_HISTORICAL_ROOT // SLOTS_PER_EPOCH) == 0:
//        historical_batch = HistoricalBatch(
//            block_roots=state.latest_block_roots,
//            state_roots=state.latest_state_roots,
//        )
//        state.historical_roots.append(hash_tree_root(historical_batch))
//    # Rotate current/previous epoch attestations
//    state.previous_epoch_attestations = state.current_epoch_attestations
//    state.current_epoch_attestations = []
func ProcessFinalUpdates(state *pb.BeaconState) (*pb.BeaconState, error) {
	currentEpoch := helpers.CurrentEpoch(state)
	nextEpoch := currentEpoch + 1

	// Reset ETH1 data votes.
	if (state.Slot+1)%params.BeaconConfig().SlotsPerHistoricalRoot == 0 {
		state.Eth1DataVotes = nil
	}

	// Update effective balances with hysteresis.
	for i, v := range state.ValidatorRegistry {
		balance := state.Balances[i]
		halfInc := params.BeaconConfig().EffectiveBalanceIncrement / 2
		if balance < v.EffectiveBalance || v.EffectiveBalance+3*halfInc < balance {
			v.EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance
			if v.EffectiveBalance > balance-balance%params.BeaconConfig().EffectiveBalanceIncrement {
				v.EffectiveBalance = balance - balance%params.BeaconConfig().EffectiveBalanceIncrement
			}
		}
	}

	// Update start shard.
	state.LatestStartShard = (state.LatestStartShard + helpers.ShardDelta(state, currentEpoch)) %
		params.BeaconConfig().ShardCount

	// Set active index root.
	activationDelay := params.BeaconConfig().ActivationExitDelay
	idxRootPosition := (nextEpoch + activationDelay) % params.BeaconConfig().LatestActiveIndexRootsLength
	idxRoot, err := ssz.TreeHash(helpers.ActiveValidatorIndices(state, nextEpoch+activationDelay))
	if err != nil {
		return nil, fmt.Errorf("could not tree hash active indices: %v", err)
	}
	state.LatestActiveIndexRoots[idxRootPosition] = idxRoot[:]

	// Set total slashed balances.
	slashedExitLength := params.BeaconConfig().LatestSlashedExitLength
	state.LatestSlashedBalances[nextEpoch%slashedExitLength] =
		state.LatestSlashedBalances[currentEpoch%slashedExitLength]

	// Set RANDAO mix.
	randaoMixLength := params.BeaconConfig().LatestRandaoMixesLength
	mix, err := helpers.RandaoMix(state, currentEpoch)
	if err != nil {
		return nil, fmt.Errorf("could not get randao mix: %v", err)
	}
	state.LatestRandaoMixes[nextEpoch%randaoMixLength] = mix

	// Set historical root accumulator.
	epochsPerHistoricalRoot := params.BeaconConfig().SlotsPerHistoricalRoot / params.BeaconConfig().SlotsPerEpoch
	if nextEpoch%epochsPerHistoricalRoot == 0 {
		historicalBatch := &pb.HistoricalBatch{
			BlockRoots: state.LatestBlockRoots,
			StateRoots: state.LatestStateRoots,
		}
		batchRoot, err := hashutil.HashProto(historicalBatch)
		if err != nil {
			return nil, fmt.Errorf("could not hash historical batch: %v", err)
		}
		state.HistoricalRoots = append(state.HistoricalRoots, batchRoot[:])
	}

	// Rotate current and previous epoch attestations.
	state.PreviousEpochAttestations = state.CurrentEpochAttestations
	state.CurrentEpochAttestations = nil

	return state, nil
}

// UpdateLatestSlashedBalances updates the latest slashed balances. It transfers
// the amount from the current epoch index to next epoch index.
//
// Spec pseudocode definition:
// Set state.latest_slashed_balances[(next_epoch) % LATEST_PENALIZED_EXIT_LENGTH] =
// 	state.latest_slashed_balances[current_epoch % LATEST_PENALIZED_EXIT_LENGTH].
func UpdateLatestSlashedBalances(state *pb.BeaconState) *pb.BeaconState {
	currentEpoch := helpers.CurrentEpoch(state) % params.BeaconConfig().LatestSlashedExitLength
	nextEpoch := helpers.NextEpoch(state) % params.BeaconConfig().LatestSlashedExitLength
	state.LatestSlashedBalances[nextEpoch] = state.LatestSlashedBalances[currentEpoch]
	return state
}

// UpdateLatestRandaoMixes updates the latest seed mixes. It transfers
// the seed mix of current epoch to next epoch.
//
// Spec pseudocode definition:
// Set state.latest_randao_mixes[next_epoch % LATEST_RANDAO_MIXES_LENGTH] =
// 	get_randao_mix(state, current_epoch).
func UpdateLatestRandaoMixes(state *pb.BeaconState) (*pb.BeaconState, error) {
	nextEpoch := helpers.NextEpoch(state) % params.BeaconConfig().LatestRandaoMixesLength
	randaoMix, err := helpers.RandaoMix(state, helpers.CurrentEpoch(state))
	if err != nil {
		return nil, fmt.Errorf("could not get randaoMix mix: %v", err)
	}

	state.LatestRandaoMixes[nextEpoch] = randaoMix
	return state, nil
}

// UnslashedAttestingIndices returns all the attesting indices from a list of attestations,
// it sorts the indices and filters out the slashed ones.
//
// Spec pseudocode definition:
// def get_unslashed_attesting_indices(state: BeaconState, attestations: List[PendingAttestation]) -> List[ValidatorIndex]:
//    output = set()
//    for a in attestations:
//        output = output.union(get_attesting_indices(state, a.data, a.aggregation_bitfield))
//    return sorted(filter(lambda index: not state.validator_registry[index].slashed, list(output)))
func UnslashedAttestingIndices(state *pb.BeaconState, atts []*pb.PendingAttestation) ([]uint64, error) {
	var setIndices []uint64
	for _, att := range atts {
		indices, err := helpers.AttestingIndices(state, att.Data, att.AggregationBitfield)
		if err != nil {
			return nil, fmt.Errorf("could not get attester indices: %v", err)
		}
		setIndices = sliceutil.UnionUint64(setIndices, indices)
	}
	// Sort the attesting set indices by increasing order.
	sort.Slice(setIndices, func(i, j int) bool { return setIndices[i] < setIndices[j] })
	// Remove the slashed validator indices.
	for i := 0; i < len(setIndices); i++ {
		if state.ValidatorRegistry[setIndices[i]].Slashed {
			setIndices = append(setIndices[:i], setIndices[i+1:]...)
		}
	}
	return setIndices, nil
}

// AttestingBalance returns the total balance from all the attesting indices.
//
// Spec pseudocode definition:
// def get_attesting_balance(state: BeaconState, attestations: List[PendingAttestation]) -> Gwei:
//    return get_total_balance(state, get_unslashed_attesting_indices(state, attestations))
func AttestingBalance(state *pb.BeaconState, atts []*pb.PendingAttestation) (uint64, error) {
	indices, err := UnslashedAttestingIndices(state, atts)
	if err != nil {
		return 0, fmt.Errorf("could not get attesting balance: %v", err)
	}
	return TotalBalance(state, indices), nil
}

// EarlistAttestation returns attestation with the earliest inclusion slot.
//
// Spec pseudocode definition:
// def get_earliest_attestation(state: BeaconState, attestations: List[PendingAttestation], index: ValidatorIndex) -> PendingAttestation:
//    return min([
//        a for a in attestations if index in get_attesting_indices(state, a.data, a.aggregation_bitfield)
//    ], key=lambda a: a.inclusion_slot)
func EarlistAttestation(state *pb.BeaconState, atts []*pb.PendingAttestation, index uint64) (*pb.PendingAttestation, error) {
	earliest := &pb.PendingAttestation{
		InclusionSlot: params.BeaconConfig().FarFutureEpoch,
	}
	for _, att := range atts {
		indices, err := helpers.AttestingIndices(state, att.Data, att.AggregationBitfield)
		if err != nil {
			return nil, fmt.Errorf("could not get attester indices: %v", err)
		}
		for _, i := range indices {
			if index == i {
				if earliest.InclusionSlot > att.InclusionSlot {
					earliest = att
				}
			}
		}
	}
	return earliest, nil
}

// MatchAttestations matches the attestations gathered in a span of an epoch
// and categorize them whether they correctly voted for source, target and head.
// We combined the individual helpers from spec for efficiency and to achieve O(N) run time.
//
// Spec pseudocode definition:
//	def get_matching_source_attestations(state: BeaconState, epoch: Epoch) -> List[PendingAttestation]:
//    assert epoch in (get_current_epoch(state), get_previous_epoch(state))
//    return state.current_epoch_attestations if epoch == get_current_epoch(state) else state.previous_epoch_attestations
//
//	def get_matching_target_attestations(state: BeaconState, epoch: Epoch) -> List[PendingAttestation]:
//    return [
//        a for a in get_matching_source_attestations(state, epoch)
//        if a.data.target_root == get_block_root(state, epoch)
//    ]
//
//	def get_matching_head_attestations(state: BeaconState, epoch: Epoch) -> List[PendingAttestation]:
//    return [
//        a for a in get_matching_source_attestations(state, epoch)
//        if a.data.beacon_block_root == get_block_root_at_slot(state, a.data.slot)
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
		return nil, fmt.Errorf("could not get block root for epoch %d: %v", epoch, err)
	}

	tgtAtts := make([]*pb.PendingAttestation, 0, len(srcAtts))
	headAtts := make([]*pb.PendingAttestation, 0, len(srcAtts))
	for _, srcAtt := range srcAtts {
		// If the target root matches attestation's target root,
		// then we know this attestation has correctly voted for target.
		if bytes.Equal(srcAtt.Data.TargetRoot, targetRoot) {
			tgtAtts = append(tgtAtts, srcAtt)
		}

		// If the block root at slot matches attestation's block root at slot,
		// then we know this attestation has correctly voted for head.
		headRoot, err := helpers.BlockRootAtSlot(state, srcAtt.Data.Slot)
		if err != nil {
			return nil, fmt.Errorf("could not get block root for slot %d: %v", srcAtt.Data.Slot, err)
		}
		if bytes.Equal(srcAtt.Data.BeaconBlockRoot, headRoot) {
			headAtts = append(headAtts, srcAtt)
		}
	}

	return &MatchedAttestations{
		source: srcAtts,
		target: tgtAtts,
		head:   headAtts,
	}, nil
}

// CrosslinkFromAttsData returns a constructed crosslink from attestation data.
//
// Spec pseudocode definition:
//	def get_crosslink_from_attestation_data(state: BeaconState, data: AttestationData) -> Crosslink:
//    return Crosslink(
//        epoch=min(slot_to_epoch(data.slot), state.current_crosslinks[data.shard].epoch + MAX_CROSSLINK_EPOCHS),
//        previous_crosslink_root=data.previous_crosslink_root,
//        crosslink_data_root=data.crosslink_data_root,
//    )
func CrosslinkFromAttsData(state *pb.BeaconState, attData *pb.AttestationData) *pb.Crosslink {
	epoch := helpers.SlotToEpoch(attData.Slot)
	if epoch > state.CurrentCrosslinks[attData.Shard].Epoch+params.BeaconConfig().MaxCrosslinkEpochs {
		epoch = state.CurrentCrosslinks[attData.Shard].Epoch + params.BeaconConfig().MaxCrosslinkEpochs
	}
	return &pb.Crosslink{
		Epoch: epoch,
		CrosslinkDataRootHash32:     attData.CrosslinkDataRoot,
		PreviousCrosslinkRootHash32: attData.PreviousCrosslinkRoot,
	}
}

// WinningCrosslink returns the most staked balance-wise crosslink of a given shard and epoch.
// Here we deviated from the spec definition and split the following to two functions
// `WinningCrosslink` and  `CrosslinkAttestingIndices` for clarity and efficiency.
//
// Spec pseudocode definition:
//	def get_winning_crosslink_and_attesting_indices(state: BeaconState, shard: Shard, epoch: Epoch) -> Tuple[Crosslink, List[ValidatorIndex]]:
//    shard_attestations = [a for a in get_matching_source_attestations(state, epoch) if a.data.shard == shard]
//    shard_crosslinks = [get_crosslink_from_attestation_data(state, a.data) for a in shard_attestations]
//    candidate_crosslinks = [
//        c for c in shard_crosslinks
//        if hash_tree_root(state.current_crosslinks[shard]) in (c.previous_crosslink_root, hash_tree_root(c))
//    ]
//    if len(candidate_crosslinks) == 0:
//        return Crosslink(epoch=GENESIS_EPOCH, previous_crosslink_root=ZERO_HASH, crosslink_data_root=ZERO_HASH), []
//
//    def get_attestations_for(crosslink: Crosslink) -> List[PendingAttestation]:
//        return [a for a in shard_attestations if get_crosslink_from_attestation_data(state, a.data) == crosslink]
//    # Winning crosslink has the crosslink data root with the most balance voting for it (ties broken lexicographically)
//    winning_crosslink = max(candidate_crosslinks, key=lambda crosslink: (
//        get_attesting_balance(state, get_attestations_for(crosslink)), crosslink.crosslink_data_root
//    ))
//
//    return winning_crosslink, get_unslashed_attesting_indices(state, get_attestations_for(winning_crosslink))
func WinningCrosslink(state *pb.BeaconState, shard uint64, epoch uint64) (*pb.Crosslink, error) {
	var shardAtts []*pb.PendingAttestation
	matchedAtts, err := MatchAttestations(state, epoch)
	if err != nil {
		return nil, fmt.Errorf("could not get matching attestations: %v", err)
	}

	// Filter out source attestations by shard.
	for _, att := range matchedAtts.source {
		if att.Data.Shard == shard {
			shardAtts = append(shardAtts, att)
		}
	}

	// Convert shard attestations to shard crosslinks.
	shardCrosslinks := make([]*pb.Crosslink, len(matchedAtts.source))
	for i := 0; i < len(shardCrosslinks); i++ {
		shardCrosslinks[i] = CrosslinkFromAttsData(state, shardAtts[i].Data)
	}

	var candidateCrosslinks []*pb.Crosslink
	// Filter out shard crosslinks with correct current or previous crosslink data.
	for _, c := range shardCrosslinks {
		cFromState := state.CurrentCrosslinks[shard]
		h, err := hashutil.HashProto(cFromState)
		if err != nil {
			return nil, fmt.Errorf("could not hash crosslink from state: %v", err)
		}
		if proto.Equal(cFromState, c) || bytes.Equal(h[:], c.PreviousCrosslinkRootHash32) {
			candidateCrosslinks = append(candidateCrosslinks, c)
		}
	}

	if len(candidateCrosslinks) == 0 {
		return &pb.Crosslink{
			Epoch: params.BeaconConfig().GenesisEpoch,
			CrosslinkDataRootHash32:     params.BeaconConfig().ZeroHash[:],
			PreviousCrosslinkRootHash32: params.BeaconConfig().ZeroHash[:],
		}, nil
	}

	var crosslinkAtts []*pb.PendingAttestation
	var winnerBalance uint64
	var winnerCrosslink *pb.Crosslink
	// Out of the existing shard crosslinks, pick the one that has the
	// most balance staked.
	crosslinkAtts = attsForCrosslink(state, candidateCrosslinks[0], shardAtts)
	winnerBalance, err = AttestingBalance(state, crosslinkAtts)
	winnerCrosslink = candidateCrosslinks[0]

	for _, c := range candidateCrosslinks {
		crosslinkAtts := crosslinkAtts[:0]
		crosslinkAtts = attsForCrosslink(state, c, shardAtts)
		attestingBalance, err := AttestingBalance(state, crosslinkAtts)
		if err != nil {
			return nil, fmt.Errorf("could not get crosslink's attesting balance: %v", err)
		}
		if attestingBalance > winnerBalance {
			winnerCrosslink = c
		}
	}

	return winnerCrosslink, nil
}

// ProcessSlashings processes the slashed validators during epoch processing,
//
// def process_slashings(state: BeaconState) -> None:
//    current_epoch = get_current_epoch(state)
//    active_validator_indices = get_active_validator_indices(state, current_epoch)
//    total_balance = get_total_balance(state, active_validator_indices)
//
//    # Compute `total_penalties`
//    total_at_start = state.latest_slashed_balances[(current_epoch + 1) % LATEST_SLASHED_EXIT_LENGTH]
//    total_at_end = state.latest_slashed_balances[current_epoch % LATEST_SLASHED_EXIT_LENGTH]
//    total_penalties = total_at_end - total_at_start
//
//    for index, validator in enumerate(state.validator_registry):
//        if validator.slashed and current_epoch == validator.withdrawable_epoch - LATEST_SLASHED_EXIT_LENGTH // 2:
//            penalty = max(
//                validator.effective_balance * min(total_penalties * 3, total_balance) // total_balance,
//                validator.effective_balance // MIN_SLASHING_PENALTY_QUOTIENT
//            )
//            decrease_balance(state, index, penalty)
func ProcessSlashings(state *pb.BeaconState) *pb.BeaconState {
	currentEpoch := helpers.CurrentEpoch(state)
	activeIndices := helpers.ActiveValidatorIndices(state, currentEpoch)
	totalBalance := helpers.TotalBalance(state, activeIndices)

	// Compute the total penalties.
	exitLength := params.BeaconConfig().LatestSlashedExitLength
	totalAtStart := state.LatestSlashedBalances[(currentEpoch+1)%exitLength]
	totalAtEnd := state.LatestSlashedBalances[currentEpoch%exitLength]
	totalPenalties := totalAtEnd - totalAtStart

	// Compute slashing for each validator.
	for index, validator := range state.ValidatorRegistry {
		correctEpoch := currentEpoch == validator.WithdrawableEpoch-exitLength/2
		if validator.Slashed && correctEpoch {
			minPenalties := totalPenalties * 3
			if minPenalties > totalBalance {
				minPenalties = totalBalance
			}
			effectiveBal := validator.EffectiveBalance
			penalty := effectiveBal * minPenalties / totalBalance
			if penalty < effectiveBal/params.BeaconConfig().MinSlashingPenaltyQuotient {
				penalty = effectiveBal / params.BeaconConfig().MinSlashingPenaltyQuotient
			}
			state = helpers.DecreaseBalance(state, uint64(index), penalty)
		}
	}
	return state
}

// CrosslinkAttestingIndices returns the attesting indices of the input crosslink.
func CrosslinkAttestingIndices(state *pb.BeaconState, crosslink *pb.Crosslink, atts []*pb.PendingAttestation) ([]uint64, error) {
	crosslinkAtts := attsForCrosslink(state, crosslink, atts)
	return UnslashedAttestingIndices(state, crosslinkAtts)
}

// BaseReward takes state and validator index and calculate
// individual validator's base reward quotient.
//
// Spec pseudocode definition:
//	def get_base_reward(state: BeaconState, index: ValidatorIndex) -> Gwei:
//    adjusted_quotient = integer_squareroot(get_total_active_balance(state)) // BASE_REWARD_QUOTIENT
//    if adjusted_quotient == 0:
//        return 0
//    return state.validator_registry[index].effective_balance // adjusted_quotient // BASE_REWARDS_PER_EPOCH
func BaseReward(state *pb.BeaconState, index uint64) uint64 {
	adjustedQuotient := mathutil.IntegerSquareRoot(totalActiveBalance(state) /
		params.BeaconConfig().BaseRewardQuotient)
	if adjustedQuotient == 0 {
		return 0
	}
	baseReward := state.ValidatorRegistry[index].EffectiveBalance / adjustedQuotient
	return baseReward / params.BeaconConfig().BaseRewardsPerEpoch
}

// attsForCrosslink returns the attestations of the input crosslink.
func attsForCrosslink(state *pb.BeaconState, crosslink *pb.Crosslink, atts []*pb.PendingAttestation) []*pb.PendingAttestation {
	var crosslinkAtts []*pb.PendingAttestation
	for _, a := range atts {
		if proto.Equal(CrosslinkFromAttsData(state, a.Data), crosslink) {
			crosslinkAtts = append(crosslinkAtts, a)
		}
	}
	return crosslinkAtts
}

// totalActiveBalance returns the combined balances of all the active validators.
//
// Spec pseudocode definition:
//	def get_total_active_balance(state: BeaconState) -> Gwei:
//    return get_total_balance(state, get_active_validator_indices(state, get_current_epoch(state)))
func totalActiveBalance(state *pb.BeaconState) uint64 {
	return TotalBalance(state, helpers.ActiveValidatorIndices(state, helpers.CurrentEpoch(state)))
}
