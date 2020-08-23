package node

import (
	"flag"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	v1 "github.com/prysmaticlabs/prysm/validator/accounts/v1"
	"github.com/urfave/cli/v2"
)

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", testutil.TempDir()+"/datadir", "the node data directory")
	dir := testutil.TempDir() + "/keystore1"
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Log(err)
		}
	}()
	defer func() {
		if err := os.RemoveAll(testutil.TempDir() + "/datadir"); err != nil {
			t.Log(err)
		}
	}()
	set.String("keystore-path", dir, "path to keystore")
	set.String("password", "1234", "validator account password")
	set.String("verbosity", "debug", "log verbosity")
	set.Bool("disable-accounts-v2", true, "disabling accounts v2")
	context := cli.NewContext(&app, set, nil)

	require.NoError(t, v1.NewValidatorAccount(dir, "1234"), "Could not create validator account")
	_, err := NewValidatorClient(context)
	require.NoError(t, err, "Failed to create ValidatorClient")
}
