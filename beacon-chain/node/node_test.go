package node

import (
	"flag"
	"fmt"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli"
	"os"
	"os/exec"
	"testing"
)

// Test that the beacon chain observer node can build with default flag values.
func TestNodeObserver_Builds(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("web3provider", "ws//127.0.0.1:8546", "web3 provider ws or IPC endpoint")
	tmp := fmt.Sprintf("%s/datadir", os.TempDir())
	set.String("datadir", tmp, "node data directory")
	set.Bool("simulator", true, "want to be a simulator?")

	context := cli.NewContext(app, set, nil)

	_, err := NewBeaconNode(context)
	if err != nil {
		t.Fatalf("Failed to create BeaconNode: %v", err)
	}

	os.RemoveAll(tmp)
}

// Test that the beacon chain validator node build fails without PoW service.
func TestNodeValidator_Builds(t *testing.T) {
	if os.Getenv("TEST_NODE_PANIC") == "1" {
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("web3provider", "ws//127.0.0.1:8546", "web3 provider ws or IPC endpoint")
		tmp := fmt.Sprintf("%s/datadir", os.TempDir())
		set.String("datadir", tmp, "node data directory")
		set.Bool("validator", true, "want to be a validator?")

		context := cli.NewContext(app, set, nil)

		NewBeaconNode(context)
	}

	// Start a subprocess to test beacon node crashes.
	cmd := exec.Command(os.Args[0], "-test.run=TestNodeValidator_Builds")
	cmd.Env = append(os.Environ(), "TEST_NODE_PANIC=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// Check beacon node program exited.
	err := cmd.Wait()
	if e, ok := err.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", err)
	}
}

func TestNodeClose(t *testing.T) {
	hook := logTest.NewGlobal()
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("web3provider", "ws//127.0.0.1:8546", "web3 provider ws or IPC endpoint")
	tmp := fmt.Sprintf("%s/datadir", os.TempDir())
	set.String("datadir", tmp, "node data directory")

	context := cli.NewContext(app, set, nil)

	node, err := NewBeaconNode(context)
	if err != nil {
		t.Fatalf("Failed to create BeaconNode: %v", err)
	}

	node.Close()

	testutil.AssertLogsContain(t, hook, "Stopping beacon node")

	os.RemoveAll(tmp)
}
