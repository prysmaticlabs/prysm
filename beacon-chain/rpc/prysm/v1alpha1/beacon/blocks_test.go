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
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/cmd"
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
			Root: make([]byte, 32),
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
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk)))
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
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk)))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

	count := types.Slot(100)
	blks := make([]block.SignedBeaconBlock, count)
	for i := types.Slot(0); i < count; i++ {
		b := util.NewBeaconBlock()
		b.Block.Slot = i
		require.NoError(t, err)
		blks[i] = wrapper.WrappedPhase0SignedBeaconBlock(b)
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
		blks[i] = wrapper.WrappedPhase0SignedBeaconBlock(b)
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
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(orphanedBlk)))

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

func TestServer_GetChainHead_NoFinalizedBlock(t *testing.T) {
	db := dbTest.SetupDB(t)

	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetSlot(1))
	require.NoError(t, s.SetPreviousJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 3, Root: bytesutil.PadTo([]byte{'A'}, 32)}))
	require.NoError(t, s.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 2, Root: bytesutil.PadTo([]byte{'B'}, 32)}))
	require.NoError(t, s.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'C'}, 32)}))

	genBlock := util.NewBeaconBlock()
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(genBlock)))
	gRoot, err := genBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), gRoot))

	bs := &Server{
		BeaconDB:    db,
		HeadFetcher: &chainMock.ChainService{Block: wrapper.WrappedPhase0SignedBeaconBlock(genBlock), State: s},
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
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(genBlock)))
	gRoot, err := genBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), gRoot))

	finalizedBlock := util.NewBeaconBlock()
	finalizedBlock.Block.Slot = 1
	finalizedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'A'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(finalizedBlock)))
	fRoot, err := finalizedBlock.Block.HashTreeRoot()
	require.NoError(t, err)

	justifiedBlock := util.NewBeaconBlock()
	justifiedBlock.Block.Slot = 2
	justifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'B'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(justifiedBlock)))
	jRoot, err := justifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err)

	prevJustifiedBlock := util.NewBeaconBlock()
	prevJustifiedBlock.Block.Slot = 3
	prevJustifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'C'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(prevJustifiedBlock)))
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
	bs := &Server{
		BeaconDB:    db,
		HeadFetcher: &chainMock.ChainService{Block: wrapper.WrappedPhase0SignedBeaconBlock(b), State: s},
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
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(genBlock)))
	gRoot, err := genBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), gRoot))

	finalizedBlock := util.NewBeaconBlock()
	finalizedBlock.Block.Slot = 32
	finalizedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'A'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(finalizedBlock)))
	fRoot, err := finalizedBlock.Block.HashTreeRoot()
	require.NoError(t, err)

	justifiedBlock := util.NewBeaconBlock()
	justifiedBlock.Block.Slot = 64
	justifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'B'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(justifiedBlock)))
	jRoot, err := justifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err)

	prevJustifiedBlock := util.NewBeaconBlock()
	prevJustifiedBlock.Block.Slot = 96
	prevJustifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'C'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(prevJustifiedBlock)))
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
	server := &Server{
		Ctx:           ctx,
		HeadFetcher:   &chainMock.ChainService{Block: wrapper.WrappedPhase0SignedBeaconBlock(b), State: s},
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
		sent = server.BlockNotifier.BlockFeed().Send(&feed.Event{
			Type: blockfeed.ReceivedBlock,
			Data: &blockfeed.ReceivedBlockData{SignedBlock: wrapper.WrappedPhase0SignedBeaconBlock(b)},
		})
	}
	<-exitRoutine
}

func TestServer_StreamBlocksVerified_OnHeadUpdated(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()
	beaconState, privs := util.DeterministicGenesisState(t, 32)
	b, err := util.GenerateFullBlock(beaconState, privs, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b)))
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
		sent = server.StateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.BlockProcessed,
			Data: &statefeed.BlockProcessedData{Slot: b.Block.Slot, BlockRoot: r, SignedBlock: wrapper.WrappedPhase0SignedBeaconBlock(b)},
		})
	}
	<-exitRoutine
}

func TestServer_GetWeakSubjectivityCheckpoint(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	db := dbTest.SetupDB(t)
	ctx := context.Background()

	// Beacon state.
	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(10))

	// Active validator set is used for computing the weak subjectivity period.
	numVals := 256 // Works with params.BeaconConfig().MinGenesisActiveValidatorCount as well, but takes longer.
	validators := make([]*ethpb.Validator, numVals)
	balances := make([]uint64, len(validators))
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			PublicKey:             make([]byte, params.BeaconConfig().BLSPubkeyLength),
			WithdrawalCredentials: make([]byte, 32),
			EffectiveBalance:      28 * 1e9,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
		}
		balances[i] = validators[i].EffectiveBalance
	}
	require.NoError(t, beaconState.SetValidators(validators))
	require.NoError(t, beaconState.SetBalances(balances))

	// Genesis block.
	genesisBlock := util.NewBeaconBlock()
	genesisBlockRoot, err := genesisBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesisBlock)))
	require.NoError(t, db.SaveState(ctx, beaconState, genesisBlockRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlockRoot))

	// Finalized checkpoint.
	finalizedEpoch := types.Epoch(1020)
	require.NoError(t, beaconState.SetSlot(types.Slot(finalizedEpoch.Mul(uint64(params.BeaconConfig().SlotsPerEpoch)))))
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: finalizedEpoch - 1,
		Root:  bytesutil.PadTo([]byte{'A'}, 32),
	}))
	require.NoError(t, beaconState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: finalizedEpoch,
		Root:  bytesutil.PadTo([]byte{'B'}, 32),
	}))

	chainService := &chainMock.ChainService{State: beaconState}
	server := &Server{
		Ctx:           ctx,
		BlockNotifier: chainService.BlockNotifier(),
		HeadFetcher:   chainService,
		BeaconDB:      db,
		StateGen:      stategen.New(db),
	}

	wsEpoch, err := helpers.ComputeWeakSubjectivityPeriod(context.Background(), beaconState)
	require.NoError(t, err)

	c, err := server.GetWeakSubjectivityCheckpoint(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	e := finalizedEpoch - (finalizedEpoch % wsEpoch)
	require.Equal(t, e, c.Epoch)
	wsState, err := server.StateGen.StateBySlot(ctx, params.BeaconConfig().SlotsPerEpoch.Mul(uint64(e)))
	require.NoError(t, err)
	sRoot, err := wsState.HashTreeRoot(ctx)
	require.NoError(t, err)
	require.DeepEqual(t, sRoot[:], c.StateRoot)
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

func TestServer_ListBlocksAltair_Genesis(t *testing.T) {
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
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk)))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))
	ctr, err := convertToBlockContainer(wrapper.WrappedPhase0SignedBeaconBlock(blk), root, true)
	assert.NoError(t, err)
	wanted := &ethpb.ListBeaconBlocksResponse{
		BlockContainers: []*ethpb.BeaconBlockContainer{ctr},
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

func TestServer_ListBlocksAltair_Genesis_MultiBlocks(t *testing.T) {
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
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk)))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

	count := types.Slot(100)
	blks := make([]block.SignedBeaconBlock, count)
	for i := types.Slot(0); i < count; i++ {
		b := util.NewBeaconBlock()
		b.Block.Slot = i
		require.NoError(t, err)
		blks[i] = wrapper.WrappedPhase0SignedBeaconBlock(b)
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

func TestServer_ListBlocksAltair_Pagination(t *testing.T) {
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
		blks[i] = wrapper.WrappedPhase0SignedBeaconBlock(b)
		ctr, err := convertToBlockContainer(blks[i], root, true)
		require.NoError(t, err)
		blkContainers[i] = ctr
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))

	orphanedBlk := util.NewBeaconBlock()
	orphanedBlk.Block.Slot = 300
	orphanedBlkRoot, err := orphanedBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(orphanedBlk)))

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
				BlockContainers: []*ethpb.BeaconBlockContainer{{Block: &ethpb.BeaconBlockContainer_Phase0Block{Phase0Block: util.HydrateSignedBeaconBlock(&ethpb.SignedBeaconBlock{
					Block: &ethpb.BeaconBlock{
						Slot: 5}})},
					BlockRoot: blkContainers[5].BlockRoot,
					Canonical: blkContainers[5].Canonical}},
				NextPageToken: "",
				TotalSize:     1,
			},
		},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Root{Root: root6[:]},
			PageSize:    3},
			res: &ethpb.ListBeaconBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{{Block: &ethpb.BeaconBlockContainer_Phase0Block{
					Phase0Block: util.HydrateSignedBeaconBlock(&ethpb.SignedBeaconBlock{
						Block: &ethpb.BeaconBlock{
							Slot: 6}})},
					BlockRoot: blkContainers[6].BlockRoot,
					Canonical: blkContainers[6].Canonical}},
				TotalSize: 1, NextPageToken: strconv.Itoa(0)}},
		{req: &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: root6[:]}},
			res: &ethpb.ListBeaconBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{{Block: &ethpb.BeaconBlockContainer_Phase0Block{
					Phase0Block: util.HydrateSignedBeaconBlock(&ethpb.SignedBeaconBlock{
						Block: &ethpb.BeaconBlock{
							Slot: 6}})},
					BlockRoot: blkContainers[6].BlockRoot,
					Canonical: blkContainers[6].Canonical}},
				TotalSize: 1, NextPageToken: strconv.Itoa(0)}},
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
				BlockContainers: []*ethpb.BeaconBlockContainer{{Block: &ethpb.BeaconBlockContainer_Phase0Block{
					Phase0Block: util.HydrateSignedBeaconBlock(&ethpb.SignedBeaconBlock{
						Block: &ethpb.BeaconBlock{
							Slot: 300}})},
					BlockRoot: orphanedBlkRoot[:],
					Canonical: false}},
				NextPageToken: "",
				TotalSize:     1}},
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
