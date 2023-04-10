package beacon

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	dbTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	mockp2p "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
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

	count := primitives.Slot(100)
	blks := make([]interfaces.ReadOnlySignedBeaconBlock, count)
	blkContainers := make([]*ethpbalpha.BeaconBlockContainer, count)
	for i := primitives.Slot(0); i < count; i++ {
		b := util.NewBeaconBlock()
		b.Block.Slot = i
		b.Block.ParentRoot = bytesutil.PadTo([]byte{uint8(i)}, 32)
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

func TestServer_GetBlockHeader(t *testing.T) {
	ctx := context.Background()
	b := util.NewBeaconBlock()
	b.Block.Slot = 123
	b.Block.ProposerIndex = 123
	b.Block.StateRoot = bytesutil.PadTo([]byte("stateroot"), 32)
	sb, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}

	t.Run("get header", func(t *testing.T) {
		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{},
		}
		bs := &Server{
			ChainInfoFetcher:      mockChainService,
			OptimisticModeFetcher: mockChainService,
			FinalizationFetcher:   mockChainService,
			Blocker:               mockBlockFetcher,
		}

		header, err := bs.GetBlockHeader(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)

		expectedBodyRoot, err := sb.Block().Body().HashTreeRoot()
		require.NoError(t, err)
		expectedParentRoot := sb.Block().ParentRoot()
		expectedHeader := &ethpbv1.BeaconBlockHeader{
			Slot:          sb.Block().Slot(),
			ProposerIndex: sb.Block().ProposerIndex(),
			ParentRoot:    expectedParentRoot[:],
			StateRoot:     bytesutil.PadTo([]byte("stateroot"), 32),
			BodyRoot:      expectedBodyRoot[:],
		}
		expectedHeaderRoot, err := expectedHeader.HashTreeRoot()
		require.NoError(t, err)
		headerRoot, err := header.Data.Header.Message.HashTreeRoot()
		require.NoError(t, err)
		assert.DeepEqual(t, expectedHeaderRoot, headerRoot)
		assert.Equal(t, sb.Block().Slot(), header.Data.Header.Message.Slot)
		expectedStateRoot := sb.Block().StateRoot()
		assert.DeepEqual(t, expectedStateRoot[:], header.Data.Header.Message.StateRoot)
		assert.DeepEqual(t, expectedParentRoot[:], header.Data.Header.Message.ParentRoot)
		assert.DeepEqual(t, expectedBodyRoot[:], header.Data.Header.Message.BodyRoot)
		assert.Equal(t, sb.Block().ProposerIndex(), header.Data.Header.Message.ProposerIndex)
	})

	t.Run("execution optimistic", func(t *testing.T) {
		r, err := sb.Block().HashTreeRoot()
		require.NoError(t, err)
		mockChainService := &mock.ChainService{
			OptimisticRoots: map[[32]byte]bool{r: true},
			FinalizedRoots:  map[[32]byte]bool{},
		}
		bs := &Server{
			ChainInfoFetcher:      mockChainService,
			OptimisticModeFetcher: mockChainService,
			FinalizationFetcher:   mockChainService,
			Blocker:               mockBlockFetcher,
		}
		header, err := bs.GetBlockHeader(ctx, &ethpbv1.BlockRequest{BlockId: []byte("head")})
		require.NoError(t, err)
		assert.Equal(t, true, header.ExecutionOptimistic)
	})

	t.Run("finalized", func(t *testing.T) {
		r, err := sb.Block().HashTreeRoot()
		require.NoError(t, err)

		t.Run("true", func(t *testing.T) {
			mockChainService := &mock.ChainService{FinalizedRoots: map[[32]byte]bool{r: true}}
			bs := &Server{
				ChainInfoFetcher:      mockChainService,
				OptimisticModeFetcher: mockChainService,
				FinalizationFetcher:   mockChainService,
				Blocker:               mockBlockFetcher,
			}

			header, err := bs.GetBlockHeader(ctx, &ethpbv1.BlockRequest{BlockId: r[:]})
			require.NoError(t, err)
			assert.Equal(t, true, header.Finalized)
		})
		t.Run("false", func(t *testing.T) {
			mockChainService := &mock.ChainService{FinalizedRoots: map[[32]byte]bool{r: false}}
			bs := &Server{
				ChainInfoFetcher:      mockChainService,
				OptimisticModeFetcher: mockChainService,
				FinalizationFetcher:   mockChainService,
				Blocker:               mockBlockFetcher,
			}

			header, err := bs.GetBlockHeader(ctx, &ethpbv1.BlockRequest{BlockId: r[:]})
			require.NoError(t, err)
			assert.Equal(t, false, header.Finalized)
		})
	})
}

func TestServer_ListBlockHeaders(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	_, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	headBlock := blkContainers[len(blkContainers)-1]

	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 30
	b1.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	util.SaveBlock(t, ctx, beaconDB, b1)
	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 30
	b2.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	util.SaveBlock(t, ctx, beaconDB, b2)
	b3 := util.NewBeaconBlock()
	b3.Block.Slot = 31
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	util.SaveBlock(t, ctx, beaconDB, b3)
	b4 := util.NewBeaconBlock()
	b4.Block.Slot = 28
	b4.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	util.SaveBlock(t, ctx, beaconDB, b4)

	t.Run("list headers", func(t *testing.T) {
		wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
		require.NoError(t, err)
		mockChainFetcher := &mock.ChainService{
			DB:                  beaconDB,
			Block:               wsb,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			FinalizedRoots:      map[[32]byte]bool{},
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainFetcher,
			OptimisticModeFetcher: mockChainFetcher,
			FinalizationFetcher:   mockChainFetcher,
		}

		tests := []struct {
			name       string
			slot       primitives.Slot
			parentRoot []byte
			want       []*ethpbalpha.SignedBeaconBlock
			wantErr    bool
		}{
			{
				name: "slot",
				slot: primitives.Slot(30),
				want: []*ethpbalpha.SignedBeaconBlock{
					blkContainers[30].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
					b1,
					b2,
				},
			},
			{
				name:       "parent root",
				parentRoot: b1.Block.ParentRoot,
				want: []*ethpbalpha.SignedBeaconBlock{
					blkContainers[1].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
					b1,
					b3,
					b4,
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
	})

	t.Run("execution optimistic", func(t *testing.T) {
		wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
		require.NoError(t, err)
		mockChainFetcher := &mock.ChainService{
			DB:                  beaconDB,
			Block:               wsb,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			Optimistic:          true,
			FinalizedRoots:      map[[32]byte]bool{},
			OptimisticRoots: map[[32]byte]bool{
				bytesutil.ToBytes32(blkContainers[30].BlockRoot): true,
			},
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainFetcher,
			OptimisticModeFetcher: mockChainFetcher,
			FinalizationFetcher:   mockChainFetcher,
		}
		slot := primitives.Slot(30)
		headers, err := bs.ListBlockHeaders(ctx, &ethpbv1.BlockHeadersRequest{
			Slot: &slot,
		})
		require.NoError(t, err)
		assert.Equal(t, true, headers.ExecutionOptimistic)
	})

	t.Run("finalized", func(t *testing.T) {
		wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
		require.NoError(t, err)
		child1 := util.NewBeaconBlock()
		child1.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		child1.Block.Slot = 999
		util.SaveBlock(t, ctx, beaconDB, child1)
		child2 := util.NewBeaconBlock()
		child2.Block.ParentRoot = bytesutil.PadTo([]byte("parent"), 32)
		child2.Block.Slot = 1000
		util.SaveBlock(t, ctx, beaconDB, child2)
		child1Root, err := child1.Block.HashTreeRoot()
		require.NoError(t, err)
		child2Root, err := child2.Block.HashTreeRoot()
		require.NoError(t, err)
		mockChainFetcher := &mock.ChainService{
			DB:                  beaconDB,
			Block:               wsb,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			FinalizedRoots:      map[[32]byte]bool{child1Root: true, child2Root: false},
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainFetcher,
			OptimisticModeFetcher: mockChainFetcher,
			FinalizationFetcher:   mockChainFetcher,
		}

		t.Run("true", func(t *testing.T) {
			slot := primitives.Slot(999)
			headers, err := bs.ListBlockHeaders(ctx, &ethpbv1.BlockHeadersRequest{
				Slot: &slot,
			})
			require.NoError(t, err)
			assert.Equal(t, true, headers.Finalized)
		})
		t.Run("false", func(t *testing.T) {
			slot := primitives.Slot(1000)
			headers, err := bs.ListBlockHeaders(ctx, &ethpbv1.BlockHeadersRequest{
				Slot: &slot,
			})
			require.NoError(t, err)
			assert.Equal(t, false, headers.Finalized)
		})
		t.Run("false when at least one not finalized", func(t *testing.T) {
			headers, err := bs.ListBlockHeaders(ctx, &ethpbv1.BlockHeadersRequest{
				ParentRoot: []byte("parent"),
			})
			require.NoError(t, err)
			assert.Equal(t, false, headers.Finalized)
		})
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

	t.Run("Capella", func(t *testing.T) {
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
		beaconChainServer := &Server{
			BeaconDB:         beaconDB,
			BlockReceiver:    c,
			ChainInfoFetcher: c,
			BlockNotifier:    c.BlockNotifier(),
			Broadcaster:      mockp2p.NewTestP2P(t),
			HeadFetcher:      c,
		}
		req := util.NewBeaconBlockCapella()
		req.Block.Slot = 5
		req.Block.ParentRoot = bsRoot[:]
		v2Block, err := migration.V1Alpha1BeaconBlockCapellaToV2(req.Block)
		require.NoError(t, err)
		util.SaveBlock(t, ctx, beaconDB, req)
		blockReq := &ethpbv2.SignedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBeaconBlockContainer_CapellaBlock{CapellaBlock: v2Block},
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

	t.Run("Capella", func(t *testing.T) {
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
		beaconChainServer := &Server{
			BeaconDB:         beaconDB,
			BlockReceiver:    c,
			ChainInfoFetcher: c,
			BlockNotifier:    c.BlockNotifier(),
			Broadcaster:      mockp2p.NewTestP2P(t),
			HeadFetcher:      c,
		}
		req := util.NewBeaconBlockCapella()
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
		_, err = beaconChainServer.SubmitBlockSSZ(sszCtx, blockReq)
		assert.NoError(t, err, "Could not propose block correctly")
	})
}

func TestServer_GetBlock(t *testing.T) {
	ctx := context.Background()
	b := util.NewBeaconBlock()
	b.Block.Slot = 123
	sb, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	bs := &Server{
		Blocker: &testutil.MockBlocker{BlockToReturn: sb},
	}

	blk, err := bs.GetBlock(ctx, &ethpbv1.BlockRequest{})
	require.NoError(t, err)
	v1Block, err := migration.V1Alpha1ToV1SignedBlock(b)
	require.NoError(t, err)
	assert.DeepEqual(t, v1Block.Block, blk.Data.Message)
}

func TestServer_GetBlockV2(t *testing.T) {
	ctx := context.Background()

	t.Run("Phase 0", func(t *testing.T) {
		b := util.NewBeaconBlock()
		b.Block.Slot = 123
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}
		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{},
		}
		bs := &Server{
			FinalizationFetcher: mockChainService,
			Blocker:             mockBlockFetcher,
		}

		blk, err := bs.GetBlockV2(ctx, &ethpbv2.BlockRequestV2{})
		require.NoError(t, err)

		v1Block, err := migration.V1Alpha1ToV1SignedBlock(b)
		require.NoError(t, err)
		phase0Block, ok := blk.Data.Message.(*ethpbv2.SignedBeaconBlockContainer_Phase0Block)
		require.Equal(t, true, ok)
		assert.DeepEqual(t, v1Block.Block, phase0Block.Phase0Block)
		assert.Equal(t, ethpbv2.Version_PHASE0, blk.Version)
	})
	t.Run("Altair", func(t *testing.T) {
		b := util.NewBeaconBlockAltair()
		b.Block.Slot = 123
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}
		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{},
		}
		bs := &Server{
			FinalizationFetcher: mockChainService,
			Blocker:             mockBlockFetcher,
		}

		blk, err := bs.GetBlockV2(ctx, &ethpbv2.BlockRequestV2{})
		require.NoError(t, err)

		v1Block, err := migration.V1Alpha1BeaconBlockAltairToV2(b.Block)
		require.NoError(t, err)
		altairBlock, ok := blk.Data.Message.(*ethpbv2.SignedBeaconBlockContainer_AltairBlock)
		require.Equal(t, true, ok)
		assert.DeepEqual(t, v1Block, altairBlock.AltairBlock)
		assert.Equal(t, ethpbv2.Version_ALTAIR, blk.Version)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		b := util.NewBeaconBlockBellatrix()
		b.Block.Slot = 123
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}
		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{},
		}
		bs := &Server{
			OptimisticModeFetcher: mockChainService,
			FinalizationFetcher:   mockChainService,
			Blocker:               mockBlockFetcher,
		}

		blk, err := bs.GetBlockV2(ctx, &ethpbv2.BlockRequestV2{})
		require.NoError(t, err)

		v1Block, err := migration.V1Alpha1BeaconBlockBellatrixToV2(b.Block)
		require.NoError(t, err)
		bellatrixBlock, ok := blk.Data.Message.(*ethpbv2.SignedBeaconBlockContainer_BellatrixBlock)
		require.Equal(t, true, ok)
		assert.DeepEqual(t, v1Block, bellatrixBlock.BellatrixBlock)
		assert.Equal(t, ethpbv2.Version_BELLATRIX, blk.Version)
	})
	t.Run("Capella", func(t *testing.T) {
		b := util.NewBeaconBlockCapella()
		b.Block.Slot = 123
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}
		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{},
		}
		bs := &Server{
			OptimisticModeFetcher: mockChainService,
			FinalizationFetcher:   mockChainService,
			Blocker:               mockBlockFetcher,
		}

		blk, err := bs.GetBlockV2(ctx, &ethpbv2.BlockRequestV2{})
		require.NoError(t, err)

		v1Block, err := migration.V1Alpha1BeaconBlockCapellaToV2(b.Block)
		require.NoError(t, err)
		bellatrixBlock, ok := blk.Data.Message.(*ethpbv2.SignedBeaconBlockContainer_CapellaBlock)
		require.Equal(t, true, ok)
		assert.DeepEqual(t, v1Block, bellatrixBlock.CapellaBlock)
		assert.Equal(t, ethpbv2.Version_CAPELLA, blk.Version)
	})
	t.Run("execution optimistic", func(t *testing.T) {
		b := util.NewBeaconBlockBellatrix()
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		r, err := sb.Block().HashTreeRoot()
		require.NoError(t, err)
		mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}
		mockChainService := &mock.ChainService{
			OptimisticRoots: map[[32]byte]bool{r: true},
			FinalizedRoots:  map[[32]byte]bool{},
		}
		bs := &Server{
			OptimisticModeFetcher: mockChainService,
			FinalizationFetcher:   mockChainService,
			Blocker:               mockBlockFetcher,
		}

		blk, err := bs.GetBlockV2(ctx, &ethpbv2.BlockRequestV2{})
		require.NoError(t, err)
		assert.Equal(t, true, blk.ExecutionOptimistic)
	})
	t.Run("finalized", func(t *testing.T) {
		b := util.NewBeaconBlock()
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		r, err := sb.Block().HashTreeRoot()
		require.NoError(t, err)
		mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}

		t.Run("true", func(t *testing.T) {
			mockChainService := &mock.ChainService{FinalizedRoots: map[[32]byte]bool{r: true}}
			bs := &Server{
				OptimisticModeFetcher: mockChainService,
				FinalizationFetcher:   mockChainService,
				Blocker:               mockBlockFetcher,
			}

			header, err := bs.GetBlockV2(ctx, &ethpbv2.BlockRequestV2{BlockId: r[:]})
			require.NoError(t, err)
			assert.Equal(t, true, header.Finalized)
		})
		t.Run("false", func(t *testing.T) {
			mockChainService := &mock.ChainService{FinalizedRoots: map[[32]byte]bool{r: false}}
			bs := &Server{
				OptimisticModeFetcher: mockChainService,
				FinalizationFetcher:   mockChainService,
				Blocker:               mockBlockFetcher,
			}

			resp, err := bs.GetBlockV2(ctx, &ethpbv2.BlockRequestV2{BlockId: r[:]})
			require.NoError(t, err)
			assert.Equal(t, false, resp.Finalized)
		})
	})
}

func TestServer_GetBlockSSZ(t *testing.T) {
	ctx := context.Background()
	b := util.NewBeaconBlock()
	b.Block.Slot = 123
	sb, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	bs := &Server{
		Blocker: &testutil.MockBlocker{BlockToReturn: sb},
	}

	resp, err := bs.GetBlockSSZ(ctx, &ethpbv1.BlockRequest{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	sszBlock, err := b.MarshalSSZ()
	require.NoError(t, err)
	assert.DeepEqual(t, sszBlock, resp.Data)
}

func TestServer_GetBlockSSZV2(t *testing.T) {
	ctx := context.Background()

	t.Run("Phase 0", func(t *testing.T) {
		b := util.NewBeaconBlock()
		b.Block.Slot = 123
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)

		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{},
		}
		bs := &Server{
			FinalizationFetcher: mockChainService,
			Blocker:             &testutil.MockBlocker{BlockToReturn: sb},
		}

		resp, err := bs.GetBlockSSZV2(ctx, &ethpbv2.BlockRequestV2{})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		sszBlock, err := b.MarshalSSZ()
		require.NoError(t, err)
		assert.DeepEqual(t, sszBlock, resp.Data)
		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)
	})
	t.Run("Altair", func(t *testing.T) {
		b := util.NewBeaconBlockAltair()
		b.Block.Slot = 123
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)

		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{},
		}
		bs := &Server{
			FinalizationFetcher: mockChainService,
			Blocker:             &testutil.MockBlocker{BlockToReturn: sb},
		}

		resp, err := bs.GetBlockSSZV2(ctx, &ethpbv2.BlockRequestV2{})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		sszBlock, err := b.MarshalSSZ()
		require.NoError(t, err)
		assert.DeepEqual(t, sszBlock, resp.Data)
		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		b := util.NewBeaconBlockBellatrix()
		b.Block.Slot = 123
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)

		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{},
		}
		bs := &Server{
			OptimisticModeFetcher: mockChainService,
			FinalizationFetcher:   mockChainService,
			Blocker:               &testutil.MockBlocker{BlockToReturn: sb},
		}

		resp, err := bs.GetBlockSSZV2(ctx, &ethpbv2.BlockRequestV2{})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		sszBlock, err := b.MarshalSSZ()
		require.NoError(t, err)
		assert.DeepEqual(t, sszBlock, resp.Data)
		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)
	})
	t.Run("Capella", func(t *testing.T) {
		b := util.NewBeaconBlockCapella()
		b.Block.Slot = 123
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)

		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{},
		}
		bs := &Server{
			OptimisticModeFetcher: mockChainService,
			FinalizationFetcher:   mockChainService,
			Blocker:               &testutil.MockBlocker{BlockToReturn: sb},
		}

		resp, err := bs.GetBlockSSZV2(ctx, &ethpbv2.BlockRequestV2{})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		sszBlock, err := b.MarshalSSZ()
		require.NoError(t, err)
		assert.DeepEqual(t, sszBlock, resp.Data)
		assert.Equal(t, ethpbv2.Version_CAPELLA, resp.Version)
	})
	t.Run("execution optimistic", func(t *testing.T) {
		b := util.NewBeaconBlockBellatrix()
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		r, err := sb.Block().HashTreeRoot()
		require.NoError(t, err)

		mockChainService := &mock.ChainService{
			OptimisticRoots: map[[32]byte]bool{r: true},
			FinalizedRoots:  map[[32]byte]bool{},
		}
		bs := &Server{
			OptimisticModeFetcher: mockChainService,
			FinalizationFetcher:   mockChainService,
			Blocker:               &testutil.MockBlocker{BlockToReturn: sb},
		}

		resp, err := bs.GetBlockSSZV2(ctx, &ethpbv2.BlockRequestV2{})
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
	t.Run("finalized", func(t *testing.T) {
		b := util.NewBeaconBlock()
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		r, err := sb.Block().HashTreeRoot()
		require.NoError(t, err)
		mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}

		t.Run("true", func(t *testing.T) {
			mockChainService := &mock.ChainService{FinalizedRoots: map[[32]byte]bool{r: true}}
			bs := &Server{
				OptimisticModeFetcher: mockChainService,
				FinalizationFetcher:   mockChainService,
				Blocker:               mockBlockFetcher,
			}

			header, err := bs.GetBlockSSZV2(ctx, &ethpbv2.BlockRequestV2{BlockId: r[:]})
			require.NoError(t, err)
			assert.Equal(t, true, header.Finalized)
		})
		t.Run("false", func(t *testing.T) {
			mockChainService := &mock.ChainService{FinalizedRoots: map[[32]byte]bool{r: false}}
			bs := &Server{
				OptimisticModeFetcher: mockChainService,
				FinalizationFetcher:   mockChainService,
				Blocker:               mockBlockFetcher,
			}

			resp, err := bs.GetBlockSSZV2(ctx, &ethpbv2.BlockRequestV2{BlockId: r[:]})
			require.NoError(t, err)
			assert.Equal(t, false, resp.Finalized)
		})
	})
}

func TestServer_GetBlockRoot(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	genBlk, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	headBlock := blkContainers[len(blkContainers)-1]

	t.Run("get root", func(t *testing.T) {
		wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
		require.NoError(t, err)
		mockChainFetcher := &mock.ChainService{
			DB:                  beaconDB,
			Block:               wsb,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			FinalizedRoots:      map[[32]byte]bool{},
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainFetcher,
			HeadFetcher:           mockChainFetcher,
			OptimisticModeFetcher: mockChainFetcher,
			FinalizationFetcher:   mockChainFetcher,
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
	})

	t.Run("execution optimistic", func(t *testing.T) {
		wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
		require.NoError(t, err)
		mockChainFetcher := &mock.ChainService{
			DB:                  beaconDB,
			Block:               wsb,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			Optimistic:          true,
			FinalizedRoots:      map[[32]byte]bool{},
			OptimisticRoots: map[[32]byte]bool{
				bytesutil.ToBytes32(headBlock.BlockRoot): true,
			},
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainFetcher,
			HeadFetcher:           mockChainFetcher,
			OptimisticModeFetcher: mockChainFetcher,
			FinalizationFetcher:   mockChainFetcher,
		}
		blockRootResp, err := bs.GetBlockRoot(ctx, &ethpbv1.BlockRequest{
			BlockId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, true, blockRootResp.ExecutionOptimistic)
	})

	t.Run("finalized", func(t *testing.T) {
		wsb, err := blocks.NewSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block)
		require.NoError(t, err)
		mockChainFetcher := &mock.ChainService{
			DB:                  beaconDB,
			Block:               wsb,
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			Optimistic:          true,
			FinalizedRoots: map[[32]byte]bool{
				bytesutil.ToBytes32(blkContainers[32].BlockRoot): true,
				bytesutil.ToBytes32(blkContainers[64].BlockRoot): false,
			},
		}
		bs := &Server{
			BeaconDB:              beaconDB,
			ChainInfoFetcher:      mockChainFetcher,
			HeadFetcher:           mockChainFetcher,
			OptimisticModeFetcher: mockChainFetcher,
			FinalizationFetcher:   mockChainFetcher,
		}

		t.Run("true", func(t *testing.T) {
			blockRootResp, err := bs.GetBlockRoot(ctx, &ethpbv1.BlockRequest{
				BlockId: []byte("32"),
			})
			require.NoError(t, err)
			assert.Equal(t, true, blockRootResp.Finalized)
		})
		t.Run("false", func(t *testing.T) {
			blockRootResp, err := bs.GetBlockRoot(ctx, &ethpbv1.BlockRequest{
				BlockId: []byte("64"),
			})
			require.NoError(t, err)
			assert.Equal(t, false, blockRootResp.Finalized)
		})
	})
}

func TestServer_ListBlockAttestations(t *testing.T) {
	ctx := context.Background()

	t.Run("Phase 0", func(t *testing.T) {
		b := util.NewBeaconBlock()
		b.Block.Body.Attestations = []*ethpbalpha.Attestation{
			{
				AggregationBits: bitfield.Bitlist{0x00},
				Data: &ethpbalpha.AttestationData{
					Slot:            123,
					CommitteeIndex:  123,
					BeaconBlockRoot: bytesutil.PadTo([]byte("root1"), 32),
					Source: &ethpbalpha.Checkpoint{
						Epoch: 123,
						Root:  bytesutil.PadTo([]byte("root1"), 32),
					},
					Target: &ethpbalpha.Checkpoint{
						Epoch: 123,
						Root:  bytesutil.PadTo([]byte("root1"), 32),
					},
				},
				Signature: bytesutil.PadTo([]byte("sig1"), 96),
			},
			{
				AggregationBits: bitfield.Bitlist{0x01},
				Data: &ethpbalpha.AttestationData{
					Slot:            456,
					CommitteeIndex:  456,
					BeaconBlockRoot: bytesutil.PadTo([]byte("root2"), 32),
					Source: &ethpbalpha.Checkpoint{
						Epoch: 456,
						Root:  bytesutil.PadTo([]byte("root2"), 32),
					},
					Target: &ethpbalpha.Checkpoint{
						Epoch: 456,
						Root:  bytesutil.PadTo([]byte("root2"), 32),
					},
				},
				Signature: bytesutil.PadTo([]byte("sig2"), 96),
			},
		}
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}
		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{},
		}
		bs := &Server{
			OptimisticModeFetcher: mockChainService,
			FinalizationFetcher:   mockChainService,
			Blocker:               mockBlockFetcher,
		}

		resp, err := bs.ListBlockAttestations(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)

		v1Block, err := migration.V1Alpha1ToV1SignedBlock(b)
		require.NoError(t, err)
		assert.DeepEqual(t, v1Block.Block.Body.Attestations, resp.Data)
	})
	t.Run("Altair", func(t *testing.T) {
		b := util.NewBeaconBlockAltair()
		b.Block.Body.Attestations = []*ethpbalpha.Attestation{
			{
				AggregationBits: bitfield.Bitlist{0x00},
				Data: &ethpbalpha.AttestationData{
					Slot:            123,
					CommitteeIndex:  123,
					BeaconBlockRoot: bytesutil.PadTo([]byte("root1"), 32),
					Source: &ethpbalpha.Checkpoint{
						Epoch: 123,
						Root:  bytesutil.PadTo([]byte("root1"), 32),
					},
					Target: &ethpbalpha.Checkpoint{
						Epoch: 123,
						Root:  bytesutil.PadTo([]byte("root1"), 32),
					},
				},
				Signature: bytesutil.PadTo([]byte("sig1"), 96),
			},
			{
				AggregationBits: bitfield.Bitlist{0x01},
				Data: &ethpbalpha.AttestationData{
					Slot:            456,
					CommitteeIndex:  456,
					BeaconBlockRoot: bytesutil.PadTo([]byte("root2"), 32),
					Source: &ethpbalpha.Checkpoint{
						Epoch: 456,
						Root:  bytesutil.PadTo([]byte("root2"), 32),
					},
					Target: &ethpbalpha.Checkpoint{
						Epoch: 456,
						Root:  bytesutil.PadTo([]byte("root2"), 32),
					},
				},
				Signature: bytesutil.PadTo([]byte("sig2"), 96),
			},
		}
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}
		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{},
		}
		bs := &Server{
			OptimisticModeFetcher: mockChainService,
			FinalizationFetcher:   mockChainService,
			Blocker:               mockBlockFetcher,
		}

		resp, err := bs.ListBlockAttestations(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)

		v1Block, err := migration.V1Alpha1BeaconBlockAltairToV2(b.Block)
		require.NoError(t, err)
		assert.DeepEqual(t, v1Block.Body.Attestations, resp.Data)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		b := util.NewBeaconBlockBellatrix()
		b.Block.Body.Attestations = []*ethpbalpha.Attestation{
			{
				AggregationBits: bitfield.Bitlist{0x00},
				Data: &ethpbalpha.AttestationData{
					Slot:            123,
					CommitteeIndex:  123,
					BeaconBlockRoot: bytesutil.PadTo([]byte("root1"), 32),
					Source: &ethpbalpha.Checkpoint{
						Epoch: 123,
						Root:  bytesutil.PadTo([]byte("root1"), 32),
					},
					Target: &ethpbalpha.Checkpoint{
						Epoch: 123,
						Root:  bytesutil.PadTo([]byte("root1"), 32),
					},
				},
				Signature: bytesutil.PadTo([]byte("sig1"), 96),
			},
			{
				AggregationBits: bitfield.Bitlist{0x01},
				Data: &ethpbalpha.AttestationData{
					Slot:            456,
					CommitteeIndex:  456,
					BeaconBlockRoot: bytesutil.PadTo([]byte("root2"), 32),
					Source: &ethpbalpha.Checkpoint{
						Epoch: 456,
						Root:  bytesutil.PadTo([]byte("root2"), 32),
					},
					Target: &ethpbalpha.Checkpoint{
						Epoch: 456,
						Root:  bytesutil.PadTo([]byte("root2"), 32),
					},
				},
				Signature: bytesutil.PadTo([]byte("sig2"), 96),
			},
		}
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}
		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{},
		}
		bs := &Server{
			OptimisticModeFetcher: mockChainService,
			FinalizationFetcher:   mockChainService,
			Blocker:               mockBlockFetcher,
		}

		resp, err := bs.ListBlockAttestations(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)

		v1Block, err := migration.V1Alpha1BeaconBlockBellatrixToV2(b.Block)
		require.NoError(t, err)
		assert.DeepEqual(t, v1Block.Body.Attestations, resp.Data)
	})
	t.Run("Capella", func(t *testing.T) {
		b := util.NewBeaconBlockCapella()
		b.Block.Body.Attestations = []*ethpbalpha.Attestation{
			{
				AggregationBits: bitfield.Bitlist{0x00},
				Data: &ethpbalpha.AttestationData{
					Slot:            123,
					CommitteeIndex:  123,
					BeaconBlockRoot: bytesutil.PadTo([]byte("root1"), 32),
					Source: &ethpbalpha.Checkpoint{
						Epoch: 123,
						Root:  bytesutil.PadTo([]byte("root1"), 32),
					},
					Target: &ethpbalpha.Checkpoint{
						Epoch: 123,
						Root:  bytesutil.PadTo([]byte("root1"), 32),
					},
				},
				Signature: bytesutil.PadTo([]byte("sig1"), 96),
			},
			{
				AggregationBits: bitfield.Bitlist{0x01},
				Data: &ethpbalpha.AttestationData{
					Slot:            456,
					CommitteeIndex:  456,
					BeaconBlockRoot: bytesutil.PadTo([]byte("root2"), 32),
					Source: &ethpbalpha.Checkpoint{
						Epoch: 456,
						Root:  bytesutil.PadTo([]byte("root2"), 32),
					},
					Target: &ethpbalpha.Checkpoint{
						Epoch: 456,
						Root:  bytesutil.PadTo([]byte("root2"), 32),
					},
				},
				Signature: bytesutil.PadTo([]byte("sig2"), 96),
			},
		}
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}
		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{},
		}
		bs := &Server{
			OptimisticModeFetcher: mockChainService,
			FinalizationFetcher:   mockChainService,
			Blocker:               mockBlockFetcher,
		}

		resp, err := bs.ListBlockAttestations(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)

		v1Block, err := migration.V1Alpha1BeaconBlockCapellaToV2(b.Block)
		require.NoError(t, err)
		assert.DeepEqual(t, v1Block.Body.Attestations, resp.Data)
	})
	t.Run("execution optimistic", func(t *testing.T) {
		b := util.NewBeaconBlockBellatrix()
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		r, err := sb.Block().HashTreeRoot()
		require.NoError(t, err)
		mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}
		mockChainService := &mock.ChainService{
			OptimisticRoots: map[[32]byte]bool{r: true},
			FinalizedRoots:  map[[32]byte]bool{},
		}
		bs := &Server{
			OptimisticModeFetcher: mockChainService,
			FinalizationFetcher:   mockChainService,
			Blocker:               mockBlockFetcher,
		}

		resp, err := bs.ListBlockAttestations(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
	t.Run("finalized", func(t *testing.T) {
		b := util.NewBeaconBlock()
		sb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		r, err := sb.Block().HashTreeRoot()
		require.NoError(t, err)
		mockBlockFetcher := &testutil.MockBlocker{BlockToReturn: sb}

		t.Run("true", func(t *testing.T) {
			mockChainService := &mock.ChainService{FinalizedRoots: map[[32]byte]bool{r: true}}
			bs := &Server{
				OptimisticModeFetcher: mockChainService,
				FinalizationFetcher:   mockChainService,
				Blocker:               mockBlockFetcher,
			}

			resp, err := bs.ListBlockAttestations(ctx, &ethpbv1.BlockRequest{BlockId: r[:]})
			require.NoError(t, err)
			assert.Equal(t, true, resp.Finalized)
		})
		t.Run("false", func(t *testing.T) {
			mockChainService := &mock.ChainService{FinalizedRoots: map[[32]byte]bool{r: false}}
			bs := &Server{
				OptimisticModeFetcher: mockChainService,
				FinalizationFetcher:   mockChainService,
				Blocker:               mockBlockFetcher,
			}

			resp, err := bs.ListBlockAttestations(ctx, &ethpbv1.BlockRequest{BlockId: r[:]})
			require.NoError(t, err)
			assert.Equal(t, false, resp.Finalized)
		})
	})
}
