package kv

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/v4/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/urfave/cli/v2"
)

var maxEpochsToPersistBlobs primitives.Epoch

// ConfigureBlobRetentionEpoch sets the for blob retention based on command-line context. It sets the local config `maxEpochsToPersistBlobs`.
// If the flag is not set, the spec default `MinEpochsForBlobsSidecarsRequest` is used.
// An error if the input epoch is smaller than the spec default value.
func ConfigureBlobRetentionEpoch(cliCtx *cli.Context) error {
	// Check if the blob retention epoch flag is set.
	if cliCtx.IsSet(flags.BlobRetentionEpoch.Name) {
		// Retrieve and cast the epoch value.
		epochValue := cliCtx.Uint64(flags.BlobRetentionEpoch.Name)
		e := primitives.Epoch(epochValue)

		// Validate the epoch value against the spec default.
		if e < params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest {
			return fmt.Errorf("input blob retention epoch smaller than spec default, %d < %d", e, params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest)
		}

		maxEpochsToPersistBlobs = e
	} else {
		// Set the blob retention epoch to the spec default.
		maxEpochsToPersistBlobs = params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest
	}

	return nil
}
