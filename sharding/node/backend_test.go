package node

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/geth-sharding/sharding"

	cli "gopkg.in/urfave/cli.v1"
)

// Verifies that ShardEthereum implements the Node interface.
var _ = sharding.Node(&ShardEthereum{})

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)

	context := cli.NewContext(app, set, nil)

	_, err := New(context)
	if err != nil {
		t.Fatalf("Failed to create ShardEthereum: %v", err)
	}
}
