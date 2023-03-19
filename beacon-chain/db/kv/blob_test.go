package kv

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestStore_BlobSidecars(t *testing.T) {
	ctx := context.Background()

	t.Run("empty", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, 0)
		require.ErrorContains(t, "nil or empty blob sidecars", db.SaveBlobSidecar(ctx, scs))
	})
	t.Run("empty by root", func(t *testing.T) {
		db := setupDB(t)
		got, err := db.BlobSidecarsByRoot(ctx, [32]byte{})
		require.NoError(t, err)
		require.DeepEqual(t, (*ethpb.BlobSidecars)(nil), got)
	})
	t.Run("empty by slot", func(t *testing.T) {
		db := setupDB(t)
		got, err := db.BlobSidecarsBySlot(ctx, 1)
		require.NoError(t, err)
		require.DeepEqual(t, (*ethpb.BlobSidecars)(nil), got)
	})
	t.Run("save and retrieve by root (one)", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, 1)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, 1, len(scs.Sidecars))
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs.Sidecars[0].BlockRoot))
		require.NoError(t, err)
		require.Equal(t, 1, len(got.Sidecars))
		require.DeepEqual(t, scs, got)
	})
	t.Run("save and retrieve by root (max)", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs.Sidecars))
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs.Sidecars[0].BlockRoot))
		require.NoError(t, err)
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(got.Sidecars))
		require.DeepEqual(t, scs, got)
	})
	t.Run("save and retrieve by slot (one)", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, 1)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, 1, len(scs.Sidecars))
		got, err := db.BlobSidecarsBySlot(ctx, scs.Sidecars[0].Slot)
		require.NoError(t, err)
		require.Equal(t, 1, len(got.Sidecars))
		require.DeepEqual(t, scs, got)
	})
	t.Run("save and retrieve by slot (max)", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs.Sidecars))
		got, err := db.BlobSidecarsBySlot(ctx, scs.Sidecars[0].Slot)
		require.NoError(t, err)
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(got.Sidecars))
		require.DeepEqual(t, scs, got)
	})
	t.Run("delete works", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs.Sidecars))
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs.Sidecars[0].BlockRoot))
		require.NoError(t, err)
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(got.Sidecars))
		require.DeepEqual(t, scs, got)
		require.NoError(t, db.DeleteBlobSidecar(ctx, bytesutil.ToBytes32(scs.Sidecars[0].BlockRoot)))
		got, err = db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs.Sidecars[0].BlockRoot))
		require.NoError(t, err)
		require.DeepEqual(t, (*ethpb.BlobSidecars)(nil), got)
	})
	t.Run("saving a blob with older slot", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs.Sidecars))
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs.Sidecars[0].BlockRoot))
		require.NoError(t, err)
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(got.Sidecars))
		require.DeepEqual(t, scs, got)
		require.ErrorContains(t, "but already have older blob with slot", db.SaveBlobSidecar(ctx, scs))
	})
	t.Run("saving a new blob for rotation", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs.Sidecars))
		oldBlockRoot := scs.Sidecars[0].BlockRoot
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs.Sidecars[0].BlockRoot))
		require.NoError(t, err)
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(got.Sidecars))
		require.DeepEqual(t, scs, got)

		newScs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		newRetentionSlot := primitives.Slot(params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest.Mul(uint64(params.BeaconConfig().SlotsPerEpoch)))
		newScs.Sidecars[0].Slot = scs.Sidecars[0].Slot + newRetentionSlot
		require.NoError(t, db.SaveBlobSidecar(ctx, newScs))

		got, err = db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(oldBlockRoot))
		require.NoError(t, err)
		require.DeepEqual(t, (*ethpb.BlobSidecars)(nil), got)

		got, err = db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(newScs.Sidecars[0].BlockRoot))
		require.NoError(t, err)
		require.DeepEqual(t, newScs, got)
	})
}

func generateBlobSidecars(t *testing.T, n uint64) *ethpb.BlobSidecars {
	blobSidecars := make([]*ethpb.BlobSidecar, n)
	for i := uint64(0); i < n; i++ {
		blobSidecars[i] = generateBlobSidecar(t)
	}
	return &ethpb.BlobSidecars{
		Sidecars: blobSidecars,
	}
}

func generateBlobSidecar(t *testing.T) *ethpb.BlobSidecar {
	blockRoot := make([]byte, 32)
	_, err := rand.Read(blockRoot)
	require.NoError(t, err)
	index := make([]byte, 8)
	_, err = rand.Read(index)
	require.NoError(t, err)
	slot := make([]byte, 8)
	_, err = rand.Read(slot)
	require.NoError(t, err)
	blockParentRoot := make([]byte, 32)
	_, err = rand.Read(blockParentRoot)
	require.NoError(t, err)
	proposerIndex := make([]byte, 8)
	_, err = rand.Read(proposerIndex)
	require.NoError(t, err)
	blobData := make([]byte, 131072)
	_, err = rand.Read(blobData)
	require.NoError(t, err)
	blob := &enginev1.Blob{
		Data: blobData,
	}
	kzgCommitment := make([]byte, 48)
	_, err = rand.Read(kzgCommitment)
	require.NoError(t, err)
	kzgProof := make([]byte, 48)
	_, err = rand.Read(kzgProof)
	require.NoError(t, err)

	return &ethpb.BlobSidecar{
		BlockRoot:       blockRoot,
		Index:           binary.LittleEndian.Uint64(index),
		Slot:            primitives.Slot(binary.LittleEndian.Uint64(slot)),
		BlockParentRoot: blockParentRoot,
		ProposerIndex:   primitives.ValidatorIndex(binary.LittleEndian.Uint64(proposerIndex)),
		Blob:            blob,
		KzgCommitment:   kzgCommitment,
		KzgProof:        kzgProof,
	}
}
