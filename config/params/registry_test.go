package params_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestRegistry_Add(t *testing.T) {
	r := params.NewRegistry()
	name := "devnet"
	cfg := testConfig(name)
	require.NoError(t, r.Add(cfg))
	c, err := r.GetByName(name)
	require.NoError(t, err)
	compareConfigs(t, cfg, c)
	require.ErrorIs(t, r.Add(cfg), params.ErrConfigNameConflict)
	cfg.ConfigName = "test"
	require.ErrorIs(t, r.Add(cfg), params.ErrRegistryCollision)
}

func TestRegistry_ReplaceMainnet(t *testing.T) {
	r := params.NewRegistry()
	mainnet := params.MainnetConfig().Copy()
	require.NoError(t, r.SetActive(mainnet))
	params.FillTestVersions(mainnet, 128)
	require.NoError(t, r.Replace(mainnet))
}

func TestRegistry_Replace(t *testing.T) {
	r := params.NewRegistry()
	mainnet := params.MainnetConfig().Copy()
	require.NoError(t, r.Add(mainnet))
	require.NoError(t, r.SetActive(mainnet))
	require.ErrorIs(t, r.Add(mainnet), params.ErrConfigNameConflict)
	c, err := r.GetByName(params.MainnetName)
	require.NoError(t, err)
	fail := c.Copy()
	fail.ConfigName = "test"
	require.ErrorIs(t, r.Replace(fail), params.ErrRegistryCollision)

	o := c.Copy()
	params.FillTestVersions(o, 128)
	o.ConfigName = params.MainnetName
	require.NoError(t, r.Replace(o))
	// mainnet is replaced, we shouldn't be able to find its fork version anymore
	_, err = r.GetByVersion(bytesutil.ToBytes4(mainnet.GenesisForkVersion))
	require.ErrorIs(t, err, params.ErrConfigNotFound)
	undo := o.Copy()
	params.FillTestVersions(undo, 127)
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

func testConfig(name string) *params.BeaconChainConfig {
	c := params.MainnetConfig().Copy()
	params.FillTestVersions(c, 127)
	c.ConfigName = name
	return c
}

