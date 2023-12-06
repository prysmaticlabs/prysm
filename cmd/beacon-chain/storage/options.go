package storage

import (
	"path"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/filesystem"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/node"
	"github.com/prysmaticlabs/prysm/v4/cmd"
	"github.com/urfave/cli/v2"
)

var (
	// BlobStoragePath defines a flag to start the beacon chain from a give genesis state file.
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
	blobsPath := blobStoragePath(c)
	bs, err := filesystem.NewBlobStorage(blobsPath)
	if err != nil {
		return nil, err
	}
	return node.WithBlobStorage(bs), nil
}

func blobStoragePath(c *cli.Context) string {
	blobsPath := c.Path(BlobStoragePath.Name)
	if blobsPath == "" {
		// append a "blobs" subdir to the end of the data dir path
		blobsPath = path.Join(c.String(cmd.DataDirFlag.Name), "blobs")
	}
	return blobsPath
}
