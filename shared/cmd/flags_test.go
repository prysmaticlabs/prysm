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

func TestValidateNoArgs(t *testing.T) {
	app := &cli.App{
		Before: ValidateNoArgs,
		Action: func(c *cli.Context) error {
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "foo",
			},
		},
		Commands: []*cli.Command{
			{
				Name: "bar",
				Subcommands: []*cli.Command{
					{
						Name: "subComm1",
						Subcommands: []*cli.Command{
							{
								Name: "subComm3",
							},
						},
					},
					{
						Name: "subComm2",
						Subcommands: []*cli.Command{
							{
								Name: "subComm4",
							},
						},
					},
				},
			},
		},
	}

	// It should not work with a bogus argument
	err := app.Run([]string{"command", "foo"})
	require.ErrorContains(t, "unrecognized argument: foo", err)
	// It should work with registered flags
	err = app.Run([]string{"command", "--foo=bar"})
	require.NoError(t, err)
	// It should work with subcommands.
	err = app.Run([]string{"command", "bar"})
	require.NoError(t, err)
	// It should fail on unregistered flag (default logic in urfave/cli).
	err = app.Run([]string{"command", "bar", "--baz"})
	require.ErrorContains(t, "flag provided but not defined", err)

	// Handle Nested Subcommands

	err = app.Run([]string{"command", "bar", "subComm1"})
	require.NoError(t, err)

	err = app.Run([]string{"command", "bar", "subComm2"})
	require.NoError(t, err)

	// Should fail from unknown subcommands.
	err = app.Run([]string{"command", "bar", "subComm3"})
	require.ErrorContains(t, "unrecognized argument: subComm3", err)

	err = app.Run([]string{"command", "bar", "subComm4"})
	require.ErrorContains(t, "unrecognized argument: subComm4", err)

	// Should fail with invalid double nested subcommands.
	err = app.Run([]string{"command", "bar", "subComm1", "subComm2"})
	require.ErrorContains(t, "unrecognized argument: subComm2", err)

	err = app.Run([]string{"command", "bar", "subComm1", "subComm4"})
	require.ErrorContains(t, "unrecognized argument: subComm4", err)

	err = app.Run([]string{"command", "bar", "subComm2", "subComm1"})
	require.ErrorContains(t, "unrecognized argument: subComm1", err)

	err = app.Run([]string{"command", "bar", "subComm2", "subComm3"})
	require.ErrorContains(t, "unrecognized argument: subComm3", err)

	// Should pass with correct nested double subcommands.
	err = app.Run([]string{"command", "bar", "subComm1", "subComm3"})
	require.NoError(t, err)

	err = app.Run([]string{"command", "bar", "subComm2", "subComm4"})
	require.NoError(t, err)
}
