package filesystem

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/kv"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func BenchmarkPruning_DB(b *testing.B) {
	blockQty := 10000
	currentSlot := primitives.Slot(150000)
	slot := primitives.Slot(0)
	p := setupDBBench(b)
	for i := 0; i < blockQty; i++ {
		p.addBlocksBench(b, slot, bytesutil.PadTo(bytesutil.ToBytes(uint64(slot), 32), 32))
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
	blockQty := 10000
	currentSlot := primitives.Slot(150000)
	slot := primitives.Slot(0)
	p := setupDBBench(b)
	for i := 0; i < blockQty; i++ {
		p.addBlocksBench(b, slot, bytesutil.PadTo(bytesutil.ToBytes(uint64(slot), 32), 32))
		slot += 100
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := p.bs.PruneBlobViaSlotFile(currentSlot)
		require.NoError(b, err)
	}
}

func BenchmarkPruning_Read(b *testing.B) {
	blockQty := 10000
	currentSlot := primitives.Slot(150000)
	slot := primitives.Slot(0)
	p := setupDBBench(b)
	for i := 0; i < blockQty; i++ {
		p.addBlocksBench(b, slot, bytesutil.PadTo(bytesutil.ToBytes(uint64(slot), 32), 32))
		slot += 100
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := p.bs.PruneBlobViaRead(currentSlot)
		require.NoError(b, err)
	}
}

func setupDBBench(t *testing.B) *blobTest {
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

func (p *blobTest) addBlocksBench(t *testing.B, slot primitives.Slot, root []byte) {
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
		blobSidecars[index] = generateBlobSidecarBench(t, slot, index, blockRoot[:])
		index++
	}

	bs := &BlobStorage{baseDir: p.baseDir}
	err = bs.SaveBlobData(blobSidecars)
	require.NoError(t, err)

	require.NoError(t, p.db.SaveBlock(p.ctx, sb))
}

func generateBlobSidecarBench(t *testing.B, slot primitives.Slot, index uint64, root []byte) *eth.BlobSidecar {
	blob := make([]byte, 131072)
	_, err := rand.Read(blob)
	require.NoError(t, err)
	kzgCommitment := make([]byte, 48)
	_, err = rand.Read(kzgCommitment)
	require.NoError(t, err)
	kzgProof := make([]byte, 48)
	_, err = rand.Read(kzgProof)
	require.NoError(t, err)
	if len(root) == 0 {
		root = bytesutil.PadTo(bytesutil.ToBytes(uint64(slot), 32), 32)
	}
	return &eth.BlobSidecar{
		BlockRoot:       root,
		Index:           index,
		Slot:            slot,
		BlockParentRoot: bytesutil.PadTo([]byte{'b'}, 32),
		ProposerIndex:   101,
		Blob:            blob,
		KzgCommitment:   kzgCommitment,
		KzgProof:        kzgProof,
	}
}
