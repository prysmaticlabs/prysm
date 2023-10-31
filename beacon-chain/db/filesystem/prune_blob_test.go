package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestBlobStorage_PruneBlob(t *testing.T) {
	currentSlot := primitives.Slot(225519)
	tempDir := t.TempDir()
	bs := &BlobStorage{
		baseDir:        tempDir,
		retentionEpoch: 4096,
	}

	blobPath := filepath.Join(tempDir, "invalid_blob.blob")
	err := os.WriteFile(blobPath, []byte("Invalid Blob Data"), 0644)
	require.NoError(t, err)

	// Simulate an error when extracting the slot from an invalid blob filename.
	err = bs.PruneBlob(currentSlot)
	require.ErrorContains(t, "failed to parse slot from filename", err)
	err = os.Remove(blobPath)
	require.NoError(t, err)

	// Create partial blob files.
	partialBlobPaths := []string{
		"12345_abcd_1_xyz.blob.partial",
		"14321_deadbeef_5_cafe.blob.partial",
		"29876_1234_3_abcd.blob.partial",
	}

	for _, path := range partialBlobPaths {
		partialBlobPath := filepath.Join(tempDir, path)
		err = os.WriteFile(partialBlobPath, []byte("Partial Blob Data"), 0644)
		require.NoError(t, err)
	}

	// Prune blobs successfully.
	err = bs.SaveBlobData(testSidecars)
	require.NoError(t, err)

	err = bs.PruneBlob(currentSlot)
	require.NoError(t, err)

	remainingFiles, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	// Expecting 2 blobs from testSidecars to remain.
	require.Equal(t, 2, len(remainingFiles))
}

func TestExtractSlotFromFileName(t *testing.T) {
	tempDir := t.TempDir()
	bs := &BlobStorage{baseDir: tempDir}
	err := bs.SaveBlobData(testSidecars)
	require.NoError(t, err)

	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)

	for _, f := range files {
		slot, err := extractSlotFromFileName(f.Name())
		require.NoError(t, err)
		sidecar := findTestSidecarsByFileName(t, f.Name())
		require.Equal(t, sidecar.Slot, slot)
	}
}
