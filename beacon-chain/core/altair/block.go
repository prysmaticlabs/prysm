package altair

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	p2pType "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

// ProcessSyncAggregate verifies sync committee aggregate signature signing over the previous slot block root.
//
// Spec code:
// def process_sync_aggregate(state: BeaconState, sync_aggregate: SyncAggregate) -> None:
//    # Verify sync committee aggregate signature signing over the previous slot block root
//    committee_pubkeys = state.current_sync_committee.pubkeys
//    participant_pubkeys = [pubkey for pubkey, bit in zip(committee_pubkeys, sync_aggregate.sync_committee_bits) if bit]
//    previous_slot = max(state.slot, Slot(1)) - Slot(1)
//    domain = get_domain(state, DOMAIN_SYNC_COMMITTEE, compute_epoch_at_slot(previous_slot))
//    signing_root = compute_signing_root(get_block_root_at_slot(state, previous_slot), domain)
//    assert eth2_fast_aggregate_verify(participant_pubkeys, signing_root, sync_aggregate.sync_committee_signature)
//
//    # Compute participant and proposer rewards
//    total_active_increments = get_total_active_balance(state) // EFFECTIVE_BALANCE_INCREMENT
//    total_base_rewards = Gwei(get_base_reward_per_increment(state) * total_active_increments)
//    max_participant_rewards = Gwei(total_base_rewards * SYNC_REWARD_WEIGHT // WEIGHT_DENOMINATOR // SLOTS_PER_EPOCH)
//    participant_reward = Gwei(max_participant_rewards // SYNC_COMMITTEE_SIZE)
//    proposer_reward = Gwei(participant_reward * PROPOSER_WEIGHT // (WEIGHT_DENOMINATOR - PROPOSER_WEIGHT))
//
//    # Apply participant and proposer rewards
//    all_pubkeys = [v.pubkey for v in state.validators]
//    committee_indices = [ValidatorIndex(all_pubkeys.index(pubkey)) for pubkey in state.current_sync_committee.pubkeys]
//    for participant_index, participation_bit in zip(committee_indices, sync_aggregate.sync_committee_bits):
//        if participation_bit:
//            increase_balance(state, participant_index, participant_reward)
//            increase_balance(state, get_beacon_proposer_index(state), proposer_reward)
//        else:
//            decrease_balance(state, participant_index, participant_reward)
func ProcessSyncAggregate(ctx context.Context, s state.BeaconState, sync *ethpb.SyncAggregate) (state.BeaconState, error) {
	votedKeys, votedIndices, didntVoteIndices, err := FilterSyncCommitteeVotes(s, sync)
	if err != nil {
		return nil, errors.Wrap(err, "could not filter sync committee votes")
	}

	if err := VerifySyncCommitteeSig(s, votedKeys, sync.SyncCommitteeSignature); err != nil {
		return nil, errors.Wrap(err, "could not verify sync committee signature")
	}

	return ApplySyncRewardsPenalties(ctx, s, votedIndices, didntVoteIndices)
}

// FilterSyncCommitteeVotes filters the validator public keys and indices for the ones that voted and didn't vote.
func FilterSyncCommitteeVotes(s state.BeaconState, sync *ethpb.SyncAggregate) (
	votedKeys []bls.PublicKey,
	votedIndices []types.ValidatorIndex,
	didntVoteIndices []types.ValidatorIndex,
	err error) {
	currentSyncCommittee, err := s.CurrentSyncCommittee()
	if err != nil {
		return nil, nil, nil, err
	}
	if currentSyncCommittee == nil {
		return nil, nil, nil, errors.New("nil current sync committee in state")
	}
	committeeKeys := currentSyncCommittee.Pubkeys
	if sync.SyncCommitteeBits.Len() > uint64(len(committeeKeys)) {
		return nil, nil, nil, errors.New("bits length exceeds committee length")
	}
	votedKeys = make([]bls.PublicKey, 0, len(committeeKeys))
	votedIndices = make([]types.ValidatorIndex, 0, len(committeeKeys))
	didntVoteIndices = make([]types.ValidatorIndex, 0) // No allocation. Expect most votes.

	for i := uint64(0); i < sync.SyncCommitteeBits.Len(); i++ {
		vIdx, exists := s.ValidatorIndexByPubkey(bytesutil.ToBytes48(committeeKeys[i]))
		// Impossible scenario.
		if !exists {
			return nil, nil, nil, errors.New("validator public key does not exist in state")
		}

		if sync.SyncCommitteeBits.BitAt(i) {
			pubKey, err := bls.PublicKeyFromBytes(committeeKeys[i])
			if err != nil {
				return nil, nil, nil, err
			}
			votedKeys = append(votedKeys, pubKey)
			votedIndices = append(votedIndices, vIdx)
		} else {
			didntVoteIndices = append(didntVoteIndices, vIdx)
		}
	}
	return
}

// VerifySyncCommitteeSig verifies sync committee signature `syncSig` is valid with respect to public keys `syncKeys`.
func VerifySyncCommitteeSig(s state.BeaconState, syncKeys []bls.PublicKey, syncSig []byte) error {
	ps := slots.PrevSlot(s.Slot())
	d, err := signing.Domain(s.Fork(), slots.ToEpoch(ps), params.BeaconConfig().DomainSyncCommittee, s.GenesisValidatorsRoot())
	if err != nil {
		return err
	}
	pbr, err := helpers.BlockRootAtSlot(s, ps)
	if err != nil {
		return err
	}
	sszBytes := p2pType.SSZBytes(pbr)
	r, err := signing.ComputeSigningRoot(&sszBytes, d)
	if err != nil {
		return err
	}
	sig, err := bls.SignatureFromBytes(syncSig)
	if err != nil {
		return err
	}
	if !sig.Eth2FastAggregateVerify(syncKeys, r) {
		return errors.New("invalid sync committee signature")
	}
	return nil
}

// ApplySyncRewardsPenalties applies rewards and penalties for proposer and sync committee participants.
func ApplySyncRewardsPenalties(ctx context.Context, s state.BeaconState, votedIndices, didntVoteIndices []types.ValidatorIndex) (state.BeaconState, error) {
	activeBalance, err := helpers.TotalActiveBalance(s)
	if err != nil {
		return nil, err
	}
	proposerReward, participantReward, err := SyncRewards(activeBalance)
	if err != nil {
		return nil, err
	}

	// Apply sync committee rewards.
	earnedProposerReward := uint64(0)
	for _, index := range votedIndices {
		if err := helpers.IncreaseBalance(s, index, participantReward); err != nil {
			return nil, err
		}
		earnedProposerReward += proposerReward
	}
	// Apply proposer rewards.
	proposerIndex, err := helpers.BeaconProposerIndex(ctx, s)
	if err != nil {
		return nil, err
	}
	if err := helpers.IncreaseBalance(s, proposerIndex, earnedProposerReward); err != nil {
		return nil, err
	}
	// Apply sync committee penalties.
	for _, index := range didntVoteIndices {
		if err := helpers.DecreaseBalance(s, index, participantReward); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// SyncRewards returns the proposer reward and the sync participant reward given the total active balance in state.
func SyncRewards(activeBalance uint64) (proposerReward, participantReward uint64, err error) {
	cfg := params.BeaconConfig()
	totalActiveIncrements := activeBalance / cfg.EffectiveBalanceIncrement
	baseRewardPerInc, err := BaseRewardPerIncrement(activeBalance)
	if err != nil {
		return 0, 0, err
	}
	totalBaseRewards := baseRewardPerInc * totalActiveIncrements
	maxParticipantRewards := totalBaseRewards * cfg.SyncRewardWeight / cfg.WeightDenominator / uint64(cfg.SlotsPerEpoch)
	participantReward = maxParticipantRewards / cfg.SyncCommitteeSize
	proposerReward = participantReward * cfg.ProposerWeight / (cfg.WeightDenominator - cfg.ProposerWeight)
	return
}
