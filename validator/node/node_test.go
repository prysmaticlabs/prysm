package node

import (
	"flag"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/urfave/cli"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		CacheTreeHash: false,
	})
}

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", testutil.TempDir()+"/datadir", "the node data directory")
	dir := testutil.TempDir() + "/keystore1"
	defer os.RemoveAll(dir)
	defer os.RemoveAll(testutil.TempDir() + "/datadir")
	set.String("keystore-path", dir, "path to keystore")
	set.String("password", "1234", "validator account password")
	context := cli.NewContext(app, set, nil)

	if err := accounts.NewValidatorAccount(dir, "1234"); err != nil {
		t.Fatalf("Could not create validator account: %v", err)
	}
	_, err := NewValidatorClient(context, "1234")
	if err != nil {
		t.Fatalf("Failed to create ValidatorClient: %v", err)
	}
}
