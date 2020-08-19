package node

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

// Ensure BeaconNode implements interfaces.
var _ = statefeed.Notifier(&BeaconNode{})

// Test that beacon chain node can close.
func TestNodeClose_OK(t *testing.T) {
	hook := logTest.NewGlobal()

	tmp := fmt.Sprintf("%s/datadirtest2", testutil.TempDir())
	if err := os.RemoveAll(tmp); err != nil {
		t.Log(err)
	}

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Bool("test-skip-pow", true, "skip pow dial")
	set.String("datadir", tmp, "node data directory")
	set.String("p2p-encoding", "ssz", "p2p encoding scheme")
	set.Bool("demo-config", true, "demo configuration")
	set.String("deposit-contract", "0x0000000000000000000000000000000000000000", "deposit contract address")

	context := cli.NewContext(&app, set, nil)

	node, err := NewBeaconNode(context)
	if err != nil {
		t.Fatalf("Failed to create BeaconNode: %v", err)
	}

	node.Close()

	require.LogsContain(t, hook, "Stopping beacon node")

	if err := os.RemoveAll(tmp); err != nil {
		t.Log(err)
	}
}

func TestBootStrapNodeFile(t *testing.T) {
	file, err := ioutil.TempFile(testutil.TempDir(), "bootstrapFile")
	if err != nil {
		t.Fatalf("Error in TempFile call:  %v", err)
	}
	defer func() {
		if err := os.Remove(file.Name()); err != nil {
			t.Log(err)
		}
	}()

	sampleNode0 := "- enr:-Ku4QMKVC_MowDsmEa20d5uGjrChI0h8_KsKXDmgVQbIbngZV0i" +
		"dV6_RL7fEtZGo-kTNZ5o7_EJI_vCPJ6scrhwX0Z4Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD" +
		"1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQJxCnE6v_x2ekgY_uo" +
		"E1rtwzvGy40mq9eD66XfHPBWgIIN1ZHCCD6A"
	sampleNode1 := "- enr:-TESTNODE2"
	sampleNode2 := "- enr:-TESTNODE3"
	err = ioutil.WriteFile(file.Name(), []byte(sampleNode0+"\n"+sampleNode1+"\n"+sampleNode2), 0644)
	if err != nil {
		t.Fatalf("Error in WriteFile call:  %v", err)
	}
	nodeList, err := readbootNodes(file.Name())
	if err != nil {
		t.Fatalf("Error in readbootNodes call:  %v", err)
	}
	if nodeList[0] != sampleNode0[2:] || nodeList[1] != sampleNode1[2:] || nodeList[2] != sampleNode2[2:] {
		// nodeList's YAML parsing will have removed the leading "- "
		t.Fatalf("TestBootStrapNodeFile failed.  Nodes do not match")
	}
}
