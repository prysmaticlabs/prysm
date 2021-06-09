package synccommittee

import (
	"context"
	"testing"
	"time"

	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestPruneExpiredSyncCommitteeSignatures(t *testing.T) {
	service := NewService(context.Background(), NewStore())
	sigs := []*eth.SyncCommitteeMessage{
		{Slot: 1, ValidatorIndex: 0, Signature: []byte{'a'}},
		{Slot: 1, ValidatorIndex: 1, Signature: []byte{'b'}},
		{Slot: 1, ValidatorIndex: 2, Signature: []byte{'c'}},
		{Slot: 2, ValidatorIndex: 0, Signature: []byte{'d'}},
		{Slot: 2, ValidatorIndex: 1, Signature: []byte{'e'}},
		{Slot: 3, ValidatorIndex: 0, Signature: []byte{'f'}},
		{Slot: 3, ValidatorIndex: 1, Signature: []byte{'g'}},
	}
	for _, sig := range sigs {
		require.NoError(t, service.store.SaveSyncCommitteeSignature(sig))
	}

	// Set 3 slots into the future.
	time := time.Now().Add(time.Duration(-3*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second)
	service.SetGenesisTime(uint64(time.Unix()))
	service.pruneExpiredSyncCommitteeSignatures()

	sigs = service.store.SyncCommitteeSignatures(1)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeMessage{}, sigs)

	sigs = service.store.SyncCommitteeSignatures(2)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeMessage{
		{Slot: 2, ValidatorIndex: 0, Signature: []byte{'d'}},
		{Slot: 2, ValidatorIndex: 1, Signature: []byte{'e'}},
	}, sigs)

	sigs = service.store.SyncCommitteeSignatures(3)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeMessage{
		{Slot: 3, ValidatorIndex: 0, Signature: []byte{'f'}},
		{Slot: 3, ValidatorIndex: 1, Signature: []byte{'g'}},
	}, sigs)
}

func TestPruneExpiredSyncCommitteeContributions(t *testing.T) {
	service := NewService(context.Background(), NewStore())
	sigs := []*eth.SyncCommitteeContribution{
		{Slot: 1, SubcommitteeIndex: 0, Signature: []byte{'a'}},
		{Slot: 1, SubcommitteeIndex: 1, Signature: []byte{'b'}},
		{Slot: 1, SubcommitteeIndex: 2, Signature: []byte{'c'}},
		{Slot: 2, SubcommitteeIndex: 0, Signature: []byte{'d'}},
		{Slot: 2, SubcommitteeIndex: 1, Signature: []byte{'e'}},
		{Slot: 3, SubcommitteeIndex: 0, Signature: []byte{'f'}},
		{Slot: 3, SubcommitteeIndex: 1, Signature: []byte{'g'}},
	}
	for _, sig := range sigs {
		require.NoError(t, service.store.SaveSyncCommitteeContribution(sig))
	}

	// Set 3 slots into the future.
	time := time.Now().Add(time.Duration(-3*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second)
	service.SetGenesisTime(uint64(time.Unix()))
	service.pruneExpiredSyncCommitteeContributions()

	sigs = service.store.SyncCommitteeContributions(1)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeContribution{}, sigs)

	sigs = service.store.SyncCommitteeContributions(2)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeContribution{
		{Slot: 2, SubcommitteeIndex: 0, Signature: []byte{'d'}},
		{Slot: 2, SubcommitteeIndex: 1, Signature: []byte{'e'}},
	}, sigs)

	sigs = service.store.SyncCommitteeContributions(3)
	require.DeepSSZEqual(t, []*eth.SyncCommitteeContribution{
		{Slot: 3, SubcommitteeIndex: 0, Signature: []byte{'f'}},
		{Slot: 3, SubcommitteeIndex: 1, Signature: []byte{'g'}},
	}, sigs)
}
