package beacon

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	builderTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/builder/testing"
	dbTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/synccommittee"
	mockp2p "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/prysm/v1alpha1/validator"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	mock2 "github.com/prysmaticlabs/prysm/v4/testing/mock"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"google.golang.org/grpc/metadata"
)

func TestServer_GetBlindedBlock(t *testing.T) {
	ctx := context.Background()

	t.Run("Phase 0", func(t *testing.T) {
		b := util.NewBeaconBlock()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)

		bs := &Server{
			FinalizationFetcher: &mock.ChainService{},
			Blocker:             &testutil.MockBlocker{BlockToReturn: blk},
		}

		expected, err := migration.V1Alpha1ToV1SignedBlock(b)
		require.NoError(t, err)
		resp, err := bs.GetBlindedBlock(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)
		phase0Block, ok := resp.Data.Message.(*ethpbv2.SignedBlindedBeaconBlockContainer_Phase0Block)
		require.Equal(t, true, ok)
		assert.DeepEqual(t, expected.Block, phase0Block.Phase0Block)
		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)
	})
	t.Run("Altair", func(t *testing.T) {
		b := util.NewBeaconBlockAltair()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)

		bs := &Server{
			FinalizationFetcher: &mock.ChainService{},
			Blocker:             &testutil.MockBlocker{BlockToReturn: blk},
		}

		expected, err := migration.V1Alpha1BeaconBlockAltairToV2(b.Block)
		require.NoError(t, err)
		resp, err := bs.GetBlindedBlock(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)
		altairBlock, ok := resp.Data.Message.(*ethpbv2.SignedBlindedBeaconBlockContainer_AltairBlock)
		require.Equal(t, true, ok)
		assert.DeepEqual(t, expected, altairBlock.AltairBlock)
		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		b := util.NewBlindedBeaconBlockBellatrix()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)

		mockChainService := &mock.ChainService{}
		bs := &Server{
			FinalizationFetcher:   mockChainService,
			Blocker:               &testutil.MockBlocker{BlockToReturn: blk},
			OptimisticModeFetcher: mockChainService,
		}

		expected, err := migration.V1Alpha1BeaconBlockBlindedBellatrixToV2Blinded(b.Block)
		require.NoError(t, err)
		resp, err := bs.GetBlindedBlock(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)
		bellatrixBlock, ok := resp.Data.Message.(*ethpbv2.SignedBlindedBeaconBlockContainer_BellatrixBlock)
		require.Equal(t, true, ok)
		assert.DeepEqual(t, expected, bellatrixBlock.BellatrixBlock)
		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)
	})
	t.Run("Capella", func(t *testing.T) {
		b := util.NewBlindedBeaconBlockCapella()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)

		mockChainService := &mock.ChainService{}
		bs := &Server{
			FinalizationFetcher:   mockChainService,
			Blocker:               &testutil.MockBlocker{BlockToReturn: blk},
			OptimisticModeFetcher: mockChainService,
		}

		expected, err := migration.V1Alpha1BeaconBlockBlindedCapellaToV2Blinded(b.Block)
		require.NoError(t, err)
		resp, err := bs.GetBlindedBlock(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)
		capellaBlock, ok := resp.Data.Message.(*ethpbv2.SignedBlindedBeaconBlockContainer_CapellaBlock)
		require.Equal(t, true, ok)
		assert.DeepEqual(t, expected, capellaBlock.CapellaBlock)
		assert.Equal(t, ethpbv2.Version_CAPELLA, resp.Version)
	})
	t.Run("execution optimistic", func(t *testing.T) {
		b := util.NewBlindedBeaconBlockBellatrix()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		r, err := blk.Block().HashTreeRoot()
		require.NoError(t, err)

		mockChainService := &mock.ChainService{
			OptimisticRoots: map[[32]byte]bool{r: true},
		}
		bs := &Server{
			FinalizationFetcher:   mockChainService,
			Blocker:               &testutil.MockBlocker{BlockToReturn: blk},
			OptimisticModeFetcher: mockChainService,
		}

		resp, err := bs.GetBlindedBlock(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
	t.Run("finalized", func(t *testing.T) {
		b := util.NewBeaconBlock()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := blk.Block().HashTreeRoot()
		require.NoError(t, err)

		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{root: true},
		}
		bs := &Server{
			FinalizationFetcher: mockChainService,
			Blocker:             &testutil.MockBlocker{BlockToReturn: blk},
		}

		resp, err := bs.GetBlindedBlock(ctx, &ethpbv1.BlockRequest{BlockId: root[:]})
		require.NoError(t, err)
		assert.Equal(t, true, resp.Finalized)
	})
	t.Run("not finalized", func(t *testing.T) {
		b := util.NewBeaconBlock()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := blk.Block().HashTreeRoot()
		require.NoError(t, err)

		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{root: false},
		}
		bs := &Server{
			FinalizationFetcher: mockChainService,
			Blocker:             &testutil.MockBlocker{BlockToReturn: blk},
		}

		resp, err := bs.GetBlindedBlock(ctx, &ethpbv1.BlockRequest{BlockId: root[:]})
		require.NoError(t, err)
		assert.Equal(t, false, resp.Finalized)
	})
}

func TestServer_GetBlindedBlockSSZ(t *testing.T) {
	ctx := context.Background()

	t.Run("Phase 0", func(t *testing.T) {
		b := util.NewBeaconBlock()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)

		bs := &Server{
			FinalizationFetcher: &mock.ChainService{},
			Blocker:             &testutil.MockBlocker{BlockToReturn: blk},
		}

		expected, err := blk.MarshalSSZ()
		require.NoError(t, err)
		resp, err := bs.GetBlindedBlockSSZ(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, expected, resp.Data)
		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)
	})
	t.Run("Altair", func(t *testing.T) {
		b := util.NewBeaconBlockAltair()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)

		bs := &Server{
			FinalizationFetcher: &mock.ChainService{},
			Blocker:             &testutil.MockBlocker{BlockToReturn: blk},
		}

		expected, err := blk.MarshalSSZ()
		require.NoError(t, err)
		resp, err := bs.GetBlindedBlockSSZ(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, expected, resp.Data)
		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		b := util.NewBlindedBeaconBlockBellatrix()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)

		mockChainService := &mock.ChainService{}
		bs := &Server{
			FinalizationFetcher:   mockChainService,
			Blocker:               &testutil.MockBlocker{BlockToReturn: blk},
			OptimisticModeFetcher: mockChainService,
		}

		expected, err := blk.MarshalSSZ()
		require.NoError(t, err)
		resp, err := bs.GetBlindedBlockSSZ(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, expected, resp.Data)
		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)
	})
	t.Run("Capella", func(t *testing.T) {
		b := util.NewBlindedBeaconBlockCapella()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)

		mockChainService := &mock.ChainService{}
		bs := &Server{
			FinalizationFetcher:   mockChainService,
			Blocker:               &testutil.MockBlocker{BlockToReturn: blk},
			OptimisticModeFetcher: mockChainService,
		}

		expected, err := blk.MarshalSSZ()
		require.NoError(t, err)
		resp, err := bs.GetBlindedBlockSSZ(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, expected, resp.Data)
		assert.Equal(t, ethpbv2.Version_CAPELLA, resp.Version)
	})
	t.Run("execution optimistic", func(t *testing.T) {
		b := util.NewBlindedBeaconBlockBellatrix()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		r, err := blk.Block().HashTreeRoot()
		require.NoError(t, err)

		mockChainService := &mock.ChainService{
			OptimisticRoots: map[[32]byte]bool{r: true},
		}
		bs := &Server{
			FinalizationFetcher:   mockChainService,
			Blocker:               &testutil.MockBlocker{BlockToReturn: blk},
			OptimisticModeFetcher: mockChainService,
		}

		resp, err := bs.GetBlindedBlockSSZ(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
	t.Run("finalized", func(t *testing.T) {
		b := util.NewBeaconBlock()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := blk.Block().HashTreeRoot()
		require.NoError(t, err)

		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{root: true},
		}
		bs := &Server{
			FinalizationFetcher: mockChainService,
			Blocker:             &testutil.MockBlocker{BlockToReturn: blk},
		}

		resp, err := bs.GetBlindedBlockSSZ(ctx, &ethpbv1.BlockRequest{BlockId: root[:]})
		require.NoError(t, err)
		assert.Equal(t, true, resp.Finalized)
	})
	t.Run("not finalized", func(t *testing.T) {
		b := util.NewBeaconBlock()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		root, err := blk.Block().HashTreeRoot()
		require.NoError(t, err)

		mockChainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{root: false},
		}
		bs := &Server{
			FinalizationFetcher: mockChainService,
			Blocker:             &testutil.MockBlocker{BlockToReturn: blk},
		}

		resp, err := bs.GetBlindedBlockSSZ(ctx, &ethpbv1.BlockRequest{BlockId: root[:]})
		require.NoError(t, err)
		assert.Equal(t, false, resp.Finalized)
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
	ctrl := gomock.NewController(t)

	t.Run("Phase 0", func(t *testing.T) {
		v1alpha1Server := mock2.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().ProposeBeaconBlock(gomock.Any(), gomock.Any())
		server := &Server{
			V1Alpha1ValidatorServer: v1alpha1Server,
		}

		blockReq := &ethpbv2.SignedBlindedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_Phase0Block{Phase0Block: &ethpbv1.BeaconBlock{}},
			Signature: []byte("sig"),
		}
		_, err := server.SubmitBlindedBlock(context.Background(), blockReq)
		assert.NoError(t, err)
	})
	t.Run("Altair", func(t *testing.T) {
		v1alpha1Server := mock2.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().ProposeBeaconBlock(gomock.Any(), gomock.Any())
		server := &Server{
			V1Alpha1ValidatorServer: v1alpha1Server,
		}

		blockReq := &ethpbv2.SignedBlindedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_AltairBlock{AltairBlock: &ethpbv2.BeaconBlockAltair{}},
			Signature: []byte("sig"),
		}
		_, err := server.SubmitBlindedBlock(context.Background(), blockReq)
		assert.NoError(t, err)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		v1alpha1Server := mock2.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().ProposeBeaconBlock(gomock.Any(), gomock.Any())
		server := &Server{
			V1Alpha1ValidatorServer: v1alpha1Server,
		}

		blockReq := &ethpbv2.SignedBlindedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_BellatrixBlock{BellatrixBlock: &ethpbv2.BlindedBeaconBlockBellatrix{}},
			Signature: []byte("sig"),
		}
		_, err := server.SubmitBlindedBlock(context.Background(), blockReq)
		assert.NoError(t, err)
	})
	t.Run("Capella", func(t *testing.T) {
		v1alpha1Server := mock2.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().ProposeBeaconBlock(gomock.Any(), gomock.Any())
		server := &Server{
			V1Alpha1ValidatorServer: v1alpha1Server,
		}

		blockReq := &ethpbv2.SignedBlindedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_CapellaBlock{CapellaBlock: &ethpbv2.BlindedBeaconBlockCapella{}},
			Signature: []byte("sig"),
		}
		_, err := server.SubmitBlindedBlock(context.Background(), blockReq)
		assert.NoError(t, err)
	})
}
