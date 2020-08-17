package node

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

// Test that slasher node can close.
func TestNodeClose_OK(t *testing.T) {
	hook := logTest.NewGlobal()

	tmp := fmt.Sprintf("%s/datadirtest2", testutil.TempDir())
	if err := os.RemoveAll(tmp); err != nil {
		t.Fatal(err)
	}

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("beacon-rpc-provider", "localhost:4232", "beacon node RPC server")
	set.String("datadir", tmp, "node data directory")

	context := cli.NewContext(&app, set, nil)

	node, err := NewSlasherNode(context)
	if err != nil {
		t.Fatalf("Failed to create SlasherNode: %v", err)
	}

	node.Close()

	require.LogsContain(t, hook, "Stopping hash slinging slasher")

	if err := os.RemoveAll(tmp); err != nil {
		t.Fatal(err)
	}
}
