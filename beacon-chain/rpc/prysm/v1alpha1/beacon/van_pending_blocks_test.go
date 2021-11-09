package beacon

import (
	"context"
	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/proto/interfaces"
	"github.com/prysmaticlabs/prysm/shared/mock"
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

	beaconState, err := testutil.NewBeaconState()
	require.NoError(t, err)
	// Genesis block.
	genesisBlock := testutil.NewBeaconBlock()
	genesisBlockRoot, err := genesisBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesisBlock)))
	require.NoError(t, db.SaveState(ctx, beaconState, genesisBlockRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlockRoot))

	ctx, cancel := context.WithCancel(ctx)

	chainService := &chainMock.ChainService{Block: wrapper.WrappedPhase0SignedBeaconBlock(genesisBlock)}
	server := &Server{
		Ctx:           ctx,
		HeadFetcher:   chainService,
		StateNotifier: chainService.StateNotifier(),
		BlockNotifier: chainService.BlockNotifier(),
		BeaconDB:      db,
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

// TODO(Atif)- Fix this test
func TestServer_StreamNewPendingBlocks_PublishPrevBlocksBatch(t *testing.T) {
	t.Skip("PublishPrevBlocksBatch event tests skipped for short time")
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)

	// Should return the proper genesis block if it exists.
	parentRoot := [32]byte{1, 2, 3}
	blk := testutil.NewBeaconBlock()
	blk.Block.ParentRoot = parentRoot[:]
	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk)))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))
	beaconState, err := testutil.NewBeaconState()
	require.NoError(t, db.SaveState(ctx, beaconState, root))

	count := types.Slot(100)
	blks := make([]interfaces.SignedBeaconBlock, count)
	var ckRoot [32]byte
	var cp *ethpb.Checkpoint
	for i := types.Slot(0); i < count; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		require.NoError(t, err)
		blks[i] = wrapper.WrappedPhase0SignedBeaconBlock(b)
		if i == 96 {
			ckRoot, err = b.Block.HashTreeRoot()
			require.NoError(t, err)
			cp = &ethpb.Checkpoint{
				Epoch: 11,
				Root:  root[:],
			}
		}
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))
	require.NoError(t, err)
	// a state is required to save checkpoint
	require.NoError(t, db.SaveState(ctx, beaconState, ckRoot))
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))
	chainService := &chainMock.ChainService{State: beaconState}
	bs := &Server{
		Ctx:           ctx,
		BeaconDB:      db,
		BlockNotifier: chainService.BlockNotifier(),
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconChain_StreamNewPendingBlocksServer(ctrl)
	blkCnt := 0
	mockStream.EXPECT().Send(gomock.Any()).Do(func(args interface{}) {
		blkCnt++
		if blkCnt == 100 {
			exitRoutine <- true
		}
	}).Return(nil).AnyTimes()
	//mockStream.EXPECT().Context().Return(ctx)
	go func(tt *testing.T) {
		bs.StreamNewPendingBlocks(&ethpb.StreamPendingBlocksRequest{
			BlockRoot: []byte{1, 2, 3},
			FromSlot:  0,
		}, mockStream)
	}(t)
	<-exitRoutine
	cancel()
}
