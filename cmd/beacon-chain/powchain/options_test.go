package powchaincmd

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
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

	endpoints := parsePowchainEndpoints(ctx)
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
	parsePowchainEndpoints(ctx)
	assert.LogsContain(t, hook, "No ETH1 node specified to run with the beacon node")
}
