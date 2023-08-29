package validator

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	mockChain "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	builderTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/builder/testing"
	dbutil "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	mockSync "github.com/prysmaticlabs/prysm/v4/beacon-chain/sync/initial-sync/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/mock"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestProduceBlockV2(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	t.Run("Phase 0", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Phase0{Phase0: &ethpbalpha.BeaconBlock{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}

		resp, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BeaconBlockContainerV2_Phase0Block)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.Phase0Block.Slot)
	})
	t.Run("Altair", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Altair{Altair: &ethpbalpha.BeaconBlockAltair{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}

		resp, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BeaconBlockContainerV2_AltairBlock)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.AltairBlock.Slot)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Bellatrix{Bellatrix: &ethpbalpha.BeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BeaconBlockContainerV2_BellatrixBlock)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.BellatrixBlock.Slot)
	})
	t.Run("Bellatrix blinded", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: &ethpbalpha.BlindedBeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared Bellatrix beacon block is blinded", err)
	})
	t.Run("Capella", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Capella{Capella: &ethpbalpha.BeaconBlockCapella{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		assert.Equal(t, ethpbv2.Version_CAPELLA, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BeaconBlockContainerV2_CapellaBlock)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.CapellaBlock.Slot)
	})
	t.Run("Capella blinded", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedCapella{BlindedCapella: &ethpbalpha.BlindedBeaconBlockCapella{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared Capella beacon block is blinded", err)
	})
	t.Run("optimistic", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Bellatrix{Bellatrix: &ethpbalpha.BeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: true},
		}

		_, err := server.ProduceBlockV2(ctx, &ethpbv1.ProduceBlockRequest{})
		require.ErrorContains(t, "The node is currently optimistic and cannot serve validators", err)
	})
	t.Run("sync not ready", func(t *testing.T) {
		chainService := &mockChain.ChainService{}
		v1Server := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: true},
			HeadFetcher:           chainService,
			TimeFetcher:           chainService,
			OptimisticModeFetcher: chainService,
		}
		_, err := v1Server.ProduceBlockV2(context.Background(), nil)
		require.ErrorContains(t, "Syncing to latest head", err)
	})
}

func TestProduceBlockV2SSZ(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	t.Run("Phase 0", func(t *testing.T) {
		b := util.HydrateBeaconBlock(&ethpbalpha.BeaconBlock{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Phase0{Phase0: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}

		resp, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Altair", func(t *testing.T) {
		b := util.HydrateBeaconBlockAltair(&ethpbalpha.BeaconBlockAltair{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Altair{Altair: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
		}

		resp, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		b := util.HydrateBeaconBlockBellatrix(&ethpbalpha.BeaconBlockBellatrix{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Bellatrix{Bellatrix: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Bellatrix blinded", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: &ethpbalpha.BlindedBeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared Bellatrix beacon block is blinded", err)
	})
	t.Run("Capella", func(t *testing.T) {
		b := util.HydrateBeaconBlockCapella(&ethpbalpha.BeaconBlockCapella{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Capella{Capella: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Capella blinded", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedCapella{BlindedCapella: &ethpbalpha.BlindedBeaconBlockCapella{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared Capella beacon block is blinded", err)
	})
	t.Run("optimistic", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Bellatrix{Bellatrix: &ethpbalpha.BeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: true},
		}

		_, err := server.ProduceBlockV2SSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.ErrorContains(t, "The node is currently optimistic and cannot serve validators", err)
	})
	t.Run("sync not ready", func(t *testing.T) {
		chainService := &mockChain.ChainService{}
		v1Server := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: true},
			HeadFetcher:           chainService,
			TimeFetcher:           chainService,
			OptimisticModeFetcher: chainService,
		}
		_, err := v1Server.ProduceBlockV2SSZ(context.Background(), nil)
		require.ErrorContains(t, "Syncing to latest head", err)
	})
}

func TestProduceBlindedBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	t.Run("Phase 0", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Phase0{Phase0: &ethpbalpha.BeaconBlock{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
			BlockBuilder:   &builderTest.MockBuilderService{HasConfigured: true},
		}

		resp, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BlindedBeaconBlockContainer_Phase0Block)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.Phase0Block.Slot)
	})
	t.Run("Altair", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Altair{Altair: &ethpbalpha.BeaconBlockAltair{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
			BlockBuilder:   &builderTest.MockBuilderService{HasConfigured: true},
		}

		resp, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BlindedBeaconBlockContainer_AltairBlock)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.AltairBlock.Slot)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: &ethpbalpha.BlindedBeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BlindedBeaconBlockContainer_BellatrixBlock)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.BellatrixBlock.Slot)
	})
	t.Run("Bellatrix full", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Bellatrix{Bellatrix: &ethpbalpha.BeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared beacon block is not blinded", err)
	})
	t.Run("Capella", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedCapella{BlindedCapella: &ethpbalpha.BlindedBeaconBlockCapella{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)

		assert.Equal(t, ethpbv2.Version_CAPELLA, resp.Version)
		containerBlock, ok := resp.Data.Block.(*ethpbv2.BlindedBeaconBlockContainer_CapellaBlock)
		require.Equal(t, true, ok)
		assert.Equal(t, primitives.Slot(123), containerBlock.CapellaBlock.Slot)
	})
	t.Run("Capella full", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Capella{Capella: &ethpbalpha.BeaconBlockCapella{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared beacon block is not blinded", err)
	})
	t.Run("optimistic", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: &ethpbalpha.BlindedBeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: true},
		}

		_, err := server.ProduceBlindedBlock(ctx, &ethpbv1.ProduceBlockRequest{})
		require.ErrorContains(t, "The node is currently optimistic and cannot serve validators", err)
	})
	t.Run("builder not configured", func(t *testing.T) {
		v1Server := &Server{
			BlockBuilder: &builderTest.MockBuilderService{HasConfigured: false},
		}
		_, err := v1Server.ProduceBlindedBlock(context.Background(), nil)
		require.ErrorContains(t, "Block builder not configured", err)
	})
	t.Run("sync not ready", func(t *testing.T) {
		chainService := &mockChain.ChainService{}
		v1Server := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: true},
			HeadFetcher:           chainService,
			TimeFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
		}
		_, err := v1Server.ProduceBlindedBlock(context.Background(), nil)
		require.ErrorContains(t, "Syncing to latest head", err)
	})
}

func TestProduceBlindedBlockSSZ(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	t.Run("Phase 0", func(t *testing.T) {
		b := util.HydrateBeaconBlock(&ethpbalpha.BeaconBlock{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Phase0{Phase0: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
			BlockBuilder:   &builderTest.MockBuilderService{HasConfigured: true},
		}

		resp, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Altair", func(t *testing.T) {
		b := util.HydrateBeaconBlockAltair(&ethpbalpha.BeaconBlockAltair{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Altair{Altair: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server: v1alpha1Server,
			SyncChecker:    &mockSync.Sync{IsSyncing: false},
			BlockBuilder:   &builderTest.MockBuilderService{HasConfigured: true},
		}

		resp, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		b := util.HydrateBlindedBeaconBlockBellatrix(&ethpbalpha.BlindedBeaconBlockBellatrix{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Bellatrix full", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Bellatrix{Bellatrix: &ethpbalpha.BeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared Bellatrix beacon block is not blinded", err)
	})
	t.Run("Capella", func(t *testing.T) {
		b := util.HydrateBlindedBeaconBlockCapella(&ethpbalpha.BlindedBeaconBlockCapella{})
		b.Slot = 123
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedCapella{BlindedCapella: b}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		resp, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.NoError(t, err)
		expectedData, err := b.MarshalSSZ()
		assert.NoError(t, err)
		assert.DeepEqual(t, expectedData, resp.Data)
	})
	t.Run("Capella full", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_Capella{Capella: &ethpbalpha.BeaconBlockCapella{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: false},
		}

		_, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		assert.ErrorContains(t, "Prepared Capella beacon block is not blinded", err)
	})
	t.Run("optimistic", func(t *testing.T) {
		blk := &ethpbalpha.GenericBeaconBlock{Block: &ethpbalpha.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: &ethpbalpha.BlindedBeaconBlockBellatrix{Slot: 123}}}
		v1alpha1Server := mock.NewMockBeaconNodeValidatorServer(ctrl)
		v1alpha1Server.EXPECT().GetBeaconBlock(gomock.Any(), gomock.Any()).Return(blk, nil)
		server := &Server{
			V1Alpha1Server:        v1alpha1Server,
			SyncChecker:           &mockSync.Sync{IsSyncing: false},
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
			OptimisticModeFetcher: &mockChain.ChainService{Optimistic: true},
		}

		_, err := server.ProduceBlindedBlockSSZ(ctx, &ethpbv1.ProduceBlockRequest{})
		require.ErrorContains(t, "The node is currently optimistic and cannot serve validators", err)
	})
	t.Run("builder not configured", func(t *testing.T) {
		v1Server := &Server{
			BlockBuilder: &builderTest.MockBuilderService{HasConfigured: false},
		}
		_, err := v1Server.ProduceBlindedBlockSSZ(context.Background(), nil)
		require.ErrorContains(t, "Block builder not configured", err)
	})
	t.Run("sync not ready", func(t *testing.T) {
		chainService := &mockChain.ChainService{}
		v1Server := &Server{
			SyncChecker:           &mockSync.Sync{IsSyncing: true},
			HeadFetcher:           chainService,
			TimeFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BlockBuilder:          &builderTest.MockBuilderService{HasConfigured: true},
		}
		_, err := v1Server.ProduceBlindedBlockSSZ(context.Background(), nil)
		require.ErrorContains(t, "Syncing to latest head", err)
	})
}

func TestPrepareBeaconProposer(t *testing.T) {
	type args struct {
		request *ethpbv1.PrepareBeaconProposerRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name: "Happy Path",
			args: args{
				request: &ethpbv1.PrepareBeaconProposerRequest{
					Recipients: []*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer{
						{
							FeeRecipient:   make([]byte, fieldparams.FeeRecipientLength),
							ValidatorIndex: 1,
						},
					},
				},
			},
			wantErr: "",
		},
		{
			name: "invalid fee recipient length",
			args: args{
				request: &ethpbv1.PrepareBeaconProposerRequest{
					Recipients: []*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer{
						{
							FeeRecipient:   make([]byte, fieldparams.BLSPubkeyLength),
							ValidatorIndex: 1,
						},
					},
				},
			},
			wantErr: "Invalid fee recipient address",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := dbutil.SetupDB(t)
			ctx := context.Background()
			hook := logTest.NewGlobal()
			server := &Server{
				BeaconDB: db,
			}
			_, err := server.PrepareBeaconProposer(ctx, tt.args.request)
			if tt.wantErr != "" {
				require.ErrorContains(t, tt.wantErr, err)
				return
			}
			require.NoError(t, err)
			address, err := server.BeaconDB.FeeRecipientByValidatorID(ctx, 1)
			require.NoError(t, err)
			require.Equal(t, common.BytesToAddress(tt.args.request.Recipients[0].FeeRecipient), address)
			indexs := make([]primitives.ValidatorIndex, len(tt.args.request.Recipients))
			for i, recipient := range tt.args.request.Recipients {
				indexs[i] = recipient.ValidatorIndex
			}
			require.LogsContain(t, hook, fmt.Sprintf(`validatorIndices="%v"`, indexs))
		})
	}
}
func TestProposer_PrepareBeaconProposerOverlapping(t *testing.T) {
	hook := logTest.NewGlobal()
	db := dbutil.SetupDB(t)
	ctx := context.Background()
	proposerServer := &Server{BeaconDB: db}

	// New validator
	f := bytesutil.PadTo([]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	req := &ethpbv1.PrepareBeaconProposerRequest{
		Recipients: []*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer{
			{FeeRecipient: f, ValidatorIndex: 1},
		},
	}
	_, err := proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Updated fee recipient addresses for validator indices")

	// Same validator
	hook.Reset()
	_, err = proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsDoNotContain(t, hook, "Updated fee recipient addresses for validator indices")

	// Same validator with different fee recipient
	hook.Reset()
	f = bytesutil.PadTo([]byte{0x01, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	req = &ethpbv1.PrepareBeaconProposerRequest{
		Recipients: []*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer{
			{FeeRecipient: f, ValidatorIndex: 1},
		},
	}
	_, err = proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Updated fee recipient addresses for validator indices")

	// More than one validator
	hook.Reset()
	f = bytesutil.PadTo([]byte{0x01, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	req = &ethpbv1.PrepareBeaconProposerRequest{
		Recipients: []*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer{
			{FeeRecipient: f, ValidatorIndex: 1},
			{FeeRecipient: f, ValidatorIndex: 2},
		},
	}
	_, err = proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Updated fee recipient addresses for validator indices")

	// Same validators
	hook.Reset()
	_, err = proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsDoNotContain(t, hook, "Updated fee recipient addresses for validator indices")
}

func BenchmarkServer_PrepareBeaconProposer(b *testing.B) {
	db := dbutil.SetupDB(b)
	ctx := context.Background()
	proposerServer := &Server{BeaconDB: db}

	f := bytesutil.PadTo([]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	recipients := make([]*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer, 0)
	for i := 0; i < 10000; i++ {
		recipients = append(recipients, &ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer{FeeRecipient: f, ValidatorIndex: primitives.ValidatorIndex(i)})
	}

	req := &ethpbv1.PrepareBeaconProposerRequest{
		Recipients: recipients,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := proposerServer.PrepareBeaconProposer(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestGetLiveness(t *testing.T) {
	ctx := context.Background()

	// Setup:
	// Epoch 0 - both validators not live
	// Epoch 1 - validator with index 1 is live
	// Epoch 2 - validator with index 0 is live
	oldSt, err := util.NewBeaconStateBellatrix()
	require.NoError(t, err)
	require.NoError(t, oldSt.AppendCurrentParticipationBits(0))
	require.NoError(t, oldSt.AppendCurrentParticipationBits(0))
	headSt, err := util.NewBeaconStateBellatrix()
	require.NoError(t, err)
	require.NoError(t, headSt.SetSlot(params.BeaconConfig().SlotsPerEpoch*2))
	require.NoError(t, headSt.AppendPreviousParticipationBits(0))
	require.NoError(t, headSt.AppendPreviousParticipationBits(1))
	require.NoError(t, headSt.AppendCurrentParticipationBits(1))
	require.NoError(t, headSt.AppendCurrentParticipationBits(0))

	server := &Server{
		HeadFetcher: &mockChain.ChainService{State: headSt},
		Stater: &testutil.MockStater{
			// We configure states for last slots of an epoch
			StatesBySlot: map[primitives.Slot]state.BeaconState{
				params.BeaconConfig().SlotsPerEpoch - 1:   oldSt,
				params.BeaconConfig().SlotsPerEpoch*3 - 1: headSt,
			},
		},
	}

	t.Run("old epoch", func(t *testing.T) {
		resp, err := server.GetLiveness(ctx, &ethpbv2.GetLivenessRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{0, 1},
		})
		require.NoError(t, err)
		data0 := resp.Data[0]
		data1 := resp.Data[1]
		assert.Equal(t, true, (data0.Index == 0 && !data0.IsLive) || (data0.Index == 1 && !data0.IsLive))
		assert.Equal(t, true, (data1.Index == 0 && !data1.IsLive) || (data1.Index == 1 && !data1.IsLive))
	})
	t.Run("previous epoch", func(t *testing.T) {
		resp, err := server.GetLiveness(ctx, &ethpbv2.GetLivenessRequest{
			Epoch: 1,
			Index: []primitives.ValidatorIndex{0, 1},
		})
		require.NoError(t, err)
		data0 := resp.Data[0]
		data1 := resp.Data[1]
		assert.Equal(t, true, (data0.Index == 0 && !data0.IsLive) || (data0.Index == 1 && data0.IsLive))
		assert.Equal(t, true, (data1.Index == 0 && !data1.IsLive) || (data1.Index == 1 && data1.IsLive))
	})
	t.Run("current epoch", func(t *testing.T) {
		resp, err := server.GetLiveness(ctx, &ethpbv2.GetLivenessRequest{
			Epoch: 2,
			Index: []primitives.ValidatorIndex{0, 1},
		})
		require.NoError(t, err)
		data0 := resp.Data[0]
		data1 := resp.Data[1]
		assert.Equal(t, true, (data0.Index == 0 && data0.IsLive) || (data0.Index == 1 && !data0.IsLive))
		assert.Equal(t, true, (data1.Index == 0 && data1.IsLive) || (data1.Index == 1 && !data1.IsLive))
	})
	t.Run("future epoch", func(t *testing.T) {
		_, err := server.GetLiveness(ctx, &ethpbv2.GetLivenessRequest{
			Epoch: 3,
			Index: []primitives.ValidatorIndex{0, 1},
		})
		require.ErrorContains(t, "Requested epoch cannot be in the future", err)
	})
	t.Run("unknown validator index", func(t *testing.T) {
		_, err := server.GetLiveness(ctx, &ethpbv2.GetLivenessRequest{
			Epoch: 0,
			Index: []primitives.ValidatorIndex{0, 1, 2},
		})
		require.ErrorContains(t, "Validator index 2 is invalid", err)
	})
}
