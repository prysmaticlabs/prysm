package beacon

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/cmd"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/mock"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/time/slots"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestServer_ListBlocks_NoResults(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	wanted := &ethpb.ListBlocksResponse{
		BlockContainers: make([]*ethpb.BeaconBlockContainer, 0),
		TotalSize:       int32(0),
		NextPageToken:   strconv.Itoa(0),
	}
	res, err := bs.ListBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Slot{
			Slot: 0,
		},
	})
	require.NoError(t, err)
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
	res, err = bs.ListBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Slot{
			Slot: 0,
		},
	})
	require.NoError(t, err)
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
	res, err = bs.ListBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Root{
			Root: make([]byte, fieldparams.RootLength),
		},
	})
	require.NoError(t, err)
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

func TestServer_ListBlocks_Genesis(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}

	// Should throw an error if no genesis block is found.
	_, err := bs.ListBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Genesis{
			Genesis: true,
		},
	})
	require.ErrorContains(t, "Could not find genesis", err)

	// Should return the proper genesis block if it exists.
	parentRoot := [32]byte{'a'}
	blk := util.NewBeaconBlock()
	blk.Block.ParentRoot = parentRoot[:]
	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := wrapper.WrappedSignedBeaconBlock(blk)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wsb))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))
	wanted := &ethpb.ListBlocksResponse{
		BlockContainers: []*ethpb.BeaconBlockContainer{
			{
				Block:     &ethpb.BeaconBlockContainer_Phase0Block{Phase0Block: blk},
				BlockRoot: root[:],
				Canonical: true,
			},
		},
		NextPageToken: "0",
		TotalSize:     1,
	}
	res, err := bs.ListBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Genesis{
			Genesis: true,
		},
	})
	require.NoError(t, err)
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

func TestServer_ListBlocks_Genesis_MultiBlocks(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	// Should return the proper genesis block if it exists.
	parentRoot := [32]byte{1, 2, 3}
	blk := util.NewBeaconBlock()
	blk.Block.ParentRoot = parentRoot[:]
	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := wrapper.WrappedSignedBeaconBlock(blk)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wsb))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

	count := types.Slot(100)
	blks := make([]block.SignedBeaconBlock, count)
	for i := types.Slot(0); i < count; i++ {
		b := util.NewBeaconBlock()
		b.Block.Slot = i
		require.NoError(t, err)
		blks[i], err = wrapper.WrappedSignedBeaconBlock(b)
		require.NoError(t, err)
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))

	// Should throw an error if more than one blk returned.
	_, err = bs.ListBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Genesis{
			Genesis: true,
		},
	})
	require.NoError(t, err)
}

func TestServer_ListBlocks_Pagination(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())

	db := dbTest.SetupDB(t)
	chain := &chainMock.ChainService{
		CanonicalRoots: map[[32]byte]bool{},
	}
	ctx := context.Background()

	count := types.Slot(100)
	blks := make([]block.SignedBeaconBlock, count)
	blkContainers := make([]*ethpb.BeaconBlockContainer, count)
	for i := types.Slot(0); i < count; i++ {
		b := util.NewBeaconBlock()
		b.Block.Slot = i
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		chain.CanonicalRoots[root] = true
		blks[i], err = wrapper.WrappedSignedBeaconBlock(b)
		require.NoError(t, err)
		blkContainers[i] = &ethpb.BeaconBlockContainer{
			Block:     &ethpb.BeaconBlockContainer_Phase0Block{Phase0Block: b},
			BlockRoot: root[:],
			Canonical: true,
		}
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))

	orphanedBlk := util.NewBeaconBlock()
	orphanedBlk.Block.Slot = 300
	orphanedBlkRoot, err := orphanedBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := wrapper.WrappedSignedBeaconBlock(orphanedBlk)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wsb))

	bs := &Server{
		BeaconDB:         db,
		CanonicalFetcher: chain,
	}

	root6, err := blks[6].Block().HashTreeRoot()
	require.NoError(t, err)

	tests := []struct {
		req *ethpb.ListBlocksRequest
		res *ethpb.ListBlocksResponse
	}{
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Slot{Slot: 5},
			PageSize:    3},
			res: &ethpb.ListBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{
					{
						Block: &ethpb.BeaconBlockContainer_Phase0Block{
							Phase0Block: util.HydrateSignedBeaconBlock(&ethpb.SignedBeaconBlock{
								Block: &ethpb.BeaconBlock{
									Slot: 5,
								},
							}),
						},
						BlockRoot: blkContainers[5].BlockRoot,
						Canonical: blkContainers[5].Canonical,
					},
				},
				NextPageToken: "",
				TotalSize:     1,
			},
		},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Root{Root: root6[:]},
			PageSize:    3},
			res: &ethpb.ListBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{
					{
						Block: &ethpb.BeaconBlockContainer_Phase0Block{
							Phase0Block: util.HydrateSignedBeaconBlock(&ethpb.SignedBeaconBlock{
								Block: &ethpb.BeaconBlock{
									Slot: 6,
								},
							}),
						},
						BlockRoot: blkContainers[6].BlockRoot,
						Canonical: blkContainers[6].Canonical,
					},
				},
				TotalSize:     1,
				NextPageToken: strconv.Itoa(0),
			},
		},
		{req: &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: root6[:]}},
			res: &ethpb.ListBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{
					{
						Block: &ethpb.BeaconBlockContainer_Phase0Block{
							Phase0Block: util.HydrateSignedBeaconBlock(&ethpb.SignedBeaconBlock{
								Block: &ethpb.BeaconBlock{
									Slot: 6,
								},
							}),
						},
						BlockRoot: blkContainers[6].BlockRoot,
						Canonical: blkContainers[6].Canonical,
					},
				},
				TotalSize:     1,
				NextPageToken: strconv.Itoa(0),
			},
		},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: 0},
			PageSize:    100},
			res: &ethpb.ListBlocksResponse{
				BlockContainers: blkContainers[0:params.BeaconConfig().SlotsPerEpoch],
				NextPageToken:   "",
				TotalSize:       int32(params.BeaconConfig().SlotsPerEpoch)}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(1),
			QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: 5},
			PageSize:    3},
			res: &ethpb.ListBlocksResponse{
				BlockContainers: blkContainers[43:46],
				NextPageToken:   "2",
				TotalSize:       int32(params.BeaconConfig().SlotsPerEpoch)}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(1),
			QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: 11},
			PageSize:    7},
			res: &ethpb.ListBlocksResponse{
				BlockContainers: blkContainers[95:96],
				NextPageToken:   "",
				TotalSize:       int32(params.BeaconConfig().SlotsPerEpoch)}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: 12},
			PageSize:    4},
			res: &ethpb.ListBlocksResponse{
				BlockContainers: blkContainers[96:100],
				NextPageToken:   "",
				TotalSize:       int32(params.BeaconConfig().SlotsPerEpoch / 2)}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Slot{Slot: 300},
			PageSize:    3},
			res: &ethpb.ListBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{
					{
						Block: &ethpb.BeaconBlockContainer_Phase0Block{
							Phase0Block: util.HydrateSignedBeaconBlock(&ethpb.SignedBeaconBlock{
								Block: &ethpb.BeaconBlock{
									Slot: 300,
								},
							}),
						},
						BlockRoot: orphanedBlkRoot[:],
						Canonical: false,
					},
				},
				NextPageToken: "",
				TotalSize:     1}},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			res, err := bs.ListBlocks(ctx, test.req)
			require.NoError(t, err)
			require.DeepSSZEqual(t, res, test.res)
		})
	}
}

func TestServer_ListBlocks_Errors(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{BeaconDB: db}
	exceedsMax := int32(cmd.Get().MaxRPCPageSize + 1)

	wanted := fmt.Sprintf("Requested page size %d can not be greater than max size %d", exceedsMax, cmd.Get().MaxRPCPageSize)
	req := &ethpb.ListBlocksRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	_, err := bs.ListBlocks(ctx, req)
	assert.ErrorContains(t, wanted, err)

	wanted = "Must specify a filter criteria for fetching"
	req = &ethpb.ListBlocksRequest{}
	_, err = bs.ListBlocks(ctx, req)
	assert.ErrorContains(t, wanted, err)

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Slot{Slot: 0}}
	res, err := bs.ListBlocks(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 0, len(res.BlockContainers), "Wanted empty list")
	assert.Equal(t, int32(0), res.TotalSize, "Wanted total size 0")

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Slot{}}
	res, err = bs.ListBlocks(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 0, len(res.BlockContainers), "Wanted empty list")
	assert.Equal(t, int32(0), res.TotalSize, "Wanted total size 0")

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: []byte{'A'}}}
	res, err = bs.ListBlocks(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 0, len(res.BlockContainers), "Wanted empty list")
	assert.Equal(t, int32(0), res.TotalSize, "Wanted total size 0")

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: []byte{'A'}}}
	res, err = bs.ListBlocks(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 0, len(res.BlockContainers), "Wanted empty list")
	assert.Equal(t, int32(0), res.TotalSize, "Wanted total size 0")
}

// ensures that if any of the checkpoints are zero-valued, an error will be generated without genesis being present
func TestServer_GetChainHead_NoGenesis(t *testing.T) {
	db := dbTest.SetupDB(t)

	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetSlot(1))

	genBlock := util.NewBeaconBlock()
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, fieldparams.RootLength)
	wsb, err := wrapper.WrappedSignedBeaconBlock(genBlock)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	gRoot, err := genBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	cases := []struct {
		name       string
		zeroSetter func(val *ethpb.Checkpoint) error
	}{
		{
			name:       "zero-value prev justified",
			zeroSetter: s.SetPreviousJustifiedCheckpoint,
		},
		{
			name:       "zero-value current justified",
			zeroSetter: s.SetCurrentJustifiedCheckpoint,
		},
		{
			name:       "zero-value finalized",
			zeroSetter: s.SetFinalizedCheckpoint,
		},
	}
	finalized := &ethpb.Checkpoint{Epoch: 1, Root: gRoot[:]}
	prevJustified := &ethpb.Checkpoint{Epoch: 2, Root: gRoot[:]}
	justified := &ethpb.Checkpoint{Epoch: 3, Root: gRoot[:]}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.NoError(t, s.SetPreviousJustifiedCheckpoint(prevJustified))
			require.NoError(t, s.SetCurrentJustifiedCheckpoint(justified))
			require.NoError(t, s.SetFinalizedCheckpoint(finalized))
			require.NoError(t, c.zeroSetter(&ethpb.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash[:]}))
		})
		wsb, err := wrapper.WrappedSignedBeaconBlock(genBlock)
		require.NoError(t, err)
		bs := &Server{
			BeaconDB:    db,
			HeadFetcher: &chainMock.ChainService{Block: wsb, State: s},
			FinalizationFetcher: &chainMock.ChainService{
				FinalizedCheckPoint:         s.FinalizedCheckpoint(),
				CurrentJustifiedCheckPoint:  s.CurrentJustifiedCheckpoint(),
				PreviousJustifiedCheckPoint: s.PreviousJustifiedCheckpoint()},
		}
		_, err = bs.GetChainHead(context.Background(), nil)
		require.ErrorContains(t, "Could not get genesis block", err)
	}
}

func TestServer_GetChainHead_NoFinalizedBlock(t *testing.T) {
	db := dbTest.SetupDB(t)

	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetSlot(1))
	require.NoError(t, s.SetPreviousJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 3, Root: bytesutil.PadTo([]byte{'A'}, fieldparams.RootLength)}))
	require.NoError(t, s.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 2, Root: bytesutil.PadTo([]byte{'B'}, fieldparams.RootLength)}))
	require.NoError(t, s.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'C'}, fieldparams.RootLength)}))

	genBlock := util.NewBeaconBlock()
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, fieldparams.RootLength)
	wsb, err := wrapper.WrappedSignedBeaconBlock(genBlock)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	gRoot, err := genBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), gRoot))

	wsb, err = wrapper.WrappedSignedBeaconBlock(genBlock)
	require.NoError(t, err)

	bs := &Server{
		BeaconDB:    db,
		HeadFetcher: &chainMock.ChainService{Block: wsb, State: s},
		FinalizationFetcher: &chainMock.ChainService{
			FinalizedCheckPoint:         s.FinalizedCheckpoint(),
			CurrentJustifiedCheckPoint:  s.CurrentJustifiedCheckpoint(),
			PreviousJustifiedCheckPoint: s.PreviousJustifiedCheckpoint()},
	}

	_, err = bs.GetChainHead(context.Background(), nil)
	require.ErrorContains(t, "Could not get finalized block", err)
}

func TestServer_GetChainHead_NoHeadBlock(t *testing.T) {
	bs := &Server{
		HeadFetcher: &chainMock.ChainService{Block: nil},
	}
	_, err := bs.GetChainHead(context.Background(), nil)
	assert.ErrorContains(t, "Head block of chain was nil", err)
}

func TestServer_GetChainHead(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())

	db := dbTest.SetupDB(t)
	genBlock := util.NewBeaconBlock()
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, fieldparams.RootLength)
	wsb, err := wrapper.WrappedSignedBeaconBlock(genBlock)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	gRoot, err := genBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), gRoot))

	finalizedBlock := util.NewBeaconBlock()
	finalizedBlock.Block.Slot = 1
	finalizedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'A'}, fieldparams.RootLength)
	wsb, err = wrapper.WrappedSignedBeaconBlock(finalizedBlock)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	fRoot, err := finalizedBlock.Block.HashTreeRoot()
	require.NoError(t, err)

	justifiedBlock := util.NewBeaconBlock()
	justifiedBlock.Block.Slot = 2
	justifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'B'}, fieldparams.RootLength)
	wsb, err = wrapper.WrappedSignedBeaconBlock(justifiedBlock)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	jRoot, err := justifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err)

	prevJustifiedBlock := util.NewBeaconBlock()
	prevJustifiedBlock.Block.Slot = 3
	prevJustifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'C'}, fieldparams.RootLength)
	wsb, err = wrapper.WrappedSignedBeaconBlock(prevJustifiedBlock)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	pjRoot, err := prevJustifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err)

	s, err := v1.InitializeFromProto(&ethpb.BeaconState{
		Slot:                        1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 3, Root: pjRoot[:]},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Epoch: 2, Root: jRoot[:]},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Epoch: 1, Root: fRoot[:]},
	})
	require.NoError(t, err)

	b := util.NewBeaconBlock()
	b.Block.Slot, err = slots.EpochStart(s.PreviousJustifiedCheckpoint().Epoch)
	require.NoError(t, err)
	b.Block.Slot++
	wsb, err = wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	bs := &Server{
		BeaconDB:    db,
		HeadFetcher: &chainMock.ChainService{Block: wsb, State: s},
		FinalizationFetcher: &chainMock.ChainService{
			FinalizedCheckPoint:         s.FinalizedCheckpoint(),
			CurrentJustifiedCheckPoint:  s.CurrentJustifiedCheckpoint(),
			PreviousJustifiedCheckPoint: s.PreviousJustifiedCheckpoint()},
	}

	head, err := bs.GetChainHead(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, types.Epoch(3), head.PreviousJustifiedEpoch, "Unexpected PreviousJustifiedEpoch")
	assert.Equal(t, types.Epoch(2), head.JustifiedEpoch, "Unexpected JustifiedEpoch")
	assert.Equal(t, types.Epoch(1), head.FinalizedEpoch, "Unexpected FinalizedEpoch")
	assert.Equal(t, types.Slot(24), head.PreviousJustifiedSlot, "Unexpected PreviousJustifiedSlot")
	assert.Equal(t, types.Slot(16), head.JustifiedSlot, "Unexpected JustifiedSlot")
	assert.Equal(t, types.Slot(8), head.FinalizedSlot, "Unexpected FinalizedSlot")
	assert.DeepEqual(t, pjRoot[:], head.PreviousJustifiedBlockRoot, "Unexpected PreviousJustifiedBlockRoot")
	assert.DeepEqual(t, jRoot[:], head.JustifiedBlockRoot, "Unexpected JustifiedBlockRoot")
	assert.DeepEqual(t, fRoot[:], head.FinalizedBlockRoot, "Unexpected FinalizedBlockRoot")
}

func TestServer_StreamChainHead_ContextCanceled(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	chainService := &chainMock.ChainService{}
	server := &Server{
		Ctx:           ctx,
		StateNotifier: chainService.StateNotifier(),
		BeaconDB:      db,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconChain_StreamChainHeadServer(ctrl)
	mockStream.EXPECT().Context().Return(ctx)
	go func(tt *testing.T) {
		assert.ErrorContains(tt, "Context canceled", server.StreamChainHead(&emptypb.Empty{}, mockStream))
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestServer_StreamChainHead_OnHeadUpdated(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	db := dbTest.SetupDB(t)
	genBlock := util.NewBeaconBlock()
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, fieldparams.RootLength)
	wsb, err := wrapper.WrappedSignedBeaconBlock(genBlock)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	gRoot, err := genBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), gRoot))

	finalizedBlock := util.NewBeaconBlock()
	finalizedBlock.Block.Slot = 32
	finalizedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'A'}, fieldparams.RootLength)
	wsb, err = wrapper.WrappedSignedBeaconBlock(finalizedBlock)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	fRoot, err := finalizedBlock.Block.HashTreeRoot()
	require.NoError(t, err)

	justifiedBlock := util.NewBeaconBlock()
	justifiedBlock.Block.Slot = 64
	justifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'B'}, fieldparams.RootLength)
	wsb, err = wrapper.WrappedSignedBeaconBlock(justifiedBlock)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	jRoot, err := justifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err)

	prevJustifiedBlock := util.NewBeaconBlock()
	prevJustifiedBlock.Block.Slot = 96
	prevJustifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'C'}, fieldparams.RootLength)
	wsb, err = wrapper.WrappedSignedBeaconBlock(prevJustifiedBlock)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	pjRoot, err := prevJustifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err)

	s, err := v1.InitializeFromProto(&ethpb.BeaconState{
		Slot:                        1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 3, Root: pjRoot[:]},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Epoch: 2, Root: jRoot[:]},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Epoch: 1, Root: fRoot[:]},
	})
	require.NoError(t, err)

	b := util.NewBeaconBlock()
	b.Block.Slot, err = slots.EpochStart(s.PreviousJustifiedCheckpoint().Epoch)
	require.NoError(t, err)

	hRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)

	chainService := &chainMock.ChainService{}
	ctx := context.Background()
	wsb, err = wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	server := &Server{
		Ctx:           ctx,
		HeadFetcher:   &chainMock.ChainService{Block: wsb, State: s},
		BeaconDB:      db,
		StateNotifier: chainService.StateNotifier(),
		FinalizationFetcher: &chainMock.ChainService{
			FinalizedCheckPoint:         s.FinalizedCheckpoint(),
			CurrentJustifiedCheckPoint:  s.CurrentJustifiedCheckpoint(),
			PreviousJustifiedCheckPoint: s.PreviousJustifiedCheckpoint()},
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconChain_StreamChainHeadServer(ctrl)
	mockStream.EXPECT().Send(
		&ethpb.ChainHead{
			HeadSlot:                   b.Block.Slot,
			HeadEpoch:                  slots.ToEpoch(b.Block.Slot),
			HeadBlockRoot:              hRoot[:],
			FinalizedSlot:              32,
			FinalizedEpoch:             1,
			FinalizedBlockRoot:         fRoot[:],
			JustifiedSlot:              64,
			JustifiedEpoch:             2,
			JustifiedBlockRoot:         jRoot[:],
			PreviousJustifiedSlot:      96,
			PreviousJustifiedEpoch:     3,
			PreviousJustifiedBlockRoot: pjRoot[:],
		},
	).Do(func(arg0 interface{}) {
		exitRoutine <- true
	})
	mockStream.EXPECT().Context().Return(ctx).AnyTimes()

	go func(tt *testing.T) {
		assert.NoError(tt, server.StreamChainHead(&emptypb.Empty{}, mockStream), "Could not call RPC method")
	}(t)

	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = server.StateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.BlockProcessed,
			Data: &statefeed.BlockProcessedData{},
		})
	}
	<-exitRoutine
}

func TestServer_StreamBlocksVerified_ContextCanceled(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	chainService := &chainMock.ChainService{}
	ctx, cancel := context.WithCancel(ctx)
	server := &Server{
		Ctx:           ctx,
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
		BeaconDB:      db,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconChain_StreamBlocksServer(ctrl)
	mockStream.EXPECT().Context().Return(ctx)
	go func(tt *testing.T) {
		assert.ErrorContains(tt, "Context canceled", server.StreamBlocks(&ethpb.StreamBlocksRequest{
			VerifiedOnly: true,
		}, mockStream))
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestServer_StreamBlocks_ContextCanceled(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	chainService := &chainMock.ChainService{}
	ctx, cancel := context.WithCancel(ctx)
	server := &Server{
		Ctx:           ctx,
		BlockNotifier: chainService.BlockNotifier(),
		HeadFetcher:   chainService,
		BeaconDB:      db,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconChain_StreamBlocksServer(ctrl)
	mockStream.EXPECT().Context().Return(ctx)
	go func(tt *testing.T) {
		assert.ErrorContains(tt, "Context canceled", server.StreamBlocks(&ethpb.StreamBlocksRequest{}, mockStream))
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestServer_StreamBlocks_OnHeadUpdated(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.UseMainnetConfig()
	ctx := context.Background()
	beaconState, privs := util.DeterministicGenesisState(t, 32)
	b, err := util.GenerateFullBlock(beaconState, privs, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)
	chainService := &chainMock.ChainService{State: beaconState}
	server := &Server{
		Ctx:           ctx,
		BlockNotifier: chainService.BlockNotifier(),
		HeadFetcher:   chainService,
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconChain_StreamBlocksServer(ctrl)
	mockStream.EXPECT().Send(b).Do(func(arg0 interface{}) {
		exitRoutine <- true
	})
	mockStream.EXPECT().Context().Return(ctx).AnyTimes()

	go func(tt *testing.T) {
		assert.NoError(tt, server.StreamBlocks(&ethpb.StreamBlocksRequest{}, mockStream), "Could not call RPC method")
	}(t)

	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		wsb, err := wrapper.WrappedSignedBeaconBlock(b)
		require.NoError(t, err)
		sent = server.BlockNotifier.BlockFeed().Send(&feed.Event{
			Type: blockfeed.ReceivedBlock,
			Data: &blockfeed.ReceivedBlockData{SignedBlock: wsb},
		})
	}
	<-exitRoutine
}

func TestServer_StreamBlocksVerified_OnHeadUpdated(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.UseMainnetConfig()
	db := dbTest.SetupDB(t)
	ctx := context.Background()
	beaconState, privs := util.DeterministicGenesisState(t, 32)
	b, err := util.GenerateFullBlock(beaconState, privs, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wsb))
	chainService := &chainMock.ChainService{State: beaconState}
	server := &Server{
		Ctx:           ctx,
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
		BeaconDB:      db,
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconChain_StreamBlocksServer(ctrl)
	mockStream.EXPECT().Send(b).Do(func(arg0 interface{}) {
		exitRoutine <- true
	})
	mockStream.EXPECT().Context().Return(ctx).AnyTimes()

	go func(tt *testing.T) {
		assert.NoError(tt, server.StreamBlocks(&ethpb.StreamBlocksRequest{
			VerifiedOnly: true,
		}, mockStream), "Could not call RPC method")
	}(t)

	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		wsb, err := wrapper.WrappedSignedBeaconBlock(b)
		require.NoError(t, err)
		sent = server.StateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.BlockProcessed,
			Data: &statefeed.BlockProcessedData{Slot: b.Block.Slot, BlockRoot: r, SignedBlock: wsb},
		})
	}
	<-exitRoutine
}

func TestServer_ListBeaconBlocks_NoResults(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	wanted := &ethpb.ListBeaconBlocksResponse{
		BlockContainers: make([]*ethpb.BeaconBlockContainer, 0),
		TotalSize:       int32(0),
		NextPageToken:   strconv.Itoa(0),
	}
	res, err := bs.ListBeaconBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Slot{
			Slot: 0,
		},
	})
	require.NoError(t, err)
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
	res, err = bs.ListBeaconBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Slot{
			Slot: 0,
		},
	})
	require.NoError(t, err)
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
	res, err = bs.ListBeaconBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Root{
			Root: make([]byte, 32),
		},
	})
	require.NoError(t, err)
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

func TestServer_ListBeaconBlocks_Genesis(t *testing.T) {
	t.Run("phase 0 block", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		blkContainer := &ethpb.BeaconBlockContainer{
			Block: &ethpb.BeaconBlockContainer_Phase0Block{Phase0Block: blk}}
		wrappedB, err := wrapper.WrappedSignedBeaconBlock(blk)
		assert.NoError(t, err)
		runListBlocksGenesis(t, wrappedB, blkContainer)
	})
	t.Run("altair block", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlockAltair()
		blk.Block.ParentRoot = parentRoot[:]
		wrapped, err := wrapper.WrappedSignedBeaconBlock(blk)
		assert.NoError(t, err)
		blkContainer := &ethpb.BeaconBlockContainer{
			Block: &ethpb.BeaconBlockContainer_AltairBlock{AltairBlock: blk}}
		runListBlocksGenesis(t, wrapped, blkContainer)
	})
	t.Run("bellatrix block", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlockBellatrix()
		blk.Block.ParentRoot = parentRoot[:]
		wrapped, err := wrapper.WrappedSignedBeaconBlock(blk)
		assert.NoError(t, err)
		blkContainer := &ethpb.BeaconBlockContainer{
			Block: &ethpb.BeaconBlockContainer_BellatrixBlock{BellatrixBlock: blk}}
		runListBlocksGenesis(t, wrapped, blkContainer)
	})
}

func runListBlocksGenesis(t *testing.T, blk block.SignedBeaconBlock, blkContainer *ethpb.BeaconBlockContainer) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}

	// Should throw an error if no genesis block is found.
	_, err := bs.ListBeaconBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Genesis{
			Genesis: true,
		},
	})
	require.ErrorContains(t, "Could not find genesis", err)

	root, err := blk.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, blk))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))
	blkContainer.BlockRoot = root[:]
	blkContainer.Canonical = true

	wanted := &ethpb.ListBeaconBlocksResponse{
		BlockContainers: []*ethpb.BeaconBlockContainer{blkContainer},
		NextPageToken:   "0",
		TotalSize:       1,
	}
	res, err := bs.ListBeaconBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Genesis{
			Genesis: true,
		},
	})
	require.NoError(t, err)
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

func TestServer_ListBeaconBlocks_Genesis_MultiBlocks(t *testing.T) {
	t.Run("phase 0 block", func(t *testing.T) {
		parentRoot := [32]byte{1, 2, 3}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		blockCreator := func(i types.Slot) block.SignedBeaconBlock {
			b := util.NewBeaconBlock()
			b.Block.Slot = i
			wrappedB, err := wrapper.WrappedSignedBeaconBlock(b)
			assert.NoError(t, err)
			return wrappedB
		}
		genB, err := wrapper.WrappedSignedBeaconBlock(blk)
		assert.NoError(t, err)
		runListBeaconBlocksGenesisMultiBlocks(t, genB, blockCreator)
	})
	t.Run("altair block", func(t *testing.T) {
		parentRoot := [32]byte{1, 2, 3}
		blk := util.NewBeaconBlockAltair()
		blk.Block.ParentRoot = parentRoot[:]
		blockCreator := func(i types.Slot) block.SignedBeaconBlock {
			b := util.NewBeaconBlockAltair()
			b.Block.Slot = i
			wrappedB, err := wrapper.WrappedSignedBeaconBlock(b)
			assert.NoError(t, err)
			return wrappedB
		}
		gBlock, err := wrapper.WrappedSignedBeaconBlock(blk)
		assert.NoError(t, err)
		runListBeaconBlocksGenesisMultiBlocks(t, gBlock, blockCreator)
	})
	t.Run("bellatrix block", func(t *testing.T) {
		parentRoot := [32]byte{1, 2, 3}
		blk := util.NewBeaconBlockBellatrix()
		blk.Block.ParentRoot = parentRoot[:]
		blockCreator := func(i types.Slot) block.SignedBeaconBlock {
			b := util.NewBeaconBlockBellatrix()
			b.Block.Slot = i
			wrappedB, err := wrapper.WrappedSignedBeaconBlock(b)
			assert.NoError(t, err)
			return wrappedB
		}
		gBlock, err := wrapper.WrappedSignedBeaconBlock(blk)
		assert.NoError(t, err)
		runListBeaconBlocksGenesisMultiBlocks(t, gBlock, blockCreator)
	})
}

func runListBeaconBlocksGenesisMultiBlocks(t *testing.T, genBlock block.SignedBeaconBlock,
	blockCreator func(i types.Slot) block.SignedBeaconBlock) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	// Should return the proper genesis block if it exists.
	root, err := genBlock.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, genBlock))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

	count := types.Slot(100)
	blks := make([]block.SignedBeaconBlock, count)
	for i := types.Slot(0); i < count; i++ {
		blks[i] = blockCreator(i)
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))

	// Should throw an error if more than one blk returned.
	_, err = bs.ListBeaconBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Genesis{
			Genesis: true,
		},
	})
	require.NoError(t, err)
}

func TestServer_ListBeaconBlocks_Pagination(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	t.Run("phase 0 block", func(t *testing.T) {
		blk := util.NewBeaconBlock()
		blk.Block.Slot = 300
		blockCreator := func(i types.Slot) block.SignedBeaconBlock {
			b := util.NewBeaconBlock()
			b.Block.Slot = i
			wrappedB, err := wrapper.WrappedSignedBeaconBlock(b)
			assert.NoError(t, err)
			return wrappedB
		}
		containerCreator := func(i types.Slot, root []byte, canonical bool) *ethpb.BeaconBlockContainer {
			b := util.NewBeaconBlock()
			b.Block.Slot = i
			ctr := &ethpb.BeaconBlockContainer{
				Block: &ethpb.BeaconBlockContainer_Phase0Block{
					Phase0Block: util.HydrateSignedBeaconBlock(b)},
				BlockRoot: root,
				Canonical: canonical}
			return ctr
		}
		wrappedB, err := wrapper.WrappedSignedBeaconBlock(blk)
		assert.NoError(t, err)
		runListBeaconBlocksPagination(t, wrappedB, blockCreator, containerCreator)
	})
	t.Run("altair block", func(t *testing.T) {
		blk := util.NewBeaconBlockAltair()
		blk.Block.Slot = 300
		blockCreator := func(i types.Slot) block.SignedBeaconBlock {
			b := util.NewBeaconBlockAltair()
			b.Block.Slot = i
			wrappedB, err := wrapper.WrappedSignedBeaconBlock(b)
			assert.NoError(t, err)
			return wrappedB
		}
		containerCreator := func(i types.Slot, root []byte, canonical bool) *ethpb.BeaconBlockContainer {
			b := util.NewBeaconBlockAltair()
			b.Block.Slot = i
			ctr := &ethpb.BeaconBlockContainer{
				Block: &ethpb.BeaconBlockContainer_AltairBlock{
					AltairBlock: util.HydrateSignedBeaconBlockAltair(b)},
				BlockRoot: root,
				Canonical: canonical}
			return ctr
		}
		orphanedB, err := wrapper.WrappedSignedBeaconBlock(blk)
		assert.NoError(t, err)
		runListBeaconBlocksPagination(t, orphanedB, blockCreator, containerCreator)
	})
	t.Run("bellatrix block", func(t *testing.T) {
		blk := util.NewBeaconBlockBellatrix()
		blk.Block.Slot = 300
		blockCreator := func(i types.Slot) block.SignedBeaconBlock {
			b := util.NewBeaconBlockBellatrix()
			b.Block.Slot = i
			wrappedB, err := wrapper.WrappedSignedBeaconBlock(b)
			assert.NoError(t, err)
			return wrappedB
		}
		containerCreator := func(i types.Slot, root []byte, canonical bool) *ethpb.BeaconBlockContainer {
			b := util.NewBeaconBlockBellatrix()
			b.Block.Slot = i
			ctr := &ethpb.BeaconBlockContainer{
				Block: &ethpb.BeaconBlockContainer_BellatrixBlock{
					BellatrixBlock: util.HydrateSignedBeaconBlockBellatrix(b)},
				BlockRoot: root,
				Canonical: canonical}
			return ctr
		}
		orphanedB, err := wrapper.WrappedSignedBeaconBlock(blk)
		assert.NoError(t, err)
		runListBeaconBlocksPagination(t, orphanedB, blockCreator, containerCreator)
	})
}

func runListBeaconBlocksPagination(t *testing.T, orphanedBlk block.SignedBeaconBlock,
	blockCreator func(i types.Slot) block.SignedBeaconBlock, containerCreator func(i types.Slot, root []byte, canonical bool) *ethpb.BeaconBlockContainer) {

	db := dbTest.SetupDB(t)
	chain := &chainMock.ChainService{
		CanonicalRoots: map[[32]byte]bool{},
	}
	ctx := context.Background()

	count := types.Slot(100)
	blks := make([]block.SignedBeaconBlock, count)
	blkContainers := make([]*ethpb.BeaconBlockContainer, count)
	for i := types.Slot(0); i < count; i++ {
		b := blockCreator(i)
		root, err := b.Block().HashTreeRoot()
		require.NoError(t, err)
		chain.CanonicalRoots[root] = true
		blks[i] = b
		ctr, err := convertToBlockContainer(blks[i], root, true)
		require.NoError(t, err)
		blkContainers[i] = ctr
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))

	orphanedBlkRoot, err := orphanedBlk.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, orphanedBlk))

	bs := &Server{
		BeaconDB:         db,
		CanonicalFetcher: chain,
	}

	root6, err := blks[6].Block().HashTreeRoot()
	require.NoError(t, err)

	tests := []struct {
		req *ethpb.ListBlocksRequest
		res *ethpb.ListBeaconBlocksResponse
	}{
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Slot{Slot: 5},
			PageSize:    3},
			res: &ethpb.ListBeaconBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{containerCreator(5, blkContainers[5].BlockRoot, blkContainers[5].Canonical)},
				NextPageToken:   "",
				TotalSize:       1,
			},
		},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Root{Root: root6[:]},
			PageSize:    3},
			res: &ethpb.ListBeaconBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{containerCreator(6, blkContainers[6].BlockRoot, blkContainers[6].Canonical)},
				TotalSize:       1,
				NextPageToken:   strconv.Itoa(0)}},
		{req: &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: root6[:]}},
			res: &ethpb.ListBeaconBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{containerCreator(6, blkContainers[6].BlockRoot, blkContainers[6].Canonical)},
				TotalSize:       1, NextPageToken: strconv.Itoa(0)}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: 0},
			PageSize:    100},
			res: &ethpb.ListBeaconBlocksResponse{
				BlockContainers: blkContainers[0:params.BeaconConfig().SlotsPerEpoch],
				NextPageToken:   "",
				TotalSize:       int32(params.BeaconConfig().SlotsPerEpoch)}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(1),
			QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: 5},
			PageSize:    3},
			res: &ethpb.ListBeaconBlocksResponse{
				BlockContainers: blkContainers[43:46],
				NextPageToken:   "2",
				TotalSize:       int32(params.BeaconConfig().SlotsPerEpoch)}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(1),
			QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: 11},
			PageSize:    7},
			res: &ethpb.ListBeaconBlocksResponse{
				BlockContainers: blkContainers[95:96],
				NextPageToken:   "",
				TotalSize:       int32(params.BeaconConfig().SlotsPerEpoch)}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: 12},
			PageSize:    4},
			res: &ethpb.ListBeaconBlocksResponse{
				BlockContainers: blkContainers[96:100],
				NextPageToken:   "",
				TotalSize:       int32(params.BeaconConfig().SlotsPerEpoch / 2)}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Slot{Slot: 300},
			PageSize:    3},
			res: &ethpb.ListBeaconBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{containerCreator(300, orphanedBlkRoot[:], false)},
				NextPageToken:   "",
				TotalSize:       1}},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			res, err := bs.ListBeaconBlocks(ctx, test.req)
			require.NoError(t, err)
			require.DeepSSZEqual(t, res, test.res)
		})
	}
}

func TestServer_ListBeaconBlocks_Errors(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	exceedsMax := int32(cmd.Get().MaxRPCPageSize + 1)

	wanted := fmt.Sprintf("Requested page size %d can not be greater than max size %d", exceedsMax, cmd.Get().MaxRPCPageSize)
	req := &ethpb.ListBlocksRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	_, err := bs.ListBlocks(ctx, req)
	assert.ErrorContains(t, wanted, err)

	wanted = "Must specify a filter criteria for fetching"
	req = &ethpb.ListBlocksRequest{}
	_, err = bs.ListBeaconBlocks(ctx, req)
	assert.ErrorContains(t, wanted, err)

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Slot{Slot: 0}}
	res, err := bs.ListBeaconBlocks(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 0, len(res.BlockContainers), "Wanted empty list")
	assert.Equal(t, int32(0), res.TotalSize, "Wanted total size 0")

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Slot{}}
	res, err = bs.ListBeaconBlocks(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 0, len(res.BlockContainers), "Wanted empty list")
	assert.Equal(t, int32(0), res.TotalSize, "Wanted total size 0")

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: []byte{'A'}}}
	res, err = bs.ListBeaconBlocks(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 0, len(res.BlockContainers), "Wanted empty list")
	assert.Equal(t, int32(0), res.TotalSize, "Wanted total size 0")

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: []byte{'A'}}}
	res, err = bs.ListBeaconBlocks(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 0, len(res.BlockContainers), "Wanted empty list")
	assert.Equal(t, int32(0), res.TotalSize, "Wanted total size 0")
}
