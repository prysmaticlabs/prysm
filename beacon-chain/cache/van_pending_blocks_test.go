package cache

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"testing"
)

// generateBeaconBlocksMap generates certain amount of beacon block with slot
func generateBeaconBlocks(amount int) map[types.Slot]*ethpb.BeaconBlock {
	generatedBlocks := make(map[types.Slot]*ethpb.BeaconBlock, amount)

	for i := 0; i < amount; i++ {
		signedBlk := newBeaconBlock()
		blk := signedBlk.Block
		blk.Slot = types.Slot(i)
		generatedBlocks[types.Slot(i)] = blk
	}
	return generatedBlocks
}

// Test_VanPendingBlocksCache_KeyFunc checks the pending block cache key
func TestPendingBlocksCache_KeyFunc(t *testing.T) {
	signedBlk := newBeaconBlock()
	blk := signedBlk.Block
	blk.Slot = types.Slot(100000)
	actualStr, err := pendingBlocksKeyFn(blk)
	require.NoError(t, err)
	assert.DeepEqual(t, "100000", actualStr)
}

// Test_VanPendingBlockCache_AddPendingBlock
func TestPendingBlockCache_AddPendingBlock(t *testing.T) {
	cache := NewPendingBlocksCache()
	blk, err := cache.PendingBlock(types.Slot(100))
	require.NoError(t, err)
	if blk != nil {
		t.Error("Expected pending block not to exist in empty cache")
	}

	blks := generateBeaconBlocks(50)
	for _, blk := range blks {
		err := cache.AddPendingBlock(blk)
		require.NoError(t, err)
	}

	blk, err = cache.PendingBlock(types.Slot(0))
	require.NoError(t, err)
	assert.DeepEqual(t, blks[types.Slot(0)], blk)
}

// TestPendingBlocksCache_PendingBlocks checking retrieval all values.
func TestPendingBlocksCache_PendingBlocks(t *testing.T) {
	cache := NewPendingBlocksCache()
	blks := generateBeaconBlocks(50)
	for i := 0; i < 50; i++ {
		err := cache.AddPendingBlock(blks[types.Slot(i)])
		require.NoError(t, err)
	}

	actualBlks, err := cache.PendingBlocks()
	require.NoError(t, err)
	actualBlocksMap := make(map[types.Slot]*ethpb.BeaconBlock, 50)
	for _, actualBlk := range actualBlks {
		actualBlocksMap[actualBlk.Slot] = actualBlk
	}
	assert.DeepEqual(t, blks, actualBlocksMap)
}

// TestPendingBlocksCache_DeleteConfirmedBlock checks delete function
func TestPendingBlocksCache_DeleteConfirmedBlock(t *testing.T) {
	cache := NewPendingBlocksCache()
	blks := generateBeaconBlocks(50)
	for i := 0; i < 50; i++ {
		err := cache.AddPendingBlock(blks[types.Slot(i)])
		require.NoError(t, err)
	}

	require.NoError(t, cache.Delete(types.Slot(20)))
	blk, err := cache.PendingBlock(types.Slot(20))
	require.NoError(t, err)
	var expected *ethpb.BeaconBlock
	assert.Equal(t, expected, blk)
}

// TestNewPendingBlocksCache_UpdateBlock
func TestNewPendingBlocksCache_UpdateBlock(t *testing.T) {
	cache := NewPendingBlocksCache()
	blk, err := cache.PendingBlock(types.Slot(100))
	require.NoError(t, err)
	if blk != nil {
		t.Error("Expected pending block not to exist in empty cache")
	}

	blks := generateBeaconBlocks(50)
	for _, blk := range blks {
		err := cache.AddPendingBlock(blk)
		require.NoError(t, err)
	}

	blk1, err := cache.PendingBlock(types.Slot(0))
	require.NoError(t, err)
	assert.DeepEqual(t, blks[types.Slot(0)], blk1)

	// update the block at slot 0
	blks[0].ParentRoot = []byte("test")
	require.NoError(t, cache.AddPendingBlock(blks[0]))

	blk2, err := cache.PendingBlock(types.Slot(0))
	require.NoError(t, err)
	assert.DeepEqual(t, blks[types.Slot(0)], blk2)
}

// newBeaconBlock creates a beacon block with minimum marshalable fields.
func newBeaconBlock() *ethpb.SignedBeaconBlock {
	return &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: make([]byte, 32),
			StateRoot:  make([]byte, 32),
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: make([]byte, 96),
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot: make([]byte, 32),
					BlockHash:   make([]byte, 32),
				},
				Graffiti:          make([]byte, 32),
				Attestations:      []*ethpb.Attestation{},
				AttesterSlashings: []*ethpb.AttesterSlashing{},
				Deposits:          []*ethpb.Deposit{},
				ProposerSlashings: []*ethpb.ProposerSlashing{},
				VoluntaryExits:    []*ethpb.SignedVoluntaryExit{},
			},
		},
		Signature: make([]byte, 96),
	}
}
