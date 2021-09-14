package node

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)

	m.Run()
}

// Test that slasher node can close.
func TestNodeClose_OK(t *testing.T) {
	hook := logTest.NewGlobal()

	tmp := fmt.Sprintf("%s/datadirtest2", t.TempDir())

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("beacon-rpc-provider", "localhost:4232", "beacon node RPC server")
	set.String("datadir", tmp, "node data directory")

	context := cli.NewContext(&app, set, nil)

	node, err := New(context)
	require.NoError(t, err, "Failed to create slasher node")

	node.Close()

	require.LogsContain(t, hook, "Stopping hash slinging slasher")
	require.NoError(t, os.RemoveAll(tmp))
}

// TestClearDB tests clearing the database
func TestClearDB(t *testing.T) {
	hook := logTest.NewGlobal()

	tmp := filepath.Join(t.TempDir(), "datadirtest")

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", tmp, "node data directory")
	set.Bool(cmd.ForceClearDB.Name, true, "force clear db")

	context := cli.NewContext(&app, set, nil)
	slasherNode, err := New(context)
	require.NoError(t, err)

	require.LogsContain(t, hook, "Removing database")
	err = slasherNode.db.Close()
	require.NoError(t, err)
	require.NoError(t, os.RemoveAll(tmp))
}
