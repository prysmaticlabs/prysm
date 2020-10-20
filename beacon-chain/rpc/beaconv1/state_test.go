package beaconv1

import (
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/proto/migration"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func fillDBTestState(ctx context.Context, t *testing.T, db db.Database) (*ethpb_alpha.SignedBeaconBlock, []*ethpb_alpha.BeaconBlockContainer) {
	genesis, keys := testutil.DeterministicGenesisState(t, 64)
	genesisBlockRoot := bytesutil.ToBytes32(nil)
	require.NoError(t, db.SaveState(ctx, genesis, genesisBlockRoot))
	stateRoot, err := genesis.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
	require.NoError(t, db.SaveBlock(ctx, genesisBlk))
	genesisBlkRoot, err := genesisBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlkRoot))

	count := uint64(100)
	blks := make([]*ethpb_alpha.SignedBeaconBlock, count)
	blkContainers := make([]*ethpb_alpha.BeaconBlockContainer, count)
	for i := uint64(1); i < count; i++ {
		b, err := testutil.GenerateFullBlock(genesis, keys, testutil.DefaultBlockGenConfig(), 1)
		assert.NoError(t, err)
		require.NoError(t, db.SaveBlock(ctx, b))

		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		blks[i] = b
		blkContainers[i] = &ethpb_alpha.BeaconBlockContainer{Block: b, BlockRoot: root[:]}
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))
	headRoot := bytesutil.ToBytes32(blkContainers[len(blks)-1].BlockRoot)
	summary := &p2ppb.StateSummary{
		Root: headRoot[:],
		Slot: blkContainers[len(blks)-1].Block.Block.Slot,
	}
	require.NoError(t, db.SaveStateSummary(ctx, summary))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, headRoot))
	return genesisBlk, blkContainers
}

func TestServer_GetStateRoot(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	genBlk, blkContainers := fillDBTestBlocks(ctx, t, db)
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	headBlock := blkContainers[len(blkContainers)-1]

	b2 := testutil.NewBeaconBlock()
	b2.Block.Slot = 30
	b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	require.NoError(t, db.SaveBlock(ctx, b2))
	b3 := testutil.NewBeaconBlock()
	b3.Block.Slot = 30
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	require.NoError(t, db.SaveBlock(ctx, b3))

	bs := &Server{
		BeaconDB: db,
		ChainInfoFetcher: &mock.ChainService{
			DB:                  db,
			Block:               headBlock.Block,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpb_alpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		},
	}

	tests := []struct {
		name    string
		blockID []byte
		want    *ethpb_alpha.SignedBeaconBlock
		wantErr bool
	}{
		{
			name:    "slot",
			blockID: []byte("30"),
			want:    blkContainers[30].Block,
		},
		{
			name:    "root",
			blockID: blkContainers[20].BlockRoot,
			want:    blkContainers[20].Block,
		},
		{
			name:    "canonical",
			blockID: []byte("30"),
			want:    blkContainers[30].Block,
		},
		{
			name:    "genesis",
			blockID: []byte("genesis"),
			want:    genBlk,
		},
		{
			name:    "genesis root",
			blockID: root[:],
			want:    genBlk,
		},
		{
			name:    "head",
			blockID: []byte("head"),
			want:    headBlock.Block,
		},
		{
			name:    "finalized",
			blockID: []byte("finalized"),
			want:    blkContainers[64].Block,
		},
		{
			name:    "no block",
			blockID: []byte("105"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, err := bs.GetBlockHeader(ctx, &ethpb.BlockRequest{
				BlockId: tt.blockID,
			})
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				require.NotEqual(t, err, nil)
				return
			}

			blkHdr, err := migration.V1Alpha1BlockToV1BlockHeader(tt.want)
			require.NoError(t, err)

			if !reflect.DeepEqual(header.Data.Header.Message, blkHdr.Header) {
				t.Error("Expected blocks to equal")
			}
		})
	}
}
