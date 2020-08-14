package kv

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_VoluntaryExits_CRUD(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	exit := &ethpb.VoluntaryExit{
		Epoch: 5,
	}
	exitRoot, err := ssz.HashTreeRoot(exit)
	require.NoError(t, err)
	retrieved, err := db.VoluntaryExit(ctx, exitRoot)
	require.NoError(t, err)
	assert.Equal(t, (*ethpb.VoluntaryExit)(nil), retrieved, "Expected nil voluntary exit")
	require.NoError(t, db.SaveVoluntaryExit(ctx, exit))
	assert.Equal(t, true, db.HasVoluntaryExit(ctx, exitRoot), "Expected voluntary exit to exist in the db")
	retrieved, err = db.VoluntaryExit(ctx, exitRoot)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(exit, retrieved), "Wanted %v, received %v", exit, retrieved)
	require.NoError(t, db.deleteVoluntaryExit(ctx, exitRoot))
	assert.Equal(t, false, db.HasVoluntaryExit(ctx, exitRoot), "Expected voluntary exit to have been deleted from the db")
}
