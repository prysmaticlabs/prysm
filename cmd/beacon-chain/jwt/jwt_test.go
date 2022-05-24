package jwt

import (
	"flag"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/urfave/cli/v2"
)

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

	// TODO: // ensure length is what we expect: https://github.com/ethereum/execution-apis/issues/162
}
