package filesystem

import (
	"bytes"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/verification"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
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
	t.Run("check pruning", func(t *testing.T) {
		fs, bs, err := NewEphemeralBlobStorageWithFs(t)
		require.NoError(t, err)

		// Slot in first half of epoch therefore should not prune
		require.Equal(t, false, bs.shouldPrune(testSidecars[0].Slot()))
		err = bs.Save(testSidecars[0])
		require.NoError(t, err)
		actual, err := bs.Get(testSidecars[0].BlockRoot(), testSidecars[0].Index)
		require.NoError(t, err)
		require.DeepSSZEqual(t, testSidecars[0], actual)
		err = pollUntil(t, fs, 1)
		require.NoError(t, err)

		_, sidecars = util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 33, fieldparams.MaxBlobsPerBlock)
		testSidecars1, err := verification.BlobSidecarSliceNoop(sidecars)
		require.NoError(t, err)
		// Slot in first half of epoch therefore should not prune
		require.Equal(t, false, bs.shouldPrune(testSidecars1[0].Slot()))
		err = bs.Save(testSidecars1[0])
		require.NoError(t, err)
		// Check previous saved sidecar was not pruned
		actual, err = bs.Get(testSidecars[0].BlockRoot(), testSidecars[0].Index)
		require.NoError(t, err)
		require.DeepSSZEqual(t, testSidecars[0], actual)
		// Check latest sidecar exists
		actual, err = bs.Get(testSidecars1[0].BlockRoot(), testSidecars1[0].Index)
		require.NoError(t, err)
		require.DeepSSZEqual(t, testSidecars1[0], actual)
		err = pollUntil(t, fs, 2) // Check correct number of files
		require.NoError(t, err)

		_, sidecars = util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 131187, fieldparams.MaxBlobsPerBlock)
		testSidecars, err = verification.BlobSidecarSliceNoop(sidecars)
		// Slot in second half of epoch therefore should prune
		require.Equal(t, true, bs.shouldPrune(testSidecars[0].Slot()))
		require.NoError(t, err)
		err = bs.Save(testSidecars[0])
		require.NoError(t, err)
		err = pollUntil(t, fs, 1)
		require.NoError(t, err)
	})
}

// pollUntil polls a condition function until it returns true or a timeout is reached.
func pollUntil(t *testing.T, fs afero.Fs, expected int) error {
	var remainingFolders []os.FileInfo
	var err error
	// Define the condition function for polling
	conditionFunc := func() bool {
		remainingFolders, err = afero.ReadDir(fs, ".")
		require.NoError(t, err)
		return len(remainingFolders) == expected
	}

	startTime := time.Now()
	for {
		if conditionFunc() {
			break // Condition met, exit the loop
		}
		if time.Since(startTime) > 30*time.Second {
			return errors.New("timeout")
		}
		time.Sleep(1 * time.Second) // Adjust the sleep interval as needed
	}
	require.Equal(t, expected, len(remainingFolders))
	return nil
}

func TestShouldPrune(t *testing.T) {
	bs := NewEphemeralBlobStorage(t)
	bs.lastPrunedEpoch = 5

	// Slot is before the midpoint of the epoch
	slot1 := primitives.Slot(100)
	p1 := bs.shouldPrune(slot1)
	require.Equal(t, false, p1)

	// Slot is after the midpoint of the epoch, but same epoch as last pruning
	slot2 := primitives.Slot(178)
	p2 := bs.shouldPrune(slot2)
	require.Equal(t, false, p2)

	// Slot is after the midpoint of the epoch
	slot3 := primitives.Slot(8018)
	p3 := bs.shouldPrune(slot3)
	require.Equal(t, true, p3)
}

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

		require.NoError(t, bs.Prune(currentSlot))

		remainingFolders, err := afero.ReadDir(fs, ".")
		require.NoError(t, err)
		require.Equal(t, 0, len(remainingFolders))
	})
	t.Run("PruneMany", func(t *testing.T) {
		blockQty := 10
		slot := primitives.Slot(0)

		for j := 0; j <= blockQty; j++ {
			root := bytesutil.ToBytes32(bytesutil.ToBytes(uint64(slot), 32))
			_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, root, slot, fieldparams.MaxBlobsPerBlock)
			testSidecars, err := verification.BlobSidecarSliceNoop(sidecars)
			require.NoError(t, err)
			require.NoError(t, bs.Save(testSidecars[0]))

			slot += 10000
		}

		require.NoError(t, bs.Prune(currentSlot))

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
		err := bs.Prune(currentSlot)
		require.NoError(b, err)
	}
}

func TestBlobStorageDelete(t *testing.T) {
	fs, bs, err := NewEphemeralBlobStorageWithFs(t)
	require.NoError(t, err)
	rawRoot := "0xcf9bb70c98f58092c9d6459227c9765f984d240be9690e85179bc5a6f60366ad"
	blockRoot, err := hexutil.Decode(rawRoot)
	require.NoError(t, err)

	_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, fieldparams.MaxBlobsPerBlock)
	testSidecars, err := verification.BlobSidecarSliceNoop(sidecars)
	require.NoError(t, err)
	for _, sidecar := range testSidecars {
		require.NoError(t, bs.Save(sidecar))
	}

	exists, err := afero.DirExists(fs, hexutil.Encode(blockRoot))
	require.NoError(t, err)
	require.Equal(t, true, exists)

	// Delete the directory corresponding to the block root
	require.NoError(t, bs.Delete(bytesutil.ToBytes32(blockRoot)))

	// Ensure that the directory no longer exists after deletion
	exists, err = afero.DirExists(fs, hexutil.Encode(blockRoot))
	require.NoError(t, err)
	require.Equal(t, false, exists)

	// Deleting a non-existent root does not return an error.
	require.NoError(t, bs.Delete(bytesutil.ToBytes32([]byte{0x1})))
}

func TestNewBlobStorage(t *testing.T) {
	_, err := NewBlobStorage(path.Join(t.TempDir(), "good"))
	require.NoError(t, err)
}
