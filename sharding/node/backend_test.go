package node

import (
	"testing"

	"flag"
	"github.com/ethereum/go-ethereum/sharding"

	cli "gopkg.in/urfave/cli.v1"
)

// Verifies that ShardEthereum implements the Node interface.
var _ = sharding.Node(&ShardEthereum{})

func TestNode_Builds(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)

	context := cli.NewContext(app, set, nil)

	s, err := New(context)
	if err != nil {
		t.Fatalf("Failed to create ShardEthereum: %v", err)
	}

	_ = s
}
