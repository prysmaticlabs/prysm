package kv

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestTips_AddNewTips(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t)

	newTips := make(map[[32]byte]types.Slot)
	newTips[[32]byte{'A'}] = types.Slot(1)
	newTips[[32]byte{'B'}] = types.Slot(2)
	newTips[[32]byte{'C'}] = types.Slot(3)

	require.NoError(t, db.UpdateValidatedTips(ctx, newTips))

	gotTips, err := db.ValidatedTips(ctx)
	require.NoError(t, err)

	require.Equal(t, true, areTipsSame(gotTips, newTips))
}

func TestTips_UpdateTipsWithoutOverlap(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t)

	oldTips := make(map[[32]byte]types.Slot)
	oldTips[[32]byte{'A'}] = types.Slot(1)
	oldTips[[32]byte{'B'}] = types.Slot(2)
	oldTips[[32]byte{'C'}] = types.Slot(3)

	require.NoError(t, db.UpdateValidatedTips(ctx, oldTips))

	// create a new non-overlapping tips to add
	newTips := make(map[[32]byte]types.Slot)
	newTips[[32]byte{'D'}] = types.Slot(4)
	newTips[[32]byte{'E'}] = types.Slot(5)
	newTips[[32]byte{'F'}] = types.Slot(6)

	require.NoError(t, db.UpdateValidatedTips(ctx, newTips))

	gotTips, err := db.ValidatedTips(ctx)
	require.NoError(t, err)

	require.Equal(t, true, areTipsSame(gotTips, newTips))

}

func TestTips_UpdateTipsWithOverlap(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t)

	oldTips := make(map[[32]byte]types.Slot)
	oldTips[[32]byte{'A'}] = types.Slot(1)
	oldTips[[32]byte{'B'}] = types.Slot(2)
	oldTips[[32]byte{'C'}] = types.Slot(3)
	require.NoError(t, db.UpdateValidatedTips(ctx, oldTips))

	// create a new overlapping tips to add
	newTips := make(map[[32]byte]types.Slot)
	newTips[[32]byte{'C'}] = types.Slot(3)
	newTips[[32]byte{'D'}] = types.Slot(4)
	newTips[[32]byte{'E'}] = types.Slot(5)
	require.NoError(t, db.UpdateValidatedTips(ctx, newTips))

	gotTips, err := db.ValidatedTips(ctx)
	require.NoError(t, err)

	require.Equal(t, true, areTipsSame(gotTips, newTips))

}

func areTipsSame(got map[[32]byte]types.Slot, required map[[32]byte]types.Slot) bool {
	if len(got) != len(required) {
		return false
	}

	for k, v := range got {
		if val, ok := required[k]; ok {
			if uint64(v) != uint64(val) {
				return false
			}
		} else {
			return false
		}
	}
	return true
}
