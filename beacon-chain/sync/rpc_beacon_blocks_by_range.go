package sync

import (
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/pkg/errors"
	p2ptypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v4/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/monitoring/tracing"
	pb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// beaconBlocksByRangeRPCHandler looks up the request blocks from the database from a given start block.
func (s *Service) beaconBlocksByRangeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.BeaconBlocksByRangeHandler")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()
	SetRPCStreamDeadlines(stream)

	m, ok := msg.(*pb.BeaconBlocksByRangeRequest)
	if !ok {
		return errors.New("message is not type *pb.BeaconBlockByRangeRequest")
	}
	start, end, size, err := validateRangeRequest(m, s.cfg.chain.CurrentSlot())
	if err != nil {
		s.writeErrorResponseToStream(responseCodeInvalidRequest, err.Error(), stream)
		s.cfg.p2p.Peers().Scorers().BadResponsesScorer().Increment(stream.Conn().RemotePeer())
		tracing.AnnotateError(span, err)
		return err
	}

	blockLimiter, err := s.rateLimiter.topicCollector(string(stream.Protocol()))
	if err != nil {
		return err
	}
	remainingBucketCapacity := blockLimiter.Remaining(stream.Conn().RemotePeer().String())
	span.AddAttributes(
		trace.Int64Attribute("start", int64(start)), // lint:ignore uintcast -- This conversion is OK for tracing.
		trace.Int64Attribute("end", int64(end)),     // lint:ignore uintcast -- This conversion is OK for tracing.
		trace.Int64Attribute("count", int64(m.Count)),
		trace.StringAttribute("peer", stream.Conn().RemotePeer().Pretty()),
		trace.Int64Attribute("remaining_capacity", remainingBucketCapacity),
	)

	// Ticker to stagger out large requests.
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	batcher := &blockRangeBatcher{
		start:       start,
		end:         end,
		size:        size,
		db:          s.cfg.beaconDB,
		limiter:     s.rateLimiter,
		isCanonical: s.cfg.chain.IsCanonical,
		ticker:      ticker,
	}

	// prevRoot is used to ensure that returned chains are strictly linear for singular steps
	// by comparing the previous root of the block in the list with the current block's parent.
	var batch blockBatch
	for batch, ok = batcher.Next(ctx, stream); ok; batch, ok = batcher.Next(ctx, stream) {
		batchStart := time.Now()
		rpcBlocksByRangeResponseLatency.Observe(float64(time.Since(batchStart).Milliseconds()))
		if err := s.writeBlockBatchToStream(ctx, batch, stream); err != nil {
			s.writeErrorResponseToStream(responseCodeServerError, p2ptypes.ErrGeneric.Error(), stream)
			return err
		}
	}
	if err := batch.Err(); err != nil {
		log.WithError(err).Debug("error in BlocksByRange batch")
		s.writeErrorResponseToStream(responseCodeServerError, p2ptypes.ErrGeneric.Error(), stream)
		tracing.AnnotateError(span, err)
		return err
	}
	closeStream(stream, log)
	return nil
}

func validateRangeRequest(r *pb.BeaconBlocksByRangeRequest, current primitives.Slot) (primitives.Slot, primitives.Slot, uint64, error) {
	start := r.StartSlot
	count := r.Count
	maxRequest := params.BeaconNetworkConfig().MaxRequestBlocks
	// Ensure all request params are within appropriate bounds
	if count == 0 || count > maxRequest {
		return 0, 0, 0, p2ptypes.ErrInvalidRequest
	}
	// Allow some wiggle room, up to double the MaxRequestBlocks past the current slot,
	// to give nodes syncing close to the head of the chain some margin for error.
	maxStart, err := current.SafeAdd(maxRequest * 2)
	if err != nil {
		return 0, 0, 0, p2ptypes.ErrInvalidRequest
	}
	if start > maxStart {
		return 0, 0, 0, p2ptypes.ErrInvalidRequest
	}
	end, err := start.SafeAdd((count - 1))
	if err != nil {
		return 0, 0, 0, p2ptypes.ErrInvalidRequest
	}

	limit := uint64(flags.Get().BlockBatchLimit)
	if limit > maxRequest {
		limit = maxRequest
	}
	batchSize := count
	if batchSize > limit {
		batchSize = limit
	}

	return start, end, batchSize, nil
}

func (s *Service) writeBlockBatchToStream(ctx context.Context, batch blockBatch, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.WriteBlockRangeToStream")
	defer span.End()

	blks := batch.Sequence()
	// If the blocks are blinded, we reconstruct the full block via the execution client.
	blindedExists := false
	blindedIndex := 0
	for i, b := range blks {
		// Since the blocks are sorted in ascending order, we assume that the following
		// blocks from the first blinded block are also ascending.
		if b.IsBlinded() {
			blindedExists = true
			blindedIndex = i
			break
		}
	}

	for _, b := range blks {
		if err := blocks.BeaconBlockIsNil(b); err != nil {
			continue
		}
		if b.IsBlinded() {
			continue
		}
		if chunkErr := s.chunkBlockWriter(stream, b); chunkErr != nil {
			log.WithError(chunkErr).Debug("Could not send a chunked response")
			return chunkErr
		}
	}

	var err error
	var reconstructedBlock []interfaces.SignedBeaconBlock
	if blindedExists {
		blinded := blks[blindedIndex:]
		unwrapped := make([]interfaces.ReadOnlySignedBeaconBlock, len(blinded))
		for i := range blinded {
			unwrapped[i] = blks[i].ReadOnlySignedBeaconBlock
		}
		reconstructedBlock, err = s.cfg.executionPayloadReconstructor.ReconstructFullBellatrixBlockBatch(ctx, unwrapped)
		if err != nil {
			log.WithError(err).Error("Could not reconstruct full bellatrix block batch from blinded bodies")
			return err
		}
	}
	for _, b := range reconstructedBlock {
		if err := blocks.BeaconBlockIsNil(b); err != nil {
			continue
		}
		if b.IsBlinded() {
			continue
		}
		if chunkErr := s.chunkBlockWriter(stream, b); chunkErr != nil {
			log.WithError(chunkErr).Debug("Could not send a chunked response")
			return chunkErr
		}
	}

	return nil
}
