package sync

import (
	"context"
	"io"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p"
	p2ptypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	pb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

// ErrInvalidFetchedData is thrown if stream fails to provide requested blocks.
var ErrInvalidFetchedData = errors.New("invalid data returned from peer")

// BeaconBlockProcessor defines a block processing function, which allows to start utilizing
// blocks even before all blocks are ready.
type BeaconBlockProcessor func(block interfaces.SignedBeaconBlock) error

type BeaconBlockAndSidecarProcessor func(blockAndSidecar *pb.SignedBeaconBlockAndBlobsSidecar) error

// SendBeaconBlocksByRangeRequest sends BeaconBlocksByRange and returns fetched blocks, if any.
func SendBeaconBlocksByRangeRequest(
	ctx context.Context, chain blockchain.ForkFetcher, p2pProvider p2p.SenderEncoder, pid peer.ID,
	req *pb.BeaconBlocksByRangeRequest, blockProcessor BeaconBlockProcessor,
) ([]interfaces.SignedBeaconBlock, error) {
	topic, err := p2p.TopicFromMessage(p2p.BeaconBlocksByRangeMessageName, slots.ToEpoch(chain.CurrentSlot()))
	if err != nil {
		return nil, err
	}
	stream, err := p2pProvider.Send(ctx, req, topic, pid)
	if err != nil {
		return nil, err
	}
	defer closeStream(stream, log)

	// Augment block processing function, if non-nil block processor is provided.
	blocks := make([]interfaces.SignedBeaconBlock, 0, req.Count)
	process := func(blk interfaces.SignedBeaconBlock) error {
		blocks = append(blocks, blk)
		if blockProcessor != nil {
			return blockProcessor(blk)
		}
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
			return nil, ErrInvalidFetchedData
		}
		// Returned blocks MUST be in the slot range [start_slot, start_slot + count * step).
		if blk.Block().Slot() < req.StartSlot || blk.Block().Slot() >= req.StartSlot.Add(req.Count*req.Step) {
			return nil, ErrInvalidFetchedData
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
			return nil, ErrInvalidFetchedData
		}
		prevSlot = blk.Block().Slot()
		if err := process(blk); err != nil {
			return nil, err
		}
	}
	return blocks, nil
}

// SendBeaconBlocksByRootRequest sends BeaconBlocksByRoot and returns fetched blocks, if any.
func SendBeaconBlocksByRootRequest(
	ctx context.Context, chain blockchain.ChainInfoFetcher, p2pProvider p2p.P2P, pid peer.ID,
	req *p2ptypes.BeaconBlockByRootsReq, blockProcessor BeaconBlockProcessor,
) ([]interfaces.SignedBeaconBlock, error) {
	topic, err := p2p.TopicFromMessage(p2p.BeaconBlocksByRootsMessageName, slots.ToEpoch(chain.CurrentSlot()))
	if err != nil {
		return nil, err
	}
	stream, err := p2pProvider.Send(ctx, req, topic, pid)
	if err != nil {
		return nil, err
	}
	defer closeStream(stream, log)

	// Augment block processing function, if non-nil block processor is provided.
	blocks := make([]interfaces.SignedBeaconBlock, 0, len(*req))
	process := func(block interfaces.SignedBeaconBlock) error {
		blocks = append(blocks, block)
		if blockProcessor != nil {
			return blockProcessor(block)
		}
		return nil
	}
	for i := 0; i < len(*req); i++ {
		// Exit if peer sends more than max request blocks.
		if uint64(i) >= params.BeaconNetworkConfig().MaxRequestBlocks {
			break
		}
		isFirstChunk := i == 0
		blk, err := ReadChunkedBlock(stream, chain, p2pProvider, isFirstChunk)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		if err := process(blk); err != nil {
			return nil, err
		}
	}
	return blocks, nil
}

func SendBlocksAndSidecarsByRootRequest(
	ctx context.Context, chain blockchain.ChainInfoFetcher, p2pProvider p2p.P2P, pid peer.ID,
	req *p2ptypes.BeaconBlockByRootsReq, processor BeaconBlockAndSidecarProcessor,
) ([]*pb.SignedBeaconBlockAndBlobsSidecar, error) {
	topic, err := p2p.TopicFromMessage(p2p.BeaconBlockAndBlobsSidecarByRootName, slots.ToEpoch(chain.CurrentSlot()))
	if err != nil {
		return nil, err
	}
	stream, err := p2pProvider.Send(ctx, req, topic, pid)
	if err != nil {
		return nil, err
	}
	defer closeStream(stream, log)

	blkAndSidecars := make([]*pb.SignedBeaconBlockAndBlobsSidecar, 0, len(*req))
	process := func(b *pb.SignedBeaconBlockAndBlobsSidecar) error {
		blkAndSidecars = append(blkAndSidecars, b)
		if processor != nil {
			return processor(b)
		}
		return nil
	}
	for i := 0; i < len(*req); i++ {
		// Exit if peer sends more than max request blocks.
		if uint64(i) >= params.BeaconNetworkConfig().MaxRequestBlocks {
			break
		}
		isFirstChunk := i == 0
		blkAndSidecar, err := ReadChunkedBlockAndBlobsSidecar(stream, chain, p2pProvider, isFirstChunk)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		if err := process(blkAndSidecar); err != nil {
			return nil, err
		}
	}
	return blkAndSidecars, nil
}

func SendBlobsSidecarsByRangeRequest(
	ctx context.Context, chain blockchain.ForkFetcher, p2pProvider p2p.SenderEncoder, pid peer.ID,
	req *pb.BlobsSidecarsByRangeRequest, sidecarProcessor BlobsSidecarProcessor) ([]*pb.BlobsSidecar, error) {
	topic, err := p2p.TopicFromMessage(p2p.BlobsSidecarsByRangeMessageName, slots.ToEpoch(chain.CurrentSlot()))
	if err != nil {
		return nil, err
	}
	stream, err := p2pProvider.Send(ctx, req, topic, pid)
	if err != nil {
		return nil, err
	}
	defer closeStream(stream, log)

	var sidecars []*pb.BlobsSidecar
	process := func(sidecar *pb.BlobsSidecar) error {
		sidecars = append(sidecars, sidecar)
		if sidecarProcessor != nil {
			return sidecarProcessor(sidecar)
		}
		return nil
	}

	var prevSlot types.Slot
	for i := uint64(0); ; i++ {
		isFirstChunk := len(sidecars) == 0
		sidecar, err := ReadChunkedBlobsSidecar(stream, chain, p2pProvider, isFirstChunk)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		if i >= req.Count || i >= params.BeaconNetworkConfig().MaxRequestBlobsSidecars {
			return nil, ErrInvalidFetchedData
		}
		if sidecar.BeaconBlockSlot < req.StartSlot || sidecar.BeaconBlockSlot >= req.StartSlot.Add(req.Count) {
			return nil, ErrInvalidFetchedData
		}
		// assert slots aren't out of order and always increasing
		if prevSlot >= sidecar.BeaconBlockSlot {
			return nil, ErrInvalidFetchedData
		}
		prevSlot = sidecar.BeaconBlockSlot

		if err := process(sidecar); err != nil {
			return nil, err
		}
	}
	return sidecars, nil
}
