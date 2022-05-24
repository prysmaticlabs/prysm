package jwt

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/urfave/cli/v2"
)

// ensure length is what we expect: https://github.com/ethereum/execution-apis/issues/162

func Test_generateJWTSecret(t *testing.T) {

	// INFO: This was created as a simple test that I could use to verify that tests are working
	//       can remove if useless
	t.Run("command should be available", func(t *testing.T) {
		generateJwtCommand := Commands
		require.Equal(t, true, generateJwtCommand.Name == "generate-jwt-secret")
	})

	t.Run("command should create file", func(t *testing.T) {
		// DEBT: This "magic string" should exist in a config file that tests + other code read from
		jwtFileName := "secret.jwt"

		// INFO: By default, the token is created within the root `prysm` directory -> /prysm/secret.jwt
		//       Because tests are executed within the jwt package directory, we can emulate default conditions by targeting /prysm/.
		//       This will need to be updated if we change the folder structure, and this test can be hardened by ensuring the working directory is /prysm/.
		jwtFileName = "../../../" + jwtFileName

		// INFO: We're testing to ensure that the file gets created, so first we need to ensure that the file doesn't exist.
		//       If it does exist, we delete it and ensure it's deleted.
		// INFO: The Stat() function returns an object that contains file information. If the file doesn’t exist, Stat() returns an error object.
		fileInfo, err := os.Stat(jwtFileName)
		if fileInfo != nil || err == nil {
			require.NoError(t, os.Remove(jwtFileName))
			fileInfo, err = os.Stat(jwtFileName)
			require.Equal(t, true, fileInfo == nil)
			require.Equal(t, true, err != nil)
		}

		// then we create the stuff we need to run the command that creates the file
		generateJwtCommand := Commands
		app := cli.App{}
		set := flag.NewFlagSet("test", 0) // <- not sure what this is or why it's needed, but without it, I get a panic. Possible to update the description and/or pattern to make the intent of this clearer to new devs like me?
		context := cli.NewContext(&app, set, nil)
		err = generateJwtCommand.Run(context) // <- hanging here, never hit breakpoint at :56
		require.NoError(t, err)
		fileInfo, err = os.Stat(jwtFileName)
		require.NoError(t, err)
		require.Equal(t, true, fileInfo != nil)
	})

	// t.Run("command invoked should generate file in root directory", func(t *testing.T) {
	// 	app := cli.App{}
	// 	set := flag.NewFlagSet("test", 0)
	// 	set.String(flags.ExecutionJWTSecretFlag.Name, "", "")
	// 	ctx := cli.NewContext(&app, set, nil)
	// 	secret, err := parseJWTSecretFromFile(ctx)
	// 	require.NoError(t, err)
	// 	require.Equal(t, true, secret == nil)
	// })
	//t.Run("no flag value specified leads to nil secret", func(t *testing.T) {
	//	app := cli.App{}
	//	set := flag.NewFlagSet("test", 0)
	//	set.String(flags.ExecutionJWTSecretFlag.Name, "", "")
	//	ctx := cli.NewContext(&app, set, nil)
	//	secret, err := parseJWTSecretFromFile(ctx)
	//	require.NoError(t, err)
	//	require.Equal(t, true, secret == nil)
	//})
	//t.Run("flag specified but no file found", func(t *testing.T) {
	//	app := cli.App{}
	//	set := flag.NewFlagSet("test", 0)
	//	set.String(flags.ExecutionJWTSecretFlag.Name, "/tmp/askdjkajsd", "")
	//	ctx := cli.NewContext(&app, set, nil)
	//	_, err := parseJWTSecretFromFile(ctx)
	//	require.ErrorContains(t, "no such file", err)
	//})
	//t.Run("empty string in file", func(t *testing.T) {
	//	app := cli.App{}
	//	set := flag.NewFlagSet("test", 0)
	//	fullPath := filepath.Join(t.TempDir(), "foohex")
	//	require.NoError(t, file.WriteFile(fullPath, []byte{}))
	//	set.String(flags.ExecutionJWTSecretFlag.Name, fullPath, "")
	//	ctx := cli.NewContext(&app, set, nil)
	//	_, err := parseJWTSecretFromFile(ctx)
	//	require.ErrorContains(t, "cannot be empty", err)
	//})
	//t.Run("less than 32 bytes", func(t *testing.T) {
	//	app := cli.App{}
	//	set := flag.NewFlagSet("test", 0)
	//	fullPath := filepath.Join(t.TempDir(), "foohex")
	//	secret := bytesutil.PadTo([]byte("foo"), 31)
	//	hexData := fmt.Sprintf("%#x", secret)
	//	require.NoError(t, file.WriteFile(fullPath, []byte(hexData)))
	//	set.String(flags.ExecutionJWTSecretFlag.Name, fullPath, "")
	//	ctx := cli.NewContext(&app, set, nil)
	//	_, err := parseJWTSecretFromFile(ctx)
	//	require.ErrorContains(t, "should be a hex string of at least 32 bytes", err)
	//})
	//t.Run("bad data", func(t *testing.T) {
	//	app := cli.App{}
	//	set := flag.NewFlagSet("test", 0)
	//	fullPath := filepath.Join(t.TempDir(), "foohex")
	//	secret := []byte("foo")
	//	require.NoError(t, file.WriteFile(fullPath, secret))
	//	set.String(flags.ExecutionJWTSecretFlag.Name, fullPath, "")
	//	ctx := cli.NewContext(&app, set, nil)
	//	_, err := parseJWTSecretFromFile(ctx)
	//	require.ErrorContains(t, "invalid byte", err)
	//})
	//t.Run("correct format", func(t *testing.T) {
	//	app := cli.App{}
	//	set := flag.NewFlagSet("test", 0)
	//	fullPath := filepath.Join(t.TempDir(), "foohex")
	//	secret := bytesutil.ToBytes32([]byte("foo"))
	//	secretHex := fmt.Sprintf("%#x", secret)
	//	require.NoError(t, file.WriteFile(fullPath, []byte(secretHex)))
	//	set.String(flags.ExecutionJWTSecretFlag.Name, fullPath, "")
	//	ctx := cli.NewContext(&app, set, nil)
	//	got, err := parseJWTSecretFromFile(ctx)
	//	require.NoError(t, err)
	//	require.DeepEqual(t, secret[:], got)
	//})
}

// DEBT: Copied from options.go, will delete if unused
func parseJWTSecretFromFile(c *cli.Context) ([]byte, error) {
	jwtSecretFile := c.String(flags.ExecutionJWTSecretFlag.Name)
	if jwtSecretFile == "" {
		return nil, nil
	}
	enc, err := file.ReadFileAsBytes(jwtSecretFile)
	if err != nil {
		return nil, err
	}
	strData := strings.TrimSpace(string(enc))
	if len(strData) == 0 {
		return nil, fmt.Errorf("provided JWT secret in file %s cannot be empty", jwtSecretFile)
	}
	secret, err := hex.DecodeString(strings.TrimPrefix(strData, "0x"))
	if err != nil {
		return nil, err
	}
	if len(secret) < 32 {
		return nil, errors.New("provided JWT secret should be a hex string of at least 32 bytes")
	}
	return secret, nil
}
