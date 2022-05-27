package jwt

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/urfave/cli/v2"
)

func Test_generateJWTSecret(t *testing.T) {
	t.Run("command should be available", func(t *testing.T) {
		generateJwtCommand := Commands
		require.Equal(t, true, generateJwtCommand.Name == "generate-auth-secret")
	})
	t.Run("junk file path fails", func(t *testing.T) {
		junk := "/adj$7@&  9a.\""
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		set.String(cmd.JwtOutputFileFlag.Name, junk, "")
		require.NoError(t, set.Set(cmd.JwtOutputFileFlag.Name, junk))

		cliCtx := cli.NewContext(&app, set, nil)
		err := generateAuthSecretInFile(cliCtx)
		// An error should return
		require.ErrorContains(t, "is not a valid file path", err)
		// when feeding junk, the call to os.Stat should fail
		_, err = os.Stat(junk)
		require.Equal(t, true, err != nil)
	})
	t.Run("should create proper file in current directory", func(t *testing.T) {
		require.NoError(t, os.RemoveAll(secretFileName))
		t.Cleanup(func() {
			require.NoError(t, os.RemoveAll(secretFileName))
		})
		app := cli.App{}
		// TODO: improve the developer experience of this NewFlagSet call, the intent of this isn't clear to me, and it probably won't be clear to others. meta: these comments are intentional ways to identify debt and todos so we can reduce risk/cost and improve clarity. before deleting, maybe we can align on how to track these todos?
		set := flag.NewFlagSet("test", 0)
		cliCtx := cli.NewContext(&app, set, nil)
		err := generateAuthSecretInFile(cliCtx)
		require.NoError(t, err)

		// We check the file has the contents we expect.
		checkAuthFileIntegrity(t, secretFileName)
	})
	t.Run("should create proper file in specified folder", func(t *testing.T) {
		customOutput := filepath.Join("data", "item.txt")
		require.NoError(t, os.RemoveAll(filepath.Dir(customOutput)))
		t.Cleanup(func() {
			require.NoError(t, os.RemoveAll(filepath.Dir(customOutput)))
		})
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		set.String(cmd.JwtOutputFileFlag.Name, customOutput, "")
		require.NoError(t, set.Set(cmd.JwtOutputFileFlag.Name, customOutput))

		cliCtx := cli.NewContext(&app, set, nil)
		err := generateAuthSecretInFile(cliCtx)
		require.NoError(t, err)

		// We check the file has the contents we expect.
		checkAuthFileIntegrity(t, customOutput)
	})
	t.Run("creates proper file in nested specified folder", func(t *testing.T) {
		customOutput := filepath.Join("data", "nest", "nested", "item.txt")
		require.NoError(t, os.RemoveAll(filepath.Dir(customOutput)))
		t.Cleanup(func() {
			require.NoError(t, os.RemoveAll(filepath.Dir(customOutput)))
		})
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		set.String(cmd.JwtOutputFileFlag.Name, customOutput, "")
		require.NoError(t, set.Set(cmd.JwtOutputFileFlag.Name, customOutput))

		cliCtx := cli.NewContext(&app, set, nil)
		err := generateAuthSecretInFile(cliCtx)
		require.NoError(t, err)

		// We check the file has the contents we expect.
		checkAuthFileIntegrity(t, customOutput)
	})
}

func checkAuthFileIntegrity(t testing.TB, fPath string) {
	fileInfo, err := os.Stat(fPath)
	require.NoError(t, err)
	require.Equal(t, true, fileInfo != nil)

	enc, err := file.ReadFileAsBytes(fPath)
	require.NoError(t, err)
	decoded, err := hexutil.Decode(string(enc))
	require.NoError(t, err)
	require.Equal(t, 32, len(decoded))
}
