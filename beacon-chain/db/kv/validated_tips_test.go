package kv

import (
	"bytes"
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestTips_AddNewTips(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t)

	tipA := [32]byte{'A'}
	tipB := [32]byte{'B'}
	tipC := [32]byte{'C'}
	newTips := [][32]byte{tipA, tipB, tipC}

	require.NoError(t, db.UpdateValidatedTips(ctx, newTips))

	gotTips, err := db.ValidatedTips(ctx)
	require.NoError(t, err)

	require.Equal(t, true, areTipsSame(gotTips, newTips))
}

func TestTips_UpdateTipsWithoutOverlap(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t)

	tipA := [32]byte{'A'}
	tipB := [32]byte{'B'}
	tipC := [32]byte{'C'}
	oldTips := [][32]byte{tipA, tipB, tipC}
	require.NoError(t, db.UpdateValidatedTips(ctx, oldTips))

	// create a new overlapping tips to add
	tipD := [32]byte{'D'}
	tipE := [32]byte{'E'}
	tipF := [32]byte{'F'}
	newTips := [][32]byte{tipD, tipE, tipF}
	require.NoError(t, db.UpdateValidatedTips(ctx, newTips))

	gotTips, err := db.ValidatedTips(ctx)
	require.NoError(t, err)

	require.Equal(t, true, areTipsSame(gotTips, newTips))

}

func TestTips_UpdateTipsWithOverlap(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t)

	tipA := [32]byte{'A'}
	tipB := [32]byte{'B'}
	tipC := [32]byte{'C'}
	oldTips := [][32]byte{tipA, tipB, tipC}
	require.NoError(t, db.UpdateValidatedTips(ctx, oldTips))

	// create a new overlapping tips to add
	tipD := [32]byte{'D'}
	tipE := [32]byte{'E'}
	newTips := [][32]byte{tipC, tipD, tipE}
	require.NoError(t, db.UpdateValidatedTips(ctx, newTips))

	gotTips, err := db.ValidatedTips(ctx)
	require.NoError(t, err)

	require.Equal(t, true, areTipsSame(gotTips, newTips))

}

func areTipsSame(got [][32]byte, required [][32]byte) bool {
	if len(got) != len(required) {
		return false
	}
	for i := 0; i < len(got); i++ {
		if !bytes.Equal(got[i][:], required[i][:]) {
			return false
		}
	}
	return true
}
