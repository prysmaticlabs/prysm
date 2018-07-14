package node

import (
	"flag"
	"testing"

	"github.com/urfave/cli"
)

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("web3provider", 0)
	context := cli.NewContext(app, set, nil)

	_, err := New(context)
	if err != nil {
		t.Fatalf("Failed to create BeaconNode: %v", err)
	}
}
