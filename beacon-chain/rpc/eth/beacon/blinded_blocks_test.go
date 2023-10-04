package beacon

import (
	"context"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"google.golang.org/grpc"
)

func TestServer_GetBlindedBlock(t *testing.T) {
	stream := &runtime.ServerTransportStream{}
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), stream)

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
	t.Run("Deneb", func(t *testing.T) {
		b := util.NewBlindedBeaconBlockDeneb()
		blk, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)

		mockChainService := &mock.ChainService{}
		bs := &Server{
			FinalizationFetcher:   mockChainService,
			Blocker:               &testutil.MockBlocker{BlockToReturn: blk},
			OptimisticModeFetcher: mockChainService,
		}

		expected, err := migration.V1Alpha1BeaconBlockBlindedDenebToV2Blinded(b.Message)
		require.NoError(t, err)
		resp, err := bs.GetBlindedBlock(ctx, &ethpbv1.BlockRequest{})
		require.NoError(t, err)
		denebBlock, ok := resp.Data.Message.(*ethpbv2.SignedBlindedBeaconBlockContainer_DenebBlock)
		require.Equal(t, true, ok)
		assert.DeepEqual(t, expected, denebBlock.DenebBlock)
		assert.Equal(t, ethpbv2.Version_DENEB, resp.Version)
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
	t.Run("Deneb", func(t *testing.T) {
		b := util.NewBlindedBeaconBlockDeneb()
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
		assert.Equal(t, ethpbv2.Version_DENEB, resp.Version)
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
