package beacon

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestServer_ListBlocks_NoResults(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
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
	db, _ := dbTest.SetupDB(t)
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
	blk := testutil.NewBeaconBlock()
	blk.Block.ParentRoot = parentRoot[:]
	root, err := stateutil.BlockRoot(blk.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, blk))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))
	wanted := &ethpb.ListBlocksResponse{
		BlockContainers: []*ethpb.BeaconBlockContainer{
			{
				Block:     blk,
				BlockRoot: root[:],
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
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	// Should return the proper genesis block if it exists.
	parentRoot := [32]byte{1, 2, 3}
	blk := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       0,
			ParentRoot: parentRoot[:],
		},
	}
	root, err := stateutil.BlockRoot(blk.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, blk))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

	count := uint64(100)
	blks := make([]*ethpb.SignedBeaconBlock, count)
	blkContainers := make([]*ethpb.BeaconBlockContainer, count)
	for i := uint64(0); i < count; i++ {
		b := &ethpb.SignedBeaconBlock{
			Block: &ethpb.BeaconBlock{
				Slot: i,
			},
		}
		root, err := stateutil.BlockRoot(b.Block)
		require.NoError(t, err)
		blks[i] = b
		blkContainers[i] = &ethpb.BeaconBlockContainer{Block: b, BlockRoot: root[:]}
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
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	count := uint64(100)
	blks := make([]*ethpb.SignedBeaconBlock, count)
	blkContainers := make([]*ethpb.BeaconBlockContainer, count)
	for i := uint64(0); i < count; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		root, err := stateutil.BlockRoot(b.Block)
		require.NoError(t, err)
		blks[i] = b
		blkContainers[i] = &ethpb.BeaconBlockContainer{Block: b, BlockRoot: root[:]}
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))

	bs := &Server{
		BeaconDB: db,
	}

	root6, err := stateutil.BlockRoot(blks[6].Block)
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
				BlockContainers: []*ethpb.BeaconBlockContainer{{Block: &ethpb.SignedBeaconBlock{
					Signature: make([]byte, 96),
					Block: &ethpb.BeaconBlock{
						ParentRoot: make([]byte, 32),
						StateRoot:  make([]byte, 32),
						Body: &ethpb.BeaconBlockBody{
							RandaoReveal: make([]byte, 96),
							Graffiti:     make([]byte, 32),
							Eth1Data: &ethpb.Eth1Data{
								BlockHash:   make([]byte, 32),
								DepositRoot: make([]byte, 32),
							},
						},
						Slot: 5}},
					BlockRoot: blkContainers[5].BlockRoot}},
				NextPageToken: "",
				TotalSize:     1}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Root{Root: root6[:]},
			PageSize:    3},
			res: &ethpb.ListBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{{Block: &ethpb.SignedBeaconBlock{
					Signature: make([]byte, 96),
					Block: &ethpb.BeaconBlock{
						ParentRoot: make([]byte, 32),
						StateRoot:  make([]byte, 32),
						Body: &ethpb.BeaconBlockBody{
							RandaoReveal: make([]byte, 96),
							Graffiti:     make([]byte, 32),
							Eth1Data: &ethpb.Eth1Data{
								BlockHash:   make([]byte, 32),
								DepositRoot: make([]byte, 32),
							},
						},
						Slot: 6}},
					BlockRoot: blkContainers[6].BlockRoot}},
				TotalSize: 1}},
		{req: &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: root6[:]}},
			res: &ethpb.ListBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{{Block: &ethpb.SignedBeaconBlock{
					Signature: make([]byte, 96),
					Block: &ethpb.BeaconBlock{
						ParentRoot: make([]byte, 32),
						StateRoot:  make([]byte, 32),
						Body: &ethpb.BeaconBlockBody{
							RandaoReveal: make([]byte, 96),
							Graffiti:     make([]byte, 32),
							Eth1Data: &ethpb.Eth1Data{
								BlockHash:   make([]byte, 32),
								DepositRoot: make([]byte, 32),
							},
						},
						Slot: 6}},
					BlockRoot: blkContainers[6].BlockRoot}},
				TotalSize: 1}},
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
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			res, err := bs.ListBlocks(ctx, test.req)
			require.NoError(t, err)
			if !proto.Equal(res, test.res) {
				t.Errorf("Incorrect blocks response, wanted %v, received %v", test.res, res)
			}
		})
	}
}

func TestServer_ListBlocks_Errors(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
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
	db, _ := dbTest.SetupDB(t)

	s, err := stateTrie.InitializeFromProto(&pbp2p.BeaconState{
		Slot:                        1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 3, Root: []byte{'A'}},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Epoch: 2, Root: []byte{'B'}},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Epoch: 1, Root: []byte{'C'}},
	})
	require.NoError(t, err)

	genBlock := testutil.NewBeaconBlock()
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), genBlock))
	gRoot, err := stateutil.BlockRoot(genBlock.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), gRoot))

	bs := &Server{
		BeaconDB:    db,
		HeadFetcher: &chainMock.ChainService{Block: &ethpb.SignedBeaconBlock{}, State: s},
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
	db, _ := dbTest.SetupDB(t)

	genBlock := testutil.NewBeaconBlock()
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), genBlock))
	gRoot, err := stateutil.BlockRoot(genBlock.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), gRoot))

	finalizedBlock := testutil.NewBeaconBlock()
	finalizedBlock.Block.Slot = 1
	finalizedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'A'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), finalizedBlock))
	fRoot, err := stateutil.BlockRoot(finalizedBlock.Block)
	require.NoError(t, err)

	justifiedBlock := testutil.NewBeaconBlock()
	justifiedBlock.Block.Slot = 2
	justifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'B'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), justifiedBlock))
	jRoot, err := stateutil.BlockRoot(justifiedBlock.Block)
	require.NoError(t, err)

	prevJustifiedBlock := testutil.NewBeaconBlock()
	prevJustifiedBlock.Block.Slot = 3
	prevJustifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'C'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), prevJustifiedBlock))
	pjRoot, err := stateutil.BlockRoot(prevJustifiedBlock.Block)
	require.NoError(t, err)

	s, err := stateTrie.InitializeFromProto(&pbp2p.BeaconState{
		Slot:                        1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 3, Root: pjRoot[:]},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Epoch: 2, Root: jRoot[:]},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Epoch: 1, Root: fRoot[:]},
	})
	require.NoError(t, err)

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.PreviousJustifiedCheckpoint().Epoch*params.BeaconConfig().SlotsPerEpoch + 1}}
	bs := &Server{
		BeaconDB:    db,
		HeadFetcher: &chainMock.ChainService{Block: b, State: s},
		FinalizationFetcher: &chainMock.ChainService{
			FinalizedCheckPoint:         s.FinalizedCheckpoint(),
			CurrentJustifiedCheckPoint:  s.CurrentJustifiedCheckpoint(),
			PreviousJustifiedCheckPoint: s.PreviousJustifiedCheckpoint()},
	}

	head, err := bs.GetChainHead(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(3), head.PreviousJustifiedEpoch, "Unexpected PreviousJustifiedEpoch")
	assert.Equal(t, uint64(2), head.JustifiedEpoch, "Unexpected JustifiedEpoch")
	assert.Equal(t, uint64(1), head.FinalizedEpoch, "Unexpected FinalizedEpoch")
	assert.Equal(t, uint64(24), head.PreviousJustifiedSlot, "Unexpected PreviousJustifiedSlot")
	assert.Equal(t, uint64(16), head.JustifiedSlot, "Unexpected JustifiedSlot")
	assert.Equal(t, uint64(8), head.FinalizedSlot, "Unexpected FinalizedSlot")
	assert.DeepEqual(t, pjRoot[:], head.PreviousJustifiedBlockRoot, "Unexpected PreviousJustifiedBlockRoot")
	assert.DeepEqual(t, jRoot[:], head.JustifiedBlockRoot, "Unexpected JustifiedBlockRoot")
	assert.DeepEqual(t, fRoot[:], head.FinalizedBlockRoot, "Unexpected FinalizedBlockRoot")
}

func TestServer_StreamChainHead_ContextCanceled(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
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
		assert.ErrorContains(tt, "Context canceled", server.StreamChainHead(&ptypes.Empty{}, mockStream))
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestServer_StreamChainHead_OnHeadUpdated(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	params.UseMainnetConfig()
	genBlock := testutil.NewBeaconBlock()
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), genBlock))
	gRoot, err := stateutil.BlockRoot(genBlock.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), gRoot))

	finalizedBlock := testutil.NewBeaconBlock()
	finalizedBlock.Block.Slot = 32
	finalizedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'A'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), finalizedBlock))
	fRoot, err := stateutil.BlockRoot(finalizedBlock.Block)
	require.NoError(t, err)

	justifiedBlock := testutil.NewBeaconBlock()
	justifiedBlock.Block.Slot = 64
	justifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'B'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), justifiedBlock))
	jRoot, err := stateutil.BlockRoot(justifiedBlock.Block)
	require.NoError(t, err)

	prevJustifiedBlock := testutil.NewBeaconBlock()
	prevJustifiedBlock.Block.Slot = 96
	prevJustifiedBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'C'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), prevJustifiedBlock))
	pjRoot, err := stateutil.BlockRoot(prevJustifiedBlock.Block)
	require.NoError(t, err)

	s, err := stateTrie.InitializeFromProto(&pbp2p.BeaconState{
		Slot:                        1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 3, Root: pjRoot[:]},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Epoch: 2, Root: jRoot[:]},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Epoch: 1, Root: fRoot[:]},
	})
	require.NoError(t, err)

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.PreviousJustifiedCheckpoint().Epoch*params.BeaconConfig().SlotsPerEpoch + 1}}
	hRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)

	chainService := &chainMock.ChainService{}
	ctx := context.Background()
	server := &Server{
		Ctx:           ctx,
		HeadFetcher:   &chainMock.ChainService{Block: b, State: s},
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
			HeadEpoch:                  helpers.SlotToEpoch(b.Block.Slot),
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
		assert.NoError(tt, server.StreamChainHead(&ptypes.Empty{}, mockStream), "Could not call RPC method")
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

func TestServer_StreamBlocks_ContextCanceled(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
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
		assert.ErrorContains(tt, "Context canceled", server.StreamBlocks(&ptypes.Empty{}, mockStream))
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestServer_StreamBlocks_OnHeadUpdated(t *testing.T) {
	ctx := context.Background()
	beaconState, privs := testutil.DeterministicGenesisState(t, 32)
	b, err := testutil.GenerateFullBlock(beaconState, privs, testutil.DefaultBlockGenConfig(), 1)
	if err != nil {
		t.Fatal(err)
	}
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
		assert.NoError(tt, server.StreamBlocks(&ptypes.Empty{}, mockStream), "Could not call RPC method")
	}(t)

	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = server.BlockNotifier.BlockFeed().Send(&feed.Event{
			Type: blockfeed.ReceivedBlock,
			Data: &blockfeed.ReceivedBlockData{SignedBlock: b},
		})
	}
	<-exitRoutine
}
