package beacon

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	mock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	builderTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/builder/testing"
	dbTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	executionTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/execution/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/synccommittee"
	mockp2p "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/prysm/v1alpha1/validator"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v3/proto/migration"
	ethpbalpha "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"google.golang.org/grpc/metadata"
)

func TestServer_GetBlindedBlock(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genBlk, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
		canonicalRoots := make(map[[32]byte]bool)

		for _, bContr := range blkContainers {
			canonicalRoots[bytesutil.ToBytes32(bContr.BlockRoot)] = true
		}
		headBlock := blkContainers[len(blkContainers)-1]
		nextSlot := headBlock.GetPhase0Block().Block.Slot + 1

		b2 := util.NewBeaconBlock()
		b2.Block.Slot = 30
		b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
		util.SaveBlock(t, ctx, beaconDB, b2)
		b3 := util.NewBeaconBlock()
		b3.Block.Slot = 30
		b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
		util.SaveBlock(t, ctx, beaconDB, b3)
		b4 := util.NewBeaconBlock()
		b4.Block.Slot = nextSlot
		b4.Block.ParentRoot = bytesutil.PadTo([]byte{8}, 32)
		util.SaveBlock(t, ctx, beaconDB, b4)

		wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
		require.NoError(t, err)

		mockChainService := &mock.ChainService{
			DB:                  beaconDB,
			Block:               wsb,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			CanonicalRoots:      canonicalRoots,
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainService,
			HeadFetcher:           mockChainService,
			OptimisticModeFetcher: mockChainService,
		}

		root, err := genBlk.Block.HashTreeRoot()
		require.NoError(t, err)

		tests := []struct {
			name    string
			blockID []byte
			want    *ethpbalpha.SignedBeaconBlock
			wantErr bool
		}{
			{
				name:    "slot",
				blockID: []byte("30"),
				want:    blkContainers[30].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
			},
			{
				name:    "bad formatting",
				blockID: []byte("3bad0"),
				wantErr: true,
			},
			{
				name:    "canonical",
				blockID: []byte("30"),
				want:    blkContainers[30].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
			},
			{
				name:    "non canonical",
				blockID: []byte(fmt.Sprintf("%d", nextSlot)),
				wantErr: true,
			},
			{
				name:    "head",
				blockID: []byte("head"),
				want:    headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
			},
			{
				name:    "finalized",
				blockID: []byte("finalized"),
				want:    blkContainers[64].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
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
				name:    "root",
				blockID: blkContainers[20].BlockRoot,
				want:    blkContainers[20].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
			},
			{
				name:    "non-existent root",
				blockID: bytesutil.PadTo([]byte("hi there"), 32),
				wantErr: true,
			},
			{
				name:    "no block",
				blockID: []byte("105"),
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				blk, err := bs.GetBlindedBlock(ctx, &ethpbv1.BlockRequest{
					BlockId: tt.blockID,
				})
				if tt.wantErr {
					require.NotEqual(t, err, nil)
					return
				}
				require.NoError(t, err)

				v1Block, err := migration.V1Alpha1ToV1SignedBlock(tt.want)
				require.NoError(t, err)

				phase0Block, ok := blk.Data.Message.(*ethpbv2.SignedBlindedBeaconBlockContainer_Phase0Block)
				require.Equal(t, true, ok)
				if !reflect.DeepEqual(phase0Block.Phase0Block, v1Block.Block) {
					t.Error("Expected blocks to equal")
				}
				assert.Equal(t, ethpbv2.Version_PHASE0, blk.Version)
			})
		}
	})

	t.Run("Altair", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genBlk, blkContainers := fillDBTestBlocksAltair(ctx, t, beaconDB)
		canonicalRoots := make(map[[32]byte]bool)

		for _, bContr := range blkContainers {
			canonicalRoots[bytesutil.ToBytes32(bContr.BlockRoot)] = true
		}
		headBlock := blkContainers[len(blkContainers)-1]
		nextSlot := headBlock.GetAltairBlock().Block.Slot + 1

		b2 := util.NewBeaconBlockAltair()
		b2.Block.Slot = 30
		b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
		util.SaveBlock(t, ctx, beaconDB, b2)
		b3 := util.NewBeaconBlockAltair()
		b3.Block.Slot = 30
		b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
		util.SaveBlock(t, ctx, beaconDB, b3)
		b4 := util.NewBeaconBlockAltair()
		b4.Block.Slot = nextSlot
		b4.Block.ParentRoot = bytesutil.PadTo([]byte{8}, 32)
		util.SaveBlock(t, ctx, beaconDB, b4)

		chainBlk, err := blocks.NewSignedBeaconBlock(headBlock.GetAltairBlock())
		require.NoError(t, err)
		mockChainService := &mock.ChainService{
			DB:                  beaconDB,
			Block:               chainBlk,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			CanonicalRoots:      canonicalRoots,
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainService,
			HeadFetcher:           mockChainService,
			OptimisticModeFetcher: mockChainService,
		}

		root, err := genBlk.Block.HashTreeRoot()
		require.NoError(t, err)

		tests := []struct {
			name    string
			blockID []byte
			want    *ethpbalpha.SignedBeaconBlockAltair
			wantErr bool
		}{
			{
				name:    "slot",
				blockID: []byte("30"),
				want:    blkContainers[30].GetAltairBlock(),
			},
			{
				name:    "bad formatting",
				blockID: []byte("3bad0"),
				wantErr: true,
			},
			{
				name:    "canonical",
				blockID: []byte("30"),
				want:    blkContainers[30].GetAltairBlock(),
			},
			{
				name:    "non canonical",
				blockID: []byte(fmt.Sprintf("%d", nextSlot)),
				wantErr: true,
			},
			{
				name:    "head",
				blockID: []byte("head"),
				want:    headBlock.GetAltairBlock(),
			},
			{
				name:    "finalized",
				blockID: []byte("finalized"),
				want:    blkContainers[64].GetAltairBlock(),
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
				name:    "root",
				blockID: blkContainers[20].BlockRoot,
				want:    blkContainers[20].GetAltairBlock(),
			},
			{
				name:    "non-existent root",
				blockID: bytesutil.PadTo([]byte("hi there"), 32),
				wantErr: true,
			},
			{
				name:    "no block",
				blockID: []byte("105"),
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				blk, err := bs.GetBlindedBlock(ctx, &ethpbv1.BlockRequest{
					BlockId: tt.blockID,
				})
				if tt.wantErr {
					require.NotEqual(t, err, nil)
					return
				}
				require.NoError(t, err)

				v2Block, err := migration.V1Alpha1BeaconBlockAltairToV2(tt.want.Block)
				require.NoError(t, err)

				altairBlock, ok := blk.Data.Message.(*ethpbv2.SignedBlindedBeaconBlockContainer_AltairBlock)
				require.Equal(t, true, ok)
				if !reflect.DeepEqual(altairBlock.AltairBlock, v2Block) {
					t.Error("Expected blocks to equal")
				}
				assert.Equal(t, ethpbv2.Version_ALTAIR, blk.Version)
			})
		}
	})

	t.Run("Bellatrix", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genBlk, blkContainers := fillDBTestBlocksBellatrixBlinded(ctx, t, beaconDB)
		canonicalRoots := make(map[[32]byte]bool)

		for _, bContr := range blkContainers {
			canonicalRoots[bytesutil.ToBytes32(bContr.BlockRoot)] = true
		}
		headBlock := blkContainers[len(blkContainers)-1]
		nextSlot := headBlock.GetBlindedBellatrixBlock().Block.Slot + 1

		b2 := util.NewBlindedBeaconBlockBellatrix()
		b2.Block.Slot = 30
		b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
		util.SaveBlock(t, ctx, beaconDB, b2)
		b3 := util.NewBlindedBeaconBlockBellatrix()
		b3.Block.Slot = 30
		b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
		util.SaveBlock(t, ctx, beaconDB, b3)
		b4 := util.NewBlindedBeaconBlockBellatrix()
		b4.Block.Slot = nextSlot
		b4.Block.ParentRoot = bytesutil.PadTo([]byte{8}, 32)
		util.SaveBlock(t, ctx, beaconDB, b4)

		chainBlk, err := blocks.NewSignedBeaconBlock(headBlock.GetBlindedBellatrixBlock())
		require.NoError(t, err)
		mockChainService := &mock.ChainService{
			DB:                  beaconDB,
			Block:               chainBlk,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			CanonicalRoots:      canonicalRoots,
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainService,
			HeadFetcher:           mockChainService,
			OptimisticModeFetcher: mockChainService,
			ExecutionPayloadReconstructor: &executionTest.EngineClient{
				ExecutionPayloadByBlockHash: map[[32]byte]*enginev1.ExecutionPayload{},
			},
		}

		root, err := genBlk.Block.HashTreeRoot()
		require.NoError(t, err)

		tests := []struct {
			name    string
			blockID []byte
			want    *ethpbalpha.SignedBlindedBeaconBlockBellatrix
			wantErr bool
		}{
			{
				name:    "slot",
				blockID: []byte("30"),
				want:    blkContainers[30].GetBlindedBellatrixBlock(),
			},
			{
				name:    "bad formatting",
				blockID: []byte("3bad0"),
				wantErr: true,
			},
			{
				name:    "canonical",
				blockID: []byte("30"),
				want:    blkContainers[30].GetBlindedBellatrixBlock(),
			},
			{
				name:    "non canonical",
				blockID: []byte(fmt.Sprintf("%d", nextSlot)),
				wantErr: true,
			},
			{
				name:    "head",
				blockID: []byte("head"),
				want:    headBlock.GetBlindedBellatrixBlock(),
			},
			{
				name:    "finalized",
				blockID: []byte("finalized"),
				want:    blkContainers[64].GetBlindedBellatrixBlock(),
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
				name:    "root",
				blockID: blkContainers[20].BlockRoot,
				want:    blkContainers[20].GetBlindedBellatrixBlock(),
			},
			{
				name:    "non-existent root",
				blockID: bytesutil.PadTo([]byte("hi there"), 32),
				wantErr: true,
			},
			{
				name:    "no block",
				blockID: []byte("105"),
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				blk, err := bs.GetBlindedBlock(ctx, &ethpbv1.BlockRequest{
					BlockId: tt.blockID,
				})
				if tt.wantErr {
					require.NotEqual(t, err, nil)
					return
				}
				require.NoError(t, err)

				v2Block, err := migration.V1Alpha1BeaconBlockBlindedBellatrixToV2Blinded(tt.want.Block)
				require.NoError(t, err)

				b, ok := blk.Data.Message.(*ethpbv2.SignedBlindedBeaconBlockContainer_BellatrixBlock)
				require.Equal(t, true, ok)
				if !reflect.DeepEqual(b.BellatrixBlock, v2Block) {
					t.Error("Expected blocks to equal")
				}
				assert.Equal(t, ethpbv2.Version_BELLATRIX, blk.Version)
			})
		}
	})

	t.Run("Capella", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genBlk, blkContainers := fillDBTestBlocksCapellaBlinded(ctx, t, beaconDB)
		canonicalRoots := make(map[[32]byte]bool)

		for _, bContr := range blkContainers {
			canonicalRoots[bytesutil.ToBytes32(bContr.BlockRoot)] = true
		}
		headBlock := blkContainers[len(blkContainers)-1]
		nextSlot := headBlock.GetBlindedCapellaBlock().Block.Slot + 1

		b2 := util.NewBlindedBeaconBlockCapella()
		b2.Block.Slot = 30
		b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
		util.SaveBlock(t, ctx, beaconDB, b2)
		b3 := util.NewBlindedBeaconBlockCapella()
		b3.Block.Slot = 30
		b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
		util.SaveBlock(t, ctx, beaconDB, b3)
		b4 := util.NewBlindedBeaconBlockCapella()
		b4.Block.Slot = nextSlot
		b4.Block.ParentRoot = bytesutil.PadTo([]byte{8}, 32)
		util.SaveBlock(t, ctx, beaconDB, b4)

		chainBlk, err := blocks.NewSignedBeaconBlock(headBlock.GetBlindedCapellaBlock())
		require.NoError(t, err)
		mockChainService := &mock.ChainService{
			DB:                  beaconDB,
			Block:               chainBlk,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			CanonicalRoots:      canonicalRoots,
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainService,
			HeadFetcher:           mockChainService,
			OptimisticModeFetcher: mockChainService,
			ExecutionPayloadReconstructor: &executionTest.EngineClient{
				ExecutionPayloadByBlockHash: map[[32]byte]*enginev1.ExecutionPayload{},
			},
		}

		root, err := genBlk.Block.HashTreeRoot()
		require.NoError(t, err)

		tests := []struct {
			name    string
			blockID []byte
			want    *ethpbalpha.SignedBlindedBeaconBlockCapella
			wantErr bool
		}{
			{
				name:    "slot",
				blockID: []byte("30"),
				want:    blkContainers[30].GetBlindedCapellaBlock(),
			},
			{
				name:    "bad formatting",
				blockID: []byte("3bad0"),
				wantErr: true,
			},
			{
				name:    "canonical",
				blockID: []byte("30"),
				want:    blkContainers[30].GetBlindedCapellaBlock(),
			},
			{
				name:    "non canonical",
				blockID: []byte(fmt.Sprintf("%d", nextSlot)),
				wantErr: true,
			},
			{
				name:    "head",
				blockID: []byte("head"),
				want:    headBlock.GetBlindedCapellaBlock(),
			},
			{
				name:    "finalized",
				blockID: []byte("finalized"),
				want:    blkContainers[64].GetBlindedCapellaBlock(),
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
				name:    "root",
				blockID: blkContainers[20].BlockRoot,
				want:    blkContainers[20].GetBlindedCapellaBlock(),
			},
			{
				name:    "non-existent root",
				blockID: bytesutil.PadTo([]byte("hi there"), 32),
				wantErr: true,
			},
			{
				name:    "no block",
				blockID: []byte("105"),
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				blk, err := bs.GetBlindedBlock(ctx, &ethpbv1.BlockRequest{
					BlockId: tt.blockID,
				})
				if tt.wantErr {
					require.NotEqual(t, err, nil)
					return
				}
				require.NoError(t, err)

				v2Block, err := migration.V1Alpha1BeaconBlockBlindedCapellaToV2Blinded(tt.want.Block)
				require.NoError(t, err)

				b, ok := blk.Data.Message.(*ethpbv2.SignedBlindedBeaconBlockContainer_CapellaBlock)
				require.Equal(t, true, ok)
				if !reflect.DeepEqual(b.CapellaBlock, v2Block) {
					t.Error("Expected blocks to equal")
				}
				assert.Equal(t, ethpbv2.Version_CAPELLA, blk.Version)
			})
		}
	})

	t.Run("execution optimistic", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		_, blkContainers := fillDBTestBlocksBellatrix(ctx, t, beaconDB)
		headBlock := blkContainers[len(blkContainers)-1]

		b2 := util.NewBeaconBlockBellatrix()
		b2.Block.Slot = 30
		b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
		util.SaveBlock(t, ctx, beaconDB, b2)
		b3 := util.NewBeaconBlockBellatrix()
		b3.Block.Slot = 30
		b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
		util.SaveBlock(t, ctx, beaconDB, b3)

		chainBlk, err := blocks.NewSignedBeaconBlock(headBlock.GetBellatrixBlock())
		require.NoError(t, err)
		mockChainService := &mock.ChainService{
			DB:                  beaconDB,
			Block:               chainBlk,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			Optimistic:          true,
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainService,
			HeadFetcher:           mockChainService,
			OptimisticModeFetcher: mockChainService,
		}

		blk, err := bs.GetBlindedBlock(ctx, &ethpbv1.BlockRequest{
			BlockId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, true, blk.ExecutionOptimistic)
	})
}

func TestServer_GetBlindedBlockSSZ(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		_, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
		headBlock := blkContainers[len(blkContainers)-1]

		b2 := util.NewBeaconBlock()
		b2.Block.Slot = 30
		b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
		util.SaveBlock(t, ctx, beaconDB, b2)

		wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
		require.NoError(t, err)

		bs := &Server{
			BeaconDB: beaconDB,
			ChainInfoFetcher: &mock.ChainService{
				DB:                  beaconDB,
				Block:               wsb,
				Root:                headBlock.BlockRoot,
				FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			},
		}

		blks, err := beaconDB.BlocksBySlot(ctx, 30)
		require.Equal(t, true, len(blks) > 0)
		require.NoError(t, err)
		sszBlock, err := blks[0].MarshalSSZ()
		require.NoError(t, err)

		resp, err := bs.GetBlindedBlockSSZ(ctx, &ethpbv1.BlockRequest{BlockId: []byte("30")})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, sszBlock, resp.Data)
		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)
	})

	t.Run("Altair", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		_, blkContainers := fillDBTestBlocksAltair(ctx, t, beaconDB)
		headBlock := blkContainers[len(blkContainers)-1]

		b2 := util.NewBeaconBlockAltair()
		b2.Block.Slot = 30
		b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
		util.SaveBlock(t, ctx, beaconDB, b2)

		chainBlk, err := blocks.NewSignedBeaconBlock(headBlock.GetAltairBlock())
		require.NoError(t, err)
		bs := &Server{
			BeaconDB: beaconDB,
			ChainInfoFetcher: &mock.ChainService{
				DB:                  beaconDB,
				Block:               chainBlk,
				Root:                headBlock.BlockRoot,
				FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			},
		}

		blks, err := beaconDB.BlocksBySlot(ctx, 30)
		require.Equal(t, true, len(blks) > 0)
		require.NoError(t, err)
		sszBlock, err := blks[0].MarshalSSZ()
		require.NoError(t, err)

		resp, err := bs.GetBlindedBlockSSZ(ctx, &ethpbv1.BlockRequest{BlockId: []byte("30")})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, sszBlock, resp.Data)
		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
	})

	t.Run("Bellatrix", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		_, blkContainers := fillDBTestBlocksBellatrixBlinded(ctx, t, beaconDB)
		headBlock := blkContainers[len(blkContainers)-1]

		b2 := util.NewBlindedBeaconBlockBellatrix()
		b2.Block.Slot = 30
		b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
		util.SaveBlock(t, ctx, beaconDB, b2)

		chainBlk, err := blocks.NewSignedBeaconBlock(headBlock.GetBlindedBellatrixBlock())
		require.NoError(t, err)
		bs := &Server{
			BeaconDB: beaconDB,
			ChainInfoFetcher: &mock.ChainService{
				DB:                  beaconDB,
				Block:               chainBlk,
				Root:                headBlock.BlockRoot,
				FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			},
			OptimisticModeFetcher: &mock.ChainService{},
		}

		blks, err := beaconDB.BlocksBySlot(ctx, 30)
		require.Equal(t, true, len(blks) > 0)
		require.NoError(t, err)
		sszBlock, err := blks[0].MarshalSSZ()
		require.NoError(t, err)

		resp, err := bs.GetBlindedBlockSSZ(ctx, &ethpbv1.BlockRequest{BlockId: []byte("30")})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, sszBlock, resp.Data)
		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)
	})

	t.Run("Capella", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		_, blkContainers := fillDBTestBlocksCapellaBlinded(ctx, t, beaconDB)
		headBlock := blkContainers[len(blkContainers)-1]

		b2 := util.NewBlindedBeaconBlockCapella()
		b2.Block.Slot = 30
		b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
		util.SaveBlock(t, ctx, beaconDB, b2)

		chainBlk, err := blocks.NewSignedBeaconBlock(headBlock.GetBlindedCapellaBlock())
		require.NoError(t, err)
		bs := &Server{
			BeaconDB: beaconDB,
			ChainInfoFetcher: &mock.ChainService{
				DB:                  beaconDB,
				Block:               chainBlk,
				Root:                headBlock.BlockRoot,
				FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			},
			OptimisticModeFetcher: &mock.ChainService{},
		}

		blks, err := beaconDB.BlocksBySlot(ctx, 30)
		require.Equal(t, true, len(blks) > 0)
		require.NoError(t, err)
		sszBlock, err := blks[0].MarshalSSZ()
		require.NoError(t, err)

		resp, err := bs.GetBlindedBlockSSZ(ctx, &ethpbv1.BlockRequest{BlockId: []byte("30")})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, sszBlock, resp.Data)
		assert.Equal(t, ethpbv2.Version_CAPELLA, resp.Version)
	})
}

func TestServer_SubmitBlindedBlockSSZ_OK(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genesis := util.NewBeaconBlock()
		util.SaveBlock(t, context.Background(), beaconDB, genesis)

		numDeposits := uint64(64)
		beaconState, _ := util.DeterministicGenesisState(t, numDeposits)
		bsRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err)
		genesisRoot, err := genesis.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

		c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
		beaconChainServer := &Server{
			BeaconDB:         beaconDB,
			BlockReceiver:    c,
			ChainInfoFetcher: c,
			BlockNotifier:    c.BlockNotifier(),
			Broadcaster:      mockp2p.NewTestP2P(t),
			HeadFetcher:      c,
		}
		req := util.NewBeaconBlock()
		req.Block.Slot = 5
		req.Block.ParentRoot = bsRoot[:]
		util.SaveBlock(t, ctx, beaconDB, req)
		blockSsz, err := req.MarshalSSZ()
		require.NoError(t, err)
		blockReq := &ethpbv2.SSZContainer{
			Data: blockSsz,
		}
		md := metadata.MD{}
		md.Set(versionHeader, "phase0")
		sszCtx := metadata.NewIncomingContext(ctx, md)
		_, err = beaconChainServer.SubmitBlindedBlockSSZ(sszCtx, blockReq)
		assert.NoError(t, err, "Could not propose block correctly")
	})

	t.Run("Altair", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genesis := util.NewBeaconBlockAltair()
		util.SaveBlock(t, context.Background(), beaconDB, genesis)

		numDeposits := uint64(64)
		beaconState, _ := util.DeterministicGenesisState(t, numDeposits)
		bsRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err)
		genesisRoot, err := genesis.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

		c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
		beaconChainServer := &Server{
			BeaconDB:         beaconDB,
			BlockReceiver:    c,
			ChainInfoFetcher: c,
			BlockNotifier:    c.BlockNotifier(),
			Broadcaster:      mockp2p.NewTestP2P(t),
			HeadFetcher:      c,
		}
		req := util.NewBeaconBlockAltair()
		req.Block.Slot = params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().AltairForkEpoch))
		req.Block.ParentRoot = bsRoot[:]
		util.SaveBlock(t, ctx, beaconDB, req)
		blockSsz, err := req.MarshalSSZ()
		require.NoError(t, err)
		blockReq := &ethpbv2.SSZContainer{
			Data: blockSsz,
		}
		md := metadata.MD{}
		md.Set(versionHeader, "altair")
		sszCtx := metadata.NewIncomingContext(ctx, md)
		_, err = beaconChainServer.SubmitBlindedBlockSSZ(sszCtx, blockReq)
		assert.NoError(t, err, "Could not propose block correctly")
	})

	t.Run("Bellatrix", func(t *testing.T) {
		// INFO: This code block can be removed once Bellatrix
		// fork epoch is set to a value other than math.MaxUint64
		params.SetupTestConfigCleanup(t)
		cfg := params.BeaconConfig()
		cfg.BellatrixForkEpoch = cfg.AltairForkEpoch + 1000
		cfg.ForkVersionSchedule[bytesutil.ToBytes4(cfg.BellatrixForkVersion)] = cfg.AltairForkEpoch + 1000
		params.OverrideBeaconConfig(cfg)

		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genesis := util.NewBeaconBlockBellatrix()
		util.SaveBlock(t, context.Background(), beaconDB, genesis)

		numDeposits := uint64(64)
		beaconState, _ := util.DeterministicGenesisState(t, numDeposits)
		bsRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err)
		genesisRoot, err := genesis.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

		c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
		alphaServer := &validator.Server{
			SyncCommitteePool: synccommittee.NewStore(),
			P2P:               &mockp2p.MockBroadcaster{},
			BlockBuilder:      &builderTest.MockBuilderService{},
			BlockReceiver:     c,
			BlockNotifier:     &mock.MockBlockNotifier{},
		}
		beaconChainServer := &Server{
			BeaconDB:                beaconDB,
			BlockReceiver:           c,
			ChainInfoFetcher:        c,
			BlockNotifier:           c.BlockNotifier(),
			Broadcaster:             mockp2p.NewTestP2P(t),
			HeadFetcher:             c,
			V1Alpha1ValidatorServer: alphaServer,
		}
		req := util.NewBlindedBeaconBlockBellatrix()
		req.Block.Slot = params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().BellatrixForkEpoch))
		req.Block.ParentRoot = bsRoot[:]
		util.SaveBlock(t, ctx, beaconDB, req)
		blockSsz, err := req.MarshalSSZ()
		require.NoError(t, err)
		blockReq := &ethpbv2.SSZContainer{
			Data: blockSsz,
		}
		md := metadata.MD{}
		md.Set(versionHeader, "bellatrix")
		sszCtx := metadata.NewIncomingContext(ctx, md)
		_, err = beaconChainServer.SubmitBlindedBlockSSZ(sszCtx, blockReq)
		assert.NoError(t, err, "Could not propose block correctly")
	})

	t.Run("Capella", func(t *testing.T) {
		t.Skip("This test needs Capella fork version configured properly")

		// INFO: This code block can be removed once Capella
		// fork epoch is set to a value other than math.MaxUint64
		params.SetupTestConfigCleanup(t)
		cfg := params.BeaconConfig()
		cfg.CapellaForkEpoch = cfg.BellatrixForkEpoch + 1000
		cfg.ForkVersionSchedule[bytesutil.ToBytes4(cfg.CapellaForkVersion)] = cfg.BellatrixForkEpoch + 1000
		params.OverrideBeaconConfig(cfg)

		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genesis := util.NewBeaconBlockCapella()
		util.SaveBlock(t, context.Background(), beaconDB, genesis)

		numDeposits := uint64(64)
		beaconState, _ := util.DeterministicGenesisState(t, numDeposits)
		bsRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err)
		genesisRoot, err := genesis.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

		c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
		alphaServer := &validator.Server{
			SyncCommitteePool: synccommittee.NewStore(),
			P2P:               &mockp2p.MockBroadcaster{},
			BlockBuilder:      &builderTest.MockBuilderService{},
			BlockReceiver:     c,
			BlockNotifier:     &mock.MockBlockNotifier{},
		}
		beaconChainServer := &Server{
			BeaconDB:                beaconDB,
			BlockReceiver:           c,
			ChainInfoFetcher:        c,
			BlockNotifier:           c.BlockNotifier(),
			Broadcaster:             mockp2p.NewTestP2P(t),
			HeadFetcher:             c,
			V1Alpha1ValidatorServer: alphaServer,
		}
		req := util.NewBlindedBeaconBlockCapella()
		req.Block.Slot = params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().CapellaForkEpoch))
		req.Block.ParentRoot = bsRoot[:]
		util.SaveBlock(t, ctx, beaconDB, req)
		blockSsz, err := req.MarshalSSZ()
		require.NoError(t, err)
		blockReq := &ethpbv2.SSZContainer{
			Data: blockSsz,
		}
		md := metadata.MD{}
		md.Set(versionHeader, "capella")
		sszCtx := metadata.NewIncomingContext(ctx, md)
		_, err = beaconChainServer.SubmitBlindedBlockSSZ(sszCtx, blockReq)
		assert.NoError(t, err, "Could not propose block correctly")
	})
}

func TestSubmitBlindedBlock(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genesis := util.NewBeaconBlock()
		util.SaveBlock(t, context.Background(), beaconDB, genesis)

		numDeposits := uint64(64)
		beaconState, _ := util.DeterministicGenesisState(t, numDeposits)
		bsRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err)
		genesisRoot, err := genesis.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

		c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
		beaconChainServer := &Server{
			BeaconDB:         beaconDB,
			BlockReceiver:    c,
			ChainInfoFetcher: c,
			BlockNotifier:    c.BlockNotifier(),
			Broadcaster:      mockp2p.NewTestP2P(t),
		}
		req := util.NewBeaconBlock()
		req.Block.Slot = 5
		req.Block.ParentRoot = bsRoot[:]
		v1Block, err := migration.V1Alpha1ToV1SignedBlock(req)
		require.NoError(t, err)
		util.SaveBlock(t, ctx, beaconDB, req)
		blockReq := &ethpbv2.SignedBlindedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_Phase0Block{Phase0Block: v1Block.Block},
			Signature: v1Block.Signature,
		}
		_, err = beaconChainServer.SubmitBlindedBlock(context.Background(), blockReq)
		assert.NoError(t, err, "Could not propose block correctly")
	})

	t.Run("Altair", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genesis := util.NewBeaconBlockAltair()
		util.SaveBlock(t, context.Background(), beaconDB, genesis)

		numDeposits := uint64(64)
		beaconState, _ := util.DeterministicGenesisState(t, numDeposits)
		bsRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err)
		genesisRoot, err := genesis.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

		c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
		beaconChainServer := &Server{
			BeaconDB:         beaconDB,
			BlockReceiver:    c,
			ChainInfoFetcher: c,
			BlockNotifier:    c.BlockNotifier(),
			Broadcaster:      mockp2p.NewTestP2P(t),
		}
		req := util.NewBeaconBlockAltair()
		req.Block.Slot = 5
		req.Block.ParentRoot = bsRoot[:]
		v2Block, err := migration.V1Alpha1BeaconBlockAltairToV2(req.Block)
		require.NoError(t, err)
		util.SaveBlock(t, ctx, beaconDB, req)
		blockReq := &ethpbv2.SignedBlindedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_AltairBlock{AltairBlock: v2Block},
			Signature: req.Signature,
		}
		_, err = beaconChainServer.SubmitBlindedBlock(context.Background(), blockReq)
		assert.NoError(t, err, "Could not propose block correctly")
	})

	t.Run("Bellatrix", func(t *testing.T) {
		transactions := [][]byte{[]byte("transaction1"), []byte("transaction2")}
		transactionsRoot, err := ssz.TransactionsRoot(transactions)
		require.NoError(t, err)

		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genesis := util.NewBeaconBlockBellatrix()
		util.SaveBlock(t, context.Background(), beaconDB, genesis)

		numDeposits := uint64(64)
		beaconState, _ := util.DeterministicGenesisState(t, numDeposits)
		bsRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err)
		genesisRoot, err := genesis.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

		c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
		alphaServer := &validator.Server{
			SyncCommitteePool: synccommittee.NewStore(),
			P2P:               &mockp2p.MockBroadcaster{},
			BlockBuilder:      &builderTest.MockBuilderService{},
			BlockReceiver:     c,
			BlockNotifier:     &mock.MockBlockNotifier{},
		}
		beaconChainServer := &Server{
			BeaconDB:                beaconDB,
			BlockReceiver:           c,
			ChainInfoFetcher:        c,
			BlockNotifier:           c.BlockNotifier(),
			Broadcaster:             mockp2p.NewTestP2P(t),
			V1Alpha1ValidatorServer: alphaServer,
		}

		blk := util.NewBeaconBlockBellatrix()
		blk.Block.Slot = 5
		blk.Block.ParentRoot = bsRoot[:]
		blk.Block.Body.ExecutionPayload.Transactions = transactions
		blindedBlk := util.NewBlindedBeaconBlockBellatrixV2()
		blindedBlk.Message.Slot = 5
		blindedBlk.Message.ParentRoot = bsRoot[:]
		blindedBlk.Message.Body.ExecutionPayloadHeader.TransactionsRoot = transactionsRoot[:]
		util.SaveBlock(t, ctx, beaconDB, blk)

		blockReq := &ethpbv2.SignedBlindedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_BellatrixBlock{BellatrixBlock: blindedBlk.Message},
			Signature: blindedBlk.Signature,
		}
		_, err = beaconChainServer.SubmitBlindedBlock(context.Background(), blockReq)
		assert.NoError(t, err)
	})

	t.Run("Capella", func(t *testing.T) {
		transactions := [][]byte{[]byte("transaction1"), []byte("transaction2")}
		transactionsRoot, err := ssz.TransactionsRoot(transactions)
		require.NoError(t, err)

		withdrawals := []*enginev1.Withdrawal{
			{
				Index:          1,
				ValidatorIndex: 1,
				Address:        bytesutil.PadTo([]byte("address1"), 20),
				Amount:         1,
			},
			{
				Index:          2,
				ValidatorIndex: 2,
				Address:        bytesutil.PadTo([]byte("address2"), 20),
				Amount:         2,
			},
		}
		withdrawalsRoot, err := ssz.WithdrawalSliceRoot(hash.CustomSHA256Hasher(), withdrawals, 16)
		require.NoError(t, err)

		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genesis := util.NewBeaconBlockCapella()
		util.SaveBlock(t, context.Background(), beaconDB, genesis)

		numDeposits := uint64(64)
		beaconState, _ := util.DeterministicGenesisState(t, numDeposits)
		bsRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err)
		genesisRoot, err := genesis.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

		c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
		alphaServer := &validator.Server{
			SyncCommitteePool: synccommittee.NewStore(),
			P2P:               &mockp2p.MockBroadcaster{},
			BlockBuilder:      &builderTest.MockBuilderService{},
			BlockReceiver:     c,
			BlockNotifier:     &mock.MockBlockNotifier{},
		}
		beaconChainServer := &Server{
			BeaconDB:                beaconDB,
			BlockReceiver:           c,
			ChainInfoFetcher:        c,
			BlockNotifier:           c.BlockNotifier(),
			Broadcaster:             mockp2p.NewTestP2P(t),
			V1Alpha1ValidatorServer: alphaServer,
		}

		blk := util.NewBeaconBlockCapella()
		blk.Block.Slot = 5
		blk.Block.ParentRoot = bsRoot[:]
		blk.Block.Body.ExecutionPayload.Transactions = transactions
		blk.Block.Body.ExecutionPayload.Withdrawals = withdrawals
		blindedBlk := util.NewBlindedBeaconBlockCapellaV2()
		blindedBlk.Message.Slot = 5
		blindedBlk.Message.ParentRoot = bsRoot[:]
		blindedBlk.Message.Body.ExecutionPayloadHeader.TransactionsRoot = transactionsRoot[:]
		blindedBlk.Message.Body.ExecutionPayloadHeader.WithdrawalsRoot = withdrawalsRoot[:]
		util.SaveBlock(t, ctx, beaconDB, blk)

		blockReq := &ethpbv2.SignedBlindedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_CapellaBlock{CapellaBlock: blindedBlk.Message},
			Signature: blindedBlk.Signature,
		}
		_, err = beaconChainServer.SubmitBlindedBlock(context.Background(), blockReq)
		assert.NoError(t, err)
	})
}
