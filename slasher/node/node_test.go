package node

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)

	os.Exit(m.Run())
}

// Test that slasher node can close.
func TestNodeClose_OK(t *testing.T) {
	hook := logTest.NewGlobal()

	tmp := fmt.Sprintf("%s/datadirtest2", testutil.TempDir())
	require.NoError(t, os.RemoveAll(tmp))

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("beacon-rpc-provider", "localhost:4232", "beacon node RPC server")
	set.String("datadir", tmp, "node data directory")

	context := cli.NewContext(&app, set, nil)

	node, err := NewSlasherNode(context)
	require.NoError(t, err, "Failed to create slasher node")

	node.Close()

	require.LogsContain(t, hook, "Stopping hash slinging slasher")
	require.NoError(t, os.RemoveAll(tmp))
}
