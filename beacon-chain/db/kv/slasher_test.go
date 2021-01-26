package kv

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_AttestationRecordForValidator_SaveRetrieve(t *testing.T) {
	beaconDB := setupDB(t)
	ctx := context.Background()
	valIdx := types.ValidatorIndex(1)
	target := uint64(5)
	source := uint64(4)
	err := beaconDB.SaveAttestationRecordForValidator(ctx, valIdx, &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Target: &ethpb.Checkpoint{
				Epoch: target,
			},
			Source: &ethpb.Checkpoint{
				Epoch: source,
			},
		},
	})
	require.NoError(t, err)
	attRecord, err := beaconDB.AttestationRecordForValidator(ctx, valIdx, types.Epoch(target))
	require.NoError(t, err)
	assert.DeepEqual(t, target, attRecord.Target)
	assert.DeepEqual(t, source, attRecord.Source)
}

func TestStore_LatestEpochAttestedForValidator(t *testing.T) {
}

func TestStore_SlasherChunk_SaveRetrieve(t *testing.T) {
}
