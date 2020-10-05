package kv

import (
	"context"
	"sort"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/slasher/db/types"
)

func TestStore_AttesterSlashingNilBucket(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	as := &ethpb.AttesterSlashing{
		Attestation_1: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: make([]byte, 32),
				Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
				Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
			},
			Signature: bytesutil.PadTo([]byte("hello"), 96),
		},
		Attestation_2: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: make([]byte, 32),
				Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
				Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
			},
			Signature: bytesutil.PadTo([]byte("hello"), 96),
		},
	}
	has, _, err := db.HasAttesterSlashing(ctx, as)
	require.NoError(t, err, "HasAttesterSlashing should not return error")
	require.Equal(t, false, has)

	p, err := db.AttesterSlashings(ctx, types.SlashingStatus(types.Active))
	require.NoError(t, err, "Failed to get attester slashing")
	require.NotNil(t, p, "Get should return empty attester slashing array for a non existent key")
	require.Equal(t, 0, len(p), "Get should return empty attester slashing array for a non existent key")
}

func TestStore_SaveAttesterSlashing(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	data := &ethpb.AttestationData{
		Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		BeaconBlockRoot: make([]byte, 32),
	}
	att := &ethpb.IndexedAttestation{Data: data, Signature: make([]byte, 96)}
	tests := []struct {
		ss types.SlashingStatus
		as *ethpb.AttesterSlashing
	}{
		{
			ss: types.Active,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello"), 96)}, Attestation_2: att},
		},
		{
			ss: types.Included,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello2"), 96)}, Attestation_2: att},
		},
		{
			ss: types.Reverted,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello3"), 96)}, Attestation_2: att},
		},
	}

	for _, tt := range tests {
		err := db.SaveAttesterSlashing(ctx, tt.ss, tt.as)
		require.NoError(t, err, "Save attester slashing failed")

		attesterSlashings, err := db.AttesterSlashings(ctx, tt.ss)
		require.NoError(t, err, "Failed to get attester slashings")
		require.NotNil(t, attesterSlashings)
		require.DeepEqual(t, tt.as, attesterSlashings[0], "Slashing: %v should be part of slashings response: %v", tt.as, attesterSlashings)
	}
}

func TestStore_SaveAttesterSlashings(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	ckpt := &ethpb.Checkpoint{Root: make([]byte, 32)}
	data := &ethpb.AttestationData{Source: ckpt, Target: ckpt, BeaconBlockRoot: make([]byte, 32)}
	att := &ethpb.IndexedAttestation{Data: data, Signature: make([]byte, 96)}
	as := []*ethpb.AttesterSlashing{
		{Attestation_1: &ethpb.IndexedAttestation{Signature: bytesutil.PadTo([]byte("1"), 96), Data: data}, Attestation_2: att},
		{Attestation_1: &ethpb.IndexedAttestation{Signature: bytesutil.PadTo([]byte("2"), 96), Data: data}, Attestation_2: att},
		{Attestation_1: &ethpb.IndexedAttestation{Signature: bytesutil.PadTo([]byte("3"), 96), Data: data}, Attestation_2: att},
	}
	err := db.SaveAttesterSlashings(ctx, types.Active, as)
	require.NoError(t, err, "Save attester slashing failed")
	attesterSlashings, err := db.AttesterSlashings(ctx, types.Active)
	require.NoError(t, err, "Failed to get attester slashings")
	sort.SliceStable(attesterSlashings, func(i, j int) bool {
		return attesterSlashings[i].Attestation_1.Signature[0] < attesterSlashings[j].Attestation_1.Signature[0]
	})
	require.NotNil(t, attesterSlashings)
	require.DeepEqual(t, as, attesterSlashings, "Slashing: %v should be part of slashings response: %v", as, attesterSlashings)
}

func TestStore_UpdateAttesterSlashingStatus(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	data := &ethpb.AttestationData{
		BeaconBlockRoot: make([]byte, 32),
		Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
	}

	tests := []struct {
		ss types.SlashingStatus
		as *ethpb.AttesterSlashing
	}{
		{
			ss: types.Active,
			as: &ethpb.AttesterSlashing{
				Attestation_1: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello"), 96)},
				Attestation_2: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello"), 96)},
			},
		},
		{
			ss: types.Active,
			as: &ethpb.AttesterSlashing{
				Attestation_1: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello2"), 96)},
				Attestation_2: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello2"), 96)},
			},
		},
		{
			ss: types.Active,
			as: &ethpb.AttesterSlashing{
				Attestation_1: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello3"), 96)},
				Attestation_2: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello2"), 96)},
			},
		},
	}

	for _, tt := range tests {
		err := db.SaveAttesterSlashing(ctx, tt.ss, tt.as)
		require.NoError(t, err, "Save attester slashing failed")
	}

	for _, tt := range tests {
		has, st, err := db.HasAttesterSlashing(ctx, tt.as)
		require.NoError(t, err, "Failed to get attester slashing")
		require.Equal(t, true, has, "Failed to find attester slashing: %v", tt.as)
		require.Equal(t, tt.ss, st, "Failed to find attester slashing with the correct status: %v", tt.as)

		err = db.SaveAttesterSlashing(ctx, types.SlashingStatus(types.Included), tt.as)
		require.NoError(t, err)
		has, st, err = db.HasAttesterSlashing(ctx, tt.as)
		require.NoError(t, err, "Failed to get attester slashing")
		require.Equal(t, true, has, "Failed to find attester slashing: %v", tt.as)
		require.Equal(t, (types.SlashingStatus)(types.Included), st, "Failed to find attester slashing with the correct status: %v", tt.as)
	}
}

func TestStore_LatestEpochDetected(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	e, err := db.GetLatestEpochDetected(ctx)
	require.NoError(t, err, "Get latest epoch detected failed")
	require.Equal(t, uint64(0), e, "Latest epoch detected should have been 0 before setting got: %d", e)
	epoch := uint64(1)
	err = db.SetLatestEpochDetected(ctx, epoch)
	require.NoError(t, err, "Set latest epoch detected failed")
	e, err = db.GetLatestEpochDetected(ctx)
	require.NoError(t, err, "Get latest epoch detected failed")
	require.Equal(t, epoch, e, "Latest epoch detected should have been: %d got: %d", epoch, e)
}
