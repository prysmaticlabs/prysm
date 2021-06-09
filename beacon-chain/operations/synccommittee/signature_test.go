package synccommittee

import (
	"testing"

	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSyncCommitteeSignatureCache_Nil(t *testing.T) {
	store := NewStore()
	require.Equal(t, nilSignatureErr, store.SaveSyncCommitteeSignature(nil))
}

func TestSyncCommitteeSignatureCache_RoundTrip(t *testing.T) {
	store := NewStore()

	sigs := []*eth.SyncCommitteeMessage{
		{Slot: 1, ValidatorIndex: 0, Signature: []byte{'a'}},
		{Slot: 1, ValidatorIndex: 1, Signature: []byte{'b'}},
		{Slot: 1, ValidatorIndex: 2, Signature: []byte{'c'}},
		{Slot: 2, ValidatorIndex: 0, Signature: []byte{'d'}},
		{Slot: 2, ValidatorIndex: 1, Signature: []byte{'e'}},
	}

	for _, sig := range sigs {
		require.NoError(t, store.SaveSyncCommitteeSignature(sig))
	}

	sigs = store.SyncCommitteeSignatures(1)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeMessage{
		{Slot: 1, ValidatorIndex: 0, Signature: []byte{'a'}},
		{Slot: 1, ValidatorIndex: 1, Signature: []byte{'b'}},
		{Slot: 1, ValidatorIndex: 2, Signature: []byte{'c'}},
	}, sigs)

	sigs = store.SyncCommitteeSignatures(2)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeMessage{
		{Slot: 2, ValidatorIndex: 0, Signature: []byte{'d'}},
		{Slot: 2, ValidatorIndex: 1, Signature: []byte{'e'}},
	}, sigs)

	store.DeleteSyncCommitteeSignatures(1)
	sigs = store.SyncCommitteeSignatures(1)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeMessage{}, sigs)

	sigs = store.SyncCommitteeSignatures(2)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeMessage{
		{Slot: 2, ValidatorIndex: 0, Signature: []byte{'d'}},
		{Slot: 2, ValidatorIndex: 1, Signature: []byte{'e'}},
	}, sigs)

	store.DeleteSyncCommitteeSignatures(2)
	sigs = store.SyncCommitteeSignatures(2)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeMessage{}, sigs)
}
