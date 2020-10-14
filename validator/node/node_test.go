package node

import (
	"crypto/rand"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", testutil.TempDir()+"/datadir", "the node data directory")
	dir := testutil.TempDir() + "/keystore1"
	defer func() {
		assert.NoError(t, os.RemoveAll(dir))
	}()
	defer func() {
		assert.NoError(t, os.RemoveAll(testutil.TempDir()+"/datadir"))
	}()
	set.String("keystore-path", dir, "path to keystore")
	set.String("password", "1234", "validator account password")
	set.String("verbosity", "debug", "log verbosity")
	context := cli.NewContext(&app, set, nil)

	valClient, err := NewValidatorClient(context)
	require.NoError(t, err, "Failed to create ValidatorClient")
	err = valClient.db.Close()
	require.NoError(t, err)
}

// TestClearDB tests clearing the database
func TestClearDB(t *testing.T) {
	hook := logTest.NewGlobal()
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err, "Could not generate random number for file path")
	tmp := filepath.Join(testutil.TempDir(), fmt.Sprintf("datadirtest%d", randPath))
	require.NoError(t, os.RemoveAll(tmp))
	err = clearDB(tmp, true)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Removing database")
}
