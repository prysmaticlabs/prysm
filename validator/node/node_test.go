package node

import (
	"flag"
	"testing"

	"github.com/urfave/cli"
)

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", "/tmp/datadir", "the node data directory")
	context := cli.NewContext(app, set, nil)

	_, err := NewShardInstance(context)
	if err != nil {
		t.Fatalf("Failed to create ShardEthereum: %v", err)
	}
}
