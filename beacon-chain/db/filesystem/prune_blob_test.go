package filesystem

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/kv"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestBlobStorage_PruneBlobWithDB(t *testing.T) {
	currentSlot := primitives.Slot(225519)
	testSidecars := generateBlobSidecars(t, []primitives.Slot{225519, 100}, fieldparams.MaxBlobsPerBlock)
	tempDir := t.TempDir()
	bs := &BlobStorage{
		baseDir:        tempDir,
		retentionEpoch: 4096,
		lastPrunedSlot: 0,
	}
	ctx := context.Background()
	db := setupDB(t)

	for _, tt := range blockTests {
		b1, err := tt.newBlock(primitives.Slot(1), nil)
		require.NoError(t, err)
		b2, err := tt.newBlock(primitives.Slot(2), nil)
		require.NoError(t, err)

		require.NoError(t, db.SaveBlock(ctx, b1))
		require.NoError(t, db.SaveBlock(ctx, b2))
	}

	blobPath := filepath.Join(tempDir, "invalid_blob.blob")
	err := os.WriteFile(blobPath, []byte("Invalid Blob Data"), 0644)
	require.NoError(t, err)

	// Simulate an error when extracting the slot from an invalid blob filename.
	err = bs.PruneBlobWithDB(ctx, currentSlot, db)
	require.NoError(t, err)
	err = os.Remove(blobPath)
	require.NoError(t, err)

	// Prune blobs successfully.
	err = bs.SaveBlobData(testSidecars)
	require.NoError(t, err)

	// Create partial blob files.
	partialBlobPaths := []string{
		"12.blob.partial",
		"13.blob.partial",
		"14.blob.partial",
	}

	for _, p := range partialBlobPaths {
		root := strings.TrimPrefix(hexutil.Encode(testSidecars[0].BlockRoot), "0x")
		partialBlobPath := filepath.Join(tempDir, "0x"+root, p)
		err = os.WriteFile(partialBlobPath, []byte("Partial Blob Data"), 0644)
		require.NoError(t, err)
	}

	err = bs.PruneBlobWithDB(ctx, currentSlot, db)
	require.NoError(t, err)

	remainingFolders, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	// Expecting 6 blobs from testSidecars to remain.
	require.Equal(t, 2, len(remainingFolders))

	// Ensure that the slot files are still present.
	for _, folder := range remainingFolders {
		if folder.IsDir() {
			files, err := os.ReadDir(path.Join(tempDir, folder.Name()))
			require.NoError(t, err)
			// Should have 6 blob files and 1 slot file.
			require.Equal(t, 10, len(files))
		}
	}
}

func TestBlobStorage_PruneBlobViaSlotFile(t *testing.T) {
	currentSlot := primitives.Slot(225519)
	testSidecars := generateBlobSidecars(t, []primitives.Slot{225519, 100}, fieldparams.MaxBlobsPerBlock)
	tempDir := t.TempDir()
	bs := &BlobStorage{
		baseDir:        tempDir,
		retentionEpoch: 4096,
	}

	// Prune blobs successfully.
	err := bs.SaveBlobData(testSidecars)
	require.NoError(t, err)

	// Create partial blob files.
	partialBlobPaths := []string{
		"12.blob.partial",
		"13.blob.partial",
		"14.blob.partial",
	}

	for _, p := range partialBlobPaths {
		root := strings.TrimPrefix(hexutil.Encode(testSidecars[6].BlockRoot), "0x")
		partialBlobPath := filepath.Join(tempDir, root, p)
		err = os.WriteFile(partialBlobPath, []byte("Partial Blob Data"), 0644)
		require.NoError(t, err)
	}

	err = bs.PruneBlobViaSlotFile(currentSlot)
	require.NoError(t, err)

	remainingFolders, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	// Expecting 6 blobs from testSidecars to remain.
	require.Equal(t, 1, len(remainingFolders))

	// Ensure that the slot files are still present.
	for _, folder := range remainingFolders {
		if folder.IsDir() {
			files, err := os.ReadDir(path.Join(tempDir, folder.Name()))
			require.NoError(t, err)
			// Should have 6 blob files and 1 slot file.
			require.Equal(t, 7, len(files))
		}
	}
}

func TestExtractSlotFromFileName(t *testing.T) {
	tempDir := t.TempDir()
	testSidecars := generateBlobSidecars(t, []primitives.Slot{225519, 100}, fieldparams.MaxBlobsPerBlock)
	bs := &BlobStorage{baseDir: tempDir}
	err := bs.SaveBlobData(testSidecars)
	require.NoError(t, err)

	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)

	for _, f := range files {
		slot, err := extractSlotFromFileName(f.Name())
		require.NoError(t, err)
		sidecar := findTestSidecarsByFileName(t, testSidecars, f.Name())
		require.Equal(t, sidecar.Slot, slot)
	}
}

// setupDB instantiates and returns a Store instance.
func setupDB(t testing.TB) *kv.Store {
	db, err := kv.NewKVStore(context.Background(), t.TempDir())
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
	})
	return db
}

var blockTests = []struct {
	name     string
	newBlock func(primitives.Slot, []byte) (interfaces.ReadOnlySignedBeaconBlock, error)
}{
	{
		name: "deneb",
		newBlock: func(slot primitives.Slot, root []byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
			b := util.NewBeaconBlockDeneb()
			b.Block.Slot = slot
			if root != nil {
				b.Block.ParentRoot = root
				b.Block.Body.BlobKzgCommitments = [][]byte{
					bytesutil.PadTo([]byte{0x01}, 48),
					bytesutil.PadTo([]byte{0x02}, 48),
					bytesutil.PadTo([]byte{0x03}, 48),
					bytesutil.PadTo([]byte{0x04}, 48),
				}
			}
			return blocks.NewSignedBeaconBlock(b)
		},
	},
	{
		name: "deneb blind",
		newBlock: func(slot primitives.Slot, root []byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
			b := util.NewBlindedBeaconBlockDeneb()
			b.Message.Slot = slot
			if root != nil {
				b.Message.ParentRoot = root
				b.Message.Body.BlobKzgCommitments = [][]byte{
					bytesutil.PadTo([]byte{0x05}, 48),
					bytesutil.PadTo([]byte{0x06}, 48),
					bytesutil.PadTo([]byte{0x07}, 48),
					bytesutil.PadTo([]byte{0x08}, 48),
				}
			}
			return blocks.NewSignedBeaconBlock(b)
		},
	},
}
