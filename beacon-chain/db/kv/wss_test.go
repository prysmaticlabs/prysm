package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/genesis"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestSaveOrigin(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	// Embedded Genesis works with Mainnet config
	params.OverrideBeaconConfig(params.MainnetConfig().Copy())

	ctx := context.Background()
	db := setupDB(t)

	st, err := genesis.State(params.MainnetName)
	require.NoError(t, err)

	sb, err := st.MarshalSSZ()
	require.NoError(t, err)
	require.NoError(t, db.LoadGenesis(ctx, sb))

	// this is necessary for mainnet, because LoadGenesis is short-circuited by the embedded state,
	// so the genesis root key is never written to the db.
	require.NoError(t, db.EnsureEmbeddedGenesis(ctx))

	cst, err := util.NewBeaconState()
	require.NoError(t, err)
	csb, err := cst.MarshalSSZ()
	require.NoError(t, err)
	cb := util.NewBeaconBlock()
	scb, err := wrapper.WrappedSignedBeaconBlock(cb)
	require.NoError(t, err)
	cbb, err := scb.MarshalSSZ()
	require.NoError(t, err)
	require.NoError(t, db.SaveOrigin(ctx, csb, cbb))

	broot, err := scb.Block().HashTreeRoot()
	require.NoError(t, err)
	require.Equal(t, true, db.IsFinalizedBlock(ctx, broot))
}
