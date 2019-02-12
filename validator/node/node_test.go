package node

import (
	"flag"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"os"
	"testing"

	"github.com/urfave/cli"
)

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", "/tmp/datadir", "the node data directory")
	dir := os.TempDir() + "/keystore1"
	defer os.RemoveAll(dir)
	set.String("keystore-path", dir, "path to keystore")
	set.String("password", "1234", "validator account password")
	context := cli.NewContext(app, set, nil)

	if err := accounts.NewValidatorAccount(dir, "1234"); err != nil {
		t.Fatalf("Could not create validator account: %v", err)
	}
	_, err := NewValidatorClient(context)
	if err != nil {
		t.Fatalf("Failed to create ValidatorClient: %v", err)
	}
}
