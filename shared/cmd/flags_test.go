package cmd

import (
	"flag"
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/urfave/cli/v2"
)

func TestLoadFlagsFromConfig(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	context := cli.NewContext(&app, set, nil)

	require.NoError(t, ioutil.WriteFile("flags_test.yaml", []byte("testflag: 100"), 0666))

	require.NoError(t, set.Parse([]string{"test-command", "--" + ConfigFileFlag.Name, "flags_test.yaml"}))
	command := &cli.Command{
		Name: "test-command",
		Flags: WrapFlags([]cli.Flag{
			&cli.StringFlag{
				Name: ConfigFileFlag.Name,
			},
			&cli.IntFlag{
				Name:  "testflag",
				Value: 0,
			},
		}),
		Before: func(cliCtx *cli.Context) error {
			return LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags)
		},
		Action: func(cliCtx *cli.Context) error {
			require.Equal(t, 100, cliCtx.Int("testflag"))
			return nil
		},
	}
	require.NoError(t, command.Run(context))
	require.NoError(t, os.Remove("flags_test.yaml"))
}
