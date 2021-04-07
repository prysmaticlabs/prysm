package altair

import (
	"errors"

	"github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	p2pType "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessSyncCommittee verifies sync committee aggregate signature signing over the previous slot block root.
//
// Spec code:
// def process_sync_committee(state: BeaconState, aggregate: SyncAggregate) -> None:
//    # Verify sync committee aggregate signature signing over the previous slot block root
//    committee_pubkeys = state.current_sync_committee.pubkeys
//    participant_pubkeys = [pubkey for pubkey, bit in zip(committee_pubkeys, aggregate.sync_committee_bits) if bit]
//    previous_slot = max(state.slot, Slot(1)) - Slot(1)
//    domain = get_domain(state, DOMAIN_SYNC_COMMITTEE, compute_epoch_at_slot(previous_slot))
//    signing_root = compute_signing_root(get_block_root_at_slot(state, previous_slot), domain)
//    assert eth2_fast_aggregate_verify(participant_pubkeys, signing_root, aggregate.sync_committee_signature)
//
//    # Compute participant and proposer rewards
//    total_active_increments = get_total_active_balance(state) // EFFECTIVE_BALANCE_INCREMENT
//    total_base_rewards = Gwei(get_base_reward_per_increment(state) * total_active_increments)
//    max_participant_rewards = Gwei(total_base_rewards * SYNC_REWARD_WEIGHT // WEIGHT_DENOMINATOR // SLOTS_PER_EPOCH)
//    participant_reward = Gwei(max_participant_rewards // SYNC_COMMITTEE_SIZE)
//    proposer_reward = Gwei(participant_reward * PROPOSER_WEIGHT // (WEIGHT_DENOMINATOR - PROPOSER_WEIGHT))
//
//    # Apply participant and proposer rewards
//    committee_indices = get_sync_committee_indices(state, get_current_epoch(state))
//    participant_indices = [index for index, bit in zip(committee_indices, aggregate.sync_committee_bits) if bit]
//    for participant_index in participant_indices:
//        increase_balance(state, participant_index, participant_reward)
//        increase_balance(state, get_beacon_proposer_index(state), proposer_reward)
func ProcessSyncCommittee(state iface.BeaconStateAltair, sync *ethpb.SyncAggregate) (iface.BeaconStateAltair, error) {
	committeeIndices, err := SyncCommitteeIndices(state, helpers.CurrentEpoch(state))
	if err != nil {
		return nil, err
	}

	currentSyncCommittee, err := state.CurrentSyncCommittee()
	if err != nil {
		return nil, err
	}
	committeeKeys := currentSyncCommittee.Pubkeys
	votedKeys := make([]bls.PublicKey, 0, len(committeeKeys))
	votedIndices := make([]types.ValidatorIndex, 0, len(committeeKeys))

	// Verify sync committee signature.
	for i := uint64(0); i < sync.SyncCommitteeBits.Len(); i++ {
		if sync.SyncCommitteeBits.BitAt(i) {
			pubKey, err := bls.PublicKeyFromBytes(committeeKeys[i])
			if err != nil {
				return nil, err
			}
			votedKeys = append(votedKeys, pubKey)
			votedIndices = append(votedIndices, committeeIndices[i])
		}
	}
	ps := helpers.PrevSlot(state.Slot())
	d, err := helpers.Domain(state.Fork(), helpers.SlotToEpoch(ps), params.BeaconConfig().DomainSyncCommittee, state.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	pbr, err := helpers.BlockRootAtSlot(state, ps)
	if err != nil {
		return nil, err
	}
	sszBytes := p2pType.SSZBytes(pbr)
	r, err := helpers.ComputeSigningRoot(&sszBytes, d)
	if err != nil {
		return nil, err
	}
	sig, err := bls.SignatureFromBytes(sync.SyncCommitteeSignature)
	if err != nil {
		return nil, err
	}
	if !sig.FastAggregateVerify(votedKeys, r) {
		return nil, errors.New("could not verify sync committee signature")
	}

	// Calculate sync committee and proposer rewards
	activeBalance, err := helpers.TotalActiveBalance(state)
	if err != nil {
		return nil, err
	}
	totalActiveIncrements := activeBalance / params.BeaconConfig().EffectiveBalanceIncrement
	totalBaseRewards := baseRewardPerIncrement(activeBalance) * totalActiveIncrements
	maxParticipantRewards := totalBaseRewards * params.BeaconConfig().SyncRewardWeight / params.BeaconConfig().WeightDenominator / uint64(params.BeaconConfig().SlotsPerEpoch)
	participantReward := maxParticipantRewards / params.BeaconConfig().SyncCommitteeSize
	proposerReward := participantReward * params.BeaconConfig().ProposerWeight / (params.BeaconConfig().WeightDenominator - params.BeaconConfig().ProposerWeight)

	// Apply sync committee rewards.
	earnedProposerReward := uint64(0)
	for _, index := range votedIndices {
		if err := helpers.IncreaseBalance(state, index, participantReward); err != nil {
			return nil, err
		}
		earnedProposerReward += proposerReward
	}
	// Apply proposer rewards.
	proposerIndex, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		return nil, err
	}
	if err := helpers.IncreaseBalance(state, proposerIndex, earnedProposerReward); err != nil {
		return nil, err
	}

	return state, nil
}

// This returns the base reward per increment for the beacon state.
//
// def get_base_reward_per_increment(state: BeaconState) -> Gwei:
//    return Gwei(EFFECTIVE_BALANCE_INCREMENT * BASE_REWARD_FACTOR // integer_squareroot(get_total_active_balance(state))
func baseRewardPerIncrement(activeBalance uint64) uint64 {
	return params.BeaconConfig().EffectiveBalanceIncrement * params.BeaconConfig().BaseRewardFactor / mathutil.IntegerSquareRoot(activeBalance)
}
