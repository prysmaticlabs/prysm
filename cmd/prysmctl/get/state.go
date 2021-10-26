package get

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/api/client/openapi"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os"
)

var getStateFlags = struct {
	BeaconNodeHost string
	Timeout        string
	StateHex       string
	StateSavePath  string
}{}

var getStateCmd = &cli.Command{
	Name:   "state",
	Usage:  "Download a state identified by slot or epoch",
	Action: cliActionGetState,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "beacon-node-host",
			Usage:       "host:port for beacon node connection",
			Destination: &getStateFlags.BeaconNodeHost,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "http-timeout",
			Usage:       "timeout for http requests made to beacon-node-url (uses duration format, ex: 2m31s). default: 2m",
			Destination: &getStateFlags.Timeout,
			Value:       "2m",
		},
		&cli.StringFlag{
			Name:        "state-root",
			Usage:       "instead of epoch, state root (in 0x hex string format) can be used to retrieve from the beacon-node and save locally.",
			Destination: &getStateFlags.StateHex,
		},
		&cli.StringFlag{
			Name:        "state-save-path",
			Usage:       "path to file where state root should be saved if specified. defaults to `state-<state_root>.ssz`",
			Destination: &getStateFlags.StateSavePath,
		},
	},
}

func saveStateByRoot(client *openapi.Client, root, path string) error {
	state, err := client.GetStateByRoot(root)
	if err != nil {
		return err
	}
	stateRoot, err := state.HashTreeRoot()
	if err != nil {
		return err
	}
	log.Printf("retrieved state for checkpoint, w/ root=%s", hexutil.Encode(stateRoot[:]))
	if path == "" {
		path = fmt.Sprintf("state-%s.ssz", root)
	}
	log.Printf("saving to %s...", path)
	blockBytes, err := state.MarshalSSZ()
	if err != nil {
		return err
	}
	return os.WriteFile(path, blockBytes, 0644)
}

func cliActionGetState(c *cli.Context) error {
	return nil
}
