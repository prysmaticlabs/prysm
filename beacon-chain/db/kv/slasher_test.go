package kv

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"

	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_AttestationRecordForValidator_SaveRetrieve(t *testing.T) {
	ctx := context.Background()
	beaconDB := setupDB(t)
	valIdx := types.ValidatorIndex(1)
	target := uint64(5)
	source := uint64(4)
	attRecord, err := beaconDB.AttestationRecordForValidator(ctx, valIdx, types.Epoch(target))
	require.NoError(t, err)
	require.Equal(t, true, attRecord == nil)

	sr := [32]byte{1}
	err = beaconDB.SaveAttestationRecordsForValidators(ctx, []types.ValidatorIndex{valIdx}, []*slashertypes.CompactAttestation{
		{
			Target:      target,
			Source:      source,
			SigningRoot: sr,
		},
	})
	require.NoError(t, err)
	attRecord, err = beaconDB.AttestationRecordForValidator(ctx, valIdx, types.Epoch(target))
	require.NoError(t, err)
	assert.DeepEqual(t, target, attRecord.Target)
	assert.DeepEqual(t, source, attRecord.Source)
	assert.DeepEqual(t, sr, attRecord.SigningRoot)
}

func TestStore_LatestEpochAttestedForValidator(t *testing.T) {
	ctx := context.Background()
	beaconDB := setupDB(t)
	indices := []types.ValidatorIndex{1, 2, 3}
	epoch := types.Epoch(5)

	for _, valIdx := range indices {
		_, exists, err := beaconDB.LatestEpochAttestedForValidator(ctx, valIdx)
		require.NoError(t, err)
		require.Equal(t, false, exists)
	}

	err := beaconDB.SaveLatestEpochAttestedForValidators(ctx, indices, epoch)
	require.NoError(t, err)

	for _, valIdx := range indices {
		retrievedEpoch, exists, err := beaconDB.LatestEpochAttestedForValidator(ctx, valIdx)
		require.NoError(t, err)
		require.Equal(t, true, exists)
		require.Equal(t, epoch, retrievedEpoch)
	}
}

func TestStore_SlasherChunk_SaveRetrieve(t *testing.T) {
	ctx := context.Background()
	beaconDB := setupDB(t)
	elemsPerChunk := 16
	totalChunks := 64
	chunkKeys := make([]uint64, totalChunks)
	chunks := make([][]uint16, totalChunks)
	for i := 0; i < totalChunks; i++ {
		chunk := make([]uint16, elemsPerChunk)
		for j := 0; j < len(chunk); j++ {
			chunk[j] = uint16(0)
		}
		chunks[i] = chunk
		chunkKeys[i] = uint64(i)
	}

	// We save chunks for min spans.
	err := beaconDB.SaveSlasherChunks(ctx, slashertypes.MinSpan, chunkKeys, chunks)
	require.NoError(t, err)

	// We expect no chunks to be stored for max spans.
	for _, key := range chunkKeys {
		_, exists, err := beaconDB.LoadSlasherChunk(ctx, slashertypes.MaxSpan, key)
		require.NoError(t, err)
		require.Equal(t, false, exists)
	}

	// We check we saved the right chunks.
	for i, key := range chunkKeys {
		chunk, exists, err := beaconDB.LoadSlasherChunk(ctx, slashertypes.MinSpan, key)
		require.NoError(t, err)
		require.Equal(t, true, exists)
		require.DeepEqual(t, chunks[i], chunk)
	}

	// We save chunks for max spans.
	err = beaconDB.SaveSlasherChunks(ctx, slashertypes.MaxSpan, chunkKeys, chunks)
	require.NoError(t, err)

	// We check we saved the right chunks.
	for i, key := range chunkKeys {
		chunk, exists, err := beaconDB.LoadSlasherChunk(ctx, slashertypes.MaxSpan, key)
		require.NoError(t, err)
		require.Equal(t, true, exists)
		require.DeepEqual(t, chunks[i], chunk)
	}
}
