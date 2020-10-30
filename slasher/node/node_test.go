package node

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/cmd"
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

// TestClearDB tests clearing the database
func TestClearDB(t *testing.T) {
	hook := logTest.NewGlobal()

	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random number for file path")
	tmp := filepath.Join(testutil.TempDir(), fmt.Sprintf("datadirtest%d", randPath))
	require.NoError(t, os.RemoveAll(tmp))

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", tmp, "node data directory")
	set.Bool(cmd.ForceClearDB.Name, true, "force clear db")

	context := cli.NewContext(&app, set, nil)
	slasherNode, err := NewSlasherNode(context)
	require.NoError(t, err)

	require.LogsContain(t, hook, "Removing database")
	err = slasherNode.db.Close()
	require.NoError(t, err)
	require.NoError(t, os.RemoveAll(tmp))
}
