package node

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/urfave/cli"
)

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("web3provider", "ws//127.0.0.1:8546", "web3 provider ws or IPC endpoint")
	tmp := fmt.Sprintf("%s/datadir", os.TempDir())
	set.String("datadir", tmp, "node data directory")

	context := cli.NewContext(app, set, nil)

	_, err := NewBeaconNode(context)
	if err != nil {
		t.Fatalf("Failed to create BeaconNode: %v", err)
	}
	os.RemoveAll(tmp)
}
