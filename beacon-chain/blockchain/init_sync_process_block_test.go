package blockchain

import (
	"context"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestService_getBlock(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	s := setupBeaconChain(t, beaconDB)
	b1 := util.NewBeaconBlock()
	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 100
	r2, err := b2.Block.HashTreeRoot()
	require.NoError(t, err)

	// block not found
	_, err = s.getBlock(ctx, [32]byte{})
	require.ErrorIs(t, err, errBlockNotFoundInCacheOrDB)

	// block in cache
	b, err := wrapper.WrappedSignedBeaconBlock(b1)
	require.NoError(t, err)
	s.saveInitSyncBlock(r1, b)
	got, err := s.getBlock(ctx, r1)
	require.NoError(t, err)
	require.DeepEqual(t, b, got)

	// block in db
	b, err = wrapper.WrappedSignedBeaconBlock(b2)
	require.NoError(t, err)
	require.NoError(t, s.cfg.BeaconDB.SaveBlock(ctx, b))
	got, err = s.getBlock(ctx, r2)
	require.NoError(t, err)
	require.DeepEqual(t, b, got)
}
