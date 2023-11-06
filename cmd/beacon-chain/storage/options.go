package storage

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/node"
	"github.com/prysmaticlabs/prysm/v4/cmd"
	"github.com/urfave/cli/v2"
)

var (
	// StatePath defines a flag to start the beacon chain from a give genesis state file.
	BlobStoragePath = &cli.PathFlag{
		Name:  "blob-path",
		Usage: "Location for blob storage. Default location will be a 'blobs' directory next to the beacon db.",
	}
)

// BeaconNodeOptions sets configuration values on the node.BeaconNode value at node startup.
// Note: we can't get the right context from cli.Context, because the beacon node setup code uses this context to
// create a cancellable context. If we switch to using App.RunContext, we can set up this cancellation in the cmd
// package instead, and allow the functional options to tap into context cancellation.
func BeaconNodeOptions(c *cli.Context) (node.Option, error) {
	blobsPath := c.Path(BlobStoragePath.Name)
	if blobsPath == "" {
		blobsPath = c.String(cmd.DataDirFlag.Name)
	}
	return func(node *node.BeaconNode) (err error) {
		node.BlobStoragePath = blobsPath
		return nil
	}, nil
}
