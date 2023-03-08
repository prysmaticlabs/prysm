package sync

import (
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	p2ptypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v3/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/monitoring/tracing"
	pb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"go.opencensus.io/trace"
)

type BlobsSidecarProcessor func(sidecar *pb.BlobsSidecar) error

func (s *Service) streamBlobBatch(ctx context.Context, batch blockBatch, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.streamBlobBatch")
	defer span.End()
	for _, b := range batch.Sequence() {
		root := b.Root()
		commitments, err := b.Block().Body().BlobKzgCommitments()
		if err != nil {
			return errors.Wrapf(err, "unable to retrieve commitments from block root %#x", root)
		}
		for i := 0; i < len(commitments); i++ {
			idx := uint64(i)
			sc, err := s.blobs.BlobSidecar(root, idx)
			if err != nil {
				if errors.Is(err, db.ErrNotFound) {
					continue
				}
				log.WithError(err).Debugf("error retrieving BlobSidecar, root=%x, idnex=%d", root, idx)
				s.writeErrorResponseToStream(responseCodeServerError, p2ptypes.ErrGeneric.Error(), stream)
				return err
			}
			SetStreamWriteDeadline(stream, defaultWriteDuration)
			if chunkErr := WriteBlobSidecarChunk(stream, s.cfg.chain, s.cfg.p2p.Encoding(), sc); chunkErr != nil {
				log.WithError(chunkErr).Debug("Could not send a chunked response")
				s.writeErrorResponseToStream(responseCodeServerError, p2ptypes.ErrGeneric.Error(), stream)
				tracing.AnnotateError(span, chunkErr)
				return chunkErr
			}
			s.rateLimiter.add(stream, 1)
		}
	}
	return nil
}

// blobsSidecarsByRangeRPCHandler looks up the request blobs from the database from a given start slot index
func (s *Service) blobsSidecarsByRangeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.BlobsSidecarsByRangeHandler")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()
	SetRPCStreamDeadlines(stream)

	r, ok := msg.(*pb.BlobSidecarsByRangeRequest)
	if !ok {
		return errors.New("message is not type *pb.BlobsSidecarsByRangeRequest")
	}
	if err := s.rateLimiter.validateRequest(stream, 1); err != nil {
		return err
	}
	start, end, size, err := validateBlobsByRange(r, s.cfg.chain.CurrentSlot())
	if err != nil {
		s.writeErrorResponseToStream(responseCodeInvalidRequest, err.Error(), stream)
		s.cfg.p2p.Peers().Scorers().BadResponsesScorer().Increment(stream.Conn().RemotePeer())
		tracing.AnnotateError(span, err)
		return err
	}

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

	var batch blockBatch
	for batch, ok = batcher.Next(ctx, stream); ok; batch, ok = batcher.Next(ctx, stream) {
		batchStart := time.Now()
		rpcBlobsByRangeResponseLatency.Observe(float64(time.Since(batchStart).Milliseconds()))
		if err := s.streamBlobBatch(ctx, batch, stream); err != nil {
			return err
		}
	}

	closeStream(stream, log)
	return nil
}

func validateBlobsByRange(r *pb.BlobSidecarsByRangeRequest, current primitives.Slot) (primitives.Slot, primitives.Slot, uint64, error) {
	start := r.StartSlot
	count := r.Count
	if r.StartSlot > current {
		return 0, 0, 0, p2ptypes.ErrInvalidRequest
	}
	minStart := params.BeaconConfig().DenebForkEpoch

	// Clients MUST keep a record of signed blobs sidecars seen on the epoch range
	// [max(current_epoch - MIN_EPOCHS_FOR_BLOB_SIDECARS_REQUESTS, DENEB_FORK_EPOCH), current_epoch]
	// where current_epoch is defined by the current wall-clock time,
	// and clients MUST support serving requests of blobs on this range.
	minReqEpochs := params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest
	currEpoch := slots.ToEpoch(current)
	if currEpoch > minReqEpochs && currEpoch-minReqEpochs > minStart {
		minStart = currEpoch - minReqEpochs
	}

	maxRequest := params.BeaconNetworkConfig().MaxRequestBlocksDeneb
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

	limit := uint64(flags.Get().BlobBatchLimit)
	if limit > maxRequest {
		limit = maxRequest
	}
	batchSize := count
	if batchSize > limit {
		batchSize = limit
	}

	return start, end, batchSize, nil
}
