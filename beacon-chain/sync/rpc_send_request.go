package sync

import (
	"context"
	"fmt"
	"io"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/encoder"
	p2ptypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	pb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// ErrInvalidFetchedData is thrown if stream fails to provide requested blocks.
var ErrInvalidFetchedData = errors.New("invalid data returned from peer")

// BeaconBlockProcessor defines a block processing function, which allows to start utilizing
// blocks even before all blocks are ready.
type BeaconBlockProcessor func(block interfaces.ReadOnlySignedBeaconBlock) error

// SendBeaconBlocksByRangeRequest sends BeaconBlocksByRange and returns fetched blocks, if any.
func SendBeaconBlocksByRangeRequest(
	ctx context.Context, tor blockchain.TemporalOracle, p2pProvider p2p.SenderEncoder, pid peer.ID,
	req *pb.BeaconBlocksByRangeRequest, blockProcessor BeaconBlockProcessor,
) ([]interfaces.ReadOnlySignedBeaconBlock, error) {
	topic, err := p2p.TopicFromMessage(p2p.BeaconBlocksByRangeMessageName, slots.ToEpoch(tor.CurrentSlot()))
	if err != nil {
		return nil, err
	}
	stream, err := p2pProvider.Send(ctx, req, topic, pid)
	if err != nil {
		return nil, err
	}
	defer closeStream(stream, log)

	// Augment block processing function, if non-nil block processor is provided.
	blocks := make([]interfaces.ReadOnlySignedBeaconBlock, 0, req.Count)
	process := func(blk interfaces.ReadOnlySignedBeaconBlock) error {
		blocks = append(blocks, blk)
		if blockProcessor != nil {
			return blockProcessor(blk)
		}
		return nil
	}
	var prevSlot primitives.Slot
	for i := uint64(0); ; i++ {
		isFirstChunk := i == 0
		blk, err := ReadChunkedBlock(stream, tor, p2pProvider, isFirstChunk)
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
	ctx context.Context, clock blockchain.TemporalOracle, p2pProvider p2p.P2P, pid peer.ID,
	req *p2ptypes.BeaconBlockByRootsReq, blockProcessor BeaconBlockProcessor,
) ([]interfaces.ReadOnlySignedBeaconBlock, error) {
	topic, err := p2p.TopicFromMessage(p2p.BeaconBlocksByRootsMessageName, slots.ToEpoch(clock.CurrentSlot()))
	if err != nil {
		return nil, err
	}
	stream, err := p2pProvider.Send(ctx, req, topic, pid)
	if err != nil {
		return nil, err
	}
	defer closeStream(stream, log)

	// Augment block processing function, if non-nil block processor is provided.
	blocks := make([]interfaces.ReadOnlySignedBeaconBlock, 0, len(*req))
	process := func(block interfaces.ReadOnlySignedBeaconBlock) error {
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
		blk, err := ReadChunkedBlock(stream, clock, p2pProvider, isFirstChunk)
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

func SendBlobsByRangeRequest(ctx context.Context, ci blockchain.ForkFetcher, p2pApi p2p.SenderEncoder, pid peer.ID, ctxMap ContextByteVersions, req *pb.BlobSidecarsByRangeRequest) ([]*pb.BlobSidecar, error) {
	topic, err := p2p.TopicFromMessage(p2p.BlobSidecarsByRangeName, slots.ToEpoch(ci.CurrentSlot()))
	if err != nil {
		return nil, err
	}
	log.WithField("topic", topic).Debug("Sending blob by range request")
	stream, err := p2pApi.Send(ctx, req, topic, pid)
	if err != nil {
		return nil, err
	}
	defer closeStream(stream, log)

	return readChunkEncodedBlobs(stream, p2pApi.Encoding(), ctxMap, blobValidatorFromRangeReq(req))
}

func SendBlobSidecarByRoot(
	ctx context.Context, tor blockchain.TemporalOracle, p2pApi p2p.P2P, pid peer.ID,
	ctxMap ContextByteVersions, req *p2ptypes.BlobSidecarsByRootReq,
) ([]*pb.BlobSidecar, error) {
	if uint64(len(*req)) > params.BeaconNetworkConfig().MaxRequestBlobSidecars {
		return nil, errors.Wrapf(p2ptypes.ErrMaxBlobReqExceeded, "length=%d", len(*req))
	}

	topic, err := p2p.TopicFromMessage(p2p.BlobSidecarsByRootName, slots.ToEpoch(tor.CurrentSlot()))
	if err != nil {
		return nil, err
	}
	log.WithField("topic", topic).Debug("Sending blob sidecar request")
	stream, err := p2pApi.Send(ctx, req, topic, pid)
	if err != nil {
		return nil, err
	}
	defer closeStream(stream, log)

	return readChunkEncodedBlobs(stream, p2pApi.Encoding(), ctxMap, blobValidatorFromRootReq(req))
}

var ErrBlobChunkedReadFailure = errors.New("failed to read stream of chunk-encoded blobs")
var ErrBlobUnmarshal = errors.New("Could not unmarshal chunk-encoded blob")
var ErrUnrequestedRoot = errors.New("Received BlobSidecar in response that was not requested")
var ErrBlobResponseOutOfBounds = errors.New("received BlobSidecar with slot outside BlobSidecarsByRangeRequest bounds")

type blobResponseValidation func(*pb.BlobSidecar) error

func blobValidatorFromRootReq(req *p2ptypes.BlobSidecarsByRootReq) blobResponseValidation {
	roots := make(map[[32]byte]bool)
	for _, sc := range *req {
		roots[bytesutil.ToBytes32(sc.BlockRoot)] = true
	}
	return func(sc *pb.BlobSidecar) error {
		if requested := roots[bytesutil.ToBytes32(sc.BlockRoot)]; !requested {
			return errors.Wrapf(ErrUnrequestedRoot, "root=%#x", sc.BlockRoot)
		}
		return nil
	}
}

func blobValidatorFromRangeReq(req *pb.BlobSidecarsByRangeRequest) blobResponseValidation {
	end := req.StartSlot + primitives.Slot(req.Count)
	return func(sc *pb.BlobSidecar) error {
		if sc.Slot < req.StartSlot || sc.Slot >= end {
			return errors.Wrapf(ErrBlobResponseOutOfBounds, "req start,end:%d,%d, resp:%d", req.StartSlot, end, sc.Slot)
		}
		return nil
	}
}

func readChunkEncodedBlobs(stream network.Stream, encoding encoder.NetworkEncoding, ctxMap ContextByteVersions, vf blobResponseValidation) ([]*pb.BlobSidecar, error) {
	decode := encoding.DecodeWithMaxLength
	max := int(params.BeaconNetworkConfig().MaxRequestBlobSidecars)
	var (
		code uint8
		msg  string
		err  error
	)
	sidecars := make([]*pb.BlobSidecar, 0)
	for i := 0; i < max; i++ {
		code, msg, err = ReadStatusCode(stream, encoding)
		if err != nil {
			break
		}
		if code != 0 {
			return nil, errors.Wrap(ErrBlobChunkedReadFailure, msg)
		}
		ctxb, err := readContextFromStream(stream)
		if err != nil {
			return nil, errors.Wrap(err, "error reading chunk context bytes from stream")
		}

		v, found := ctxMap[bytesutil.ToBytes4(ctxb)]
		if !found {
			return nil, errors.Wrapf(ErrBlobUnmarshal, fmt.Sprintf("unrecognized fork digest %#x", ctxb))
		}
		if v != version.Deneb {
			return nil, fmt.Errorf("unexpected context bytes for deneb BlobSidecar, ctx=%#x, v=%s", ctxb, version.String(v))
		}
		sc := &pb.BlobSidecar{}
		if err := decode(stream, sc); err != nil {
			return nil, errors.Wrap(err, "failed to decode the protobuf-encoded BlobSidecar message from RPC chunk stream")
		}
		if err := vf(sc); err != nil {
			return nil, errors.Wrap(err, "validation failure decoding blob RPC response")
		}
		sidecars = append(sidecars, sc)
	}
	if !errors.Is(err, io.EOF) {
		return nil, err
	}
	return sidecars, nil
}
