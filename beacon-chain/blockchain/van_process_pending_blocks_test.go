package blockchain

import (
	"context"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	blockchainTesting "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"testing"
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
