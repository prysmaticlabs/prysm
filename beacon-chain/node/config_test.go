package node

import (
	"flag"
	"fmt"
	"strconv"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
	require.NoError(t, set.Set(flags.DepositContractFlag.Name, "deposit"))
	cliCtx := cli.NewContext(&app, set, nil)

	configureProofOfWork(cliCtx)

	assert.Equal(t, uint64(100), params.BeaconConfig().DepositChainID)
	assert.Equal(t, uint64(200), params.BeaconConfig().DepositNetworkID)
	assert.Equal(t, "deposit", params.BeaconConfig().DepositContractAddress)
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
