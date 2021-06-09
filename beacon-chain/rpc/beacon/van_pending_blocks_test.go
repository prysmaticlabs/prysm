package beacon

import (
	"context"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
		assert.ErrorContains(tt, "Context canceled", server.StreamNewPendingBlocks(&ptypes.Empty{}, mockStream))
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

// TestServer_StreamNewPendingBlocks_OnNewBlock checks the pipeline of the new pending block publishing from blockchain
func TestServer_StreamNewPendingBlocks_OnNewBlock(t *testing.T) {
	ctx := context.Background()
	beaconState, privs := testutil.DeterministicGenesisState(t, 32)
	b, err := testutil.GenerateFullBlock(beaconState, privs, testutil.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)
	chainService := &chainMock.ChainService{State: beaconState}
	server := &Server{
		Ctx:                     ctx,
		BlockNotifier:           chainService.BlockNotifier(),
		HeadFetcher:             chainService,
		UnconfirmedBlockFetcher: chainService,
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := vmock.NewMockBeaconChain_StreamNewPendingBlocksServer(ctrl)
	mockStream.EXPECT().Send(b.Block).Do(func(arg0 interface{}) {
		exitRoutine <- true
	})
	mockStream.EXPECT().Context().Return(ctx).AnyTimes()

	go func(tt *testing.T) {
		assert.NoError(tt, server.StreamNewPendingBlocks(&ptypes.Empty{}, mockStream), "Could not call RPC method")
	}(t)

	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = server.BlockNotifier.BlockFeed().Send(&feed.Event{
			Type: blockfeed.UnConfirmedBlock,
			Data: &blockfeed.UnConfirmedBlockData{Block: b.Block},
		})
	}
	<-exitRoutine
}

// TestServer_GetCanonicalBlock_NoFinalizedBlock
func TestServer_GetCanonicalBlock_NoFinalizedBlock(t *testing.T) {
	db := dbTest.SetupDB(t)

	s, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetSlot(1))
	require.NoError(t, s.SetPreviousJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 3, Root: bytesutil.PadTo([]byte{'A'}, 32)}))
	require.NoError(t, s.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: 2, Root: bytesutil.PadTo([]byte{'B'}, 32)}))
	require.NoError(t, s.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'C'}, 32)}))

	genBlock := testutil.NewBeaconBlock()
	genBlock.Block.ParentRoot = bytesutil.PadTo([]byte{'G'}, 32)
	require.NoError(t, db.SaveBlock(context.Background(), genBlock))
	gRoot, err := genBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), gRoot))

	bs := &Server{
		BeaconDB:    db,
		HeadFetcher: &chainMock.ChainService{Block: genBlock, State: s},
		FinalizationFetcher: &chainMock.ChainService{
			FinalizedCheckPoint:         s.FinalizedCheckpoint(),
			CurrentJustifiedCheckPoint:  s.CurrentJustifiedCheckpoint(),
			PreviousJustifiedCheckPoint: s.PreviousJustifiedCheckpoint()},
	}

	_, err = bs.GetCanonicalBlock(context.Background(), nil)
	require.ErrorContains(t, "Could not get finalized block", err)
}

func TestServer_GetCanonicalBlock_NoHeadBlock(t *testing.T) {
	bs := &Server{
		HeadFetcher: &chainMock.ChainService{Block: nil},
	}
	_, err := bs.GetChainHead(context.Background(), nil)
	assert.ErrorContains(t, "Head block of chain was nil", err)
}
