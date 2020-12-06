package blocks

import (
	"errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessLightClientAggregate verifies sync committee aggregate signature signing over the previous slot block root.
//
// def process_sync_committee(state: BeaconState, body: BeaconBlockBody) -> None:
//    # Verify sync committee aggregate signature signing over the previous slot block root
//    previous_slot = max(state.slot, Slot(1)) - Slot(1)
//    committee_indices = get_sync_committee_indices(state, get_current_epoch(state))
//    participant_indices = [committee_indices[i] for i in range(len(committee_indices)) if body.sync_committee_bits[i]]
//    participant_pubkeys = [state.validators[participant_index].pubkey for participant_index in participant_indices]
//    domain = get_domain(state, DOMAIN_SYNC_COMMITTEE, compute_epoch_at_slot(previous_slot))
//    signing_root = compute_signing_root(get_block_root_at_slot(state, previous_slot), domain)
//    assert bls.FastAggregateVerify(participant_pubkeys, signing_root, body.sync_committee_signature)
//
//    # Reward sync committee participants
//    participant_rewards = Gwei(0)
//    active_validator_count = uint64(len(get_active_validator_indices(state, get_current_epoch(state))))
//    for participant_index in participant_indices:
//        base_reward = get_base_reward(state, participant_index)
//        reward = Gwei(base_reward * active_validator_count // len(committee_indices) // SLOTS_PER_EPOCH)
//        increase_balance(state, participant_index, reward)
//        participant_rewards += reward
//
//    # Reward beacon proposer
//    increase_balance(state, get_beacon_proposer_index(state), Gwei(participant_rewards // PROPOSER_REWARD_QUOTIENT))
func ProcessLightClientAggregate(state *state.BeaconState, body *ethpb.BeaconBlockBody) (*state.BeaconState, error) {
	indices, err := helpers.SyncCommitteeIndices(state, helpers.CurrentEpoch(state))
	if err != nil {
		return nil, err
	}
	pPubKeys := make([]bls.PublicKey, 0, len(indices))
	pIndices := make([]uint64, 0, len(indices))
	for i, index := range indices {
		if body.SyncCommitteeBits.BitAt(uint64(i)) {
			v, err := state.ValidatorAtIndex(index)
			if err != nil {
				return nil, err
			}
			p, err := bls.PublicKeyFromBytes(v.PublicKey)
			if err != nil {
				return nil, err
			}
			pPubKeys = append(pPubKeys, p)
			pIndices = append(pIndices, index)
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
	r, err := helpers.ComputeSigningRoot(pbr, d)
	if err != nil {
		return nil, err
	}
	sig, err := bls.SignatureFromBytes(body.SyncCommitteeSignature)
	if err != nil {
		return nil, err
	}
	if !sig.FastAggregateVerify(pPubKeys, r) {
		return nil, errors.New("could not verify sync committee signature")
	}

	proposerRewards := uint64(0)
	aCount, err := helpers.ActiveValidatorCount(state, helpers.CurrentEpoch(state))
	if err != nil {
		return nil, err
	}
	for _, index := range pIndices {
		br, err := epoch.BaseReward(state, index)
		if err != nil {
			return nil, err
		}
		r := br * aCount / uint64(len(indices)) / params.BeaconConfig().SlotsPerEpoch
		if err := helpers.IncreaseBalance(state, index, r); err != nil {
			return nil, err
		}
		proposerRewards += r
	}
	proposerIndex, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		return nil, err
	}
	if err := helpers.IncreaseBalance(state, proposerIndex, proposerRewards/params.BeaconConfig().ProposerRewardQuotient); err != nil {
		return nil, err
	}
	return state, nil
}
