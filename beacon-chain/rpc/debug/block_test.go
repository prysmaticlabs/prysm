package debug

import (
	"context"
	"testing"

	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestServer_GetBlock(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	b := testutil.NewBeaconBlock()
	b.Block.Slot = 100
	require.NoError(t, db.SaveBlock(ctx, b))
	blockRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	bs := &Server{
		BeaconDB: db,
	}
	res, err := bs.GetBlock(ctx, &pbrpc.BlockRequest{
		BlockRoot: blockRoot[:],
	})
	require.NoError(t, err)
	wanted, err := b.MarshalSSZ()
	require.NoError(t, err)
	assert.DeepEqual(t, wanted, res.Encoded)

	// Checking for nil block.
	blockRoot = [32]byte{}
	res, err = bs.GetBlock(ctx, &pbrpc.BlockRequest{
		BlockRoot: blockRoot[:],
	})
	require.NoError(t, err)
	assert.DeepEqual(t, []byte{}, res.Encoded)
}
