package node

import (
	"flag"
	"fmt"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

func TestConfigureHistoricalSlasher(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	hook := logTest.NewGlobal()

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Bool(flags.HistoricalSlasherNode.Name, true, "")
	cliCtx := cli.NewContext(&app, set, nil)

	configureHistoricalSlasher(cliCtx)

	assert.Equal(t, params.BeaconConfig().SlotsPerEpoch*4, params.BeaconConfig().SlotsPerArchivedPoint)
	assert.LogsContain(t, hook,
		fmt.Sprintf(
			"Setting %d slots per archive point and %d max RPC page size for historical slasher usage",
			params.BeaconConfig().SlotsPerArchivedPoint,
			int(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations))),
	)
}

func TestConfigureSafeSlotsToImportOptimistically(t *testing.T) {
	params.SetupTestConfigCleanup(t)

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Int(flags.SafeSlotsToImportOptimistically.Name, 0, "")
	require.NoError(t, set.Set(flags.SafeSlotsToImportOptimistically.Name, strconv.Itoa(128)))
	cliCtx := cli.NewContext(&app, set, nil)

	configureSafeSlotsToImportOptimistically(cliCtx)

	assert.Equal(t, types.Slot(128), params.BeaconConfig().SafeSlotsToImportOptimistically)
}

func TestConfigureSlotsPerArchivedPoint(t *testing.T) {
	params.SetupTestConfigCleanup(t)

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Int(flags.SlotsPerArchivedPoint.Name, 0, "")
	require.NoError(t, set.Set(flags.SlotsPerArchivedPoint.Name, strconv.Itoa(100)))
	cliCtx := cli.NewContext(&app, set, nil)

	configureSlotsPerArchivedPoint(cliCtx)

	assert.Equal(t, types.Slot(100), params.BeaconConfig().SlotsPerArchivedPoint)
}

func TestConfigureProofOfWork(t *testing.T) {
	params.SetupTestConfigCleanup(t)

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Uint64(flags.ChainID.Name, 0, "")
	set.Uint64(flags.NetworkID.Name, 0, "")
	set.String(flags.DepositContractFlag.Name, "", "")
	require.NoError(t, set.Set(flags.ChainID.Name, strconv.Itoa(100)))
	require.NoError(t, set.Set(flags.NetworkID.Name, strconv.Itoa(200)))
	require.NoError(t, set.Set(flags.DepositContractFlag.Name, "deposit-contract"))
	cliCtx := cli.NewContext(&app, set, nil)

	configureEth1Config(cliCtx)

	assert.Equal(t, uint64(100), params.BeaconConfig().DepositChainID)
	assert.Equal(t, uint64(200), params.BeaconConfig().DepositNetworkID)
	assert.Equal(t, "deposit-contract", params.BeaconConfig().DepositContractAddress)
}

func TestConfigureExecutionSetting(t *testing.T) {
	params.SetupTestConfigCleanup(t)

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.SuggestedFeeRecipient.Name, "", "")
	require.NoError(t, set.Set(flags.SuggestedFeeRecipient.Name, "0xB"))
	cliCtx := cli.NewContext(&app, set, nil)
	err := configureExecutionSetting(cliCtx)
	require.ErrorContains(t, "0xB is not a valid fee recipient address", err)

	require.NoError(t, set.Set(flags.SuggestedFeeRecipient.Name, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"))
	cliCtx = cli.NewContext(&app, set, nil)
	err = configureExecutionSetting(cliCtx)
	require.NoError(t, err)
	assert.Equal(t, common.HexToAddress("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"), params.BeaconConfig().DefaultFeeRecipient)

	require.NoError(t, set.Set(flags.SuggestedFeeRecipient.Name, "0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"))
	cliCtx = cli.NewContext(&app, set, nil)
	err = configureExecutionSetting(cliCtx)
	require.NoError(t, err)
	assert.Equal(t, common.HexToAddress("0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"), params.BeaconConfig().DefaultFeeRecipient)
}

func TestConfigureNetwork(t *testing.T) {
	params.SetupTestConfigCleanup(t)

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	bootstrapNodes := cli.StringSlice{}
	set.Var(&bootstrapNodes, cmd.BootstrapNode.Name, "")
	set.Int(flags.ContractDeploymentBlock.Name, 0, "")
	require.NoError(t, set.Set(cmd.BootstrapNode.Name, "node1"))
	require.NoError(t, set.Set(cmd.BootstrapNode.Name, "node2"))
	require.NoError(t, set.Set(flags.ContractDeploymentBlock.Name, strconv.Itoa(100)))
	cliCtx := cli.NewContext(&app, set, nil)

	configureNetwork(cliCtx)

	assert.DeepEqual(t, []string{"node1", "node2"}, params.BeaconNetworkConfig().BootstrapNodes)
	assert.Equal(t, uint64(100), params.BeaconNetworkConfig().ContractDeploymentBlock)
}

func TestConfigureInterop(t *testing.T) {
	params.SetupTestConfigCleanup(t)

	tests := []struct {
		name       string
		flagSetter func() *cli.Context
		configName string
	}{
		{
			"nothing set",
			func() *cli.Context {
				app := cli.App{}
				set := flag.NewFlagSet("test", 0)
				return cli.NewContext(&app, set, nil)
			},
			"mainnet",
		},
		{
			"mock votes set",
			func() *cli.Context {
				app := cli.App{}
				set := flag.NewFlagSet("test", 0)
				set.Bool(flags.InteropMockEth1DataVotesFlag.Name, false, "")
				assert.NoError(t, set.Set(flags.InteropMockEth1DataVotesFlag.Name, "true"))
				return cli.NewContext(&app, set, nil)
			},
			"interop",
		},
		{
			"validators set",
			func() *cli.Context {
				app := cli.App{}
				set := flag.NewFlagSet("test", 0)
				set.Uint64(flags.InteropNumValidatorsFlag.Name, 0, "")
				assert.NoError(t, set.Set(flags.InteropNumValidatorsFlag.Name, "20"))
				return cli.NewContext(&app, set, nil)
			},
			"interop",
		},
		{
			"genesis time set",
			func() *cli.Context {
				app := cli.App{}
				set := flag.NewFlagSet("test", 0)
				set.Uint64(flags.InteropGenesisTimeFlag.Name, 0, "")
				assert.NoError(t, set.Set(flags.InteropGenesisTimeFlag.Name, "200"))
				return cli.NewContext(&app, set, nil)
			},
			"interop",
		},
		{
			"genesis state set",
			func() *cli.Context {
				app := cli.App{}
				set := flag.NewFlagSet("test", 0)
				set.String(flags.InteropGenesisStateFlag.Name, "", "")
				assert.NoError(t, set.Set(flags.InteropGenesisStateFlag.Name, "/path/"))
				return cli.NewContext(&app, set, nil)
			},
			"interop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configureInteropConfig(tt.flagSetter())
			assert.DeepEqual(t, tt.configName, params.BeaconConfig().ConfigName)
		})
	}
}
