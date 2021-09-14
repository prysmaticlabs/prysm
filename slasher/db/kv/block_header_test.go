package kv

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core"
	"github.com/prysmaticlabs/prysm/encoding/bytes"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestNilDBHistoryBlkHdr(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	slot := types.Slot(1)
	validatorIndex := types.ValidatorIndex(1)

	require.Equal(t, false, db.HasBlockHeader(ctx, slot, validatorIndex))

	bPrime, err := db.BlockHeaders(ctx, slot, validatorIndex)
	require.NoError(t, err, "Failed to get block")
	require.DeepEqual(t, []*ethpb.SignedBeaconBlockHeader(nil), bPrime, "Should return nil for a non existent key")
}

func TestSaveHistoryBlkHdr(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	tests := []struct {
		bh *ethpb.SignedBeaconBlockHeader
	}{
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in 2nd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 1}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in 3rd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch + 1, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in 3rd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 1, ProposerIndex: 0}},
		},
	}

	for _, tt := range tests {
		err := db.SaveBlockHeader(ctx, tt.bh)
		require.NoError(t, err, "Save block failed")

		bha, err := db.BlockHeaders(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)
		require.NoError(t, err, "Failed to get block")
		require.NotNil(t, bha)
		require.DeepEqual(t, tt.bh, bha[0], "Should return bh")
	}
}

func TestDeleteHistoryBlkHdr(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	tests := []struct {
		bh *ethpb.SignedBeaconBlockHeader
	}{
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in 2nd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 1}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in 3rd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch + 1, ProposerIndex: 0}},
		},
	}
	for _, tt := range tests {
		err := db.SaveBlockHeader(ctx, tt.bh)
		require.NoError(t, err, "Save block failed")
	}

	for _, tt := range tests {
		bha, err := db.BlockHeaders(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)
		require.NoError(t, err, "Failed to get block")
		require.NotNil(t, bha)
		require.DeepEqual(t, tt.bh, bha[0], "Should return bh")

		err = db.DeleteBlockHeader(ctx, tt.bh)
		require.NoError(t, err, "Save block failed")
		bh, err := db.BlockHeaders(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)
		require.NoError(t, err)
		assert.DeepEqual(t, []*ethpb.SignedBeaconBlockHeader(nil), bh, "Expected block to have been deleted")
	}
}

func TestHasHistoryBlkHdr(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	tests := []struct {
		bh *ethpb.SignedBeaconBlockHeader
	}{
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in 2nd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 1}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in 3rd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch + 1, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in 4th"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 1, ProposerIndex: 0}},
		},
	}
	for _, tt := range tests {
		found := db.HasBlockHeader(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)
		require.Equal(t, false, found, "has block header should return false for block headers that are not in db")
		err := db.SaveBlockHeader(ctx, tt.bh)
		require.NoError(t, err, "Save block failed")
	}
	for _, tt := range tests {
		err := db.SaveBlockHeader(ctx, tt.bh)
		require.NoError(t, err, "Save block failed")

		found := db.HasBlockHeader(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)
		require.Equal(t, true, found, "Block header should exist")
	}
}

func TestPruneHistoryBlkHdr(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	tests := []struct {
		bh *ethpb.SignedBeaconBlockHeader
	}{
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in 2nd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: 0, ProposerIndex: 1}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in 3rd"), 96), Header: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch + 1, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in 4th"), 96), Header: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch*2 + 1, ProposerIndex: 0}},
		},
		{
			bh: &ethpb.SignedBeaconBlockHeader{Signature: bytes.PadTo([]byte("let me in 5th"), 96), Header: &ethpb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch*3 + 1, ProposerIndex: 0}},
		},
	}

	for _, tt := range tests {
		err := db.SaveBlockHeader(ctx, tt.bh)
		require.NoError(t, err, "Save block header failed")

		bha, err := db.BlockHeaders(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)
		require.NoError(t, err, "Failed to get block header")
		require.NotNil(t, bha)
		require.DeepEqual(t, tt.bh, bha[0], "Should return bh")
	}
	currentEpoch := types.Epoch(3)
	historyToKeep := types.Epoch(2)
	err := db.PruneBlockHistory(ctx, currentEpoch, historyToKeep)
	require.NoError(t, err, "Failed to prune")

	for _, tt := range tests {
		bha, err := db.BlockHeaders(ctx, tt.bh.Header.Slot, tt.bh.Header.ProposerIndex)
		require.NoError(t, err, "Failed to get block header")
		if core.SlotToEpoch(tt.bh.Header.Slot) >= currentEpoch-historyToKeep {
			require.NotNil(t, bha)
			require.DeepEqual(t, tt.bh, bha[0], "Should return bh")
		} else {
			require.Equal(t, 0, len(bha), "Block header should have been pruned")
			require.DeepEqual(t, []*ethpb.SignedBeaconBlockHeader(nil), bha, "Block header should have been pruned")
		}
	}
}
