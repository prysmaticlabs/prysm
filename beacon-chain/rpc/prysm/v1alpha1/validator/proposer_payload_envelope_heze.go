package validator

import (
	"context"
	"fmt"
	"time"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/cache"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/heze"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/peerdas"
	"github.com/OffchainLabs/prysm/v7/config/params"
	consensusblocks "github.com/OffchainLabs/prysm/v7/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	"github.com/OffchainLabs/prysm/v7/io/logs"
	"github.com/OffchainLabs/prysm/v7/monitoring/tracing/trace"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// storeExecutionPayloadEnvelopeHeze creates and caches the Heze execution payload envelope
// (EIP-8411) after the block is fully built, returning the envelope for the caller to bundle.
// Alongside the data column sidecars, the 64 PayloadChunk gossip messages are precomputed so
// the publish hot path only restamps their routing hints.
func (vs *Server) storeExecutionPayloadEnvelopeHeze(
	sBlk interfaces.SignedBeaconBlock,
	local *consensusblocks.GetPayloadResponse,
) (*ethpb.ExecutionPayloadEnvelopeHeze, error) {
	blockRoot, err := sBlk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block hash tree root")
	}

	payload := extractExecutionPayloadGloas(local)

	parentRoot := sBlk.Block().ParentRoot()
	envelope := &ethpb.ExecutionPayloadEnvelopeHeze{
		Payload:               payload,
		ExecutionRequests:     local.ExecutionRequestsGloas,
		BuilderIndex:          params.BeaconConfig().BuilderIndexSelfBuild,
		ParentBeaconBlockRoot: parentRoot[:],
	}

	// Precompute the chunk messages (during ProposeBeaconBlock slack) so publish stays fast,
	// and cross-check the commitment against the bid the block actually carries.
	chunks, chunksRoot, err := heze.BuildPayloadChunkMessages(envelope, sBlk.Block().Slot(), blockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not build payload chunk messages")
	}
	sbid, err := sBlk.Block().Body().SignedExecutionPayloadBid()
	if err != nil {
		return nil, errors.Wrap(err, "could not get signed execution payload bid")
	}
	bid, err := sbid.Bid()
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution payload bid")
	}
	if bid.PayloadChunksRoot() != chunksRoot {
		return nil, errors.Errorf("envelope chunk commitment %#x does not match bid payload_chunks_root %#x", chunksRoot, bid.PayloadChunksRoot())
	}

	// Precompute sidecars here (during ProposeBeaconBlock slack) so publish stays fast.
	var roSidecars []consensusblocks.RODataColumn
	if bundle := local.BlobsBundler; bundle != nil && len(bundle.GetBlobs()) > 0 {
		cellsPerBlob, proofsPerBlob, err := peerdas.ComputeCellsAndProofsFromFlat(bundle.GetBlobs(), bundle.GetProofs())
		if err != nil {
			return nil, errors.Wrap(err, "compute cells and proofs from blobs bundle")
		}
		roSidecars, err = peerdas.DataColumnSidecarsGloas(cellsPerBlob, proofsPerBlob, sBlk.Block().Slot(), blockRoot)
		if err != nil {
			return nil, errors.Wrap(err, "build gloas data column sidecars")
		}
	}

	var partialColumns []consensusblocks.PartialDataColumn
	if len(roSidecars) > 0 && vs.ExecutionEngineCaller.PartialColumnsSupported() {
		commitments, err := sBlk.Block().Body().BlobKzgCommitments()
		if err != nil {
			log.WithError(err).Error("Failed to get blob kzg commitments for partial columns")
		} else if partialColumns, err = partialColumnsFromSidecars(roSidecars, commitments); err != nil {
			log.WithError(err).Error("Failed to build partial columns")
		}
	}

	vs.ExecutionPayloadEnvelopeCache.Set(&cache.ExecutionPayloadContents{
		EnvelopeHeze:    envelope,
		BeaconBlockRoot: blockRoot,
		Chunks:          chunks,
		DataColumns:     roSidecars,
		PartialColumns:  partialColumns,
	})
	return envelope, nil
}

// GetExecutionPayloadEnvelopeV3 returns the cached Heze execution payload envelope for the
// requested slot so the proposer can sign and publish it. The Heze container carries no beacon
// block root (EIP-8411), so the requested root is checked against the cached binding.
func (vs *Server) GetExecutionPayloadEnvelopeV3(
	ctx context.Context,
	req *ethpb.ExecutionPayloadEnvelopeRequest,
) (*ethpb.ExecutionPayloadEnvelopeHezeResponse, error) {
	_, span := trace.StartSpan(ctx, "ProposerServer.GetExecutionPayloadEnvelopeV3")
	defer span.End()

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}
	span.SetAttributes(trace.StringAttribute("slot", fmt.Sprintf("%d", req.Slot)))

	if slots.ToEpoch(req.Slot) < params.BeaconConfig().HezeForkEpoch {
		return nil, status.Errorf(codes.InvalidArgument,
			"Heze execution payload envelopes are not supported before Heze fork (slot %d)", req.Slot)
	}

	contents, ok := vs.ExecutionPayloadEnvelopeCache.Contents()
	if !ok || contents.EnvelopeHeze == nil || contents.EnvelopeHeze.Payload.SlotNumber != req.Slot {
		return nil, status.Errorf(codes.NotFound,
			"execution payload envelope not found for slot %d", req.Slot)
	}
	if len(req.BeaconBlockRoot) > 0 && bytesutil.ToBytes32(req.BeaconBlockRoot) != contents.BeaconBlockRoot {
		return nil, status.Errorf(codes.NotFound,
			"cached execution payload envelope does not belong to the requested block for slot %d", req.Slot)
	}

	return &ethpb.ExecutionPayloadEnvelopeHezeResponse{
		Envelope: contents.EnvelopeHeze,
	}, nil
}

// PublishExecutionPayloadEnvelopeV3 validates and reveals a signed Heze execution payload
// envelope: data column sidecars are verified and broadcast as in V2, then the envelope is
// broadcast as its 64 PayloadChunk messages (EIP-8411) instead of a single envelope message,
// and finally imported in the background.
func (vs *Server) PublishExecutionPayloadEnvelopeV3(
	ctx context.Context,
	req *ethpb.GenericSignedExecutionPayloadEnvelopeHeze,
) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.PublishExecutionPayloadEnvelopeV3")
	defer span.End()
	start := time.Now()

	signed, blobs, kzgProofs, cached, err := vs.resolveEnvelopeToPublishHeze(req)
	if err != nil {
		return nil, err
	}

	envSlot := primitives.Slot(signed.Message.Payload.SlotNumber)
	if slots.ToEpoch(envSlot) < params.BeaconConfig().HezeForkEpoch {
		return nil, status.Errorf(codes.InvalidArgument,
			"Heze execution payload envelopes are not supported before Heze fork (slot %d)", envSlot)
	}

	// The Heze container carries no beacon block root; take the binding from the request,
	// falling back to the cached binding recorded at block production.
	var beaconBlockRoot [32]byte
	switch {
	case len(req.BeaconBlockRoot) == 32:
		beaconBlockRoot = bytesutil.ToBytes32(req.BeaconBlockRoot)
	case cached != nil:
		beaconBlockRoot = cached.BeaconBlockRoot
	default:
		return nil, status.Error(codes.InvalidArgument, "beacon block root is required when the beacon node has no cached envelope")
	}

	span.SetAttributes(
		trace.StringAttribute("slot", fmt.Sprintf("%d", envSlot)),
		trace.StringAttribute("builderIndex", fmt.Sprintf("%d", signed.Message.BuilderIndex)),
		trace.StringAttribute("beaconBlockRoot", fmt.Sprintf("%#x", beaconBlockRoot[:8])),
	)

	log := log.WithFields(logrus.Fields{
		"slot":            envSlot,
		"builderIndex":    logs.BuilderIndexLabel(signed.Message.BuilderIndex),
		"beaconBlockRoot": fmt.Sprintf("%#x", beaconBlockRoot[:8]),
	})
	log.Debug("Publishing execution payload envelope")

	// KZG verification stays synchronous, never gossip unverified sidecars. A cache match means
	// the columns were built locally from the engine's own bundle, so verification is skipped.
	var sidecars []consensusblocks.RODataColumn
	var partialColumns []consensusblocks.PartialDataColumn
	if cached != nil {
		sidecars = cached.DataColumns
		partialColumns = cached.PartialColumns
	} else if len(blobs) > 0 {
		sidecars, partialColumns, err = vs.sidecarsFromContents(blobs, kzgProofs, envSlot, beaconBlockRoot)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid execution payload envelope contents: %v", err)
		}
	}

	roSigned, err := consensusblocks.WrappedROSignedExecutionPayloadEnvelopeHeze(signed, beaconBlockRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not wrap signed envelope: %v", err)
	}

	// Locally built or verified above, safe to upgrade.
	verifiedSidecars := make([]consensusblocks.VerifiedRODataColumn, 0, len(sidecars))
	for _, sidecar := range sidecars {
		verifiedSidecars = append(verifiedSidecars, consensusblocks.NewVerifiedRODataColumn(sidecar))
	}
	if len(verifiedSidecars) > 0 {
		log.WithFields(logrus.Fields{
			"columns":  len(sidecars),
			"partials": len(partialColumns),
		}).Debug("Broadcasting Gloas data column sidecars")
		if err := vs.P2P.BroadcastDataColumnSidecars(ctx, verifiedSidecars, partialColumns); err != nil {
			log.WithError(err).Error("Failed to broadcast Gloas data column sidecars")
		}
	}

	// EIP-8411: the envelope travels as chunk messages. Cached chunk sets only need their
	// unauthenticated routing hints stamped; otherwise chunk the submitted envelope.
	var chunks []*ethpb.PayloadChunk
	if cached != nil && len(cached.Chunks) > 0 {
		chunks = cached.Chunks
		heze.StampPayloadChunkHints(chunks, envSlot, beaconBlockRoot)
	} else {
		chunks, _, err = heze.BuildPayloadChunkMessages(signed.Message, envSlot, beaconBlockRoot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not build payload chunk messages: %v", err)
		}
	}
	for _, chunk := range chunks {
		if err := vs.P2P.Broadcast(ctx, chunk); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to broadcast payload chunk %d: %v", chunk.Index, err)
		}
	}

	// Import in the background so the reveal is not delayed past the PTC deadline.
	go vs.importPublishedEnvelope(log, verifiedSidecars, roSigned)

	log.WithField("duration", time.Since(start)).Info("Published execution payload envelope")

	return &emptypb.Empty{}, nil
}

// resolveEnvelopeToPublishHeze extracts the signed Heze envelope plus any caller-supplied blobs.
// It mirrors resolveEnvelopeToPublish: the stateful signed_envelope arm must match the cached
// envelope so its precomputed data columns and chunks apply. The cached contents are returned
// when the submitted envelope is the cached one (matched by envelope root), nil otherwise.
func (vs *Server) resolveEnvelopeToPublishHeze(req *ethpb.GenericSignedExecutionPayloadEnvelopeHeze) (*ethpb.SignedExecutionPayloadEnvelopeHeze, [][]byte, [][]byte, *cache.ExecutionPayloadContents, error) {
	cachedMatch := func(envelope *ethpb.ExecutionPayloadEnvelopeHeze) (*cache.ExecutionPayloadContents, error) {
		cached, ok := vs.ExecutionPayloadEnvelopeCache.Contents()
		if !ok || cached.EnvelopeHeze == nil {
			return nil, nil
		}
		cachedRoot, err := cached.EnvelopeHeze.HashTreeRoot()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not hash cached envelope: %v", err)
		}
		submittedRoot, err := envelope.HashTreeRoot()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not hash submitted envelope: %v", err)
		}
		if cachedRoot != submittedRoot {
			return nil, nil
		}
		return cached, nil
	}

	switch {
	case req.GetContents() != nil:
		c := req.GetContents()
		if c.SignedExecutionPayloadEnvelope == nil || c.SignedExecutionPayloadEnvelope.Message == nil ||
			c.SignedExecutionPayloadEnvelope.Message.Payload == nil {
			return nil, nil, nil, nil, status.Error(codes.InvalidArgument, "signed envelope or payload cannot be nil")
		}
		cached, err := cachedMatch(c.SignedExecutionPayloadEnvelope.Message)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		return c.SignedExecutionPayloadEnvelope, c.Blobs, c.KzgProofs, cached, nil
	case req.GetSignedEnvelope() != nil:
		signed := req.GetSignedEnvelope()
		if signed.Message == nil || signed.Message.Payload == nil {
			return nil, nil, nil, nil, status.Error(codes.InvalidArgument, "signed envelope or payload cannot be nil")
		}
		cached, err := cachedMatch(signed.Message)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if cached == nil {
			return nil, nil, nil, nil, status.Error(codes.InvalidArgument, "cached execution payload envelope does not match submitted envelope")
		}
		return signed, nil, nil, cached, nil
	default:
		return nil, nil, nil, nil, status.Error(codes.InvalidArgument, "generic signed execution payload envelope must set contents or signed_envelope")
	}
}
