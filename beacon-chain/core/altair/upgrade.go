package altair

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state/types"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/attestation"
)

// UpgradeToAltair updates input state to return the version Altair state.
//
// Spec code:
// def upgrade_to_altair(pre: phase0.BeaconState) -> BeaconState:
//
//	epoch = phase0.get_current_epoch(pre)
//	post = BeaconState(
//	    # Versioning
//	    genesis_time=pre.genesis_time,
//	    genesis_validators_root=pre.genesis_validators_root,
//	    slot=pre.slot,
//	    fork=Fork(
//	        previous_version=pre.fork.current_version,
//	        current_version=ALTAIR_FORK_VERSION,
//	        epoch=epoch,
//	    ),
//	    # History
//	    latest_block_header=pre.latest_block_header,
//	    block_roots=pre.block_roots,
//	    state_roots=pre.state_roots,
//	    historical_roots=pre.historical_roots,
//	    # Eth1
//	    eth1_data=pre.eth1_data,
//	    eth1_data_votes=pre.eth1_data_votes,
//	    eth1_deposit_index=pre.eth1_deposit_index,
//	    # Registry
//	    validators=pre.validators,
//	    balances=pre.balances,
//	    # Randomness
//	    randao_mixes=pre.randao_mixes,
//	    # Slashings
//	    slashings=pre.slashings,
//	    # Participation
//	    previous_epoch_participation=[ParticipationFlags(0b0000_0000) for _ in range(len(pre.validators))],
//	    current_epoch_participation=[ParticipationFlags(0b0000_0000) for _ in range(len(pre.validators))],
//	    # Finality
//	    justification_bits=pre.justification_bits,
//	    previous_justified_checkpoint=pre.previous_justified_checkpoint,
//	    current_justified_checkpoint=pre.current_justified_checkpoint,
//	    finalized_checkpoint=pre.finalized_checkpoint,
//	    # Inactivity
//	    inactivity_scores=[uint64(0) for _ in range(len(pre.validators))],
//	)
//	# Fill in previous epoch participation from the pre state's pending attestations
//	translate_participation(post, pre.previous_epoch_attestations)
//
//	# Fill in sync committees
//	# Note: A duplicate committee is assigned for the current and next committee at the fork boundary
//	post.current_sync_committee = get_next_sync_committee(post)
//	post.next_sync_committee = get_next_sync_committee(post)
//	return post
func UpgradeToAltair(ctx context.Context, st types.BeaconState) (types.BeaconState, error) {
	epoch := time.CurrentEpoch(st)

	numValidators := st.NumValidators()
	hrs, err := st.HistoricalRoots()
	if err != nil {
		return nil, err
	}
	s := &ethpb.BeaconStateAltair{
		GenesisTime:           st.GenesisTime(),
		GenesisValidatorsRoot: st.GenesisValidatorsRoot(),
		Slot:                  st.Slot(),
		Fork: &ethpb.Fork{
			PreviousVersion: st.Fork().CurrentVersion,
			CurrentVersion:  params.BeaconConfig().AltairForkVersion,
			Epoch:           epoch,
		},
		LatestBlockHeader:           st.LatestBlockHeader(),
		BlockRoots:                  st.BlockRoots(),
		StateRoots:                  st.StateRoots(),
		HistoricalRoots:             hrs,
		Eth1Data:                    st.Eth1Data(),
		Eth1DataVotes:               st.Eth1DataVotes(),
		Eth1DepositIndex:            st.Eth1DepositIndex(),
		Validators:                  st.Validators(),
		Balances:                    st.Balances(),
		RandaoMixes:                 st.RandaoMixes(),
		Slashings:                   st.Slashings(),
		PreviousEpochParticipation:  make([]byte, numValidators),
		CurrentEpochParticipation:   make([]byte, numValidators),
		JustificationBits:           st.JustificationBits(),
		PreviousJustifiedCheckpoint: st.PreviousJustifiedCheckpoint(),
		CurrentJustifiedCheckpoint:  st.CurrentJustifiedCheckpoint(),
		FinalizedCheckpoint:         st.FinalizedCheckpoint(),
		InactivityScores:            make([]uint64, numValidators),
	}

	newState, err := state.InitializeFromProtoUnsafeAltair(s)
	if err != nil {
		return nil, err
	}
	prevEpochAtts, err := st.PreviousEpochAttestations()
	if err != nil {
		return nil, err
	}
	newState, err = TranslateParticipation(ctx, newState, prevEpochAtts)
	if err != nil {
		return nil, err
	}

	committee, err := NextSyncCommittee(ctx, newState)
	if err != nil {
		return nil, err
	}
	if err := newState.SetCurrentSyncCommittee(committee); err != nil {
		return nil, err
	}
	if err := newState.SetNextSyncCommittee(committee); err != nil {
		return nil, err
	}
	return newState, nil
}

// TranslateParticipation translates pending attestations into participation bits, then inserts the bits into beacon state.
// This is helper function to convert phase 0 beacon state(pending_attestations) to Altair beacon state(participation_bits).
//
// Spec code:
// def translate_participation(state: BeaconState, pending_attestations: Sequence[phase0.PendingAttestation]) -> None:
//
//	for attestation in pending_attestations:
//	    data = attestation.data
//	    inclusion_delay = attestation.inclusion_delay
//	    # Translate attestation inclusion info to flag indices
//	    participation_flag_indices = get_attestation_participation_flag_indices(state, data, inclusion_delay)
//
//	    # Apply flags to all attesting validators
//	    epoch_participation = state.previous_epoch_participation
//	    for index in get_attesting_indices(state, data, attestation.aggregation_bits):
//	        for flag_index in participation_flag_indices:
//	            epoch_participation[index] = add_flag(epoch_participation[index], flag_index)
func TranslateParticipation(ctx context.Context, st types.BeaconState, atts []*ethpb.PendingAttestation) (types.BeaconState, error) {
	epochParticipation, err := st.PreviousEpochParticipation()
	if err != nil {
		return nil, err
	}

	for _, att := range atts {
		participatedFlags, err := AttestationParticipationFlagIndices(st, att.Data, att.InclusionDelay)
		if err != nil {
			return nil, err
		}
		committee, err := helpers.BeaconCommitteeFromState(ctx, st, att.Data.Slot, att.Data.CommitteeIndex)
		if err != nil {
			return nil, err
		}
		indices, err := attestation.AttestingIndices(att.AggregationBits, committee)
		if err != nil {
			return nil, err
		}
		cfg := params.BeaconConfig()
		sourceFlagIndex := cfg.TimelySourceFlagIndex
		targetFlagIndex := cfg.TimelyTargetFlagIndex
		headFlagIndex := cfg.TimelyHeadFlagIndex
		for _, index := range indices {
			has, err := HasValidatorFlag(epochParticipation[index], sourceFlagIndex)
			if err != nil {
				return nil, err
			}
			if participatedFlags[sourceFlagIndex] && !has {
				epochParticipation[index], err = AddValidatorFlag(epochParticipation[index], sourceFlagIndex)
				if err != nil {
					return nil, err
				}
			}
			has, err = HasValidatorFlag(epochParticipation[index], targetFlagIndex)
			if err != nil {
				return nil, err
			}
			if participatedFlags[targetFlagIndex] && !has {
				epochParticipation[index], err = AddValidatorFlag(epochParticipation[index], targetFlagIndex)
				if err != nil {
					return nil, err
				}
			}
			has, err = HasValidatorFlag(epochParticipation[index], headFlagIndex)
			if err != nil {
				return nil, err
			}
			if participatedFlags[headFlagIndex] && !has {
				epochParticipation[index], err = AddValidatorFlag(epochParticipation[index], headFlagIndex)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if err := st.SetPreviousParticipationBits(epochParticipation); err != nil {
		return nil, err
	}

	return st, nil
}
