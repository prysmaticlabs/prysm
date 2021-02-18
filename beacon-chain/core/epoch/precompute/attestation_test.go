package precompute_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestUpdateBalance(t *testing.T) {
	vp := []*precompute.Validator{
		{CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsCurrentEpochTargetAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsCurrentEpochTargetAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsPrevEpochSourceAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsPrevEpochSourceAttester: true, IsPrevEpochTargetAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsPrevEpochHeadAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsPrevEpochSourceAttester: true, IsPrevEpochHeadAttester: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
		{IsSlashed: true, CurrentEpochEffectiveBalance: 100 * params.BeaconConfig().EffectiveBalanceIncrement},
	}
	wantedPBal := &precompute.Balance{
		ActiveCurrentEpoch:         params.BeaconConfig().EffectiveBalanceIncrement,
		ActivePrevEpoch:            params.BeaconConfig().EffectiveBalanceIncrement,
		CurrentEpochTargetAttested: 200 * params.BeaconConfig().EffectiveBalanceIncrement,
		PrevEpochSourceAttested:    300 * params.BeaconConfig().EffectiveBalanceIncrement,
		PrevEpochTargetAttested:    100 * params.BeaconConfig().EffectiveBalanceIncrement,
		PrevEpochHeadAttested:      200 * params.BeaconConfig().EffectiveBalanceIncrement,
	}
	pBal := precompute.UpdateBalance(vp, &precompute.Balance{})
	assert.DeepEqual(t, wantedPBal, pBal, "Incorrect balance calculations")
}

func TestProcessAttestations(t *testing.T) {
}

func TestEnsureBalancesLowerBound(t *testing.T) {
	b := &precompute.Balance{}
	b = precompute.EnsureBalancesLowerBound(b)
	balanceIncrement := params.BeaconConfig().EffectiveBalanceIncrement
	assert.Equal(t, balanceIncrement, b.ActiveCurrentEpoch, "Did not get wanted active current balance")
	assert.Equal(t, balanceIncrement, b.ActivePrevEpoch, "Did not get wanted active previous balance")
	assert.Equal(t, balanceIncrement, b.CurrentEpochTargetAttested, "Did not get wanted target attested balance")
	assert.Equal(t, balanceIncrement, b.PrevEpochSourceAttested, "Did not get wanted prev attested balance")
	assert.Equal(t, balanceIncrement, b.PrevEpochTargetAttested, "Did not get wanted prev target attested balance")
	assert.Equal(t, balanceIncrement, b.PrevEpochHeadAttested, "Did not get wanted prev head attested balance")
}
