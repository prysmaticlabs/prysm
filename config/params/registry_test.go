package params_test

import (
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestRegistry_Add(t *testing.T) {
	r := params.NewRegistry()
	require.NoError(t, r.Add(params.MainnetConfig()))
	c, err := r.GetByName(params.MainnetName)
	require.NoError(t, err)
	compareConfigs(t, params.MainnetConfig(), c)
	require.ErrorIs(t, r.Add(params.MainnetConfig()), params.ErrRegistryCollision)
	c = c.Copy()
	c.ConfigName = "test"
	require.ErrorIs(t, r.Add(params.MainnetConfig()), params.ErrRegistryCollision)
}

func TestRegistry_Replace(t *testing.T) {
	r := params.NewRegistry()
	mainnet := params.MainnetConfig().Copy()
	fillTestVersions(mainnet, 128)
	require.ErrorIs(t, r.Add(mainnet), params.ErrConfigNameConflict)
	c, err := r.GetByName(params.MainnetName)
	require.NoError(t, err)
	fail := c.Copy()
	fail.ConfigName = "test"
	require.ErrorIs(t, r.Replace(fail), params.ErrRegistryCollision)

	o := c.Copy()
	o.ConfigName = params.MainnetName
	require.NoError(t, r.Replace(o))
	// mainnet is replaced, we shouldn't be able to find its fork version anymore
	_, err = r.GetByVersion(bytesutil.ToBytes4(mainnet.GenesisForkVersion))
	require.ErrorIs(t, err, params.ErrConfigNotFound)
	undo := o.Copy()
	fillTestVersions(undo, 127)
	undoFunc, err := r.ReplaceWithUndo(undo)
	require.NoError(t, err)
	u, err := r.GetByName(undo.ConfigName)
	require.NoError(t, err)
	require.Equal(t, undo, u)
	u, err = r.GetByVersion(bytesutil.ToBytes4(undo.GenesisForkVersion))
	require.NoError(t, err)
	require.Equal(t, undo, u)
	_, err = r.GetByVersion(bytesutil.ToBytes4(o.GenesisForkVersion))
	require.ErrorIs(t, err, params.ErrConfigNotFound)
	require.NoError(t, undoFunc())
	// replaced config restored by undoFunc, lookup should now succeed
	_, err = r.GetByVersion(bytesutil.ToBytes4(o.GenesisForkVersion))
	require.NoError(t, err)
}

func fillTestVersions(c *params.BeaconChainConfig, b byte) {
	c.GenesisForkVersion = make([]byte, fieldparams.VersionLength)
	c.GenesisForkVersion[0] = 0
	c.GenesisForkVersion[fieldparams.VersionLength-1] = b
	c.AltairForkVersion[0] = 1
	c.AltairForkVersion[fieldparams.VersionLength-1] = b
	c.BellatrixForkVersion[0] = 2
	c.BellatrixForkVersion[fieldparams.VersionLength-1] = b
	c.ShardingForkVersion[0] = 3
	c.ShardingForkVersion[fieldparams.VersionLength-1] = b
}
