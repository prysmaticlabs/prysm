package beacon

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	vmock "github.com/prysmaticlabs/prysm/shared/van_mock"
	"testing"
)

// TestServer_StreamNewPendingBlocks_ContextCanceled
func TestServer_StreamNewPendingBlocks_ContextCanceled(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	chainService := &chainMock.ChainService{}
	server := &Server{
		Ctx:                     ctx,
		StateNotifier:           chainService.StateNotifier(),
		BlockNotifier:           chainService.BlockNotifier(),
		BeaconDB:                db,
		UnconfirmedBlockFetcher: chainService,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := vmock.NewMockBeaconChain_StreamNewPendingBlocksServer(ctrl)
	mockStream.EXPECT().Context().Return(ctx)
	go func(tt *testing.T) {
		assert.ErrorContains(tt, "Context canceled", server.StreamNewPendingBlocks(&ethpb.StreamPendingBlocksRequest{}, mockStream))
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestServer_StreamNewPendingBlocks_CheckPreviousBlocksSending(t *testing.T) {
	db := dbTest.SetupDB(t)
	ctx := context.Background()
	chainService := &chainMock.ChainService{}

	server := &Server{
		Ctx:                     ctx,
		StateNotifier:           chainService.StateNotifier(),
		BlockNotifier:           chainService.BlockNotifier(),
		BeaconDB:                db,
		UnconfirmedBlockFetcher: chainService,
	}
	// Should return the proper genesis block if it exists.
	parentRoot := [32]byte{1, 2, 3}
	blk := testutil.NewBeaconBlock()
	blk.Block.ParentRoot = parentRoot[:]
	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, blk))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

	count := types.Slot(100)
	blks := make([]*ethpb.SignedBeaconBlock, count)
	for i := types.Slot(0); i < count; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		require.NoError(t, err)
		blks[i] = b
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		if i == 16 {
			require.NoError(t, db.SaveStateSummary(ctx, &pb.StateSummary{Root: root[:]}))
			require.NoError(t, db.SaveFinalizedCheckpoint(server.Ctx, &ethpb.Checkpoint{Epoch: types.Epoch(2), Root: root[:]}))
		}
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStream := vmock.NewMockBeaconChain_StreamNewPendingBlocksServer(ctrl)
	mockStream.EXPECT().Send(gomock.Any()).Do(func(arg0 interface{}) {
		exitRoutine <- true
	}).AnyTimes()
	mockStream.EXPECT().Context().Return(ctx).AnyTimes()

	tests := []struct {
		req *ethpb.StreamPendingBlocksRequest
		res string
	}{
		{
			req: &ethpb.StreamPendingBlocksRequest{
				BlockRoot: []byte{1, 2, 3},
				FromSlot:  types.Slot(0),
			},
			res: "",
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			go func(tt *testing.T) {
				err := server.StreamNewPendingBlocks(test.req, mockStream)
				assert.ErrorContains(tt, test.res, err)
			}(t)

			for i := test.req.FromSlot; i < 23; i++ {
				<-exitRoutine
			}
			// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
			for sent := 0; sent == 0; {
				sent = server.BlockNotifier.BlockFeed().Send(&feed.Event{
					Type: blockfeed.UnConfirmedBlock,
					Data: &blockfeed.UnConfirmedBlockData{Block: blks[99].Block},
				})
			}
			for i := 0; i <= 75; i++ {
				<-exitRoutine
			}
		})
	}
}
