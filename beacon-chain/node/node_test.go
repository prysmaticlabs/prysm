package node

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli"
)

// Test that the beacon chain validator node build fails without PoW service.
func TestNodeValidator_Builds(t *testing.T) {
	tmp := fmt.Sprintf("%s/datadirtest1", os.TempDir())
	os.RemoveAll(tmp)

	if os.Getenv("TEST_NODE_PANIC") == "1" {
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("web3provider", "ws//127.0.0.1:8546", "web3 provider ws or IPC endpoint")
		tmp := fmt.Sprintf("%s/datadirtest1", os.TempDir())
		set.String("datadir", tmp, "node data directory")
		set.Bool("enable-powchain", true, "enable powchain")

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
	os.RemoveAll(tmp)
}

// Test that beacon chain node can close.
func TestNodeClose(t *testing.T) {
	hook := logTest.NewGlobal()

	tmp := fmt.Sprintf("%s/datadirtest2", os.TempDir())
	os.RemoveAll(tmp)

	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("web3provider", "ws//127.0.0.1:8546", "web3 provider ws or IPC endpoint")
	set.String("datadir", tmp, "node data directory")
	set.Bool("demo-config", true, "demo configuration")

	context := cli.NewContext(app, set, nil)

	node, err := NewBeaconNode(context)
	if err != nil {
		t.Fatalf("Failed to create BeaconNode: %v", err)
	}

	node.Close()

	testutil.AssertLogsContain(t, hook, "Stopping beacon node")

	os.RemoveAll(tmp)
}
