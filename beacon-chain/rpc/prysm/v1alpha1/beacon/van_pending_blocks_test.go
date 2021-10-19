package beacon

import (
	"context"
	"github.com/golang/mock/gomock"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
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
