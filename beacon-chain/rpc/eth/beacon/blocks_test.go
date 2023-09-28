package beacon

import (
	"context"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/testutil"
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
	"google.golang.org/grpc"
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
	stream := &runtime.ServerTransportStream{}
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), stream)
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
