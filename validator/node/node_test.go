package node

import (
	"flag"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
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
	context := cli.NewContext(&app, set, nil)

	if err := v1.NewValidatorAccount(dir, "1234"); err != nil {
		t.Fatalf("Could not create validator account: %v", err)
	}
	_, err := NewValidatorClient(context)
	if err != nil {
		t.Fatalf("Failed to create ValidatorClient: %v", err)
	}
}
