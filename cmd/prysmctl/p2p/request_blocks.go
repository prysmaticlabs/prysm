package p2p

import (
	"context"
	"fmt"
	"strings"
	"time"

	corenet "github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/urfave/cli/v2"
)

var requestBlocksFlags = struct {
	Peer        string
	APIEndpoint string
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
		&cli.StringFlag{
			Name:        "prysm-api-endpoint",
			Usage:       "gRPC API endpoint for a Prysm node",
			Destination: &requestBlocksFlags.APIEndpoint,
			Value:       "localhost:4000",
		},
	},
}

func cliActionRequestBlocks(_ *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	c, err := newClient(requestBlocksFlags.APIEndpoint)
	if err != nil {
		return err
	}
	defer c.Close()
	mockChain, err := c.initializeMockChainService(ctx)
	if err != nil {
		return err
	}
	c.registerHandshakeHandlers()
	if err := c.connectToPeers(ctx, requestBlocksFlags.Peer); err != nil {
		return err
	}

	// Submit requests.
	for _, pr := range c.host.Peerstore().Peers() {
		req := &pb.BeaconBlocksByRangeRequest{
			StartSlot: 0,
			Count:     10,
			Step:      1,
		}
		blocks, err := sync.SendBeaconBlocksByRangeRequest(
			ctx,
			mockChain,
			c,
			pr,
			req,
			nil /* no extra block processing */,
		)
		if err != nil {
			if strings.Contains(err.Error(), "dial to self attempted") {
				continue
			}
			return err
		}
		fmt.Println("Got blocks", blocks)
	}

	time.Sleep(time.Minute * 10)
	// Process responses and measure latency.
	return nil
}

func closeStream(stream corenet.Stream) {
	if err := stream.Close(); err != nil {
		log.Println(err)
	}
}
