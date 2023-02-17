package rewards

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

type slashValidatorFunc func(
	ctx context.Context,
	st state.BeaconState,
	vid primitives.ValidatorIndex,
	penaltyQuotient,
	proposerRewardQuotient uint64,
) (state.BeaconState, uint64, error)

func processProposerSlashings(
	ctx context.Context,
	beaconState state.BeaconState,
	slashings []*ethpb.ProposerSlashing,
	slashFunc slashValidatorFunc,
	r *BlockRewardsInfo,
) (state.BeaconState, error) {
	var err error
	var totalReward uint64
	for _, slashing := range slashings {
		var reward uint64
		beaconState, reward, err = processProposerSlashing(ctx, beaconState, slashing, slashFunc)
		if err != nil {
			return nil, err
		}
		totalReward += reward
	}

	r.ProposerSlashings = totalReward
	return beaconState, nil
}

func processProposerSlashing(
	ctx context.Context,
	beaconState state.BeaconState,
	slashing *ethpb.ProposerSlashing,
	slashFunc slashValidatorFunc,
) (state.BeaconState, uint64, error) {
	var err error
	if slashing == nil {
		return nil, 0, errors.New("nil proposer slashings in block body")
	}
	cfg := params.BeaconConfig()
	var slashingQuotient uint64
	switch {
	case beaconState.Version() == version.Phase0:
		slashingQuotient = cfg.MinSlashingPenaltyQuotient
	case beaconState.Version() == version.Altair:
		slashingQuotient = cfg.MinSlashingPenaltyQuotientAltair
	case beaconState.Version() >= version.Bellatrix:
		slashingQuotient = cfg.MinSlashingPenaltyQuotientBellatrix
	default:
		return nil, 0, errors.New("unknown state version")
	}
	var reward uint64
	beaconState, reward, err = slashFunc(ctx, beaconState, slashing.Header_1.Header.ProposerIndex, slashingQuotient, cfg.ProposerRewardQuotient)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "could not slash proposer index %d", slashing.Header_1.Header.ProposerIndex)
	}
	return beaconState, reward, nil
}

func processAttesterSlashings(
	ctx context.Context,
	beaconState state.BeaconState,
	slashings []*ethpb.AttesterSlashing,
	slashFunc slashValidatorFunc,
	r *BlockRewardsInfo,
) (state.BeaconState, error) {
	var err error
	var totalReward uint64
	for _, slashing := range slashings {
		var reward uint64
		beaconState, reward, err = processAttesterSlashing(ctx, beaconState, slashing, slashFunc)
		if err != nil {
			return nil, err
		}
		totalReward += reward
	}

	r.AttesterSlashings = totalReward
	return beaconState, nil
}

func processAttesterSlashing(
	ctx context.Context,
	beaconState state.BeaconState,
	slashing *ethpb.AttesterSlashing,
	slashFunc slashValidatorFunc,
) (state.BeaconState, uint64, error) {
	slashableIndices := blocks.SlashableAttesterIndices(slashing)
	sort.SliceStable(slashableIndices, func(i, j int) bool {
		return slashableIndices[i] < slashableIndices[j]
	})
	currentEpoch := slots.ToEpoch(beaconState.Slot())
	var err error
	var slashedAny bool
	var val state.ReadOnlyValidator
	var totalReward uint64
	for _, validatorIndex := range slashableIndices {
		val, err = beaconState.ValidatorAtIndexReadOnly(primitives.ValidatorIndex(validatorIndex))
		if err != nil {
			return nil, 0, err
		}
		if helpers.IsSlashableValidator(val.ActivationEpoch(), val.WithdrawableEpoch(), val.Slashed(), currentEpoch) {
			cfg := params.BeaconConfig()
			var slashingQuotient uint64
			switch {
			case beaconState.Version() == version.Phase0:
				slashingQuotient = cfg.MinSlashingPenaltyQuotient
			case beaconState.Version() == version.Altair:
				slashingQuotient = cfg.MinSlashingPenaltyQuotientAltair
			case beaconState.Version() >= version.Bellatrix:
				slashingQuotient = cfg.MinSlashingPenaltyQuotientBellatrix
			default:
				return nil, 0, errors.New("unknown state version")
			}
			var reward uint64
			beaconState, reward, err = slashFunc(ctx, beaconState, primitives.ValidatorIndex(validatorIndex), slashingQuotient, cfg.ProposerRewardQuotient)
			if err != nil {
				return nil, 0, errors.Wrapf(err, "could not slash validator index %d",
					validatorIndex)
			}
			totalReward += reward
			slashedAny = true
		}
	}
	if !slashedAny {
		return nil, 0, errors.New("unable to slash any validator despite confirmed attester slashing")
	}
	return beaconState, totalReward, nil
}

func slashValidator(
	ctx context.Context,
	s state.BeaconState,
	slashedIdx primitives.ValidatorIndex,
	penaltyQuotient uint64,
	proposerRewardQuotient uint64) (state.BeaconState, uint64, error) {
	currentEpoch := slots.ToEpoch(s.Slot())
	validator, err := s.ValidatorAtIndex(slashedIdx)
	if err != nil {
		return nil, 0, err
	}
	validator.Slashed = true
	maxWithdrawableEpoch := primitives.MaxEpoch(validator.WithdrawableEpoch, currentEpoch+params.BeaconConfig().EpochsPerSlashingsVector)
	validator.WithdrawableEpoch = maxWithdrawableEpoch

	if err := s.UpdateValidatorAtIndex(slashedIdx, validator); err != nil {
		return nil, 0, err
	}

	// The slashing amount is represented by epochs per slashing vector. The validator's effective balance is then applied to that amount.
	slashings := s.Slashings()
	currentSlashing := slashings[currentEpoch%params.BeaconConfig().EpochsPerSlashingsVector]
	if err := s.UpdateSlashingsAtIndex(
		uint64(currentEpoch%params.BeaconConfig().EpochsPerSlashingsVector),
		currentSlashing+validator.EffectiveBalance,
	); err != nil {
		return nil, 0, err
	}
	if err := helpers.DecreaseBalance(s, slashedIdx, validator.EffectiveBalance/penaltyQuotient); err != nil {
		return nil, 0, err
	}

	proposerIdx, err := helpers.BeaconProposerIndex(ctx, s)
	if err != nil {
		return nil, 0, errors.Wrap(err, "could not get proposer idx")
	}
	whistleBlowerIdx := proposerIdx
	whistleblowerReward := validator.EffectiveBalance / params.BeaconConfig().WhistleBlowerRewardQuotient
	proposerReward := whistleblowerReward / proposerRewardQuotient
	err = helpers.IncreaseBalance(s, proposerIdx, proposerReward)
	if err != nil {
		return nil, 0, err
	}
	err = helpers.IncreaseBalance(s, whistleBlowerIdx, whistleblowerReward-proposerReward)
	if err != nil {
		return nil, 0, err
	}
	return s, proposerReward, nil
}
