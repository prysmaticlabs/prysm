package blockchain

import (
	"context"
	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	blockchainTesting "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	vanTypes "github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/van_mock"
	"math/rand"
	"sort"
	"testing"
	"time"
)

// TestService_PublishAndStorePendingBlock checks PublishAndStorePendingBlock method
func TestService_PublishAndStorePendingBlock(t *testing.T) {
	ctx := context.Background()
	cfg := &Config{
		BlockNotifier: &blockchainTesting.MockBlockNotifier{},
	}
	s, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b := testutil.NewBeaconBlock()
	require.NoError(t, s.publishAndStorePendingBlock(ctx, b.Block))
	cachedBlock, err := s.pendingBlockCache.PendingBlock(b.Block.GetSlot())
	require.NoError(t, err)
	assert.DeepEqual(t, b.Block, cachedBlock)
}

// TestService_PublishAndStorePendingBlockBatch checks PublishAndStorePendingBlockBatch method
func TestService_PublishAndStorePendingBlockBatch(t *testing.T) {
	ctx := context.Background()
	cfg := &Config{
		BlockNotifier: &blockchainTesting.MockBlockNotifier{},
	}
	s, err := NewService(ctx, cfg)
	require.NoError(t, err)

	blks := make([]*ethpb.SignedBeaconBlock, 10)
	for i := 0; i < 10; i++ {
		b := testutil.NewBeaconBlock()
		blks[i] = b
	}

	require.NoError(t, s.publishAndStorePendingBlockBatch(ctx, blks))
	require.NoError(t, err)
	for _, blk := range blks {
		cachedBlock, err := s.pendingBlockCache.PendingBlock(blk.Block.GetSlot())
		require.NoError(t, err)
		assert.DeepEqual(t, blk.Block, cachedBlock)
	}
}

// TestService_SortedUnConfirmedBlocksFromCache checks SortedUnConfirmedBlocksFromCache method
func TestService_SortedUnConfirmedBlocksFromCache(t *testing.T) {
	ctx := context.Background()
	cfg := &Config{
		BlockNotifier: &blockchainTesting.MockBlockNotifier{},
	}
	s, err := NewService(ctx, cfg)
	require.NoError(t, err)

	blks := make([]*ethpb.BeaconBlock, 10)
	for i := 0; i < 10; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = types.Slot(rand.Uint64() % 100)
		blks[i] = b.Block
		require.NoError(t, s.pendingBlockCache.AddPendingBlock(b.Block))
	}

	sort.Slice(blks, func(i, j int) bool {
		return blks[i].Slot < blks[j].Slot
	})

	sortedBlocks, err := s.SortedUnConfirmedBlocksFromCache()
	require.NoError(t, err)
	require.DeepEqual(t, blks, sortedBlocks)
}

// TestService_fetchOrcConfirmations checks fetchOrcConfirmations
func TestService_fetchOrcConfirmations(t *testing.T) {
	ctx := context.Background()
	var mockedOrcClient *van_mock.MockClient
	ctrl := gomock.NewController(t)
	mockedOrcClient = van_mock.NewMockClient(ctrl)
	cfg := &Config{
		BlockNotifier:      &blockchainTesting.MockBlockNotifier{RecordEvents: true},
		OrcRPCClient:       mockedOrcClient,
		EnableVanguardNode: true,
	}

	confirmationStatus := make([]*vanTypes.ConfirmationResData, 10)
	for i := 0; i < 10; i++ {
		confirmationStatus[i] = &vanTypes.ConfirmationResData{Slot: types.Slot(i), Status: vanTypes.Verified}
	}
	mockedOrcClient.EXPECT().ConfirmVanBlockHashes(
		gomock.Any(),
		gomock.Any(),
	).AnyTimes().Return(confirmationStatus, nil)

	s, err := NewService(ctx, cfg)
	require.NoError(t, err)
	blks := make([]*ethpb.BeaconBlock, 10)
	for i := 0; i < 10; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = types.Slot(i)
		blks[i] = b.Block
		confirmationStatus[i] = &vanTypes.ConfirmationResData{Slot: types.Slot(i), Status: vanTypes.Verified}
		require.NoError(t, s.pendingBlockCache.AddPendingBlock(b.Block))
	}

	time.Sleep(2 * time.Second)
	if recvd := len(s.blockNotifier.(*blockchainTesting.MockBlockNotifier).ReceivedEvents()); recvd < 1 {
		t.Errorf("Received %d pending block notifications, expected at least 1", recvd)
	}
}

// TestService_waitForConfirmationBlock checks waitForConfirmationBlock method
// When the confirmation result of the block is verified then waitForConfirmationBlock gives you error return
// Not delete the invalid block because, when node gets an valid block, then it will be replaced and then it will be deleted
func TestService_waitForConfirmationBlock(t *testing.T) {
	tests := []struct {
		name                 string
		pendingBlocksInQueue []*ethpb.SignedBeaconBlock
		incomingBlock        *ethpb.SignedBeaconBlock
		confirmationStatus   []*vanTypes.ConfirmationResData
		expectedOutput       string
	}{
		{
			name:                 "Returns nil when orchestrator sends verified status for all blocks",
			pendingBlocksInQueue: getBeaconBlocks(0, 3),
			incomingBlock:        getBeaconBlock(2),
			confirmationStatus: []*vanTypes.ConfirmationResData{
				{
					Slot:   0,
					Status: vanTypes.Verified,
				},
				{
					Slot:   1,
					Status: vanTypes.Verified,
				},
				{
					Slot:   2,
					Status: vanTypes.Verified,
				},
			},
			expectedOutput: "",
		},
		{
			name:                 "Returns error when orchestrator sends invalid status",
			pendingBlocksInQueue: getBeaconBlocks(0, 3),
			incomingBlock:        getBeaconBlock(1),
			confirmationStatus: []*vanTypes.ConfirmationResData{
				{
					Slot:   0,
					Status: vanTypes.Verified,
				},
				{
					Slot:   1,
					Status: vanTypes.Invalid,
				},
				{
					Slot:   2,
					Status: vanTypes.Verified,
				},
			},
			expectedOutput: "invalid block found, discarded block batch",
		},
		{
			name:                 "Retry for the block with pending status",
			pendingBlocksInQueue: getBeaconBlocks(0, 3),
			incomingBlock:        getBeaconBlock(1),
			confirmationStatus: []*vanTypes.ConfirmationResData{
				{
					Slot:   0,
					Status: vanTypes.Verified,
				},
				{
					Slot:   1,
					Status: vanTypes.Pending,
				},
				{
					Slot:   2,
					Status: vanTypes.Verified,
				},
			},
			expectedOutput: "maximum wait is exceeded and orchestrator can not verify the block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var mockedOrcClient *van_mock.MockClient
			ctrl := gomock.NewController(t)
			mockedOrcClient = van_mock.NewMockClient(ctrl)

			cfg := &Config{
				BlockNotifier:      &blockchainTesting.MockBlockNotifier{},
				OrcRPCClient:       mockedOrcClient,
				EnableVanguardNode: true,
			}
			mockedOrcClient.EXPECT().ConfirmVanBlockHashes(
				gomock.Any(),
				gomock.Any(),
			).AnyTimes().Return(tt.confirmationStatus, nil)
			s, err := NewService(ctx, cfg)
			require.NoError(t, err)
			for i := 0; i < len(tt.pendingBlocksInQueue); i++ {
				require.NoError(t, s.pendingBlockCache.AddPendingBlock(tt.pendingBlocksInQueue[i].Block))
			}

			if tt.expectedOutput == "" {
				require.NoError(t, s.waitForConfirmationBlock(ctx, tt.incomingBlock))
			} else {
				require.ErrorContains(t, tt.expectedOutput, s.waitForConfirmationBlock(ctx, tt.incomingBlock))
			}
		})
	}
}

// TestService_waitForConfirmationBlockBatch
func TestService_waitForConfirmationBlockBatch(t *testing.T) {
	tests := []struct {
		name                 string
		pendingBlocksInQueue []*ethpb.SignedBeaconBlock
		incomingBlocksBatch  []*ethpb.SignedBeaconBlock
		confirmationStatus   []*vanTypes.ConfirmationResData
		expectedOutput       string
	}{
		{
			name:                 "Returns nil when orchestrator sends verified status for all blocks",
			pendingBlocksInQueue: getBeaconBlocks(5, 8),
			incomingBlocksBatch:  getBeaconBlocks(5, 8),
			confirmationStatus: []*vanTypes.ConfirmationResData{
				{
					Slot:   5,
					Status: vanTypes.Verified,
				},
				{
					Slot:   6,
					Status: vanTypes.Verified,
				},
				{
					Slot:   7,
					Status: vanTypes.Verified,
				},
			},
			expectedOutput: "",
		},
		{
			name:                 "Returns error when orchestrator sends invalid status",
			pendingBlocksInQueue: getBeaconBlocks(0, 3),
			incomingBlocksBatch:  getBeaconBlocks(0, 3),
			confirmationStatus: []*vanTypes.ConfirmationResData{
				{
					Slot:   0,
					Status: vanTypes.Verified,
				},
				{
					Slot:   1,
					Status: vanTypes.Invalid,
				},
				{
					Slot:   2,
					Status: vanTypes.Verified,
				},
			},
			expectedOutput: "invalid block found, discarded block batch",
		},
		{
			name:                 "Returns error after certain time when orchestrator repeatedly sends pending status",
			pendingBlocksInQueue: getBeaconBlocks(0, 3),
			incomingBlocksBatch:  getBeaconBlocks(0, 3),
			confirmationStatus: []*vanTypes.ConfirmationResData{
				{
					Slot:   0,
					Status: vanTypes.Verified,
				},
				{
					Slot:   1,
					Status: vanTypes.Pending,
				},
				{
					Slot:   2,
					Status: vanTypes.Verified,
				},
			},
			expectedOutput: "maximum wait is exceeded and orchestrator can not verify the block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var mockedOrcClient *van_mock.MockClient
			ctrl := gomock.NewController(t)
			mockedOrcClient = van_mock.NewMockClient(ctrl)

			cfg := &Config{
				BlockNotifier:      &blockchainTesting.MockBlockNotifier{},
				OrcRPCClient:       mockedOrcClient,
				EnableVanguardNode: true,
			}
			mockedOrcClient.EXPECT().ConfirmVanBlockHashes(
				gomock.Any(),
				gomock.Any(),
			).AnyTimes().Return(tt.confirmationStatus, nil)
			s, err := NewService(ctx, cfg)
			require.NoError(t, err)
			for i := 0; i < len(tt.pendingBlocksInQueue); i++ {
				require.NoError(t, s.pendingBlockCache.AddPendingBlock(tt.pendingBlocksInQueue[i].Block))
			}
			if tt.expectedOutput == "" {
				require.NoError(t, s.waitForConfirmationsBlockBatch(ctx, tt.incomingBlocksBatch))
			} else {
				require.ErrorContains(t, tt.expectedOutput, s.waitForConfirmationsBlockBatch(ctx, tt.incomingBlocksBatch))
			}
		})
	}
}

// Helper method to generate pending queue with random blocks
func getBeaconBlocks(from, to int) []*ethpb.SignedBeaconBlock {
	pendingBlks := make([]*ethpb.SignedBeaconBlock, to-from)
	for i := 0; i < to-from; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = types.Slot(from + i)
		pendingBlks[i] = b
	}
	return pendingBlks
}

// Helper method to generate pending queue with random block
func getBeaconBlock(slot types.Slot) *ethpb.SignedBeaconBlock {
	b := testutil.NewBeaconBlock()
	b.Block.Slot = types.Slot(slot)
	return b
}
