package p2p

import (
	"context"
	"strings"
	"time"

	corenet "github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var requestBlocksFlags = struct {
	Peer        string
	ClientPort  uint
	APIEndpoint string
	StartSlot   uint64
	Count       uint64
	Step        uint64
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
		&cli.UintFlag{
			Name:        "client-port",
			Usage:       "port to use for the client as a libp2p host",
			Destination: &requestBlocksFlags.ClientPort,
			Value:       13001,
		},
		&cli.StringFlag{
			Name:        "prysm-api-endpoint",
			Usage:       "gRPC API endpoint for a Prysm node",
			Destination: &requestBlocksFlags.APIEndpoint,
			Value:       "localhost:4000",
		},
		&cli.Uint64Flag{
			Name:        "start-slot",
			Usage:       "start slot for blocks by range request. If unset, will use start_slot(current_epoch-1)",
			Destination: &requestBlocksFlags.StartSlot,
			Value:       0,
		},
		&cli.Uint64Flag{
			Name:        "count",
			Usage:       "number of blocks to request, (default 32)",
			Destination: &requestBlocksFlags.Count,
			Value:       32,
		},
		&cli.Uint64Flag{
			Name:        "step",
			Usage:       "number of steps of blocks in the range request, (default 1)",
			Destination: &requestBlocksFlags.Step,
			Value:       1,
		},
	},
}

func cliActionRequestBlocks(_ *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	c, err := newClient(requestBlocksFlags.APIEndpoint, requestBlocksFlags.ClientPort)
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

	startSlot := types.Slot(requestBlocksFlags.StartSlot)
	if startSlot == 0 {
		headResp, err := c.beaconClient.GetChainHead(ctx, nil)
		if err != nil {
			return err
		}
		startSlot, err = slots.EpochStart(headResp.HeadEpoch.Sub(1))
		if err != nil {
			return err
		}
	}

	// Submit requests.
	for _, pr := range c.host.Peerstore().Peers() {
		req := &pb.BeaconBlocksByRangeRequest{
			StartSlot: startSlot,
			Count:     requestBlocksFlags.Count,
			Step:      requestBlocksFlags.Step,
		}
		log.WithFields(logrus.Fields{
			"startSlot": startSlot,
			"count":     requestBlocksFlags.Count,
			"step":      requestBlocksFlags.Step,
			"peer":      pr.String(),
		}).Info("Sending blocks by range p2p request to peer")
		start := time.Now()
		blocks, err := sync.SendBeaconBlocksByRangeRequest(
			ctx,
			mockChain,
			c,
			pr,
			req,
			nil, /* no extra block processing */
		)
		if err != nil {
			if strings.Contains(err.Error(), "dial to self attempted") {
				log.Info("Ignoring sending a request to self")
				continue
			}
			return err
		}
		log.WithFields(logrus.Fields{
			"numBlocks":                           len(blocks),
			"peer":                                pr.String(),
			"timeFromSendingToProcessingResponse": time.Since(start),
		}).Info("Received blocks from peer")
		for _, blk := range blocks {
			blk.Version()
		}
	}
	return nil
}

func closeStream(stream corenet.Stream) {
	if err := stream.Close(); err != nil {
		log.Println(err)
	}
}
