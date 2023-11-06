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
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// ErrInvalidFetchedData is used to signal that an error occurred which should result in peer downscoring.
var ErrInvalidFetchedData = errors.New("invalid data returned from peer")

var errMaxRequestBlobSidecarsExceeded = errors.Wrap(ErrInvalidFetchedData, "peer exceeded req blob chunk tx limit")
var errBlobChunkedReadFailure = errors.New("failed to read stream of chunk-encoded blobs")
var errBlobUnmarshal = errors.New("Could not unmarshal chunk-encoded blob")
var errUnrequestedRoot = errors.New("Received BlobSidecar in response that was not requested")
var errBlobResponseOutOfBounds = errors.New("received BlobSidecar with slot outside BlobSidecarsByRangeRequest bounds")

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
		currentEpoch := slots.ToEpoch(tor.CurrentSlot())
		if i >= req.Count || i >= params.MaxRequestBlock(currentEpoch) {
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
	currentEpoch := slots.ToEpoch(clock.CurrentSlot())
	for i := 0; i < len(*req); i++ {
		// Exit if peer sends more than max request blocks.
		if uint64(i) >= params.MaxRequestBlock(currentEpoch) {
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

func SendBlobsByRangeRequest(ctx context.Context, tor blockchain.TemporalOracle, p2pApi p2p.SenderEncoder, pid peer.ID, ctxMap ContextByteVersions, req *pb.BlobSidecarsByRangeRequest) ([]blocks.ROBlob, error) {
	topic, err := p2p.TopicFromMessage(p2p.BlobSidecarsByRangeName, slots.ToEpoch(tor.CurrentSlot()))
	if err != nil {
		return nil, err
	}
	log.WithField("topic", topic).Debug("Sending blob by range request")
	stream, err := p2pApi.Send(ctx, req, topic, pid)
	if err != nil {
		return nil, err
	}
	defer closeStream(stream, log)

	max := params.BeaconNetworkConfig().MaxRequestBlobSidecars
	if max > req.Count*fieldparams.MaxBlobsPerBlock {
		max = req.Count * fieldparams.MaxBlobsPerBlock
	}
	return readChunkEncodedBlobs(stream, p2pApi.Encoding(), ctxMap, blobValidatorFromRangeReq(req), max)
}

func SendBlobSidecarByRoot(
	ctx context.Context, tor blockchain.TemporalOracle, p2pApi p2p.P2P, pid peer.ID,
	ctxMap ContextByteVersions, req *p2ptypes.BlobSidecarsByRootReq,
) ([]blocks.ROBlob, error) {
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

	max := params.BeaconNetworkConfig().MaxRequestBlobSidecars
	if max > uint64(len(*req))*fieldparams.MaxBlobsPerBlock {
		max = uint64(len(*req)) * fieldparams.MaxBlobsPerBlock
	}
	return readChunkEncodedBlobs(stream, p2pApi.Encoding(), ctxMap, blobValidatorFromRootReq(req), max)
}

type blobResponseValidation func(blocks.ROBlob) error

func blobValidatorFromRootReq(req *p2ptypes.BlobSidecarsByRootReq) blobResponseValidation {
	roots := make(map[[32]byte]bool)
	for _, sc := range *req {
		roots[bytesutil.ToBytes32(sc.BlockRoot)] = true
	}
	return func(sc blocks.ROBlob) error {
		if requested := roots[sc.BlockRoot()]; !requested {
			return errors.Wrapf(errUnrequestedRoot, "root=%#x", sc.BlockRoot())
		}
		return nil
	}
}

func blobValidatorFromRangeReq(req *pb.BlobSidecarsByRangeRequest) blobResponseValidation {
	end := req.StartSlot + primitives.Slot(req.Count)
	return func(sc blocks.ROBlob) error {
		if sc.Slot() < req.StartSlot || sc.Slot() >= end {
			return errors.Wrapf(errBlobResponseOutOfBounds, "req start,end:%d,%d, resp:%d", req.StartSlot, end, sc.Slot())
		}
		return nil
	}
}

func readChunkEncodedBlobs(stream network.Stream, encoding encoder.NetworkEncoding, ctxMap ContextByteVersions, vf blobResponseValidation, max uint64) ([]blocks.ROBlob, error) {
	sidecars := make([]blocks.ROBlob, 0)
	// Attempt an extra read beyond max to check if the peer is violating the spec by
	// sending more than MAX_REQUEST_BLOB_SIDECARS, or more blobs than requested.
	for i := uint64(0); i < max+1; i++ {
		sc, err := readChunkedBlobSidecar(stream, encoding, ctxMap, vf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if i == max {
			// We have read an extra sidecar beyond what the spec allows. Since this is a spec violation, we return
			// an error that wraps ErrInvalidFetchedData. The part of the state machine that handles rpc peer scoring
			// will downscore the peer if the request ends in an error that wraps that one.
			return nil, errMaxRequestBlobSidecarsExceeded
		}
		sidecars = append(sidecars, sc)
	}

	return sidecars, nil
}

func readChunkedBlobSidecar(stream network.Stream, encoding encoder.NetworkEncoding, ctxMap ContextByteVersions, vf blobResponseValidation) (blocks.ROBlob, error) {
	var b blocks.ROBlob
	pb := &ethpb.BlobSidecar{}
	decode := encoding.DecodeWithMaxLength
	var (
		code uint8
		msg  string
	)
	code, msg, err := ReadStatusCode(stream, encoding)
	if err != nil {
		return b, err
	}
	if code != 0 {
		return b, errors.Wrap(errBlobChunkedReadFailure, msg)
	}
	ctxb, err := readContextFromStream(stream)
	if err != nil {
		return b, errors.Wrap(err, "error reading chunk context bytes from stream")
	}

	v, found := ctxMap[bytesutil.ToBytes4(ctxb)]
	if !found {
		return b, errors.Wrapf(errBlobUnmarshal, fmt.Sprintf("unrecognized fork digest %#x", ctxb))
	}
	// Only deneb is supported at this time, because we lack a fork-spanning interface/union type for blobs.
	if v != version.Deneb {
		return b, fmt.Errorf("unexpected context bytes for deneb BlobSidecar, ctx=%#x, v=%s", ctxb, version.String(v))
	}
	if err := decode(stream, pb); err != nil {
		return b, errors.Wrap(err, "failed to decode the protobuf-encoded BlobSidecar message from RPC chunk stream")
	}

	rob, err := blocks.NewROBlob(pb)
	if err != nil {
		return b, errors.Wrap(err, "unexpected error initializing ROBlob")
	}
	if err := vf(rob); err != nil {
		return b, errors.Wrap(err, "validation failure decoding blob RPC response")
	}

	return rob, nil
}
