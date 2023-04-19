package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/proto/dbval"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"google.golang.org/protobuf/proto"
)

func TestBackfillRoundtrip(t *testing.T) {
	db := setupDB(t)
	b := &dbval.BackfillStatus{}
	b.HighSlot = 23
	b.LowSlot = 13
	b.HighRoot = bytesutil.PadTo([]byte("high"), 42)
	b.LowRoot = bytesutil.PadTo([]byte("low"), 24)
	m, err := proto.Marshal(b)
	require.NoError(t, err)
	ub := &dbval.BackfillStatus{}
	require.NoError(t, proto.Unmarshal(m, ub))
	require.Equal(t, b.HighSlot, ub.HighSlot)
	require.DeepEqual(t, b.HighRoot, ub.HighRoot)
	require.Equal(t, b.LowSlot, ub.LowSlot)
	require.DeepEqual(t, b.LowRoot, ub.LowRoot)

	ctx := context.Background()
	require.NoError(t, db.SaveBackfillStatus(ctx, b))
	dbub, err := db.BackfillStatus(ctx)

	require.Equal(t, b.HighSlot, dbub.HighSlot)
	require.DeepEqual(t, b.HighRoot, dbub.HighRoot)
	require.Equal(t, b.LowSlot, dbub.LowSlot)
	require.DeepEqual(t, b.LowRoot, dbub.LowRoot)
}
