package jwt

import (
	"flag"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/urfave/cli/v2"
)

func Test_generateJWTSecret(t *testing.T) {
	t.Run("command should be available", func(t *testing.T) {
		generateJwtCommand := Commands
		require.Equal(t, true, generateJwtCommand.Name == "generate-auth-secret")
	})
	t.Run("command should create file", func(t *testing.T) {
		fileInfo, err := os.Stat(secretFileName)
		if fileInfo != nil || err == nil {
			require.NoError(t, os.Remove(secretFileName))
			fileInfo, err = os.Stat(secretFileName)
			require.Equal(t, true, fileInfo == nil)
			require.Equal(t, true, err != nil)
		}
		generateJwtCommand := Commands
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		context := cli.NewContext(&app, set, nil)
		err = generateJwtCommand.Run(context)
		require.NoError(t, err)
		fileInfo, err = os.Stat(secretFileName)
		require.NoError(t, err)
		require.Equal(t, true, fileInfo != nil)

	})
}
