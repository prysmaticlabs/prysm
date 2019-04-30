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
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "core/state")

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
		shardsProcessed := helpers.CurrentEpochCommitteeCount(state) * params.BeaconConfig().SlotsPerEpoch
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

// ProcessEth1Data processes eth1 block deposit roots by checking its vote count.
// With sufficient votes (>2*EPOCHS_PER_ETH1_VOTING_PERIOD), it then
// marks the voted Eth1 data as the latest data set.
//
// Official spec definition:
//     if eth1_data_vote.vote_count * 2 > EPOCHS_PER_ETH1_VOTING_PERIOD * SLOTS_PER_EPOCH for
//       some eth1_data_vote in state.eth1_data_votes.
//       (ie. more than half the votes in this voting period were for that value)
//       Set state.latest_eth1_data = eth1_data_vote.eth1_data.
//		 Set state.eth1_data_votes = [].
//
func ProcessEth1Data(state *pb.BeaconState) *pb.BeaconState {
	for _, eth1DataVote := range state.Eth1DataVotes {
		if eth1DataVote.VoteCount*2 > params.BeaconConfig().SlotsPerEpoch*
			params.BeaconConfig().EpochsPerEth1VotingPeriod {
			state.LatestEth1Data = eth1DataVote.Eth1Data
		}
	}
	state.Eth1DataVotes = make([]*pb.Eth1DataVote, 0)
	return state
}

// ProcessJustificationAndFinalization processes for justified slot by comparing
// epoch boundary balance and total balance.
//   First, update the justification bitfield:
//     Let new_justified_epoch = state.justified_epoch.
//     Set state.justification_bitfield = state.justification_bitfield << 1.
//     Set state.justification_bitfield |= 2 and new_justified_epoch = previous_epoch if
//       3 * previous_epoch_boundary_attesting_balance >= 2 * previous_total_balance.
//     Set state.justification_bitfield |= 1 and new_justified_epoch = current_epoch if
//       3 * current_epoch_boundary_attesting_balance >= 2 * current_total_balance.
//   Next, update last finalized epoch if possible:
//     Set state.finalized_epoch = state.previous_justified_epoch if (state.justification_bitfield >> 1) % 8
//       == 0b111 and state.previous_justified_epoch == previous_epoch - 2.
//     Set state.finalized_epoch = state.previous_justified_epoch if (state.justification_bitfield >> 1) % 4
//       == 0b11 and state.previous_justified_epoch == previous_epoch - 1.
//     Set state.finalized_epoch = state.justified_epoch if (state.justification_bitfield >> 0) % 8
//       == 0b111 and state.justified_epoch == previous_epoch - 1.
//     Set state.finalized_epoch = state.justified_epoch if (state.justification_bitfield >> 0) % 4
//       == 0b11 and state.justified_epoch == previous_epoch.
//   Finally, update the following:
//     Set state.previous_justified_epoch = state.justified_epoch.
//     Set state.justified_epoch = new_justified_epoch
func ProcessJustificationAndFinalization(
	state *pb.BeaconState,
	thisEpochBoundaryAttestingBalance uint64,
	prevEpochBoundaryAttestingBalance uint64,
	prevTotalBalance uint64,
	totalBalance uint64,
) (*pb.BeaconState, error) {

	newJustifiedEpoch := state.JustifiedEpoch
	newFinalizedEpoch := state.FinalizedEpoch
	prevEpoch := helpers.PrevEpoch(state)
	currentEpoch := helpers.CurrentEpoch(state)
	// Shifts all the bits over one to create a new bit for the recent epoch.
	state.JustificationBitfield <<= 1
	// If prev prev epoch was justified then we ensure the 2nd bit in the bitfield is set,
	// assign new justified slot to 2 * SLOTS_PER_EPOCH before.
	if 3*prevEpochBoundaryAttestingBalance >= 2*prevTotalBalance {
		state.JustificationBitfield |= 2
		newJustifiedEpoch = prevEpoch
	}
	// If this epoch was justified then we ensure the 1st bit in the bitfield is set,
	// assign new justified slot to 1 * SLOTS_PER_EPOCH before.
	if 3*thisEpochBoundaryAttestingBalance >= 2*totalBalance {
		state.JustificationBitfield |= 1
		newJustifiedEpoch = currentEpoch
	}

	// Process finality.
	// When the 2nd, 3rd and 4th most epochs are all justified, the 2nd can finalize the 4th epoch
	// as a source.
	if state.PreviousJustifiedEpoch == prevEpoch-2 &&
		(state.JustificationBitfield>>1)%8 == 7 {
		newFinalizedEpoch = state.PreviousJustifiedEpoch
	}
	// When the 2nd and 3rd most epochs are all justified, the 2nd can finalize the 3rd epoch
	// as a source.
	if state.PreviousJustifiedEpoch == prevEpoch-1 &&
		(state.JustificationBitfield>>1)%4 == 3 {
		newFinalizedEpoch = state.PreviousJustifiedEpoch
	}
	// When the 1st, 2nd and 3rd most epochs are all justified, the 1st can finalize the 3rd epoch
	// as a source.
	if state.JustifiedEpoch == prevEpoch-1 &&
		(state.JustificationBitfield>>0)%8 == 7 {
		newFinalizedEpoch = state.JustifiedEpoch
	}
	// When the 1st and 2nd most epochs are all justified, the 1st can finalize the 2nd epoch
	// as a source.
	if state.JustifiedEpoch == prevEpoch &&
		(state.JustificationBitfield>>0)%4 == 3 {
		newFinalizedEpoch = state.JustifiedEpoch
	}
	state.PreviousJustifiedEpoch = state.JustifiedEpoch
	state.PreviousJustifiedRoot = state.JustifiedRoot
	if newJustifiedEpoch != state.JustifiedEpoch {
		state.JustifiedEpoch = newJustifiedEpoch
		newJustifedRoot, err := blocks.BlockRoot(state, helpers.StartSlot(newJustifiedEpoch))
		if err != nil {
			return state, err
		}
		state.JustifiedRoot = newJustifedRoot
	}
	if newFinalizedEpoch != state.FinalizedEpoch {
		state.FinalizedEpoch = newFinalizedEpoch
		newFinalizedRoot, err := blocks.BlockRoot(state, helpers.StartSlot(newFinalizedEpoch))
		if err != nil {
			return state, err
		}
		state.FinalizedRoot = newFinalizedRoot
	}
	return state, nil
}

// ProcessCrosslinks goes through each crosslink committee and check
// crosslink committee's attested balance * 3 is greater than total balance *2.
// If it's greater then beacon node updates crosslink committee with
// the state epoch and wining root.
//
// Spec pseudocode definition:
//	For every slot in range(get_epoch_start_slot(previous_epoch), get_epoch_start_slot(next_epoch)),
// 	let `crosslink_committees_at_slot = get_crosslink_committees_at_slot(state, slot)`.
// 		For every `(crosslink_committee, shard)` in `crosslink_committees_at_slot`, compute:
// 			Set state.latest_crosslinks[shard] = Crosslink(
// 			epoch=slot_to_epoch(slot), crosslink_data_root=winning_root(crosslink_committee))
// 			if 3 * total_attesting_balance(crosslink_committee) >= 2 * total_balance(crosslink_committee)
func ProcessCrosslinks(
	state *pb.BeaconState,
	thisEpochAttestations []*pb.PendingAttestation,
	prevEpochAttestations []*pb.PendingAttestation) (*pb.BeaconState, error) {

	prevEpoch := helpers.PrevEpoch(state)
	currentEpoch := helpers.CurrentEpoch(state)
	nextEpoch := helpers.NextEpoch(state)
	startSlot := helpers.StartSlot(prevEpoch)
	endSlot := helpers.StartSlot(nextEpoch)

	for i := startSlot; i < endSlot; i++ {
		// RegistryChange is a no-op when requesting slot in current and previous epoch.
		// ProcessCrosslinks will never ask for slot in next epoch.
		crosslinkCommittees, err := helpers.CrosslinkCommitteesAtSlot(state, i, false /* registryChange */)
		if err != nil {
			return nil, fmt.Errorf("could not get committees for slot %d: %v", i-params.BeaconConfig().GenesisSlot, err)
		}
		for _, crosslinkCommittee := range crosslinkCommittees {
			shard := crosslinkCommittee.Shard
			committee := crosslinkCommittee.Committee
			attestingBalance, err := TotalAttestingBalance(state, shard, thisEpochAttestations, prevEpochAttestations)
			if err != nil {
				return nil, fmt.Errorf("could not get attesting balance for shard committee %d: %v", shard, err)
			}
			totalBalance := TotalBalance(state, committee)
			if attestingBalance*3 >= totalBalance*2 {
				winningRoot, err := winningRoot(state, shard, thisEpochAttestations, prevEpochAttestations)
				if err != nil {
					return nil, fmt.Errorf("could not get winning root: %v", err)
				}
				state.LatestCrosslinks[shard] = &pb.Crosslink{
					Epoch:                   currentEpoch,
					CrosslinkDataRootHash32: winningRoot,
				}
			}
		}
	}
	return state, nil
}

// ProcessEjections iterates through every validator and find the ones below
// ejection balance and eject them.
//
// Spec pseudocode definition:
//	def process_ejections(state: BeaconState) -> None:
//    """
//    Iterate through the validator registry
//    and eject active validators with balance below ``EJECTION_BALANCE``.
//    """
//    for index in get_active_validator_indices(state.validator_registry, current_epoch(state)):
//        if state.validator_balances[index] < EJECTION_BALANCE:
//            exit_validator(state, index)
func ProcessEjections(state *pb.BeaconState, enableLogging bool) (*pb.BeaconState, error) {
	activeValidatorIndices := helpers.ActiveValidatorIndices(state, helpers.CurrentEpoch(state))
	for _, index := range activeValidatorIndices {
		if state.Balances[index] < params.BeaconConfig().EjectionBalance {
			if enableLogging {
				log.WithFields(logrus.Fields{
					"pubKey": fmt.Sprintf("%#x", state.ValidatorRegistry[index].Pubkey),
					"index":  index}).Info("Validator ejected")
			}
			state = validators.ExitValidator(state, index)
		}
	}
	return state, nil
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

// ProcessCurrSlotShardSeed sets the current shuffling information in the beacon state.
//   Set state.current_shuffling_start_shard = (state.current_shuffling_start_shard +
//     get_current_epoch_committee_count(state)) % SHARD_COUNT
//   Set state.current_shuffling_epoch = next_epoch
//   Set state.current_shuffling_seed = generate_seed(state, state.current_shuffling_epoch)
func ProcessCurrSlotShardSeed(state *pb.BeaconState) (*pb.BeaconState, error) {
	state.CurrentShufflingStartShard = (state.CurrentShufflingStartShard +
		helpers.CurrentEpochCommitteeCount(state)) % params.BeaconConfig().ShardCount
	// TODO(#2072)we have removed the generation of a new seed for the timebeing to get it stable for the testnet.
	// this will be handled in Q2.
	state.CurrentShufflingEpoch = helpers.NextEpoch(state)
	return state, nil
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
	state.LatestIndexRootHash32S[nextEpoch%params.BeaconConfig().LatestActiveIndexRootsLength] =
		indexRoot[:]
	return state, nil
}

// ProcessJustification checks if there has been a new justified epoch.
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
	// There's no reason to process justification until the 2nd epoch.
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
	if oldCurrJustifiedEpoch == currentEpoch-2 && (bitfield>>1)%8 == 7 {
		state.FinalizedEpoch = oldCurrJustifiedEpoch
		state.FinalizedRoot = oldCurrJustifiedRoot
	}
	// when 1st, 2nd most recent epochs are all justified,
	// 1st epoch can finalize 2nd as a source.
	if oldCurrJustifiedEpoch == currentEpoch-1 && (bitfield>>1)%4 == 3 {
		state.FinalizedEpoch = oldCurrJustifiedEpoch
		state.FinalizedRoot = oldCurrJustifiedRoot
	}
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
		indices, err := helpers.AttestationParticipants(state, att.Data, att.AggregationBitfield)
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
		indices, err := helpers.AttestationParticipants(state, att.Data, att.AggregationBitfield)
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
		Epoch:                       epoch,
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
			Epoch:                       params.BeaconConfig().GenesisEpoch,
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

// CrosslinkAttestingIndices returns the attesting indices of the input crosslink.
func CrosslinkAttestingIndices(state *pb.BeaconState, crosslink *pb.Crosslink, atts []*pb.PendingAttestation) ([]uint64, error) {
	crosslinkAtts := attsForCrosslink(state, crosslink, atts)
	return UnslashedAttestingIndices(state, crosslinkAtts)
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

// TotalActiveBalance returns the combined balances of all the active validators.
// Spec pseudocode definition:
//	def get_total_active_balance(state: BeaconState) -> Gwei:
//    return get_total_balance(state, get_active_validator_indices(state, get_current_epoch(state)))
func totalActiveBalance(state *pb.BeaconState) uint64 {
	return TotalBalance(state, helpers.ActiveValidatorIndices(state, helpers.CurrentEpoch(state)))
}
