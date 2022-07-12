package p2p

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
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
		blocks, err := sendBeaconBlocksByRangeRequest(ctx, mockChain, c, pr, req)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("Got blocks", blocks)
	}

	time.Sleep(time.Minute * 10)
	// Process responses and measure latency.
	return nil
}

func sendBeaconBlocksByRangeRequest(
	ctx context.Context, chain blockchain.ChainInfoFetcher, p2pProvider *client, pid peer.ID,
	req *pb.BeaconBlocksByRangeRequest,
) ([]interfaces.SignedBeaconBlock, error) {
	sinceGenesis := slots.SinceGenesis(chain.GenesisTime())
	topic, err := p2p.TopicFromMessage(p2p.BeaconBlocksByRangeMessageName, slots.ToEpoch(sinceGenesis))
	if err != nil {
		return nil, errors.Wrap(err, "topic cannot find")
	}
	stream, err := p2pProvider.SendP2PRequest(ctx, req, topic, pid)
	if err != nil {
		return nil, errors.Wrap(err, "cannot send")
	}
	defer closeStream(stream)

	// Augment block processing function, if non-nil block processor is provided.
	blocks := make([]interfaces.SignedBeaconBlock, 0, req.Count)
	process := func(blk interfaces.SignedBeaconBlock) error {
		blocks = append(blocks, blk)
		return nil
	}
	var prevSlot types.Slot
	for i := uint64(0); ; i++ {
		isFirstChunk := i == 0
		blk, err := ReadChunkedBlock(stream, chain, p2pProvider, isFirstChunk)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		// The response MUST contain no more than `count` blocks, and no more than
		// MAX_REQUEST_BLOCKS blocks.
		if i >= req.Count || i >= params.BeaconNetworkConfig().MaxRequestBlocks {
			return nil, errors.New("invalid data")
		}
		// Returned blocks MUST be in the slot range [start_slot, start_slot + count * step).
		if blk.Block().Slot() < req.StartSlot || blk.Block().Slot() >= req.StartSlot.Add(req.Count*req.Step) {
			return nil, errors.New("invalid data")
		}
		// Returned blocks, where they exist, MUST be sent in a consecutive order.
		// Consecutive blocks MUST have values in `step` increments (slots may be skipped in between).
		isSlotOutOfOrder := false
		if prevSlot >= blk.Block().Slot() {
			isSlotOutOfOrder = true
		} else if req.Step != 0 && blk.Block().Slot().SubSlot(prevSlot).Mod(req.Step) != 0 {
			isSlotOutOfOrder = true
		}
		if !isFirstChunk && isSlotOutOfOrder {
			return nil, errors.New("invalid data")
		}
		prevSlot = blk.Block().Slot()
		if err := process(blk); err != nil {
			return nil, err
		}
	}
	return blocks, nil
}
