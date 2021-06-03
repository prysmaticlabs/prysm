package synccommittee

import (
	"testing"

	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSyncCommitteeContributionCache_Nil(t *testing.T) {
	store := NewStore()
	require.Equal(t, nilContributionErr, store.SaveSyncCommitteeContribution(nil))
}

func TestSyncCommitteeContributionCache_RoundTrip(t *testing.T) {
	store := NewStore()

	sigs := []*eth.SyncCommitteeContribution{
		{Slot: 1, SubcommitteeIndex: 0, Signature: []byte{'a'}},
		{Slot: 1, SubcommitteeIndex: 1, Signature: []byte{'b'}},
		{Slot: 1, SubcommitteeIndex: 2, Signature: []byte{'c'}},
		{Slot: 2, SubcommitteeIndex: 0, Signature: []byte{'d'}},
		{Slot: 2, SubcommitteeIndex: 1, Signature: []byte{'e'}},
	}

	for _, sig := range sigs {
		require.NoError(t, store.SaveSyncCommitteeContribution(sig))
	}

	sigs = store.SyncCommitteeContributions(1)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeContribution{
		{Slot: 1, SubcommitteeIndex: 0, Signature: []byte{'a'}},
		{Slot: 1, SubcommitteeIndex: 1, Signature: []byte{'b'}},
		{Slot: 1, SubcommitteeIndex: 2, Signature: []byte{'c'}},
	}, sigs)

	sigs = store.SyncCommitteeContributions(2)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeContribution{
		{Slot: 2, SubcommitteeIndex: 0, Signature: []byte{'d'}},
		{Slot: 2, SubcommitteeIndex: 1, Signature: []byte{'e'}},
	}, sigs)

	store.DeleteSyncCommitteeContributions(1)
	sigs = store.SyncCommitteeContributions(1)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeContribution{}, sigs)

	sigs = store.SyncCommitteeContributions(2)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeContribution{
		{Slot: 2, SubcommitteeIndex: 0, Signature: []byte{'d'}},
		{Slot: 2, SubcommitteeIndex: 1, Signature: []byte{'e'}},
	}, sigs)

	store.DeleteSyncCommitteeContributions(2)
	sigs = store.SyncCommitteeContributions(2)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeContribution{}, sigs)
}
