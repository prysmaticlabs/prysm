package tos

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/urfave/cli/v2"
)

func TestVerifyTosAcceptedOrPrompt(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(cmd.DataDirFlag.Name, "./tmpdir/", "")
	context := cli.NewContext(&app, set, nil)

	// replacing stdin
	tmpfile, err := ioutil.TempFile("", "tmp")
	require.NoError(t, err)
	origStdin := os.Stdin
	os.Stdin = tmpfile
	defer func() { os.Stdin = origStdin }()

	// prompt decline
	_, err = tmpfile.Write([]byte("decline"))
	require.NoError(t, err)
	_, err = tmpfile.Seek(0, 0)
	require.NoError(t, err)
	require.ErrorContains(t, "you have to accept Terms and Conditions", VerifyTosAcceptedOrPrompt(context))

	// prompt accept
	err = tmpfile.Truncate(0)
	require.NoError(t, err)
	_, err = tmpfile.Seek(0, 0)
	require.NoError(t, err)
	_, err = tmpfile.Write([]byte("accept"))
	require.NoError(t, err)
	_, err = tmpfile.Seek(0, 0)
	require.NoError(t, err)
	require.NoError(t, VerifyTosAcceptedOrPrompt(context))
	require.NoError(t, os.Remove(filepath.Join(context.String(cmd.DataDirFlag.Name), acceptTosFilename)))

	require.NoError(t, tmpfile.Close())
	require.NoError(t, os.Remove(tmpfile.Name()))

	// saved in file
	require.NoError(t, ioutil.WriteFile(filepath.Join(context.String(cmd.DataDirFlag.Name), acceptTosFilename), []byte(""), 0666))
	require.NoError(t, VerifyTosAcceptedOrPrompt(context))
	require.NoError(t, os.RemoveAll(context.String(cmd.DataDirFlag.Name)))

	// flag is set
	set.Bool(cmd.AcceptTosFlag.Name, true, "")
	require.NoError(t, VerifyTosAcceptedOrPrompt(context))
	require.NoError(t, os.RemoveAll(context.String(cmd.DataDirFlag.Name)))
}
