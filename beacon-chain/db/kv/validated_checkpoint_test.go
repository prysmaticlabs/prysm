package kv

import (
	"bytes"
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestValidateCheckpoint(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t)

	checkpointA := [32]byte{'A'}
	slotA := types.Slot(1)

	checkpointB := [32]byte{'B'}
	slotB := types.Slot(2)

	// add first checkpoint
	require.NoError(t, db.saveLastValidatedCheckpoint(ctx, checkpointA, slotA))

	rcvdRoot, rcvdSlot, err := db.LastValidatedCheckpoint(ctx)
	require.NoError(t, err)

	require.Equal(t, 0, bytes.Compare(checkpointA[:], rcvdRoot[:]))
	require.Equal(t, true, uint64(slotA) == uint64(rcvdSlot))

	// update the checkpoint and slot
	require.NoError(t, db.saveLastValidatedCheckpoint(ctx, checkpointB, slotB))

	rcvdRoot, rcvdSlot, err = db.LastValidatedCheckpoint(ctx)
	require.NoError(t, err)

	require.Equal(t, 0, bytes.Compare(checkpointB[:], rcvdRoot[:]))
	require.Equal(t, true, uint64(slotB) == uint64(rcvdSlot))
}
