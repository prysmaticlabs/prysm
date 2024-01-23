package backfill

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/node"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/sync/backfill"
	"github.com/urfave/cli/v2"
)

var (
	backfillBatchSizeName   = "backfill-batch-size"
	backfillWorkerCountName = "backfill-worker-count"
	// EnableExperimentalBackfill enables backfill for checkpoint synced nodes.
	// This flag will be removed onced backfill is enabled by default.
	EnableExperimentalBackfill = &cli.BoolFlag{
		Name: "enable-experimental-backfill",
		Usage: "Backfill is still experimental at this time." +
			"It will only be enabled if this flag is specified and the node was started using checkpoint sync.",
	}
	// BackfillBatchSize allows users to tune block backfill request sizes to maximize network utilization
	// at the cost of higher memory.
	BackfillBatchSize = &cli.Uint64Flag{
		Name: backfillBatchSizeName,
		Usage: "Number of blocks per backfill batch. " +
			"A larger number will request more blocks at once from peers, but also consume more system memory to " +
			"hold batches in memory during processing. This has a multiplicative effect with " + backfillWorkerCountName,
		Value: 64,
	}
	// BackfillWorkerCount allows users to tune the number of concurrent backfill batches to download, to maximize
	// network utilization at the cost of higher memory.
	BackfillWorkerCount = &cli.IntFlag{
		Name: backfillWorkerCountName,
		Usage: "Number of concurrent backfill batch requests. " +
			"A larger number will better utilize network resources, up to a system-dependent limit, but will also " +
			"consume more system memory to hold batches in memory during processing. Multiply by backfill-batch-size and " +
			"average block size (~2MB before deneb) to find the right number for your system. " +
			"This has a multiplicatice effect with " + backfillBatchSizeName,
		Value: 2,
	}
)

// BeaconNodeOptions sets the appropriate functional opts on the *node.BeaconNode value, to decouple options
// from flag parsing.
func BeaconNodeOptions(c *cli.Context) ([]node.Option, error) {
	opt := func(node *node.BeaconNode) (err error) {
		node.BackfillOpts = []backfill.ServiceOption{
			backfill.WithBatchSize(c.Uint64(BackfillBatchSize.Name)),
			backfill.WithWorkerCount(c.Int(BackfillWorkerCount.Name)),
			backfill.WithEnableBackfill(c.Bool(EnableExperimentalBackfill.Name)),
		}
		return nil
	}
	return []node.Option{opt}, nil
}
