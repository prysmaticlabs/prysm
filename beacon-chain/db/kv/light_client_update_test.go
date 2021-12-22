package kv

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestStore_LatestLightClientUpdate(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	_, err := db.LatestLightClientUpdate(ctx)
	require.ErrorContains(t, "no latest light client update found", err)

	h := &ethpb.BeaconBlockHeader{Slot: 1}
	u := &ethpb.LightClientUpdate{AttestedHeader: h}
	require.NoError(t, db.SaveLightClientUpdate(ctx, u))
	got, err := db.LatestLightClientUpdate(ctx)
	require.NoError(t, err)
	require.DeepSSZEqual(t, u, got)

	h1 := &ethpb.BeaconBlockHeader{Slot: 100}
	u1 := &ethpb.LightClientUpdate{AttestedHeader: h1}
	require.NoError(t, db.SaveLightClientUpdate(ctx, u1))
	got, err = db.LatestLightClientUpdate(ctx)
	require.NoError(t, err)
	require.DeepSSZEqual(t, u1, got)

	h2 := &ethpb.BeaconBlockHeader{Slot: 2}
	u2 := &ethpb.LightClientUpdate{AttestedHeader: h2}
	require.NoError(t, db.SaveLightClientUpdate(ctx, u2))
	got, err = db.LatestLightClientUpdate(ctx)
	require.NoError(t, err)
	require.DeepSSZEqual(t, u1, got)
}

func TestStore_LatestFinalizedLightClientUpdate(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	_, err := db.LatestFinalizedLightClientUpdate(ctx)
	require.ErrorContains(t, "no finalized light client update found", err)

	h := &ethpb.BeaconBlockHeader{Slot: 1}
	u := &ethpb.LightClientUpdate{AttestedHeader: h}
	require.NoError(t, db.SaveLatestFinalizedLightClientUpdate(ctx, u))
	got, err := db.LatestFinalizedLightClientUpdate(ctx)
	require.NoError(t, err)
	require.DeepSSZEqual(t, u, got)

	h1 := &ethpb.BeaconBlockHeader{Slot: 100}
	u1 := &ethpb.LightClientUpdate{AttestedHeader: h1}
	require.NoError(t, db.SaveLatestFinalizedLightClientUpdate(ctx, u1))
	got, err = db.LatestFinalizedLightClientUpdate(ctx)
	require.NoError(t, err)
	require.DeepSSZEqual(t, u1, got)

	h2 := &ethpb.BeaconBlockHeader{Slot: 2}
	u2 := &ethpb.LightClientUpdate{AttestedHeader: h2}
	require.NoError(t, db.SaveLatestFinalizedLightClientUpdate(ctx, u2))
	got, err = db.LatestFinalizedLightClientUpdate(ctx)
	require.NoError(t, err)
	require.DeepSSZEqual(t, u1, got)
}

func TestStore_LightClientUpdates(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	for i := types.Slot(0); i < 5*slotsPerEpoch; i++ {
		h := &ethpb.BeaconBlockHeader{Slot: i}
		u := &ethpb.LightClientUpdate{AttestedHeader: h}
		require.NoError(t, db.SaveLightClientUpdate(ctx, u))
	}

	tests := []struct {
		name   string
		filter *filters.QueryFilter
		want   []types.Slot
	}{
		{
			name:   "slot 0",
			filter: filters.NewFilter().SetStartSlot(0).SetEndSlot(0),
			want:   []types.Slot{0},
		},
		{
			name:   "slot 1",
			filter: filters.NewFilter().SetStartSlot(1).SetEndSlot(1),
			want:   []types.Slot{1},
		},
		{
			name:   "slot out of range",
			filter: filters.NewFilter().SetStartSlot(6 * slotsPerEpoch).SetEndSlot(6 * slotsPerEpoch),
			want:   []types.Slot{},
		},
		{
			name:   "slot 0 to slot 3",
			filter: filters.NewFilter().SetStartSlot(0).SetEndSlot(3),
			want:   []types.Slot{0, 1, 2, 3},
		},
		{
			name:   "slot 30 to slot 34",
			filter: filters.NewFilter().SetStartSlot(30).SetEndSlot(34),
			want:   []types.Slot{30, 31, 32, 33, 34},
		},
		{
			name:   "slot 158 to out of range",
			filter: filters.NewFilter().SetStartSlot(158).SetEndSlot(162),
			want:   []types.Slot{158, 159},
		},
		{
			name:   "epoch 0",
			filter: filters.NewFilter().SetStartEpoch(0).SetEndEpoch(0),
			want:   []types.Slot{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		},
		{
			name:   "epoch 1",
			filter: filters.NewFilter().SetStartEpoch(1).SetEndEpoch(1),
			want:   []types.Slot{32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63},
		},
		{
			name:   "epoch 2 to epoch 3",
			filter: filters.NewFilter().SetStartEpoch(2).SetEndEpoch(3),
			want: []types.Slot{64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93,
				94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126, 127},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updates, err := db.LightClientUpdates(ctx, tt.filter)
			require.NoError(t, err)
			require.Equal(t, len(tt.want), len(updates))
			for i, update := range updates {
				require.Equal(t, tt.want[i], update.AttestedHeader.Slot)
			}
		})
	}
}

func TestStore_DeleteLightClientUpdates(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	for i := types.Slot(0); i < 10; i++ {
		h := &ethpb.BeaconBlockHeader{Slot: i}
		u := &ethpb.LightClientUpdate{AttestedHeader: h}
		require.NoError(t, db.SaveLightClientUpdate(ctx, u))
	}

	require.NoError(t, db.DeleteLightClientUpdates(ctx, []types.Slot{1, 3, 5, 7, 9}))
	updates, err := db.LightClientUpdates(ctx, filters.NewFilter().SetStartEpoch(0).SetEndEpoch(10))
	require.NoError(t, err)
	for _, update := range updates {
		require.Equal(t, types.Slot(0), update.AttestedHeader.Slot%2)
	}
}
