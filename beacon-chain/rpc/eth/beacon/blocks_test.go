package beacon

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	mock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	dbTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	executionTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/execution/testing"
	mockp2p "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
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

func fillDBTestBlocks(ctx context.Context, t *testing.T, beaconDB db.Database) (*ethpbalpha.SignedBeaconBlock, []*ethpbalpha.BeaconBlockContainer) {
	parentRoot := [32]byte{1, 2, 3}
	genBlk := util.NewBeaconBlock()
	genBlk.Block.ParentRoot = parentRoot[:]
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, beaconDB, genBlk)
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, root))

	count := types.Slot(100)
	blks := make([]interfaces.SignedBeaconBlock, count)
	blkContainers := make([]*ethpbalpha.BeaconBlockContainer, count)
	for i := types.Slot(0); i < count; i++ {
		b := util.NewBeaconBlock()
		b.Block.Slot = i
		b.Block.ParentRoot = bytesutil.PadTo([]byte{uint8(i)}, 32)
		att1 := util.NewAttestation()
		att1.Data.Slot = i
		att1.Data.CommitteeIndex = types.CommitteeIndex(i)
		att2 := util.NewAttestation()
		att2.Data.Slot = i
		att2.Data.CommitteeIndex = types.CommitteeIndex(i + 1)
		b.Block.Body.Attestations = []*ethpbalpha.Attestation{att1, att2}
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		blks[i], err = blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		blkContainers[i] = &ethpbalpha.BeaconBlockContainer{
			Block:     &ethpbalpha.BeaconBlockContainer_Phase0Block{Phase0Block: b},
			BlockRoot: root[:],
		}
	}
	require.NoError(t, beaconDB.SaveBlocks(ctx, blks))
	headRoot := bytesutil.ToBytes32(blkContainers[len(blks)-1].BlockRoot)
	summary := &ethpbalpha.StateSummary{
		Root: headRoot[:],
		Slot: blkContainers[len(blks)-1].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block.Block.Slot,
	}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, summary))
	require.NoError(t, beaconDB.SaveHeadBlockRoot(ctx, headRoot))
	return genBlk, blkContainers
}

func fillDBTestBlocksAltair(ctx context.Context, t *testing.T, beaconDB db.Database) (*ethpbalpha.SignedBeaconBlockAltair, []*ethpbalpha.BeaconBlockContainer) {
	parentRoot := [32]byte{1, 2, 3}
	genBlk := util.NewBeaconBlockAltair()
	genBlk.Block.ParentRoot = parentRoot[:]
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, beaconDB, genBlk)
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, root))

	count := types.Slot(100)
	blks := make([]interfaces.SignedBeaconBlock, count)
	blkContainers := make([]*ethpbalpha.BeaconBlockContainer, count)
	for i := types.Slot(0); i < count; i++ {
		b := util.NewBeaconBlockAltair()
		b.Block.Slot = i
		b.Block.ParentRoot = bytesutil.PadTo([]byte{uint8(i)}, 32)
		att1 := util.NewAttestation()
		att1.Data.Slot = i
		att1.Data.CommitteeIndex = types.CommitteeIndex(i)
		att2 := util.NewAttestation()
		att2.Data.Slot = i
		att2.Data.CommitteeIndex = types.CommitteeIndex(i + 1)
		b.Block.Body.Attestations = []*ethpbalpha.Attestation{att1, att2}
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		signedB, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		blks[i] = signedB
		blkContainers[i] = &ethpbalpha.BeaconBlockContainer{
			Block: &ethpbalpha.BeaconBlockContainer_AltairBlock{AltairBlock: b}, BlockRoot: root[:]}
	}
	require.NoError(t, beaconDB.SaveBlocks(ctx, blks))
	headRoot := bytesutil.ToBytes32(blkContainers[len(blks)-1].BlockRoot)
	summary := &ethpbalpha.StateSummary{
		Root: headRoot[:],
		Slot: blkContainers[len(blks)-1].Block.(*ethpbalpha.BeaconBlockContainer_AltairBlock).AltairBlock.Block.Slot,
	}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, summary))
	require.NoError(t, beaconDB.SaveHeadBlockRoot(ctx, headRoot))
	return genBlk, blkContainers
}

func fillDBTestBlocksBellatrix(ctx context.Context, t *testing.T, beaconDB db.Database) (*ethpbalpha.SignedBeaconBlockBellatrix, []*ethpbalpha.BeaconBlockContainer) {
	parentRoot := [32]byte{1, 2, 3}
	genBlk := util.NewBeaconBlockBellatrix()
	genBlk.Block.ParentRoot = parentRoot[:]
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, beaconDB, genBlk)
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, root))

	count := types.Slot(100)
	blks := make([]interfaces.SignedBeaconBlock, count)
	blkContainers := make([]*ethpbalpha.BeaconBlockContainer, count)
	for i := types.Slot(0); i < count; i++ {
		b := util.NewBeaconBlockBellatrix()
		b.Block.Slot = i
		b.Block.ParentRoot = bytesutil.PadTo([]byte{uint8(i)}, 32)
		att1 := util.NewAttestation()
		att1.Data.Slot = i
		att1.Data.CommitteeIndex = types.CommitteeIndex(i)
		att2 := util.NewAttestation()
		att2.Data.Slot = i
		att2.Data.CommitteeIndex = types.CommitteeIndex(i + 1)
		b.Block.Body.Attestations = []*ethpbalpha.Attestation{att1, att2}
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		signedB, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		blks[i] = signedB
		blkContainers[i] = &ethpbalpha.BeaconBlockContainer{
			Block: &ethpbalpha.BeaconBlockContainer_BellatrixBlock{BellatrixBlock: b}, BlockRoot: root[:]}
	}
	require.NoError(t, beaconDB.SaveBlocks(ctx, blks))
	headRoot := bytesutil.ToBytes32(blkContainers[len(blks)-1].BlockRoot)
	summary := &ethpbalpha.StateSummary{
		Root: headRoot[:],
		Slot: blkContainers[len(blks)-1].Block.(*ethpbalpha.BeaconBlockContainer_BellatrixBlock).BellatrixBlock.Block.Slot,
	}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, summary))
	require.NoError(t, beaconDB.SaveHeadBlockRoot(ctx, headRoot))
	return genBlk, blkContainers
}

func TestServer_GetBlockHeader(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	genBlk, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	headBlock := blkContainers[len(blkContainers)-1]

	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 30
	b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	util.SaveBlock(t, ctx, beaconDB, b2)
	b3 := util.NewBeaconBlock()
	b3.Block.Slot = 30
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	util.SaveBlock(t, ctx, beaconDB, b3)
	wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
	require.NoError(t, err)

	mockChainService := &mock.ChainService{
		DB:                  beaconDB,
		Block:               wsb,
		Root:                headBlock.BlockRoot,
		FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
	}
	bs := &Server{
		BeaconDB:              beaconDB,
		ChainInfoFetcher:      mockChainService,
		HeadFetcher:           mockChainService,
		OptimisticModeFetcher: mockChainService,
	}

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
			name:    "root",
			blockID: blkContainers[20].BlockRoot,
			want:    blkContainers[20].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
		},
		{
			name:    "canonical",
			blockID: []byte("30"),
			want:    blkContainers[30].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
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
			want:    headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
		},
		{
			name:    "finalized",
			blockID: []byte("finalized"),
			want:    blkContainers[64].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
		},
		{
			name:    "no block",
			blockID: []byte("105"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, err := bs.GetBlockHeader(ctx, &ethpbv1.BlockRequest{
				BlockId: tt.blockID,
			})
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				require.NotEqual(t, err, nil)
				return
			}

			expectedBodyRoot, err := tt.want.Block.Body.HashTreeRoot()
			require.NoError(t, err)
			expectedHeader := &ethpbv1.BeaconBlockHeader{
				Slot:          tt.want.Block.Slot,
				ProposerIndex: tt.want.Block.ProposerIndex,
				ParentRoot:    tt.want.Block.ParentRoot,
				StateRoot:     make([]byte, 32),
				BodyRoot:      expectedBodyRoot[:],
			}
			expectedHeaderRoot, err := expectedHeader.HashTreeRoot()
			require.NoError(t, err)
			headerRoot, err := header.Data.Header.Message.HashTreeRoot()
			require.NoError(t, err)
			assert.DeepEqual(t, expectedHeaderRoot, headerRoot)

			assert.Equal(t, tt.want.Block.Slot, header.Data.Header.Message.Slot)
			assert.DeepEqual(t, tt.want.Block.StateRoot, header.Data.Header.Message.StateRoot)
			assert.DeepEqual(t, tt.want.Block.ParentRoot, header.Data.Header.Message.ParentRoot)
			expectedRoot, err := tt.want.Block.Body.HashTreeRoot()
			require.NoError(t, err)
			assert.DeepEqual(t, expectedRoot[:], header.Data.Header.Message.BodyRoot)
			assert.Equal(t, tt.want.Block.ProposerIndex, header.Data.Header.Message.ProposerIndex)
		})
	}

	t.Run("execution optimistic", func(t *testing.T) {
		wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
		require.NoError(t, err)
		mockChainService := &mock.ChainService{
			DB:                  beaconDB,
			Block:               wsb,
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
		header, err := bs.GetBlockHeader(ctx, &ethpbv1.BlockRequest{BlockId: []byte("head")})
		require.NoError(t, err)
		assert.Equal(t, true, header.ExecutionOptimistic)
	})
}

func TestServer_ListBlockHeaders(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	_, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	headBlock := blkContainers[len(blkContainers)-1]
	wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
	require.NoError(t, err)
	mockChainFetcher := &mock.ChainService{
		DB:                  beaconDB,
		Block:               wsb,
		Root:                headBlock.BlockRoot,
		FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
	}
	bs := &Server{
		BeaconDB:              beaconDB,
		ChainInfoFetcher:      mockChainFetcher,
		HeadFetcher:           mockChainFetcher,
		OptimisticModeFetcher: mockChainFetcher,
	}

	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 30
	b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	util.SaveBlock(t, ctx, beaconDB, b2)
	b3 := util.NewBeaconBlock()
	b3.Block.Slot = 30
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	util.SaveBlock(t, ctx, beaconDB, b3)
	b4 := util.NewBeaconBlock()
	b4.Block.Slot = 31
	b4.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	util.SaveBlock(t, ctx, beaconDB, b4)
	b5 := util.NewBeaconBlock()
	b5.Block.Slot = 28
	b5.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	util.SaveBlock(t, ctx, beaconDB, b5)

	tests := []struct {
		name       string
		slot       types.Slot
		parentRoot []byte
		want       []*ethpbalpha.SignedBeaconBlock
		wantErr    bool
	}{
		{
			name: "slot",
			slot: types.Slot(30),
			want: []*ethpbalpha.SignedBeaconBlock{
				blkContainers[30].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
				b2,
				b3,
			},
		},
		{
			name:       "parent root",
			parentRoot: b2.Block.ParentRoot,
			want: []*ethpbalpha.SignedBeaconBlock{
				blkContainers[1].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
				b2,
				b4,
				b5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers, err := bs.ListBlockHeaders(ctx, &ethpbv1.BlockHeadersRequest{
				Slot:       &tt.slot,
				ParentRoot: tt.parentRoot,
			})
			require.NoError(t, err)

			require.Equal(t, len(tt.want), len(headers.Data))
			for i, blk := range tt.want {
				expectedBodyRoot, err := blk.Block.Body.HashTreeRoot()
				require.NoError(t, err)
				expectedHeader := &ethpbv1.BeaconBlockHeader{
					Slot:          blk.Block.Slot,
					ProposerIndex: blk.Block.ProposerIndex,
					ParentRoot:    blk.Block.ParentRoot,
					StateRoot:     make([]byte, 32),
					BodyRoot:      expectedBodyRoot[:],
				}
				expectedHeaderRoot, err := expectedHeader.HashTreeRoot()
				require.NoError(t, err)
				headerRoot, err := headers.Data[i].Header.Message.HashTreeRoot()
				require.NoError(t, err)
				assert.DeepEqual(t, expectedHeaderRoot, headerRoot)

				assert.Equal(t, blk.Block.Slot, headers.Data[i].Header.Message.Slot)
				assert.DeepEqual(t, blk.Block.StateRoot, headers.Data[i].Header.Message.StateRoot)
				assert.DeepEqual(t, blk.Block.ParentRoot, headers.Data[i].Header.Message.ParentRoot)
				expectedRoot, err := blk.Block.Body.HashTreeRoot()
				require.NoError(t, err)
				assert.DeepEqual(t, expectedRoot[:], headers.Data[i].Header.Message.BodyRoot)
				assert.Equal(t, blk.Block.ProposerIndex, headers.Data[i].Header.Message.ProposerIndex)
			}
		})
	}

	t.Run("execution optimistic", func(t *testing.T) {
		wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
		require.NoError(t, err)
		mockChainFetcher := &mock.ChainService{
			DB:                  beaconDB,
			Block:               wsb,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			Optimistic:          true,
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainFetcher,
			HeadFetcher:           mockChainFetcher,
			OptimisticModeFetcher: mockChainFetcher,
		}
		slot := types.Slot(30)
		headers, err := bs.ListBlockHeaders(ctx, &ethpbv1.BlockHeadersRequest{
			Slot: &slot,
		})
		require.NoError(t, err)
		assert.Equal(t, true, headers.ExecutionOptimistic)
	})
}

func TestServer_SubmitBlock_OK(t *testing.T) {
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
		v1Block, err := migration.V1Alpha1ToV1SignedBlock(req)
		require.NoError(t, err)
		util.SaveBlock(t, ctx, beaconDB, req)
		blockReq := &ethpbv2.SignedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBeaconBlockContainer_Phase0Block{Phase0Block: v1Block.Block},
			Signature: v1Block.Signature,
		}
		_, err = beaconChainServer.SubmitBlock(context.Background(), blockReq)
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
		req.Block.Slot = 5
		req.Block.ParentRoot = bsRoot[:]
		v2Block, err := migration.V1Alpha1BeaconBlockAltairToV2(req.Block)
		require.NoError(t, err)
		util.SaveBlock(t, ctx, beaconDB, req)
		blockReq := &ethpbv2.SignedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBeaconBlockContainer_AltairBlock{AltairBlock: v2Block},
			Signature: req.Signature,
		}
		_, err = beaconChainServer.SubmitBlock(context.Background(), blockReq)
		assert.NoError(t, err, "Could not propose block correctly")
	})

	t.Run("Bellatrix", func(t *testing.T) {
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
		beaconChainServer := &Server{
			BeaconDB:         beaconDB,
			BlockReceiver:    c,
			ChainInfoFetcher: c,
			BlockNotifier:    c.BlockNotifier(),
			Broadcaster:      mockp2p.NewTestP2P(t),
			HeadFetcher:      c,
		}
		req := util.NewBeaconBlockBellatrix()
		req.Block.Slot = 5
		req.Block.ParentRoot = bsRoot[:]
		v2Block, err := migration.V1Alpha1BeaconBlockBellatrixToV2(req.Block)
		require.NoError(t, err)
		util.SaveBlock(t, ctx, beaconDB, req)
		blockReq := &ethpbv2.SignedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBeaconBlockContainer_BellatrixBlock{BellatrixBlock: v2Block},
			Signature: req.Signature,
		}
		_, err = beaconChainServer.SubmitBlock(context.Background(), blockReq)
		assert.NoError(t, err, "Could not propose block correctly")
	})
}

func TestServer_SubmitBlockSSZ_OK(t *testing.T) {
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
		_, err = beaconChainServer.SubmitBlockSSZ(sszCtx, blockReq)
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
		_, err = beaconChainServer.SubmitBlockSSZ(sszCtx, blockReq)
		assert.NoError(t, err, "Could not propose block correctly")
	})

	t.Run("Bellatrix", func(t *testing.T) {
		// INFO: This code block can be removed once Bellatrix
		// fork epoch is set to a value other than math.MaxUint64
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
		beaconChainServer := &Server{
			BeaconDB:         beaconDB,
			BlockReceiver:    c,
			ChainInfoFetcher: c,
			BlockNotifier:    c.BlockNotifier(),
			Broadcaster:      mockp2p.NewTestP2P(t),
			HeadFetcher:      c,
		}
		req := util.NewBeaconBlockBellatrix()
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
		_, err = beaconChainServer.SubmitBlockSSZ(sszCtx, blockReq)
		assert.NoError(t, err, "Could not propose block correctly")
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
		beaconChainServer := &Server{
			BeaconDB:         beaconDB,
			BlockReceiver:    c,
			ChainInfoFetcher: c,
			BlockNotifier:    c.BlockNotifier(),
			Broadcaster:      mockp2p.NewTestP2P(t),
			HeadFetcher:      c,
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

	t.Run("Bellatrix - OK", func(t *testing.T) {
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
		beaconChainServer := &Server{
			BeaconDB:         beaconDB,
			BlockReceiver:    c,
			ChainInfoFetcher: c,
			BlockNotifier:    c.BlockNotifier(),
			Broadcaster:      mockp2p.NewTestP2P(t),
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
}

func TestServer_GetBlock(t *testing.T) {
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

	bs := &Server{
		BeaconDB: beaconDB,
		ChainInfoFetcher: &mock.ChainService{
			DB:                  beaconDB,
			Block:               wsb,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			CanonicalRoots:      canonicalRoots,
		},
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
			blk, err := bs.GetBlock(ctx, &ethpbv1.BlockRequest{
				BlockId: tt.blockID,
			})
			if tt.wantErr {
				require.NotEqual(t, err, nil, "no error has been returned")
				return
			}
			require.NoError(t, err)

			v1Block, err := migration.V1Alpha1ToV1SignedBlock(tt.want)
			require.NoError(t, err)

			if !reflect.DeepEqual(blk.Data.Message, v1Block.Block) {
				t.Error("Expected blocks to equal")
			}
		})
	}
}

func TestServer_GetBlockV2(t *testing.T) {
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
				blk, err := bs.GetBlockV2(ctx, &ethpbv2.BlockRequestV2{
					BlockId: tt.blockID,
				})
				if tt.wantErr {
					require.NotEqual(t, err, nil)
					return
				}
				require.NoError(t, err)

				v1Block, err := migration.V1Alpha1ToV1SignedBlock(tt.want)
				require.NoError(t, err)

				phase0Block, ok := blk.Data.Message.(*ethpbv2.SignedBeaconBlockContainer_Phase0Block)
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
				blk, err := bs.GetBlockV2(ctx, &ethpbv2.BlockRequestV2{
					BlockId: tt.blockID,
				})
				if tt.wantErr {
					require.NotEqual(t, err, nil)
					return
				}
				require.NoError(t, err)

				v2Block, err := migration.V1Alpha1BeaconBlockAltairToV2(tt.want.Block)
				require.NoError(t, err)

				altairBlock, ok := blk.Data.Message.(*ethpbv2.SignedBeaconBlockContainer_AltairBlock)
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

		genBlk, blkContainers := fillDBTestBlocksBellatrix(ctx, t, beaconDB)
		canonicalRoots := make(map[[32]byte]bool)

		for _, bContr := range blkContainers {
			canonicalRoots[bytesutil.ToBytes32(bContr.BlockRoot)] = true
		}
		headBlock := blkContainers[len(blkContainers)-1]
		nextSlot := headBlock.GetBellatrixBlock().Block.Slot + 1

		b2 := util.NewBeaconBlockBellatrix()
		b2.Block.Slot = 30
		b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
		util.SaveBlock(t, ctx, beaconDB, b2)
		b3 := util.NewBeaconBlockBellatrix()
		b3.Block.Slot = 30
		b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
		util.SaveBlock(t, ctx, beaconDB, b3)
		b4 := util.NewBeaconBlockBellatrix()
		b4.Block.Slot = nextSlot
		b4.Block.ParentRoot = bytesutil.PadTo([]byte{8}, 32)
		util.SaveBlock(t, ctx, beaconDB, b4)

		chainBlk, err := blocks.NewSignedBeaconBlock(headBlock.GetBellatrixBlock())
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
			want    *ethpbalpha.SignedBeaconBlockBellatrix
			wantErr bool
		}{
			{
				name:    "slot",
				blockID: []byte("30"),
				want:    blkContainers[30].GetBellatrixBlock(),
			},
			{
				name:    "bad formatting",
				blockID: []byte("3bad0"),
				wantErr: true,
			},
			{
				name:    "canonical",
				blockID: []byte("30"),
				want:    blkContainers[30].GetBellatrixBlock(),
			},
			{
				name:    "non canonical",
				blockID: []byte(fmt.Sprintf("%d", nextSlot)),
				wantErr: true,
			},
			{
				name:    "head",
				blockID: []byte("head"),
				want:    headBlock.GetBellatrixBlock(),
			},
			{
				name:    "finalized",
				blockID: []byte("finalized"),
				want:    blkContainers[64].GetBellatrixBlock(),
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
				want:    blkContainers[20].GetBellatrixBlock(),
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
				blk, err := bs.GetBlockV2(ctx, &ethpbv2.BlockRequestV2{
					BlockId: tt.blockID,
				})
				if tt.wantErr {
					require.NotEqual(t, err, nil)
					return
				}
				require.NoError(t, err)

				v2Block, err := migration.V1Alpha1BeaconBlockBellatrixToV2(tt.want.Block)
				require.NoError(t, err)

				b, ok := blk.Data.Message.(*ethpbv2.SignedBeaconBlockContainer_BellatrixBlock)
				require.Equal(t, true, ok)
				if !reflect.DeepEqual(b.BellatrixBlock, v2Block) {
					t.Error("Expected blocks to equal")
				}
				assert.Equal(t, ethpbv2.Version_BELLATRIX, blk.Version)
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

		blk, err := bs.GetBlockV2(ctx, &ethpbv2.BlockRequestV2{
			BlockId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, true, blk.ExecutionOptimistic)
	})
}

func TestServer_GetBlockSSZ(t *testing.T) {
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

	blocks, err := beaconDB.BlocksBySlot(ctx, 30)
	require.Equal(t, true, len(blocks) > 0)
	require.NoError(t, err)
	sszBlock, err := blocks[0].MarshalSSZ()
	require.NoError(t, err)

	resp, err := bs.GetBlockSSZ(ctx, &ethpbv1.BlockRequest{BlockId: []byte("30")})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.DeepEqual(t, sszBlock, resp.Data)
}

func TestServer_GetBlockSSZV2(t *testing.T) {
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

		blocks, err := beaconDB.BlocksBySlot(ctx, 30)
		require.Equal(t, true, len(blocks) > 0)
		require.NoError(t, err)
		sszBlock, err := blocks[0].MarshalSSZ()
		require.NoError(t, err)

		resp, err := bs.GetBlockSSZV2(ctx, &ethpbv2.BlockRequestV2{BlockId: []byte("30")})
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

		blocks, err := beaconDB.BlocksBySlot(ctx, 30)
		require.Equal(t, true, len(blocks) > 0)
		require.NoError(t, err)
		sszBlock, err := blocks[0].MarshalSSZ()
		require.NoError(t, err)

		resp, err := bs.GetBlockSSZV2(ctx, &ethpbv2.BlockRequestV2{BlockId: []byte("30")})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, sszBlock, resp.Data)
		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
	})

	t.Run("Bellatrix", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		_, blkContainers := fillDBTestBlocksBellatrix(ctx, t, beaconDB)
		headBlock := blkContainers[len(blkContainers)-1]

		b2 := util.NewBeaconBlockBellatrix()
		b2.Block.Slot = 30
		b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
		util.SaveBlock(t, ctx, beaconDB, b2)

		chainBlk, err := blocks.NewSignedBeaconBlock(headBlock.GetBellatrixBlock())
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

		blocks, err := beaconDB.BlocksBySlot(ctx, 30)
		require.Equal(t, true, len(blocks) > 0)
		require.NoError(t, err)
		sszBlock, err := blocks[0].MarshalSSZ()
		require.NoError(t, err)

		resp, err := bs.GetBlockSSZV2(ctx, &ethpbv2.BlockRequestV2{BlockId: []byte("30")})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, sszBlock, resp.Data)
		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)
	})
}

func TestServer_GetBlockRoot(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	genBlk, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	headBlock := blkContainers[len(blkContainers)-1]
	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 30
	b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	util.SaveBlock(t, ctx, beaconDB, b2)
	b3 := util.NewBeaconBlock()
	b3.Block.Slot = 30
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	util.SaveBlock(t, ctx, beaconDB, b3)

	wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
	require.NoError(t, err)

	mockChainFetcher := &mock.ChainService{
		DB:                  beaconDB,
		Block:               wsb,
		Root:                headBlock.BlockRoot,
		FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
	}
	bs := &Server{
		BeaconDB:              beaconDB,
		ChainInfoFetcher:      mockChainFetcher,
		HeadFetcher:           mockChainFetcher,
		OptimisticModeFetcher: mockChainFetcher,
	}

	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)

	tests := []struct {
		name    string
		blockID []byte
		want    []byte
		wantErr bool
	}{
		{
			name:    "bad formatting",
			blockID: []byte("3bad0"),
			wantErr: true,
		},
		{
			name:    "canonical slot",
			blockID: []byte("30"),
			want:    blkContainers[30].BlockRoot,
		},
		{
			name:    "head",
			blockID: []byte("head"),
			want:    headBlock.BlockRoot,
		},
		{
			name:    "finalized",
			blockID: []byte("finalized"),
			want:    blkContainers[64].BlockRoot,
		},
		{
			name:    "genesis",
			blockID: []byte("genesis"),
			want:    root[:],
		},
		{
			name:    "genesis root",
			blockID: root[:],
			want:    root[:],
		},
		{
			name:    "root",
			blockID: blkContainers[20].BlockRoot,
			want:    blkContainers[20].BlockRoot,
		},
		{
			name:    "non-existent root",
			blockID: bytesutil.PadTo([]byte("hi there"), 32),
			wantErr: true,
		},
		{
			name:    "slot",
			blockID: []byte("40"),
			want:    blkContainers[40].BlockRoot,
		},
		{
			name:    "no block",
			blockID: []byte("105"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockRootResp, err := bs.GetBlockRoot(ctx, &ethpbv1.BlockRequest{
				BlockId: tt.blockID,
			})
			if tt.wantErr {
				require.NotEqual(t, err, nil)
				return
			}
			require.NoError(t, err)
			assert.DeepEqual(t, tt.want, blockRootResp.Data.Root)
		})
	}

	t.Run("execution optimistic", func(t *testing.T) {
		wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
		require.NoError(t, err)
		mockChainFetcher := &mock.ChainService{
			DB:                  beaconDB,
			Block:               wsb,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			Optimistic:          true,
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainFetcher,
			HeadFetcher:           mockChainFetcher,
			OptimisticModeFetcher: mockChainFetcher,
		}
		blockRootResp, err := bs.GetBlockRoot(ctx, &ethpbv1.BlockRequest{
			BlockId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, true, blockRootResp.ExecutionOptimistic)
	})
}

func TestServer_ListBlockAttestations(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		_, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
		headBlock := blkContainers[len(blkContainers)-1]
		wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
		require.NoError(t, err)
		mockChainService := &mock.ChainService{
			DB:                  beaconDB,
			Block:               wsb,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainService,
			HeadFetcher:           mockChainService,
			OptimisticModeFetcher: mockChainService,
		}

		genBlk, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
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
				name:    "slot",
				blockID: []byte("40"),
				want:    blkContainers[40].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
			},
			{
				name:    "no block",
				blockID: []byte("105"),
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				blk, err := bs.ListBlockAttestations(ctx, &ethpbv1.BlockRequest{
					BlockId: tt.blockID,
				})
				if tt.wantErr {
					require.NotEqual(t, err, nil)
					return
				}
				require.NoError(t, err)

				v1Block, err := migration.V1Alpha1ToV1SignedBlock(tt.want)
				require.NoError(t, err)
				blkAtts := blk.Data
				if len(blkAtts) == 0 {
					blkAtts = nil
				}
				if !reflect.DeepEqual(blkAtts, v1Block.Block.Body.Attestations) {
					t.Error("Expected attestations to equal")
				}
			})
		}
	})

	t.Run("Altair", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		_, blkContainers := fillDBTestBlocksAltair(ctx, t, beaconDB)
		headBlock := blkContainers[len(blkContainers)-1]
		blk, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_AltairBlock).AltairBlock)
		require.NoError(t, err)
		mockChainService := &mock.ChainService{
			DB:                  beaconDB,
			Block:               blk,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainService,
			HeadFetcher:           mockChainService,
			OptimisticModeFetcher: mockChainService,
		}

		genBlk, blkContainers := fillDBTestBlocksAltair(ctx, t, beaconDB)
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
				want:    blkContainers[30].Block.(*ethpbalpha.BeaconBlockContainer_AltairBlock).AltairBlock,
			},
			{
				name:    "bad formatting",
				blockID: []byte("3bad0"),
				wantErr: true,
			},
			{
				name:    "head",
				blockID: []byte("head"),
				want:    headBlock.Block.(*ethpbalpha.BeaconBlockContainer_AltairBlock).AltairBlock,
			},
			{
				name:    "finalized",
				blockID: []byte("finalized"),
				want:    blkContainers[64].Block.(*ethpbalpha.BeaconBlockContainer_AltairBlock).AltairBlock,
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
				want:    blkContainers[20].Block.(*ethpbalpha.BeaconBlockContainer_AltairBlock).AltairBlock,
			},
			{
				name:    "non-existent root",
				blockID: bytesutil.PadTo([]byte("hi there"), 32),
				wantErr: true,
			},
			{
				name:    "slot",
				blockID: []byte("40"),
				want:    blkContainers[40].Block.(*ethpbalpha.BeaconBlockContainer_AltairBlock).AltairBlock,
			},
			{
				name:    "no block",
				blockID: []byte("105"),
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				blk, err := bs.ListBlockAttestations(ctx, &ethpbv1.BlockRequest{
					BlockId: tt.blockID,
				})
				if tt.wantErr {
					require.NotEqual(t, err, nil)
					return
				}
				require.NoError(t, err)

				v1Block, err := migration.V1Alpha1BeaconBlockAltairToV2(tt.want.Block)
				require.NoError(t, err)

				blkAtts := blk.Data
				if len(blkAtts) == 0 {
					blkAtts = nil
				}
				if !reflect.DeepEqual(blkAtts, v1Block.Body.Attestations) {
					t.Error("Expected attestations to equal")
				}
			})
		}
	})

	t.Run("Bellatrix", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		_, blkContainers := fillDBTestBlocksBellatrix(ctx, t, beaconDB)
		headBlock := blkContainers[len(blkContainers)-1]
		blk, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_BellatrixBlock).BellatrixBlock)
		require.NoError(t, err)
		mockChainService := &mock.ChainService{
			DB:                  beaconDB,
			Block:               blk,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainService,
			HeadFetcher:           mockChainService,
			OptimisticModeFetcher: mockChainService,
		}

		genBlk, blkContainers := fillDBTestBlocksBellatrix(ctx, t, beaconDB)
		root, err := genBlk.Block.HashTreeRoot()
		require.NoError(t, err)

		tests := []struct {
			name    string
			blockID []byte
			want    *ethpbalpha.SignedBeaconBlockBellatrix
			wantErr bool
		}{
			{
				name:    "slot",
				blockID: []byte("30"),
				want:    blkContainers[30].Block.(*ethpbalpha.BeaconBlockContainer_BellatrixBlock).BellatrixBlock,
			},
			{
				name:    "bad formatting",
				blockID: []byte("3bad0"),
				wantErr: true,
			},
			{
				name:    "head",
				blockID: []byte("head"),
				want:    headBlock.Block.(*ethpbalpha.BeaconBlockContainer_BellatrixBlock).BellatrixBlock,
			},
			{
				name:    "finalized",
				blockID: []byte("finalized"),
				want:    blkContainers[64].Block.(*ethpbalpha.BeaconBlockContainer_BellatrixBlock).BellatrixBlock,
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
				want:    blkContainers[20].Block.(*ethpbalpha.BeaconBlockContainer_BellatrixBlock).BellatrixBlock,
			},
			{
				name:    "non-existent root",
				blockID: bytesutil.PadTo([]byte("hi there"), 32),
				wantErr: true,
			},
			{
				name:    "slot",
				blockID: []byte("40"),
				want:    blkContainers[40].Block.(*ethpbalpha.BeaconBlockContainer_BellatrixBlock).BellatrixBlock,
			},
			{
				name:    "no block",
				blockID: []byte("105"),
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				blk, err := bs.ListBlockAttestations(ctx, &ethpbv1.BlockRequest{
					BlockId: tt.blockID,
				})
				if tt.wantErr {
					require.NotEqual(t, err, nil)
					return
				}
				require.NoError(t, err)

				v1Block, err := migration.V1Alpha1BeaconBlockBellatrixToV2(tt.want.Block)
				require.NoError(t, err)

				blkAtts := blk.Data
				if len(blkAtts) == 0 {
					blkAtts = nil
				}
				if !reflect.DeepEqual(blkAtts, v1Block.Body.Attestations) {
					t.Error("Expected attestations to equal")
				}
			})
		}
	})

	t.Run("execution optimistic", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		_, blkContainers := fillDBTestBlocksBellatrix(ctx, t, beaconDB)
		headBlock := blkContainers[len(blkContainers)-1]
		blk, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_BellatrixBlock).BellatrixBlock)
		require.NoError(t, err)
		mockChainService := &mock.ChainService{
			DB:                  beaconDB,
			Block:               blk,
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
		resp, err := bs.ListBlockAttestations(ctx, &ethpbv1.BlockRequest{
			BlockId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
}
