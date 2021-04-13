package registration

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

func TestPowchainPreregistration(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.HTTPWeb3ProviderFlag.Name, "primary", "")
	fallback := cli.StringSlice{}
	err := fallback.Set("fallback1")
	require.NoError(t, err)
	err = fallback.Set("fallback2")
	require.NoError(t, err)
	set.Var(&fallback, flags.FallbackWeb3ProviderFlag.Name, "")
	ctx := cli.NewContext(&app, set, nil)

	address, endpoints, err := PowchainPreregistration(ctx)
	require.NoError(t, err)
	assert.Equal(t, params.BeaconConfig().DepositContractAddress, address)
	assert.DeepEqual(t, []string{"primary", "fallback1", "fallback2"}, endpoints)
}

func TestDepositContractAddress_Ok(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.HTTPWeb3ProviderFlag.Name, "provider", "")
	ctx := cli.NewContext(&app, set, nil)

	address, err := DepositContractAddress(ctx)
	require.NoError(t, err)
	assert.Equal(t, params.BeaconConfig().DepositContractAddress, address)
}

func TestDepositContractAddress_EmptyAddress(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.DepositContractAddress = ""
	params.OverrideBeaconConfig(config)
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(&app, set, nil)

	_, err := DepositContractAddress(ctx)
	assert.ErrorContains(t, "valid deposit contract is required", err)
}

func TestDepositContractAddress_NotHexAddress(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.DepositContractAddress = "abc?!"
	params.OverrideBeaconConfig(config)
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.HTTPWeb3ProviderFlag.Name, "", "")
	ctx := cli.NewContext(&app, set, nil)

	_, err := DepositContractAddress(ctx)
	assert.ErrorContains(t, "invalid deposit contract address given", err)
}

func TestDepositContractAddress_EmptyWeb3Provider(t *testing.T) {
	hook := logTest.NewGlobal()
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(&app, set, nil)

	address, err := DepositContractAddress(ctx)
	require.NoError(t, err)
	assert.Equal(t, params.BeaconConfig().DepositContractAddress, address)
	assert.LogsContain(t, hook, "No ETH1 node specified to run with the beacon node")
}
