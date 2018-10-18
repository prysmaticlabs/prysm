package node

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli"
)

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	hook := logTest.NewGlobal()
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", "/tmp/datadir", "the node data directory")
	set.String("pukey", "f24f252", "set public key of validator")
	context := cli.NewContext(app, set, nil)

	_, err := NewValidatorClient(context)
	if err != nil {
		t.Fatalf("Failed to create ValidatorClient: %v", err)
	}

	testutil.AssertLogsContain(t, hook, "PublicKey not detected, generating a new one")
}
