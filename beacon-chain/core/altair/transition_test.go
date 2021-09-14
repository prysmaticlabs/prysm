package altair_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/shared/params"
	testing2 "github.com/prysmaticlabs/prysm/testing"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestProcessEpoch_CanProcess(t *testing.T) {
	st, _ := testing2.DeterministicGenesisStateAltair(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	require.NoError(t, st.SetSlot(10*params.BeaconConfig().SlotsPerEpoch))
	newState, err := altair.ProcessEpoch(context.Background(), st)
	require.NoError(t, err)
	require.Equal(t, uint64(0), newState.Slashings()[2], "Unexpected slashed balance")

	b := st.Balances()
	require.Equal(t, params.BeaconConfig().MaxValidatorsPerCommittee, uint64(len(b)))
	require.Equal(t, uint64(31999841265), b[0])

	s, err := st.InactivityScores()
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MaxValidatorsPerCommittee, uint64(len(s)))

	p, err := st.PreviousEpochParticipation()
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MaxValidatorsPerCommittee, uint64(len(p)))

	p, err = st.CurrentEpochParticipation()
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MaxValidatorsPerCommittee, uint64(len(p)))

	sc, err := st.CurrentSyncCommittee()
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().SyncCommitteeSize, uint64(len(sc.Pubkeys)))

	sc, err = st.NextSyncCommittee()
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().SyncCommitteeSize, uint64(len(sc.Pubkeys)))
}
