package filesystem

import (
	"bytes"
	"os"
	"path"
	"sync"
	"testing"

	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/verification"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	"github.com/spf13/afero"
)

func TestBlobStorage_SaveBlobData(t *testing.T) {
	_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, fieldparams.MaxBlobsPerBlock)
	testSidecars, err := verification.BlobSidecarSliceNoop(sidecars)
	require.NoError(t, err)

	t.Run("no error for duplicate", func(t *testing.T) {
		fs, bs, err := NewEphemeralBlobStorageWithFs(t)
		require.NoError(t, err)
		existingSidecar := testSidecars[0]

		blobPath := namerForSidecar(existingSidecar).path()
		// Serialize the existing BlobSidecar to binary data.
		existingSidecarData, err := ssz.MarshalSSZ(existingSidecar)
		require.NoError(t, err)

		require.NoError(t, bs.Save(existingSidecar))
		// No error when attempting to write twice.
		require.NoError(t, bs.Save(existingSidecar))

		content, err := afero.ReadFile(fs, blobPath)
		require.NoError(t, err)
		require.Equal(t, true, bytes.Equal(existingSidecarData, content))

		// Deserialize the BlobSidecar from the saved file data.
		savedSidecar := &ethpb.BlobSidecar{}
		err = savedSidecar.UnmarshalSSZ(content)
		require.NoError(t, err)

		// Compare the original Sidecar and the saved Sidecar.
		require.DeepSSZEqual(t, existingSidecar.BlobSidecar, savedSidecar)

	})
	t.Run("indices", func(t *testing.T) {
		bs := NewEphemeralBlobStorage(t)
		sc := testSidecars[2]
		require.NoError(t, bs.Save(sc))
		actualSc, err := bs.Get(sc.BlockRoot(), sc.Index)
		require.NoError(t, err)
		expectedIdx := [fieldparams.MaxBlobsPerBlock]bool{false, false, true}
		actualIdx, err := bs.Indices(actualSc.BlockRoot())
		require.NoError(t, err)
		require.Equal(t, expectedIdx, actualIdx)
	})

	t.Run("round trip write then read", func(t *testing.T) {
		bs := NewEphemeralBlobStorage(t)
		err := bs.Save(testSidecars[0])
		require.NoError(t, err)

		expected := testSidecars[0]
		actual, err := bs.Get(expected.BlockRoot(), expected.Index)
		require.NoError(t, err)
		require.DeepSSZEqual(t, expected, actual)
	})

	t.Run("round trip write, read and delete", func(t *testing.T) {
		bs := NewEphemeralBlobStorage(t)
		err := bs.Save(testSidecars[0])
		require.NoError(t, err)

		expected := testSidecars[0]
		actual, err := bs.Get(expected.BlockRoot(), expected.Index)
		require.NoError(t, err)
		require.DeepSSZEqual(t, expected, actual)

		require.NoError(t, bs.Remove(expected.BlockRoot()))
		_, err = bs.Get(expected.BlockRoot(), expected.Index)
		require.ErrorContains(t, "file does not exist", err)
	})

	t.Run("clear", func(t *testing.T) {
		blob := testSidecars[0]
		b := NewEphemeralBlobStorage(t)
		require.NoError(t, b.Save(blob))
		res, err := b.Get(blob.BlockRoot(), blob.Index)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NoError(t, b.Clear())
		// After clearing, the blob should not exist in the db.
		_, err = b.Get(blob.BlockRoot(), blob.Index)
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("race conditions", func(t *testing.T) {
		// There was a bug where saving the same blob in multiple go routines would cause a partial blob
		// to be empty. This test ensures that several routines can safely save the same blob at the
		// same time. This isn't ideal behavior from the caller, but should be handled safely anyway.
		// See https://github.com/prysmaticlabs/prysm/pull/13648
		b, err := NewBlobStorage(WithBasePath(t.TempDir()))
		require.NoError(t, err)
		blob := testSidecars[0]

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				require.NoError(t, b.Save(blob))
			}()
		}

		wg.Wait()
		res, err := b.Get(blob.BlockRoot(), blob.Index)
		require.NoError(t, err)
		require.DeepSSZEqual(t, blob, res)
	})
}

// pollUntil polls a condition function until it returns true or a timeout is reached.

func TestBlobIndicesBounds(t *testing.T) {
	fs, bs, err := NewEphemeralBlobStorageWithFs(t)
	require.NoError(t, err)
	root := [32]byte{}

	okIdx := uint64(fieldparams.MaxBlobsPerBlock - 1)
	writeFakeSSZ(t, fs, root, okIdx)
	indices, err := bs.Indices(root)
	require.NoError(t, err)
	var expected [fieldparams.MaxBlobsPerBlock]bool
	expected[okIdx] = true
	for i := range expected {
		require.Equal(t, expected[i], indices[i])
	}

	oobIdx := uint64(fieldparams.MaxBlobsPerBlock)
	writeFakeSSZ(t, fs, root, oobIdx)
	_, err = bs.Indices(root)
	require.ErrorIs(t, err, errIndexOutOfBounds)
}

func writeFakeSSZ(t *testing.T, fs afero.Fs, root [32]byte, idx uint64) {
	namer := blobNamer{root: root, index: idx}
	require.NoError(t, fs.MkdirAll(namer.dir(), 0700))
	fh, err := fs.Create(namer.path())
	require.NoError(t, err)
	_, err = fh.Write([]byte("derp"))
	require.NoError(t, err)
	require.NoError(t, fh.Close())
}

func TestBlobStoragePrune(t *testing.T) {
	currentSlot := primitives.Slot(200000)
	fs, bs, err := NewEphemeralBlobStorageWithFs(t)
	require.NoError(t, err)

	t.Run("PruneOne", func(t *testing.T) {
		_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 300, fieldparams.MaxBlobsPerBlock)
		testSidecars, err := verification.BlobSidecarSliceNoop(sidecars)
		require.NoError(t, err)

		for _, sidecar := range testSidecars {
			require.NoError(t, bs.Save(sidecar))
		}

		require.NoError(t, bs.pruner.prune(currentSlot-bs.pruner.windowSize))

		remainingFolders, err := afero.ReadDir(fs, ".")
		require.NoError(t, err)
		require.Equal(t, 0, len(remainingFolders))
	})
	t.Run("Prune dangling blob", func(t *testing.T) {
		_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 299, fieldparams.MaxBlobsPerBlock)
		testSidecars, err := verification.BlobSidecarSliceNoop(sidecars)
		require.NoError(t, err)

		for _, sidecar := range testSidecars[4:] {
			require.NoError(t, bs.Save(sidecar))
		}

		require.NoError(t, bs.pruner.prune(currentSlot-bs.pruner.windowSize))

		remainingFolders, err := afero.ReadDir(fs, ".")
		require.NoError(t, err)
		require.Equal(t, 0, len(remainingFolders))
	})
	t.Run("PruneMany", func(t *testing.T) {
		blockQty := 10
		slot := primitives.Slot(1)

		for j := 0; j <= blockQty; j++ {
			root := bytesutil.ToBytes32(bytesutil.ToBytes(uint64(slot), 32))
			_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, root, slot, fieldparams.MaxBlobsPerBlock)
			testSidecars, err := verification.BlobSidecarSliceNoop(sidecars)
			require.NoError(t, err)
			require.NoError(t, bs.Save(testSidecars[0]))

			slot += 10000
		}

		require.NoError(t, bs.pruner.prune(currentSlot-bs.pruner.windowSize))

		remainingFolders, err := afero.ReadDir(fs, ".")
		require.NoError(t, err)
		require.Equal(t, 4, len(remainingFolders))
	})
}

func BenchmarkPruning(b *testing.B) {
	var t *testing.T
	_, bs, err := NewEphemeralBlobStorageWithFs(t)
	require.NoError(t, err)

	blockQty := 10000
	currentSlot := primitives.Slot(150000)
	slot := primitives.Slot(0)

	for j := 0; j <= blockQty; j++ {
		root := bytesutil.ToBytes32(bytesutil.ToBytes(uint64(slot), 32))
		_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, root, slot, fieldparams.MaxBlobsPerBlock)
		testSidecars, err := verification.BlobSidecarSliceNoop(sidecars)
		require.NoError(t, err)
		require.NoError(t, bs.Save(testSidecars[0]))

		slot += 100
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := bs.pruner.prune(currentSlot)
		require.NoError(b, err)
	}
}

func TestNewBlobStorage(t *testing.T) {
	_, err := NewBlobStorage()
	require.ErrorIs(t, err, errNoBasePath)
	_, err = NewBlobStorage(WithBasePath(path.Join(t.TempDir(), "good")))
	require.NoError(t, err)
}
