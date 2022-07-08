package p2p

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
)

var requestBlocksFlags = struct {
	Peer    string
	Timeout time.Duration
}{}

var requestBlocksCmd = &cli.Command{
	Name:   "request-blocks",
	Usage:  "Request a range of blocks from a beacon node via a p2p connection",
	Action: cliActionRequestBlocks,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "peer",
			Usage:       "peer multiaddr to connect to for p2p requests",
			Destination: &requestBlocksFlags.Peer,
			Value:       "",
		},
	},
}

func cliActionRequestBlocks(_ *cli.Context) error {
	fmt.Println("hello world")

	// Set up p2p credentials.

	// Initialize p2p host.

	// Connect to peers.

	// Submit requests.

	// Process responses and measure latency.

	return nil
}
