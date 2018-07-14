package node

import (
	"testing"

	"github.com/urfave/cli"
)

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.NewApp()
	context := cli.NewContext(app, nil, nil)

	_, err := New(context)
	if err != nil {
		t.Fatalf("Failed to create BeaconNode: %v", err)
	}
}
