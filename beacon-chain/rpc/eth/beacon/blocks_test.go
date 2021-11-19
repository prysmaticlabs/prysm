package beacon

import (
	"context"
	"reflect"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/proto/migration"
	ethpbalpha "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func fillDBTestBlocks(ctx context.Context, t *testing.T, beaconDB db.Database) (*ethpbalpha.SignedBeaconBlock, []*ethpbalpha.BeaconBlockContainer) {
	parentRoot := [32]byte{1, 2, 3}
	genBlk := util.NewBeaconBlock()
	genBlk.Block.ParentRoot = parentRoot[:]
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genBlk)))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, root))

	count := types.Slot(100)
	blks := make([]block.SignedBeaconBlock, count)
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
		blks[i] = wrapper.WrappedPhase0SignedBeaconBlock(b)
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
	signedBlk, err := wrapper.WrappedAltairSignedBeaconBlock(genBlk)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, signedBlk))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, root))

	count := types.Slot(100)
	blks := make([]block.SignedBeaconBlock, count)
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
		signedB, err := wrapper.WrappedAltairSignedBeaconBlock(b)
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
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))
	b3 := util.NewBeaconBlock()
	b3.Block.Slot = 30
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b3)))

	bs := &Server{
		BeaconDB: beaconDB,
		ChainInfoFetcher: &mock.ChainService{
			DB:                  beaconDB,
			Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block),
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		},
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

			assert.Equal(t, tt.want.Block.Slot, header.Data.Header.Message.Slot)
			assert.DeepEqual(t, tt.want.Block.StateRoot, header.Data.Header.Message.StateRoot)
			assert.DeepEqual(t, tt.want.Block.ParentRoot, header.Data.Header.Message.ParentRoot)
			expectedRoot, err := tt.want.Block.Body.HashTreeRoot()
			require.NoError(t, err)
			assert.DeepEqual(t, expectedRoot[:], header.Data.Header.Message.BodyRoot)
			assert.Equal(t, tt.want.Block.ProposerIndex, header.Data.Header.Message.ProposerIndex)
		})
	}
}

func TestServer_ListBlockHeaders(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	_, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	headBlock := blkContainers[len(blkContainers)-1]
	bs := &Server{
		BeaconDB: beaconDB,
		ChainInfoFetcher: &mock.ChainService{
			DB:                  beaconDB,
			Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block),
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		},
	}

	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 30
	b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))
	b3 := util.NewBeaconBlock()
	b3.Block.Slot = 30
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b3)))
	b4 := util.NewBeaconBlock()
	b4.Block.Slot = 31
	b4.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b4)))
	b5 := util.NewBeaconBlock()
	b5.Block.Slot = 28
	b5.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b5)))

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
}

func TestServer_ProposeBlock_OK(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genesis := util.NewBeaconBlock()
		require.NoError(t, beaconDB.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(genesis)), "Could not save genesis block")

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
		require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(req)))
		blockReq := &ethpbv2.SignedBeaconBlockContainerV2{
			Message:   &ethpbv2.SignedBeaconBlockContainerV2_Phase0Block{Phase0Block: v1Block.Block},
			Signature: v1Block.Signature,
		}
		_, err = beaconChainServer.SubmitBlock(context.Background(), blockReq)
		assert.NoError(t, err, "Could not propose block correctly")
	})

	t.Run("Altair", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		genesis := util.NewBeaconBlockAltair()
		wrapped, err := wrapper.WrappedAltairSignedBeaconBlock(genesis)
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveBlock(context.Background(), wrapped), "Could not save genesis block")

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
		wrapped, err = wrapper.WrappedAltairSignedBeaconBlock(req)
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveBlock(ctx, wrapped))
		blockReq := &ethpbv2.SignedBeaconBlockContainerV2{
			Message:   &ethpbv2.SignedBeaconBlockContainerV2_AltairBlock{AltairBlock: v2Block},
			Signature: req.Signature,
		}
		_, err = beaconChainServer.SubmitBlock(context.Background(), blockReq)
		assert.NoError(t, err, "Could not propose block correctly")
	})
}

func TestServer_GetBlock(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	_, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	headBlock := blkContainers[len(blkContainers)-1]

	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 30
	b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))
	b3 := util.NewBeaconBlock()
	b3.Block.Slot = 30
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b3)))

	bs := &Server{
		BeaconDB: beaconDB,
		ChainInfoFetcher: &mock.ChainService{
			DB:                  beaconDB,
			Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block),
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		},
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
			name:    "canonical",
			blockID: []byte("30"),
			want:    blkContainers[30].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
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
				require.NotEqual(t, err, nil)
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

		_, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
		headBlock := blkContainers[len(blkContainers)-1]

		b2 := util.NewBeaconBlock()
		b2.Block.Slot = 30
		b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
		require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))
		b3 := util.NewBeaconBlock()
		b3.Block.Slot = 30
		b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
		require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b3)))

		bs := &Server{
			BeaconDB: beaconDB,
			ChainInfoFetcher: &mock.ChainService{
				DB:                  beaconDB,
				Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block),
				Root:                headBlock.BlockRoot,
				FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			},
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
				name:    "canonical",
				blockID: []byte("30"),
				want:    blkContainers[30].Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block,
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

				phase0Block, ok := blk.Data.Message.(*ethpbv2.SignedBeaconBlockContainerV2_Phase0Block)
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

		_, blkContainers := fillDBTestBlocksAltair(ctx, t, beaconDB)
		headBlock := blkContainers[len(blkContainers)-1]

		b2 := util.NewBeaconBlockAltair()
		b2.Block.Slot = 30
		b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
		signedBlk, err := wrapper.WrappedAltairSignedBeaconBlock(b2)
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveBlock(ctx, signedBlk))
		b3 := util.NewBeaconBlockAltair()
		b3.Block.Slot = 30
		b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
		signedBlk, err = wrapper.WrappedAltairSignedBeaconBlock(b2)
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveBlock(ctx, signedBlk))

		chainBlk, err := wrapper.WrappedAltairSignedBeaconBlock(headBlock.GetAltairBlock())
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

				altairBlock, ok := blk.Data.Message.(*ethpbv2.SignedBeaconBlockContainerV2_AltairBlock)
				require.Equal(t, true, ok)
				if !reflect.DeepEqual(altairBlock.AltairBlock, v2Block) {
					t.Error("Expected blocks to equal")
				}
				assert.Equal(t, ethpbv2.Version_ALTAIR, blk.Version)
			})
		}
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
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))

	bs := &Server{
		BeaconDB: beaconDB,
		ChainInfoFetcher: &mock.ChainService{
			DB:                  beaconDB,
			Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block),
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		},
	}

	ok, blocks, err := beaconDB.BlocksBySlot(ctx, 30)
	require.Equal(t, true, ok)
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
		require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))

		bs := &Server{
			BeaconDB: beaconDB,
			ChainInfoFetcher: &mock.ChainService{
				DB:                  beaconDB,
				Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block),
				Root:                headBlock.BlockRoot,
				FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			},
		}

		ok, blocks, err := beaconDB.BlocksBySlot(ctx, 30)
		require.Equal(t, true, ok)
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
		signedBlk, err := wrapper.WrappedAltairSignedBeaconBlock(b2)
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveBlock(ctx, signedBlk))

		chainBlk, err := wrapper.WrappedAltairSignedBeaconBlock(headBlock.GetAltairBlock())
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

		ok, blocks, err := beaconDB.BlocksBySlot(ctx, 30)
		require.Equal(t, true, ok)
		require.NoError(t, err)
		sszBlock, err := blocks[0].MarshalSSZ()
		require.NoError(t, err)

		resp, err := bs.GetBlockSSZV2(ctx, &ethpbv2.BlockRequestV2{BlockId: []byte("30")})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, sszBlock, resp.Data)
		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
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
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))
	b3 := util.NewBeaconBlock()
	b3.Block.Slot = 30
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b3)))

	bs := &Server{
		BeaconDB: beaconDB,
		ChainInfoFetcher: &mock.ChainService{
			DB:                  beaconDB,
			Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block),
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		},
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
}

func TestServer_ListBlockAttestations(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		beaconDB := dbTest.SetupDB(t)
		ctx := context.Background()

		_, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
		headBlock := blkContainers[len(blkContainers)-1]
		bs := &Server{
			BeaconDB: beaconDB,
			ChainInfoFetcher: &mock.ChainService{
				DB:                  beaconDB,
				Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_Phase0Block).Phase0Block),
				Root:                headBlock.BlockRoot,
				FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			},
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

				if !reflect.DeepEqual(blk.Data, v1Block.Block.Body.Attestations) {
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
		blk, err := wrapper.WrappedAltairSignedBeaconBlock(headBlock.Block.(*ethpbalpha.BeaconBlockContainer_AltairBlock).AltairBlock)
		require.NoError(t, err)
		bs := &Server{
			BeaconDB: beaconDB,
			ChainInfoFetcher: &mock.ChainService{
				DB:                  beaconDB,
				Block:               blk,
				Root:                headBlock.BlockRoot,
				FinalizedCheckPoint: &ethpbalpha.Checkpoint{Root: blkContainers[64].BlockRoot},
			},
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

				if !reflect.DeepEqual(blk.Data, v1Block.Body.Attestations) {
					t.Error("Expected attestations to equal")
				}
			})
		}
	})
}
