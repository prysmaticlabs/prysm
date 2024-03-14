package storage

import (
	"path"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/db/filesystem"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/node"
	"github.com/prysmaticlabs/prysm/v5/cmd"
	"github.com/prysmaticlabs/prysm/v5/cmd/beacon-chain/storage/flags"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/urfave/cli/v2"
)

// BeaconNodeOptions sets configuration values on the node.BeaconNode value at node startup.
// Note: we can't get the right context from cli.Context, because the beacon node setup code uses this context to
// create a cancellable context. If we switch to using App.RunContext, we can set up this cancellation in the cmd
// package instead, and allow the functional options to tap into context cancellation.
func BeaconNodeOptions(c *cli.Context) ([]node.Option, error) {
	e, err := blobRetentionEpoch(c)
	if err != nil {
		return nil, err
	}
	opts := []node.Option{node.WithBlobStorageOptions(
		filesystem.WithBlobRetentionEpochs(e), filesystem.WithBasePath(blobStoragePath(c)),
	)}
	return opts, nil
}

func blobStoragePath(c *cli.Context) string {
	blobsPath := c.Path(flags.BlobStoragePathFlag.Name)
	if blobsPath == "" {
		// append a "blobs" subdir to the end of the data dir path
		blobsPath = path.Join(c.String(cmd.DataDirFlag.Name), "blobs")
	}
	return blobsPath
}

var errInvalidBlobRetentionEpochs = errors.New("value is smaller than spec minimum")

// blobRetentionEpoch returns the spec default MIN_EPOCHS_FOR_BLOB_SIDECARS_REQUEST
// or a user-specified flag overriding this value. If a user-specified override is
// smaller than the spec default, an error will be returned.
func blobRetentionEpoch(cliCtx *cli.Context) (primitives.Epoch, error) {
	spec := params.BeaconConfig().MinEpochsForBlobsSidecarsRequest
	if !cliCtx.IsSet(flags.BlobRetentionEpochFlag.Name) {
		return spec, nil
	}

	re := primitives.Epoch(cliCtx.Uint64(flags.BlobRetentionEpochFlag.Name))
	// Validate the epoch value against the spec default.
	if re < params.BeaconConfig().MinEpochsForBlobsSidecarsRequest {
		return spec, errors.Wrapf(errInvalidBlobRetentionEpochs, "%s=%d, spec=%d", flags.BlobRetentionEpochFlag.Name, re, spec)
	}

	return re, nil
}
