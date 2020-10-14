package tos

import (
	"flag"
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/urfave/cli/v2"
)

func TestVerifyTosAcceptedOrPrompt(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(cmd.DataDirFlag.Name, ".", "")
	context := cli.NewContext(&app, set, nil)

	// saved in file
	require.NoError(t, ioutil.WriteFile(context.String(cmd.DataDirFlag.Name)+"/"+acceptTosFilename, []byte(""), 0666))
	require.NoError(t, VerifyTosAcceptedOrPrompt(context))
	require.NoError(t, os.Remove(context.String(cmd.DataDirFlag.Name)+"/"+acceptTosFilename))

	// not set
	require.ErrorContains(t, "could not scan text input", VerifyTosAcceptedOrPrompt(context))

	// is set
	set.Bool(cmd.AcceptTosFlag.Name, true, "")
	require.NoError(t, VerifyTosAcceptedOrPrompt(context))

	require.NoError(t, os.Remove(context.String(cmd.DataDirFlag.Name)+"/"+acceptTosFilename))
}
