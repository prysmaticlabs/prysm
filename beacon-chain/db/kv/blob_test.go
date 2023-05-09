package kv

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assertions"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func equalBlobSlices(expect []*ethpb.BlobSidecar, got []*ethpb.BlobSidecar) error {
	if len(expect) != len(got) {
		return fmt.Errorf("mismatched lengths, expect=%d, got=%d", len(expect), len(got))
	}
	for i := 0; i < len(expect); i++ {
		es := expect[i]
		gs := got[i]
		var e string
		assertions.DeepEqual(assertions.SprintfAssertionLoggerFn(&e), es, gs)
		if e != "" {
			return errors.New(e)
		}
	}
	return nil
}

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
		require.ErrorIs(t, ErrNotFound, err)
		require.Equal(t, 0, len(got))
	})
	t.Run("empty by slot", func(t *testing.T) {
		db := setupDB(t)
		got, err := db.BlobSidecarsBySlot(ctx, 1)
		require.ErrorIs(t, ErrNotFound, err)
		require.Equal(t, 0, len(got))
	})
	t.Run("save and retrieve by root (one)", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, 1)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, 1, len(scs))
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
	})
	t.Run("save and retrieve by root (max)", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs))
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
	})
	t.Run("save and retrieve valid subset by root", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs))

		// we'll request indices 0 and 3, so make a slice with those indices for comparison
		expect := make([]*ethpb.BlobSidecar, 2)
		expect[0] = scs[0]
		expect[1] = scs[3]

		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot), 0, 3)
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(expect, got))
		require.Equal(t, uint64(0), got[0].Index)
		require.Equal(t, uint64(3), got[1].Index)
	})
	t.Run("error for invalid index when retrieving by root", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs))

		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot), uint64(len(scs)))
		require.ErrorIs(t, err, ErrNotFound)
		require.Equal(t, 0, len(got))
	})
	t.Run("save and retrieve by slot (one)", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, 1)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, 1, len(scs))
		got, err := db.BlobSidecarsBySlot(ctx, scs[0].Slot)
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
	})
	t.Run("save and retrieve by slot (max)", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs))
		got, err := db.BlobSidecarsBySlot(ctx, scs[0].Slot)
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
	})
	t.Run("save and retrieve valid subset by slot", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs))

		// we'll request indices 0 and 3, so make a slice with those indices for comparison
		expect := make([]*ethpb.BlobSidecar, 2)
		expect[0] = scs[0]
		expect[1] = scs[3]

		got, err := db.BlobSidecarsBySlot(ctx, scs[0].Slot, 0, 3)
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(expect, got))

		require.Equal(t, uint64(0), got[0].Index)
		require.Equal(t, uint64(3), got[1].Index)
	})
	t.Run("error for invalid index when retrieving by slot", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs))

		got, err := db.BlobSidecarsBySlot(ctx, scs[0].Slot, uint64(len(scs)))
		require.ErrorIs(t, err, ErrNotFound)
		require.Equal(t, 0, len(got))
	})
	t.Run("delete works", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs))
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
		require.NoError(t, db.DeleteBlobSidecar(ctx, bytesutil.ToBytes32(scs[0].BlockRoot)))
		got, err = db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.ErrorIs(t, ErrNotFound, err)
		require.Equal(t, 0, len(got))
	})
	t.Run("saving a blob with older slot", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs))
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
		require.ErrorContains(t, "but already have older blob with slot", db.SaveBlobSidecar(ctx, scs))
	})
	t.Run("saving a new blob for rotation", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(params.BeaconConfig().MaxBlobsPerBlock), len(scs))
		oldBlockRoot := scs[0].BlockRoot
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(oldBlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))

		newScs := generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock)
		newRetentionSlot := primitives.Slot(params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest.Mul(uint64(params.BeaconConfig().SlotsPerEpoch)))
		newScs[0].Slot = scs[0].Slot + newRetentionSlot
		require.NoError(t, db.SaveBlobSidecar(ctx, newScs))

		_, err = db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(oldBlockRoot))
		require.ErrorIs(t, ErrNotFound, err)

		got, err = db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(newScs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(newScs, got))
	})
}

func generateBlobSidecars(t *testing.T, n uint64) []*ethpb.BlobSidecar {
	blobSidecars := make([]*ethpb.BlobSidecar, n)
	for i := uint64(0); i < n; i++ {
		blobSidecars[i] = generateBlobSidecar(t, i)
	}
	return blobSidecars
}

func generateBlobSidecar(t *testing.T, index uint64) *ethpb.BlobSidecar {
	blockRoot := make([]byte, 32)
	_, err := rand.Read(blockRoot)
	require.NoError(t, err)
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
	blob := make([]byte, 131072)
	_, err = rand.Read(blob)
	require.NoError(t, err)
	kzgCommitment := make([]byte, 48)
	_, err = rand.Read(kzgCommitment)
	require.NoError(t, err)
	kzgProof := make([]byte, 48)
	_, err = rand.Read(kzgProof)
	require.NoError(t, err)

	return &ethpb.BlobSidecar{
		BlockRoot:       blockRoot,
		Index:           index,
		Slot:            primitives.Slot(binary.LittleEndian.Uint64(slot)),
		BlockParentRoot: blockParentRoot,
		ProposerIndex:   primitives.ValidatorIndex(binary.LittleEndian.Uint64(proposerIndex)),
		Blob:            blob,
		KzgCommitment:   kzgCommitment,
		KzgProof:        kzgProof,
	}
}
