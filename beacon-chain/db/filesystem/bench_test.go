package filesystem

import (
	"context"
	"testing"

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

type pruneBenchmark struct {
	ctx     context.Context
	db      *kv.Store
	baseDir string
	bs      *BlobStorage
}

func (p *pruneBenchmark) addBlocks(t *testing.B, slot primitives.Slot, root []byte) {
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
	blockRoot, err := b.HashTreeRoot()
	require.NoError(t, err)
	for i := 0; i < fieldparams.MaxBlobsPerBlock; i++ {
		blobSidecars[index] = generateBlobSidecar(t, slot, index, blockRoot[:])
		index++
	}

	bs := &BlobStorage{baseDir: p.baseDir}
	err = bs.SaveBlobData(blobSidecars)
	require.NoError(t, err)

	sb, err := convertToReadOnlySignedBeaconBlock(b)
	require.NoError(t, err)

	require.NoError(t, p.db.SaveBlock(p.ctx, sb))
}

func setupBench(t *testing.B) *pruneBenchmark {
	ctx := context.Background()
	tempDir := t.TempDir()
	db, err := kv.NewKVStore(ctx, tempDir)
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
	})
	return &pruneBenchmark{
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

func convertToReadOnlySignedBeaconBlock(b *ethpb.SignedBeaconBlockDeneb) (interfaces.ReadOnlySignedBeaconBlock, error) {
	return blocks.NewSignedBeaconBlock(b)
}

func BenchmarkPruning_DB(b *testing.B) {
	blockNum := 1000
	currentSlot := primitives.Slot(10000)
	slot := primitives.Slot(0)
	p := setupBench(b)
	for i := 0; i < blockNum; i++ {
		p.addBlocks(b, slot, bytesutil.PadTo(bytesutil.ToBytes(uint64(slot), 32), 32))
		slot += 100
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := p.bs.PruneBlobWithDB(p.ctx, currentSlot, p.db)
		require.NoError(b, err)
	}
}

func BenchmarkPruning_Slot(b *testing.B) {
	blockNum := 1000
	currentSlot := primitives.Slot(10000)
	slot := primitives.Slot(0)
	p := setupBench(b)
	for i := 0; i < blockNum; i++ {
		p.addBlocks(b, slot, bytesutil.PadTo(bytesutil.ToBytes(uint64(slot), 32), 32))
		slot += 100
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := p.bs.PruneBlobViaSlotFile(currentSlot)
		require.NoError(b, err)
	}
}
