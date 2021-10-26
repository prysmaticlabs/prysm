package get

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/api/client/openapi"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os"
	"time"
)

var getBlockFlags = struct {
	BeaconNodeHost string
	Timeout        string
	BlockHex       string
	BlockSavePath  string
}{}

var getBlockCmd = &cli.Command{
	Name:   "block",
	Usage:  "Retrieve ssz-encoded block data from a beacon node.",
	Action: cliActionGetBlock,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "beacon-node-host",
			Usage:       "host:port for beacon node connection",
			Destination: &getBlockFlags.BeaconNodeHost,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "http-timeout",
			Usage:       "timeout for http requests made to beacon-node-url (uses duration format, ex: 2m31s). default: 2m",
			Destination: &getBlockFlags.Timeout,
			Value:       "2m",
		},
		&cli.StringFlag{
			Name:        "block-root",
			Usage:       "block root (in 0x hex string format) used to retrieve the SignedBeaconBlock for checkpoint state.",
			Destination: &getBlockFlags.BlockHex,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "block-save-path",
			Usage:       "path to file where block root should be saved. defaults to `block-<block_root>.ssz`",
			Destination: &getBlockFlags.BlockSavePath,
		},
	},
}

func cliActionGetBlock(c *cli.Context) error {
	f := getBlockFlags
	if f.BlockHex != "" {
	}
	opts := make([]openapi.ClientOpt, 0)
	log.Printf("beacon-node-url=%s", f.BeaconNodeHost)
	timeout, err := time.ParseDuration(f.Timeout)
	if err != nil {
		return err
	}
	opts = append(opts, openapi.WithTimeout(timeout))
	client, err := openapi.NewClient(f.BeaconNodeHost, opts...)
	if err != nil {
		return err
	}

	return saveBlock(client, f.BlockHex, f.BlockSavePath)
}

func saveBlock(client *openapi.Client, root, path string) error {
	block, err := client.GetBlockByRoot(root)
	if err != nil {
		return err
	}
	blockRoot, err := block.Block.HashTreeRoot()
	if err != nil {
		return err
	}
	log.Printf("retrieved block for checkpoint, w/ block (header) root=%s", hexutil.Encode(blockRoot[:]))
	if path == "" {
		path = fmt.Sprintf("block-%s.ssz", root)
	}
	log.Printf("saving to %s...", path)
	blockBytes, err := block.MarshalSSZ()
	if err != nil {
		return err
	}
	return os.WriteFile(path, blockBytes, 0644)
}
