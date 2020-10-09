package cmd

import (
	"flag"
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/urfave/cli/v2"
)

func TestLoadFlagsFromConfig(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	context := cli.NewContext(&app, set, nil)

	assert.NoError(t, ioutil.WriteFile("flags_test.yaml", []byte("testflag: 100"), 0666))

	assert.NoError(t, set.Parse([]string{"test-command", "--" + ConfigFileFlag.Name, "flags_test.yaml"}))
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
			assert.Equal(t, 100, cliCtx.Int("testflag"))
			return nil
		},
	}
	assert.NoError(t, command.Run(context))
	assert.NoError(t, os.Remove("flags_test.yaml"))
}
