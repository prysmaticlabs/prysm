package blocks

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/math"
)

type validatorInfo struct {
	isPrevEpochAttester bool
	isSlashed           bool
}

func AttestationRewards(st state.ReadOnlyBeaconState, b interfaces.BeaconBlockBody) (uint64, error) {
	currEpochActiveBalance := uint64(0)
	if err := st.ReadFromEveryValidator(func(idx int, val state.ReadOnlyValidator) error {
		currEpochActiveBalance += val.EffectiveBalance()
		return nil
	}); err != nil {
		return 0, errors.Wrap(err, "failed to read validator data")
	}
	balanceSqrt := math.IntegerSquareRoot(currEpochActiveBalance)
	// Balance square root cannot be 0, this prevents division by 0.
	if balanceSqrt == 0 {
		balanceSqrt = 1
	}

	baseRewardFactor := params.BeaconConfig().BaseRewardFactor
	baseRewardsPerEpoch := params.BeaconConfig().BaseRewardsPerEpoch
	proposerRewardQuotient := params.BeaconConfig().ProposerRewardQuotient
	baseReward := currEpochActiveBalance * baseRewardFactor / balanceSqrt / baseRewardsPerEpoch
	proposerReward := baseReward / proposerRewardQuotient

	return proposerReward * uint64(len(b.Attestations())), nil
}
