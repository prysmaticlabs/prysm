package checkpoint

import (
	"fmt"
	"github.com/prysmaticlabs/prysm/beacon-chain/node"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/checkpoint"
	"github.com/urfave/cli/v2"
	"os"
)

var (
	// StatePath defines a flag to start the beacon chain from a give genesis state file.
	StatePath = &cli.PathFlag{
		Name: "checkpoint-state",
		Usage: "Rather than syncing from genesis, you can start processing from a ssz-serialized BeaconState+Block." +
			" This flag allows you to specify a local file containing the checkpoint BeaconState to load.",
	}
	// BlockPath is required when using StatePath to also provide the latest integrated block.
	BlockPath = &cli.PathFlag{
		Name: "checkpoint-block",
		Usage: "Rather than syncing from genesis, you can start processing from a ssz-serialized BeaconState+Block." +
			" This flag allows you to specify a local file containing the checkpoint Block to load.",
	}
)

func BeaconNodeOptions(c *cli.Context) (node.Option, error) {
	blockPath := c.Path(BlockPath.Name)
	statePath := c.Path(StatePath.Name)
	if blockPath == "" && statePath == "" {
		return nil, nil
	}
	if blockPath != "" && statePath == "" {
		return nil, fmt.Errorf("--checkpoint-block specified, but not --checkpoint-state. both are required")
	}
	if blockPath == "" && statePath != "" {
		return nil, fmt.Errorf("--checkpoint-state specified, but not --checkpoint-block. both are required")
	}

	return func(node *node.BeaconNode) error {
		blockFH, err := os.Open(blockPath)
		if err != nil {
			return err
		}
		stateFH, err := os.Open(statePath)
		if err != nil {
			return err
		}

		node.CheckpointInitializer = &checkpoint.Initializer{
			BlockReadCloser: blockFH,
			StateReadCloser: stateFH,
		}
		return nil
	}, nil
}
