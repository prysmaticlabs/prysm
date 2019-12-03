package beacon

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/statefeed"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockRPC "github.com/prysmaticlabs/prysm/beacon-chain/rpc/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestServer_ListBlocks_NoResults(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

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
		QueryFilter: &ethpb.ListBlocksRequest_Epoch{
			Epoch: 0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
	res, err = bs.ListBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Slot{
			Slot: 0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
	res, err = bs.ListBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Root{
			Root: make([]byte, 32),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

func TestServer_ListBlocks_Genesis(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	ctx := context.Background()
	bs := &Server{
		BeaconDB: db,
	}

	// Should throw an error if no genesis block is found.
	if _, err := bs.ListBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Genesis{
			Genesis: true,
		},
	}); err != nil && !strings.Contains(err.Error(), "Could not find genesis") {
		t.Fatal(err)
	}

	// Should return the proper genesis block if it exists.
	parentRoot := [32]byte{1, 2, 3}
	blk := &ethpb.BeaconBlock{
		Slot:       0,
		ParentRoot: parentRoot[:],
	}
	root, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}

	// Should throw an error if there is more than 1 block
	// for the genesis slot.
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}
	if _, err := bs.ListBlocks(ctx, &ethpb.ListBlocksRequest{
		QueryFilter: &ethpb.ListBlocksRequest_Genesis{
			Genesis: true,
		},
	}); err != nil && !strings.Contains(err.Error(), "Found more than 1") {
		t.Fatal(err)
	}
}

func TestServer_ListBlocks_Pagination(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()

	count := uint64(100)
	blks := make([]*ethpb.BeaconBlock, count)
	blkContainers := make([]*ethpb.BeaconBlockContainer, count)
	for i := uint64(0); i < count; i++ {
		b := &ethpb.BeaconBlock{
			Slot: i,
		}
		root, err := ssz.SigningRoot(b)
		if err != nil {
			t.Fatal(err)
		}
		blks[i] = b
		blkContainers[i] = &ethpb.BeaconBlockContainer{Block: b, BlockRoot: root[:]}
	}
	if err := db.SaveBlocks(ctx, blks); err != nil {
		t.Fatal(err)
	}

	root6, err := ssz.SigningRoot(&ethpb.BeaconBlock{Slot: 6})
	if err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		BeaconDB: db,
	}

	tests := []struct {
		req *ethpb.ListBlocksRequest
		res *ethpb.ListBlocksResponse
	}{
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Slot{Slot: 5},
			PageSize:    3},
			res: &ethpb.ListBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{{Block: &ethpb.BeaconBlock{Slot: 5}, BlockRoot: blkContainers[5].BlockRoot}},
				NextPageToken:   "",
				TotalSize:       1}},
		{req: &ethpb.ListBlocksRequest{
			PageToken:   strconv.Itoa(0),
			QueryFilter: &ethpb.ListBlocksRequest_Root{Root: root6[:]},
			PageSize:    3},
			res: &ethpb.ListBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{{Block: &ethpb.BeaconBlock{Slot: 6}, BlockRoot: blkContainers[6].BlockRoot}},
				TotalSize:       1}},
		{req: &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: root6[:]}},
			res: &ethpb.ListBlocksResponse{
				BlockContainers: []*ethpb.BeaconBlockContainer{{Block: &ethpb.BeaconBlock{Slot: 6}, BlockRoot: blkContainers[6].BlockRoot}},
				TotalSize:       1}},
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
				NextPageToken:   "1",
				TotalSize:       int32(params.BeaconConfig().SlotsPerEpoch / 2)}},
	}

	for _, test := range tests {
		res, err := bs.ListBlocks(ctx, test.req)
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(res, test.res) {
			t.Errorf("Incorrect blocks response, wanted %v, received %v", test.res, res)
		}
	}
}

func TestServer_ListBlocks_Errors(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()

	bs := &Server{BeaconDB: db}
	exceedsMax := int32(params.BeaconConfig().MaxPageSize + 1)

	wanted := fmt.Sprintf("Requested page size %d can not be greater than max size %d", exceedsMax, params.BeaconConfig().MaxPageSize)
	req := &ethpb.ListBlocksRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	if _, err := bs.ListBlocks(ctx, req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}

	wanted = "Must specify a filter criteria for fetching"
	req = &ethpb.ListBlocksRequest{}
	if _, err := bs.ListBlocks(ctx, req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Epoch{}}
	res, err := bs.ListBlocks(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.BlockContainers) != 0 {
		t.Errorf("wanted empty list, got a list of %d", len(res.BlockContainers))
	}
	if res.TotalSize != 0 {
		t.Errorf("wanted total size 0, got size %d", res.TotalSize)
	}

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Slot{}}
	res, err = bs.ListBlocks(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.BlockContainers) != 0 {
		t.Errorf("wanted empty list, got a list of %d", len(res.BlockContainers))
	}
	if res.TotalSize != 0 {
		t.Errorf("wanted total size 0, got size %d", res.TotalSize)

	}

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: []byte{'A'}}}
	res, err = bs.ListBlocks(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.BlockContainers) != 0 {
		t.Errorf("wanted empty list, got a list of %d", len(res.BlockContainers))
	}
	if res.TotalSize != 0 {
		t.Errorf("wanted total size 0, got size %d", res.TotalSize)

	}

	req = &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Root{Root: []byte{'A'}}}
	res, err = bs.ListBlocks(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.BlockContainers) != 0 {
		t.Errorf("wanted empty list, got a list of %d", len(res.BlockContainers))
	}
	if res.TotalSize != 0 {
		t.Errorf("wanted total size 0, got size %d", res.TotalSize)

	}
}

func TestServer_GetChainHead_NoFinalizedBlock(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	s := &pbp2p.BeaconState{
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 3, Root: []byte{'A'}},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Epoch: 2, Root: []byte{'B'}},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Epoch: 1, Root: []byte{'C'}},
	}

	bs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mock.ChainService{Block: &ethpb.BeaconBlock{}, State: s},
	}

	if _, err := bs.GetChainHead(context.Background(), nil); !strings.Contains(err.Error(), "Could not get finalized block") {
		t.Fatal("Did not get wanted error")
	}
}

func TestServer_GetChainHead(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	finalizedBlock := &ethpb.BeaconBlock{Slot: 1, ParentRoot: []byte{'A'}}
	db.SaveBlock(context.Background(), finalizedBlock)
	fRoot, _ := ssz.SigningRoot(finalizedBlock)
	justifiedBlock := &ethpb.BeaconBlock{Slot: 2, ParentRoot: []byte{'B'}}
	db.SaveBlock(context.Background(), justifiedBlock)
	jRoot, _ := ssz.SigningRoot(justifiedBlock)
	prevJustifiedBlock := &ethpb.BeaconBlock{Slot: 3, ParentRoot: []byte{'C'}}
	db.SaveBlock(context.Background(), prevJustifiedBlock)
	pjRoot, _ := ssz.SigningRoot(prevJustifiedBlock)

	s := &pbp2p.BeaconState{
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 3, Root: pjRoot[:]},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Epoch: 2, Root: jRoot[:]},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Epoch: 1, Root: fRoot[:]},
	}

	b := &ethpb.BeaconBlock{Slot: s.PreviousJustifiedCheckpoint.Epoch*params.BeaconConfig().SlotsPerEpoch + 1}
	bs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mock.ChainService{Block: b, State: s},
	}

	head, err := bs.GetChainHead(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if head.PreviousJustifiedEpoch != 3 {
		t.Errorf("Wanted PreviousJustifiedEpoch: %d, got: %d",
			3*params.BeaconConfig().SlotsPerEpoch, head.PreviousJustifiedEpoch)
	}
	if head.JustifiedEpoch != 2 {
		t.Errorf("Wanted JustifiedEpoch: %d, got: %d",
			2*params.BeaconConfig().SlotsPerEpoch, head.JustifiedEpoch)
	}
	if head.FinalizedEpoch != 1 {
		t.Errorf("Wanted FinalizedEpoch: %d, got: %d",
			1*params.BeaconConfig().SlotsPerEpoch, head.FinalizedEpoch)
	}
	if head.PreviousJustifiedSlot != 3 {
		t.Errorf("Wanted PreviousJustifiedSlot: %d, got: %d",
			3, head.PreviousJustifiedSlot)
	}
	if head.JustifiedSlot != 2 {
		t.Errorf("Wanted JustifiedSlot: %d, got: %d",
			2, head.JustifiedSlot)
	}
	if head.FinalizedSlot != 1 {
		t.Errorf("Wanted FinalizedSlot: %d, got: %d",
			1, head.FinalizedSlot)
	}
	if !bytes.Equal(pjRoot[:], head.PreviousJustifiedBlockRoot) {
		t.Errorf("Wanted PreviousJustifiedBlockRoot: %v, got: %v",
			pjRoot[:], head.PreviousJustifiedBlockRoot)
	}
	if !bytes.Equal(jRoot[:], head.JustifiedBlockRoot) {
		t.Errorf("Wanted JustifiedBlockRoot: %v, got: %v",
			jRoot[:], head.JustifiedBlockRoot)
	}
	if !bytes.Equal(fRoot[:], head.FinalizedBlockRoot) {
		t.Errorf("Wanted FinalizedBlockRoot: %v, got: %v",
			fRoot[:], head.FinalizedBlockRoot)
	}
}

func TestServer_StreamChainHead_ContextCanceled(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	chainService := &mock.ChainService{}
	server := &Server{
		Ctx:           ctx,
		StateNotifier: chainService.StateNotifier(),
		BeaconDB:      db,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockBeaconChain_StreamChainHeadServer(ctrl)
	mockStream.EXPECT().Context().Return(ctx)
	go func(tt *testing.T) {
		if err := server.StreamChainHead(&ptypes.Empty{}, mockStream); !strings.Contains(err.Error(), "Context canceled") {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestServer_StreamChainHead_OnHeadUpdated(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)

	finalizedBlock := &ethpb.BeaconBlock{Slot: 1, ParentRoot: []byte{'A'}}
	db.SaveBlock(context.Background(), finalizedBlock)
	fRoot, _ := ssz.SigningRoot(finalizedBlock)
	justifiedBlock := &ethpb.BeaconBlock{Slot: 2, ParentRoot: []byte{'B'}}
	db.SaveBlock(context.Background(), justifiedBlock)
	jRoot, _ := ssz.SigningRoot(justifiedBlock)
	prevJustifiedBlock := &ethpb.BeaconBlock{Slot: 3, ParentRoot: []byte{'C'}}
	db.SaveBlock(context.Background(), prevJustifiedBlock)
	pjRoot, _ := ssz.SigningRoot(prevJustifiedBlock)

	s := &pbp2p.BeaconState{
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Epoch: 3, Root: pjRoot[:]},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Epoch: 2, Root: jRoot[:]},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Epoch: 1, Root: fRoot[:]},
	}
	b := &ethpb.BeaconBlock{Slot: s.PreviousJustifiedCheckpoint.Epoch*params.BeaconConfig().SlotsPerEpoch + 1}
	hRoot, _ := ssz.SigningRoot(b)

	chainService := &mock.ChainService{}
	server := &Server{
		Ctx:           context.Background(),
		HeadFetcher:   &mock.ChainService{Block: b, State: s},
		BeaconDB:      db,
		StateNotifier: chainService.StateNotifier(),
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockBeaconChain_StreamChainHeadServer(ctrl)
	mockStream.EXPECT().Context().Return(context.Background())
	mockStream.EXPECT().Send(
		&ethpb.ChainHead{
			HeadSlot:                   b.Slot,
			HeadEpoch:                  helpers.SlotToEpoch(b.Slot),
			HeadBlockRoot:              hRoot[:],
			FinalizedSlot:              1,
			FinalizedEpoch:             1,
			FinalizedBlockRoot:         fRoot[:],
			JustifiedSlot:              2,
			JustifiedEpoch:             2,
			JustifiedBlockRoot:         jRoot[:],
			PreviousJustifiedSlot:      3,
			PreviousJustifiedEpoch:     3,
			PreviousJustifiedBlockRoot: pjRoot[:],
		},
	).Return(nil)
	go func(tt *testing.T) {
		if err := server.StreamChainHead(&ptypes.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)

	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = server.StateNotifier.StateFeed().Send(&statefeed.Event{
			Type: statefeed.BlockProcessed,
			Data: &statefeed.BlockProcessedData{},
		})
	}
	exitRoutine <- true
}
