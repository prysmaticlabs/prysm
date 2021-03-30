package node

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

// Ensure BeaconNode implements interfaces.
var _ statefeed.Notifier = (*BeaconNode)(nil)

// Test that beacon chain node can close.
func TestNodeClose_OK(t *testing.T) {
	hook := logTest.NewGlobal()

	tmp := fmt.Sprintf("%s/datadirtest2", t.TempDir())

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Bool("test-skip-pow", true, "skip pow dial")
	set.String("datadir", tmp, "node data directory")
	set.String("p2p-encoding", "ssz", "p2p encoding scheme")
	set.Bool("demo-config", true, "demo configuration")
	set.String("deposit-contract", "0x0000000000000000000000000000000000000000", "deposit contract address")

	context := cli.NewContext(&app, set, nil)

	node, err := New(context)
	require.NoError(t, err)

	node.Close()

	require.LogsContain(t, hook, "Stopping beacon node")
	require.NoError(t, os.RemoveAll(tmp))
}

func TestBootStrapNodeFile(t *testing.T) {
	file, err := ioutil.TempFile(t.TempDir(), "bootstrapFile")
	require.NoError(t, err)

	sampleNode0 := "- enr:-Ku4QMKVC_MowDsmEa20d5uGjrChI0h8_KsKXDmgVQbIbngZV0i" +
		"dV6_RL7fEtZGo-kTNZ5o7_EJI_vCPJ6scrhwX0Z4Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD" +
		"1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQJxCnE6v_x2ekgY_uo" +
		"E1rtwzvGy40mq9eD66XfHPBWgIIN1ZHCCD6A"
	sampleNode1 := "- enr:-TESTNODE2"
	sampleNode2 := "- enr:-TESTNODE3"
	err = ioutil.WriteFile(file.Name(), []byte(sampleNode0+"\n"+sampleNode1+"\n"+sampleNode2), 0644)
	require.NoError(t, err, "Error in WriteFile call")
	nodeList, err := readbootNodes(file.Name())
	require.NoError(t, err, "Error in readbootNodes call")
	assert.Equal(t, sampleNode0[2:], nodeList[0], "Unexpected nodes")
	assert.Equal(t, sampleNode1[2:], nodeList[1], "Unexpected nodes")
	assert.Equal(t, sampleNode2[2:], nodeList[2], "Unexpected nodes")
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
	_, err := New(context)
	require.NoError(t, err)

	require.LogsContain(t, hook, "Removing database")
	require.NoError(t, os.RemoveAll(tmp))
}
