package kv

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assertions"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	bolt "go.etcd.io/bbolt"
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
		for _, sc := range newScs {
			sc.Slot = sc.Slot + newRetentionSlot
		}
		require.NoError(t, db.SaveBlobSidecar(ctx, newScs))

		_, err = db.BlobSidecarsBySlot(ctx, 100)
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
	blob := make([]byte, 131072)
	_, err := rand.Read(blob)
	require.NoError(t, err)
	kzgCommitment := make([]byte, 48)
	_, err = rand.Read(kzgCommitment)
	require.NoError(t, err)
	kzgProof := make([]byte, 48)
	_, err = rand.Read(kzgProof)
	require.NoError(t, err)

	return &ethpb.BlobSidecar{
		BlockRoot:       bytesutil.PadTo([]byte{'a'}, 32),
		Index:           index,
		Slot:            100,
		BlockParentRoot: bytesutil.PadTo([]byte{'b'}, 32),
		ProposerIndex:   101,
		Blob:            blob,
		KzgCommitment:   kzgCommitment,
		KzgProof:        kzgProof,
	}
}

func TestStore_verifySideCars(t *testing.T) {
	s := setupDB(t)
	tests := []struct {
		name  string
		scs   []*ethpb.BlobSidecar
		error string
	}{
		{name: "empty", scs: []*ethpb.BlobSidecar{}, error: "nil or empty blob sidecars"},
		{name: "too many sidecars", scs: generateBlobSidecars(t, params.BeaconConfig().MaxBlobsPerBlock+1), error: "too many sidecars: 5 > 4"},
		{name: "invalid slot", scs: []*ethpb.BlobSidecar{{Slot: 1}, {Slot: 2}}, error: "sidecar slot mismatch: 2 != 1"},
		{name: "invalid proposer index", scs: []*ethpb.BlobSidecar{{ProposerIndex: 1}, {ProposerIndex: 2}}, error: "sidecar proposer index mismatch: 2 != 1"},
		{name: "invalid root", scs: []*ethpb.BlobSidecar{{BlockRoot: []byte{1}}, {BlockRoot: []byte{2}}}, error: "sidecar root mismatch: 02 != 01"},
		{name: "invalid parent root", scs: []*ethpb.BlobSidecar{{BlockParentRoot: []byte{1}}, {BlockParentRoot: []byte{2}}}, error: "sidecar parent root mismatch: 02 != 01"},
		{name: "invalid side index", scs: []*ethpb.BlobSidecar{{Index: 0}, {Index: 0}}, error: "sidecar index mismatch: 0 != 1"},
		{name: "happy path", scs: []*ethpb.BlobSidecar{{Index: 0}, {Index: 1}}, error: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.verifySideCars(tt.scs)
			if tt.error != "" {
				require.Equal(t, tt.error, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStore_sortSidecars(t *testing.T) {
	scs := []*ethpb.BlobSidecar{
		{Index: 6},
		{Index: 4},
		{Index: 2},
		{Index: 1},
		{Index: 3},
		{Index: 5},
		{},
	}
	sortSideCars(scs)
	for i := 0; i < len(scs)-1; i++ {
		require.Equal(t, uint64(i), scs[i].Index)
	}
}

func BenchmarkStore_BlobSidecarsByRoot(b *testing.B) {
	s := setupDB(b)
	ctx := context.Background()
	require.NoError(b, s.SaveBlobSidecar(ctx, []*ethpb.BlobSidecar{
		{BlockRoot: bytesutil.PadTo([]byte{'a'}, 32), Slot: 0},
	}))

	err := s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blobsBucket)
		for i := 1; i < 131071; i++ {
			r := make([]byte, 32)
			_, err := rand.Read(r)
			require.NoError(b, err)
			scs := []*ethpb.BlobSidecar{
				{BlockRoot: r, Slot: primitives.Slot(i)},
			}
			k := blobSidecarKey(scs[0])
			encodedBlobSidecar, err := encode(ctx, &ethpb.BlobSidecars{Sidecars: scs})
			require.NoError(b, err)
			require.NoError(b, bkt.Put(k, encodedBlobSidecar))
		}
		return nil
	})
	require.NoError(b, err)

	require.NoError(b, s.SaveBlobSidecar(ctx, []*ethpb.BlobSidecar{
		{BlockRoot: bytesutil.PadTo([]byte{'b'}, 32), Slot: 131071},
	}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.BlobSidecarsByRoot(ctx, [32]byte{'b'})
		require.NoError(b, err)
	}
}

func Test_checkEpochsForBlobSidecarsRequestBucket(t *testing.T) {
	dbStore := setupDB(t)

	require.NoError(t, checkEpochsForBlobSidecarsRequestBucket(dbStore.db)) // First write
	require.NoError(t, checkEpochsForBlobSidecarsRequestBucket(dbStore.db)) // First check

	nConfig := params.BeaconNetworkConfig()
	nConfig.MinEpochsForBlobsSidecarsRequest = 42069
	params.OverrideBeaconNetworkConfig(nConfig)
	require.ErrorContains(t, "epochs for blobs request value in DB 4096 does not match config value 42069", checkEpochsForBlobSidecarsRequestBucket(dbStore.db))
}
