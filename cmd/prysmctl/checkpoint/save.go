package checkpoint

import (
	"fmt"
	"os"
	"time"

	"github.com/prysmaticlabs/prysm/api/client/openapi"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const SLOTS_PER_EPOCH = 32

var saveFlags = struct {
	BeaconNodeHost string
	Timeout        string
	BlockHex       string
	BlockSavePath  string
	StateHex       string
	Epoch          int
}{}

var saveCmd = &cli.Command{
	Name:   "save",
	Usage:  "query for the current weak subjectivity period epoch, then download the corresponding state and block. To be used for checkpoint sync.",
	Action: cliActionSave,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "beacon-node-host",
			Usage:       "host:port for beacon node connection",
			Destination: &saveFlags.BeaconNodeHost,
			Value:       "localhost:3500",
		},
		&cli.StringFlag{
			Name:        "http-timeout",
			Usage:       "timeout for http requests made to beacon-node-url (uses duration format, ex: 2m31s). default: 2m",
			Destination: &saveFlags.Timeout,
			Value:       "4m",
		},
		&cli.IntFlag{
			Name:        "epoch",
			Usage:       "instead of state-root, epoch can be used to find the BeaconState for the slot at the epoch boundary.",
			Destination: &saveFlags.Epoch,
		},
	},
}

func cliActionSave(c *cli.Context) error {
	f := saveFlags
	opts := make([]openapi.ClientOpt, 0)
	log.Printf("--beacon-node-url=%s", f.BeaconNodeHost)
	timeout, err := time.ParseDuration(f.Timeout)
	if err != nil {
		return err
	}
	opts = append(opts, openapi.WithTimeout(timeout))
	client, err := openapi.NewClient(saveFlags.BeaconNodeHost, opts...)
	if err != nil {
		return err
	}

	if saveFlags.Epoch > 0 {
		return saveCheckpointByEpoch(client, uint64(saveFlags.Epoch))
	}

	return saveCheckpoint(client)
}

func saveCheckpoint(client *openapi.Client) error {
	epoch, err := client.GetWeakSubjectivityCheckpointEpoch()
	if err != nil {
		return err
	}

	log.Printf("Beacon node computes the current weak subjectivity checkpoint as epoch = %d", epoch)
	return saveCheckpointByEpoch(client, epoch)
}

func saveCheckpointByEpoch(client *openapi.Client, epoch uint64) error {
	slot := epoch * SLOTS_PER_EPOCH

	block, err := client.GetBlockBySlot(slot)
	blockRoot, err := block.Block.HashTreeRoot()
	if err != nil {
		return err
	}
	blockRootHex := fmt.Sprintf("%#x", blockRoot)
	log.Printf("retrieved block at slot %d with root=%s", slot, fmt.Sprintf("%#x", blockRoot))
	blockStateRoot := block.Block.StateRoot
	log.Printf("retrieved block has state root %s", fmt.Sprintf("%#x", blockStateRoot))

	// assigning this variable to make it extra obvious that the state slot is different
	stateSlot := slot + 1
	// using the state at (slot % 32 = 1) instead of the epoch boundary ensures the
	// next block applied to the state will have the block at the weak subjectivity checkpoint
	// as its parent, satisfying prysm's sync code current verification that the parent block is present in the db
	state, err := client.GetStateBySlot(stateSlot)
	if err != nil {
		return err
	}
	stateRoot, err := state.HashTreeRoot()
	if err != nil {
		return err
	}
	log.Printf("retrieved state for checkpoint at slot %d, w/ root=%s", slot, fmt.Sprintf("%#x", stateRoot))
	latestBlockRoot, err := state.LatestBlockHeader.HashTreeRoot()
	if err != nil {
		return err
	}
	// we only want to provide checkpoints+state pairs where the state integrates the checkpoint block as its latest root
	// this ensures that when syncing begins from the provided state, the next block in the chain can find the
	// latest block in the db.
	if blockRoot == latestBlockRoot {
		log.Printf("State latest_block_header root matches block root=%#x", latestBlockRoot)
	} else {
		return fmt.Errorf("fatal error, state latest_block_header root=%#x, does not match block root=%#x", latestBlockRoot)
	}

	bb, err := block.MarshalSSZ()
	if err != nil {
		return err
	}
	blockPath := fmt.Sprintf("block-%s.ssz", blockRootHex)
	log.Printf("saving ssz-encoded block to to %s", blockPath)
	err = os.WriteFile(blockPath, bb, 0644)
	if err != nil {
		return err
	}

	sb, err := state.MarshalSSZ()
	if err != nil {
		return err
	}
	statePath := fmt.Sprintf("state-%s.ssz", fmt.Sprintf("%#x", stateRoot))
	log.Printf("saving ssz-encoded state to to %s", statePath)
	err = os.WriteFile(statePath, sb, 0644)
	if err != nil {
		return err
	}

	fmt.Println("To validate that your client is using this checkpoint, specify the following flag when starting prysm:")
	fmt.Printf("--weak-subjectivity-checkpoint=%s:%d\n\n", blockRootHex, epoch)
	fmt.Println("To sync a new beacon node starting from the checkpoint state, you may specify the following flags (assuming the files are in your current working directory)")
	fmt.Printf("--checkpoint-state=%s --checkpoint-block=%s\n", statePath, blockPath)
	return nil
}
