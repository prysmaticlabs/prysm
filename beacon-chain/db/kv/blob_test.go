package kv

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/cmd/beacon-chain/flags"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assertions"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/urfave/cli/v2"
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
	t.Run("save and retrieve by root (max), per batch", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, fieldparams.MaxBlobsPerBlock, len(scs))
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
	})
	t.Run("save and retrieve by root, max and individually", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		for _, sc := range scs {
			require.NoError(t, db.SaveBlobSidecar(ctx, []*ethpb.BlobSidecar{sc}))
		}
		require.Equal(t, fieldparams.MaxBlobsPerBlock, len(scs))
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
	})
	t.Run("save and retrieve valid subset by root", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(fieldparams.MaxBlobsPerBlock), len(scs))

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
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(fieldparams.MaxBlobsPerBlock), len(scs))

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
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(fieldparams.MaxBlobsPerBlock), len(scs))
		got, err := db.BlobSidecarsBySlot(ctx, scs[0].Slot)
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
	})
	t.Run("save and retrieve by slot, max and individually", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		for _, sc := range scs {
			require.NoError(t, db.SaveBlobSidecar(ctx, []*ethpb.BlobSidecar{sc}))
		}
		require.Equal(t, fieldparams.MaxBlobsPerBlock, len(scs))
		got, err := db.BlobSidecarsBySlot(ctx, scs[0].Slot)
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
	})
	t.Run("save and retrieve valid subset by slot", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(fieldparams.MaxBlobsPerBlock), len(scs))

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
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(fieldparams.MaxBlobsPerBlock), len(scs))

		got, err := db.BlobSidecarsBySlot(ctx, scs[0].Slot, uint64(len(scs)))
		require.ErrorIs(t, err, ErrNotFound)
		require.Equal(t, 0, len(got))
	})
	t.Run("delete works", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, int(fieldparams.MaxBlobsPerBlock), len(scs))
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
		require.NoError(t, db.DeleteBlobSidecar(ctx, bytesutil.ToBytes32(scs[0].BlockRoot)))
		got, err = db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.ErrorIs(t, ErrNotFound, err)
		require.Equal(t, 0, len(got))
	})
	t.Run("saving blob different times", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)

		for i := 0; i < fieldparams.MaxBlobsPerBlock; i++ {
			scs[i].Slot = primitives.Slot(i)
			scs[i].BlockRoot = bytesutil.PadTo([]byte{byte(i)}, 32)
			require.NoError(t, db.SaveBlobSidecar(ctx, []*ethpb.BlobSidecar{scs[i]}))
			br := bytesutil.ToBytes32(scs[i].BlockRoot)
			saved, err := db.BlobSidecarsByRoot(ctx, br)
			require.NoError(t, err)
			require.NoError(t, equalBlobSlices([]*ethpb.BlobSidecar{scs[i]}, saved))
		}
	})
	t.Run("saving a new blob for rotation (batch)", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, fieldparams.MaxBlobsPerBlock, len(scs))
		oldBlockRoot := scs[0].BlockRoot
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(oldBlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))

		newScs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
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
	t.Run("save multiple blobs after new rotation (individually)", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		for _, sc := range scs {
			require.NoError(t, db.SaveBlobSidecar(ctx, []*ethpb.BlobSidecar{sc}))
		}
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))

		scs = generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		newRetentionSlot := primitives.Slot(params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest.Mul(uint64(params.BeaconConfig().SlotsPerEpoch)))
		for _, sc := range scs {
			sc.Slot = sc.Slot + newRetentionSlot
		}
		for _, sc := range scs {
			require.NoError(t, db.SaveBlobSidecar(ctx, []*ethpb.BlobSidecar{sc}))
		}

		_, err = db.BlobSidecarsBySlot(ctx, 100)
		require.ErrorIs(t, ErrNotFound, err)

		got, err = db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
	})
	t.Run("save multiple blobs after new rotation (batch then individually)", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))
		require.Equal(t, fieldparams.MaxBlobsPerBlock, len(scs))
		oldBlockRoot := scs[0].BlockRoot
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(oldBlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))

		scs = generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		newRetentionSlot := primitives.Slot(params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest.Mul(uint64(params.BeaconConfig().SlotsPerEpoch)))
		for _, sc := range scs {
			sc.Slot = sc.Slot + newRetentionSlot
		}
		for _, sc := range scs {
			require.NoError(t, db.SaveBlobSidecar(ctx, []*ethpb.BlobSidecar{sc}))
		}

		_, err = db.BlobSidecarsBySlot(ctx, 100)
		require.ErrorIs(t, ErrNotFound, err)

		got, err = db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
	})
	t.Run("save multiple blobs after new rotation (individually then batch)", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		for _, sc := range scs {
			require.NoError(t, db.SaveBlobSidecar(ctx, []*ethpb.BlobSidecar{sc}))
		}
		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))

		scs = generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock)
		newRetentionSlot := primitives.Slot(params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest.Mul(uint64(params.BeaconConfig().SlotsPerEpoch)))
		for _, sc := range scs {
			sc.Slot = sc.Slot + newRetentionSlot
		}
		require.NoError(t, db.SaveBlobSidecar(ctx, scs))

		_, err = db.BlobSidecarsBySlot(ctx, 100)
		require.ErrorIs(t, ErrNotFound, err)

		got, err = db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))
	})
	t.Run("save equivocating blobs", func(t *testing.T) {
		db := setupDB(t)
		scs := generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock/2)
		eScs := generateEquivocatingBlobSidecars(t, fieldparams.MaxBlobsPerBlock/2)

		for i, sc := range scs {
			require.NoError(t, db.SaveBlobSidecar(ctx, []*ethpb.BlobSidecar{sc}))
			require.NoError(t, db.SaveBlobSidecar(ctx, []*ethpb.BlobSidecar{eScs[i]}))
		}

		got, err := db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(scs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(scs, got))

		got, err = db.BlobSidecarsByRoot(ctx, bytesutil.ToBytes32(eScs[0].BlockRoot))
		require.NoError(t, err)
		require.NoError(t, equalBlobSlices(eScs, got))
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

func generateEquivocatingBlobSidecars(t *testing.T, n uint64) []*ethpb.BlobSidecar {
	blobSidecars := make([]*ethpb.BlobSidecar, n)
	for i := uint64(0); i < n; i++ {
		blobSidecars[i] = generateEquivocatingBlobSidecar(t, i)
	}
	return blobSidecars
}

func generateEquivocatingBlobSidecar(t *testing.T, index uint64) *ethpb.BlobSidecar {
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
		BlockRoot:       bytesutil.PadTo([]byte{'c'}, 32),
		Index:           index,
		Slot:            100,
		BlockParentRoot: bytesutil.PadTo([]byte{'b'}, 32),
		ProposerIndex:   102,
		Blob:            blob,
		KzgCommitment:   kzgCommitment,
		KzgProof:        kzgProof,
	}
}

func Test_validUniqueSidecars_validation(t *testing.T) {
	tests := []struct {
		name string
		scs  []*ethpb.BlobSidecar
		err  error
	}{
		{name: "empty", scs: []*ethpb.BlobSidecar{}, err: errEmptySidecar},
		{name: "too many sidecars", scs: generateBlobSidecars(t, fieldparams.MaxBlobsPerBlock+1), err: errBlobSidecarLimit},
		{name: "invalid slot", scs: []*ethpb.BlobSidecar{{Slot: 1}, {Slot: 2}}, err: errBlobSlotMismatch},
		{name: "invalid proposer index", scs: []*ethpb.BlobSidecar{{ProposerIndex: 1}, {ProposerIndex: 2}}, err: errBlobProposerMismatch},
		{name: "invalid root", scs: []*ethpb.BlobSidecar{{BlockRoot: []byte{1}}, {BlockRoot: []byte{2}}}, err: errBlobRootMismatch},
		{name: "invalid parent root", scs: []*ethpb.BlobSidecar{{BlockParentRoot: []byte{1}}, {BlockParentRoot: []byte{2}}}, err: errBlobParentMismatch},
		{name: "happy path", scs: []*ethpb.BlobSidecar{{Index: 0}, {Index: 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validUniqueSidecars(tt.scs)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_validUniqueSidecars_dedup(t *testing.T) {
	cases := []struct {
		name     string
		scs      []*ethpb.BlobSidecar
		expected []*ethpb.BlobSidecar
		err      error
	}{
		{
			name:     "duplicate sidecar",
			scs:      []*ethpb.BlobSidecar{{Index: 1}, {Index: 1}},
			expected: []*ethpb.BlobSidecar{{Index: 1}},
		},
		{
			name:     "single sidecar",
			scs:      []*ethpb.BlobSidecar{{Index: 1}},
			expected: []*ethpb.BlobSidecar{{Index: 1}},
		},
		{
			name:     "multiple duplicates",
			scs:      []*ethpb.BlobSidecar{{Index: 1}, {Index: 2}, {Index: 2}, {Index: 3}, {Index: 3}},
			expected: []*ethpb.BlobSidecar{{Index: 1}, {Index: 2}, {Index: 3}},
		},
		{
			name:     "ok number after de-dupe, > 6 before",
			scs:      []*ethpb.BlobSidecar{{Index: 1}, {Index: 2}, {Index: 2}, {Index: 2}, {Index: 2}, {Index: 3}, {Index: 3}},
			expected: []*ethpb.BlobSidecar{{Index: 1}, {Index: 2}, {Index: 3}},
		},
		{
			name:     "max unique, no dupes",
			scs:      []*ethpb.BlobSidecar{{Index: 1}, {Index: 2}, {Index: 3}, {Index: 4}, {Index: 5}, {Index: 6}},
			expected: []*ethpb.BlobSidecar{{Index: 1}, {Index: 2}, {Index: 3}, {Index: 4}, {Index: 5}, {Index: 6}},
		},
		{
			name: "too many unique",
			scs:  []*ethpb.BlobSidecar{{Index: 1}, {Index: 2}, {Index: 3}, {Index: 4}, {Index: 5}, {Index: 6}, {Index: 7}},
			err:  errBlobSidecarLimit,
		},
		{
			name: "too many unique with dupes",
			scs:  []*ethpb.BlobSidecar{{Index: 1}, {Index: 1}, {Index: 1}, {Index: 2}, {Index: 3}, {Index: 4}, {Index: 5}, {Index: 6}, {Index: 7}},
			err:  errBlobSidecarLimit,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			u, err := validUniqueSidecars(c.scs)
			if c.err != nil {
				require.ErrorIs(t, err, c.err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, len(c.expected), len(u))
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
	sortSidecars(scs)
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

	params.SetupTestConfigCleanup(t)
	set := flag.NewFlagSet("test", 0)
	set.Uint64(flags.BlobRetentionEpoch.Name, 0, "")
	require.NoError(t, set.Set(flags.BlobRetentionEpoch.Name, strconv.FormatUint(42069, 10)))
	cliCtx := cli.NewContext(&cli.App{}, set, nil)
	require.NoError(t, ConfigureBlobRetentionEpoch(cliCtx))
	require.ErrorContains(t, "epochs for blobs request value in DB 4096 does not match config value 42069", checkEpochsForBlobSidecarsRequestBucket(dbStore.db))
}

func TestBlobRotatingKey(t *testing.T) {
	k := blobSidecarKey(&ethpb.BlobSidecar{
		Slot:      1,
		BlockRoot: []byte{2},
	})

	require.Equal(t, types.Slot(1), k.Slot())
	require.DeepEqual(t, []byte{2}, k.BlockRoot())
	require.DeepEqual(t, slotKey(types.Slot(1)), k.BufferPrefix())
}
