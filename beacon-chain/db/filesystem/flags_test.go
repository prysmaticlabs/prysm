package filesystem

import (
	"flag"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/urfave/cli/v2"
)

func TestConfigureBlobRetentionEpoch(t *testing.T) {
	MaxEpochsToPersistBlobs = params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest
	params.SetupTestConfigCleanup(t)
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)

	// Test case: Spec default.
	require.NoError(t, ConfigureBlobRetentionEpoch(cli.NewContext(&app, set, nil)))
	require.Equal(t, params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest, MaxEpochsToPersistBlobs)

	set.Uint64(flags.BlobRetentionEpoch.Name, 0, "")
	minEpochsForSidecarRequest := uint64(params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest)
	require.NoError(t, set.Set(flags.BlobRetentionEpoch.Name, strconv.FormatUint(2*minEpochsForSidecarRequest, 10)))
	cliCtx := cli.NewContext(&app, set, nil)

	// Test case: Input epoch is greater than or equal to spec value.
	require.NoError(t, ConfigureBlobRetentionEpoch(cliCtx))
	require.Equal(t, primitives.Epoch(2*minEpochsForSidecarRequest), MaxEpochsToPersistBlobs)

	// Test case: Input epoch is less than spec value.
	require.NoError(t, set.Set(flags.BlobRetentionEpoch.Name, strconv.FormatUint(minEpochsForSidecarRequest-1, 10)))
	cliCtx = cli.NewContext(&app, set, nil)
	err := ConfigureBlobRetentionEpoch(cliCtx)
	require.ErrorContains(t, "blob-retention-epochs smaller than spec default", err)
}
