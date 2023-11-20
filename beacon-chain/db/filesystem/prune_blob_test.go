package filesystem

import (
	"os"
	"path"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestBlobStorage_Prune(t *testing.T) {
	bs := &BlobStorage{
		baseDir:        t.TempDir(),
		retentionEpoch: 4096,
	}
	blockQty := 10
	currentSlot := primitives.Slot(150000)
	blobSidecars := make([]*eth.BlobSidecar, fieldparams.MaxBlobsPerBlock*blockQty)

	slot := primitives.Slot(0)
	index := uint64(0)
	for j := 0; j < blockQty; j++ {
		for i := 0; i < fieldparams.MaxBlobsPerBlock; i++ {
			blobSidecars[index] = generateBlobSidecar(t, slot, uint64(i), bytesutil.PadTo(bytesutil.ToBytes(uint64(slot), 32), 32))
			index++
		}
		slot += 2500
	}
	// Prune blobs successfully.
	err := bs.SaveBlobData(blobSidecars)
	require.NoError(t, err)

	err = bs.Prune(currentSlot)
	require.NoError(t, err)

	remainingFolders, err := os.ReadDir(bs.baseDir)
	require.NoError(t, err)
	require.Equal(t, 2, len(remainingFolders))
	// Ensure that the blob files are still present.
	for _, folder := range remainingFolders {
		if folder.IsDir() {
			files, err := os.ReadDir(path.Join(bs.baseDir, folder.Name()))
			require.NoError(t, err)
			// Should have 6 blob files.
			require.Equal(t, 6, len(files))
		}
	}
}

func BenchmarkPruning(b *testing.B) {
	bs := &BlobStorage{
		baseDir:        b.TempDir(),
		retentionEpoch: 4096,
	}
	blockQty := 10000
	currentSlot := primitives.Slot(150000)
	slot := primitives.Slot(0)

	blobSidecars := make([]*eth.BlobSidecar, fieldparams.MaxBlobsPerBlock)
	for j := 0; j <= blockQty; j++ {
		for i := 0; i < fieldparams.MaxBlobsPerBlock; i++ {
			blobSidecars[i] = generateBlobSidecar(b, slot, uint64(i), bytesutil.PadTo(bytesutil.ToBytes(uint64(slot), 32), 32))
		}
		slot += 100
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := bs.Prune(currentSlot)
		require.NoError(b, err)
	}
}
