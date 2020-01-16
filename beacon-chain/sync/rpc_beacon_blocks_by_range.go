package sync

import (
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// beaconBlocksByRangeRPCHandler looks up the request blocks from the database from a given start block.
func (r *Service) beaconBlocksByRangeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.BeaconBlocksByRangeHandler")
	defer span.End()
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)
	log := log.WithField("handler", "beacon_blocks_by_range")

	m := msg.(*pb.BeaconBlocksByRangeRequest)

	startSlot := m.StartSlot
	endSlot := startSlot + (m.Step * (m.Count - 1))
	remainingBucketCapacity := r.blocksRateLimiter.Remaining(stream.Conn().RemotePeer().String())

	span.AddAttributes(
		trace.Int64Attribute("start", int64(startSlot)),
		trace.Int64Attribute("end", int64(endSlot)),
		trace.Int64Attribute("step", int64(m.Step)),
		trace.Int64Attribute("count", int64(m.Count)),
		trace.StringAttribute("peer", stream.Conn().RemotePeer().Pretty()),
		trace.Int64Attribute("remaining_capacity", remainingBucketCapacity),
	)

	if m.Count > uint64(remainingBucketCapacity) {
		r.p2p.Peers().IncrementBadResponses(stream.Conn().RemotePeer())
		if r.p2p.Peers().IsBad(stream.Conn().RemotePeer()) {
			log.Debug("Disconnecting bad peer")
			defer r.p2p.Disconnect(stream.Conn().RemotePeer())
		}
		resp, err := r.generateErrorResponse(responseCodeInvalidRequest, rateLimitedError)
		if err != nil {
			log.WithError(err).Error("Failed to generate a response error")
		} else {
			if _, err := stream.Write(resp); err != nil {
				log.WithError(err).Errorf("Failed to write to stream")
			}
		}
		return errors.New(rateLimitedError)
	}

	r.blocksRateLimiter.Add(stream.Conn().RemotePeer().String(), int64(m.Count))

	// TODO(3147): Update this with reasonable constraints.
	if endSlot-startSlot > 1000 || m.Step == 0 {
		resp, err := r.generateErrorResponse(responseCodeInvalidRequest, "invalid range or step")
		if err != nil {
			log.WithError(err).Error("Failed to generate a response error")
		} else {
			if _, err := stream.Write(resp); err != nil {
				log.WithError(err).Errorf("Failed to write to stream")
			}
		}
		err = errors.New("invalid range or step")
		traceutil.AnnotateError(span, err)
		return err
	}

	var errResponse = func() {
		resp, err := r.generateErrorResponse(responseCodeServerError, genericError)
		if err != nil {
			log.WithError(err).Error("Failed to generate a response error")
		} else {
			if _, err := stream.Write(resp); err != nil {
				log.WithError(err).Errorf("Failed to write to stream")
			}
		}
	}

	filter := filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot).SetSlotStep(m.Step)
	blks, err := r.db.Blocks(ctx, filter)
	if err != nil {
		log.WithError(err).Error("Failed to retrieve blocks")
		errResponse()
		traceutil.AnnotateError(span, err)
		return err
	}
	roots, err := r.db.BlockRoots(ctx, filter)
	if err != nil {
		log.WithError(err).Error("Failed to retrieve block roots")
		errResponse()
		traceutil.AnnotateError(span, err)
		return err
	}
	checkpoint, err := r.db.FinalizedCheckpoint(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to retrieve finalized checkpoint")
		errResponse()
		traceutil.AnnotateError(span, err)
		return err
	}
	for i, b := range blks {
		if b == nil || b.Block == nil {
			continue
		}
		blk := b.Block

		isRequestedSlotStep := (blk.Slot-startSlot)%m.Step == 0
		isRecentUnfinalizedSlot := blk.Slot >= helpers.StartSlot(checkpoint.Epoch+1) || checkpoint.Epoch == 0
		if isRequestedSlotStep && (isRecentUnfinalizedSlot || r.db.IsFinalizedBlock(ctx, roots[i])) {
			if err := r.chunkWriter(stream, b); err != nil {
				log.WithError(err).Error("Failed to send a chunked response")
				return err
			}
		}
	}
	return err
}
