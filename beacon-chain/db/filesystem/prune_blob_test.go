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
	"github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestBlobStorage_PruneBlobWithDB(t *testing.T) {
	currentSlot := primitives.Slot(140000)
	slot := primitives.Slot(0)
	blockQty := 10
	b := setupDBTest(t)

	blobPath := filepath.Join(b.baseDir, "invalid_blob.blob")
	err := os.WriteFile(blobPath, []byte("Invalid Blob Data"), 0644)
	require.NoError(t, err)

	// Simulate invalid filename being ignored.
	err = b.bs.PruneBlobWithDB(b.ctx, currentSlot, b.db)
	require.NoError(t, err)
	err = os.Remove(blobPath)
	require.NoError(t, err)

	for i := 0; i < blockQty; i++ {
		b.addBlocks(t, slot, bytesutil.PadTo(bytesutil.ToBytes(uint64(slot), 32), 32))
		slot += 1000
	}

	err = b.bs.PruneBlobWithDB(b.ctx, currentSlot, b.db)
	require.NoError(t, err)

	remainingFolders, err := os.ReadDir(b.baseDir)
	require.NoError(t, err)
	// 1 folder and 1 database file
	require.Equal(t, 2, len(remainingFolders))

	// Ensure that the slot files are still present.
	for _, folder := range remainingFolders {
		if folder.IsDir() {
			files, err := os.ReadDir(path.Join(b.baseDir, folder.Name()))
			require.NoError(t, err)
			// Should have 6 blob files and 1 slot file.
			require.Equal(t, 7, len(files))
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

func TestBlobStorage_PruneBlobViaRead(t *testing.T) {
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

	err = bs.PruneBlobViaRead(currentSlot)
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

type blobTest struct {
	ctx     context.Context
	db      *kv.Store
	baseDir string
	bs      *BlobStorage
}

func setupDBTest(t *testing.T) *blobTest {
	ctx := context.Background()
	tempDir := t.TempDir()
	db, err := kv.NewKVStore(ctx, tempDir)
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
	})
	return &blobTest{
		ctx:     ctx,
		db:      db,
		baseDir: tempDir,
		bs: &BlobStorage{
			baseDir:        tempDir,
			retentionEpoch: 4096,
			lastPrunedSlot: 0,
		},
	}
}

func (p *blobTest) addBlocks(t *testing.T, slot primitives.Slot, root []byte) {
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
	blobSidecars := make([]*eth.BlobSidecar, fieldparams.MaxBlobsPerBlock)
	index := uint64(0)
	sb, err := convertToReadOnlySignedBeaconBlock(b)
	require.NoError(t, err)
	blockRoot, err := sb.Block().HashTreeRoot()
	require.NoError(t, err)
	for i := 0; i < fieldparams.MaxBlobsPerBlock; i++ {
		blobSidecars[index] = generateBlobSidecar(t, slot, index, blockRoot[:])
		index++
	}

	bs := &BlobStorage{baseDir: p.baseDir}
	err = bs.SaveBlobData(blobSidecars)
	require.NoError(t, err)

	require.NoError(t, p.db.SaveBlock(p.ctx, sb))
}

func convertToReadOnlySignedBeaconBlock(b *ethpb.SignedBeaconBlockDeneb) (interfaces.ReadOnlySignedBeaconBlock, error) {
	return blocks.NewSignedBeaconBlock(b)
}
