package altair_test

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/shared/params"
	altairState "github.com/prysmaticlabs/prysm/shared/testutil/altair"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProcessSyncCommitteeUpdates_CanRotate(t *testing.T) {
	s, _ := altairState.DeterministicGenesisStateAltair(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	postState, err := altair.ProcessSyncCommitteeUpdates(s)
	require.NoError(t, err)
	current, err := postState.CurrentSyncCommittee()
	require.NoError(t, err)
	next, err := postState.NextSyncCommittee()
	require.NoError(t, err)
	require.DeepEqual(t, current, next)

	require.NoError(t, s.SetSlot(params.BeaconConfig().SlotsPerEpoch))
	postState, err = altair.ProcessSyncCommitteeUpdates(s)
	require.NoError(t, err)
	c, err := postState.CurrentSyncCommittee()
	require.NoError(t, err)
	n, err := postState.NextSyncCommittee()
	require.NoError(t, err)
	require.DeepEqual(t, current, c)
	require.DeepEqual(t, next, n)

	require.NoError(t, s.SetSlot(types.Slot(params.BeaconConfig().EpochsPerSyncCommitteePeriod)*params.BeaconConfig().SlotsPerEpoch-1))
	postState, err = altair.ProcessSyncCommitteeUpdates(s)
	require.NoError(t, err)
	c, err = postState.CurrentSyncCommittee()
	require.NoError(t, err)
	n, err = postState.NextSyncCommittee()
	require.NoError(t, err)
	require.NotEqual(t, current, c)
	require.NotEqual(t, next, c)
	require.DeepEqual(t, next, c)
}
