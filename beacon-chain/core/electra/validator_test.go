package electra_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestQueueExcessActiveBalance_Ok(t *testing.T) {
	st, _ := util.DeterministicGenesisStateElectra(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	bals := st.Balances()
	bals[0] = params.BeaconConfig().MinActivationBalance + 1000
	require.NoError(t, st.SetBalances(bals))

	err := electra.QueueExcessActiveBalance(st, 0)
	require.NoError(t, err)

	pbd, err := st.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, uint64(1000), pbd[0].Amount)

	bals = st.Balances()
	require.Equal(t, params.BeaconConfig().MinActivationBalance, bals[0])
}

func TestQueueEntireBalanceAndResetValidator_Ok(t *testing.T) {
	st, _ := util.DeterministicGenesisStateElectra(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	val, err := st.ValidatorAtIndex(0)
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MaxEffectiveBalance, val.EffectiveBalance)
	pbd, err := st.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, 0, len(pbd))
	err = electra.QueueEntireBalanceAndResetValidator(st, 0)
	require.NoError(t, err)

	pbd, err = st.PendingBalanceDeposits()
	require.NoError(t, err)
	require.Equal(t, 1, len(pbd))

	val, err = st.ValidatorAtIndex(0)
	require.NoError(t, err)
	require.Equal(t, uint64(0), val.EffectiveBalance)
}
