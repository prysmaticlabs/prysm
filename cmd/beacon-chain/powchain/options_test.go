package powchaincmd

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

func TestPowchainCmd(t *testing.T) {
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

func TestPowchainPreregistration_EmptyWeb3Provider(t *testing.T) {
	hook := logTest.NewGlobal()
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.HTTPWeb3ProviderFlag.Name, "", "")
	fallback := cli.StringSlice{}
	set.Var(&fallback, flags.FallbackWeb3ProviderFlag.Name, "")
	ctx := cli.NewContext(&app, set, nil)

	_, _, err := PowchainPreregistration(ctx)
	require.NoError(t, err)
	assert.LogsContain(t, hook, "No ETH1 node specified to run with the beacon node")
}

func TestDepositContractAddress_Ok(t *testing.T) {
	address, err := depositContractAddress()
	require.NoError(t, err)
	assert.Equal(t, params.BeaconConfig().DepositContractAddress, address)
}

func TestDepositContractAddress_EmptyAddress(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.DepositContractAddress = ""
	params.OverrideBeaconConfig(config)

	_, err := depositContractAddress()
	assert.ErrorContains(t, "valid deposit contract is required", err)
}

func TestDepositContractAddress_NotHexAddress(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.DepositContractAddress = "abc?!"
	params.OverrideBeaconConfig(config)

	_, err := depositContractAddress()
	assert.ErrorContains(t, "invalid deposit contract address given", err)
}
