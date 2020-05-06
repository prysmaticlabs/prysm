package sync

import (
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// beaconBlocksByRangeRPCHandler looks up the request blocks from the database from a given start block.
func (r *Service) beaconBlocksByRangeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.BeaconBlocksByRangeHandler")
	defer span.End()
	defer func() {
		if err := stream.Close(); err != nil {
			log.WithError(err).Error("Failed to close stream")
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)
	log := log.WithField("handler", "beacon_blocks_by_range")

	// Ticker to stagger out large requests.
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	m, ok := msg.(*pb.BeaconBlocksByRangeRequest)
	if !ok {
		return errors.New("message is not type *pb.BeaconBlockByRangeRequest")
	}

	// The initial count for the first batch to be returned back.
	count := m.Count
	if count > uint64(allowedBlocksPerSecond) {
		count = uint64(allowedBlocksPerSecond)
	}
	// initial batch start and end slots to be returned to remote peer.
	startSlot := m.StartSlot
	endSlot := startSlot + (m.Step * (count - 1))

	// The final requested slot from remote peer.
	endReqSlot := startSlot + (m.Step * (m.Count - 1))

	remainingBucketCapacity := r.blocksRateLimiter.Remaining(stream.Conn().RemotePeer().String())
	span.AddAttributes(
		trace.Int64Attribute("start", int64(startSlot)),
		trace.Int64Attribute("end", int64(endReqSlot)),
		trace.Int64Attribute("step", int64(m.Step)),
		trace.Int64Attribute("count", int64(m.Count)),
		trace.StringAttribute("peer", stream.Conn().RemotePeer().Pretty()),
		trace.Int64Attribute("remaining_capacity", remainingBucketCapacity),
	)
	for startSlot <= endReqSlot {
		remainingBucketCapacity = r.blocksRateLimiter.Remaining(stream.Conn().RemotePeer().String())

		if int64(allowedBlocksPerSecond) > remainingBucketCapacity {
			r.p2p.Peers().IncrementBadResponses(stream.Conn().RemotePeer())
			if r.p2p.Peers().IsBad(stream.Conn().RemotePeer()) {
				log.Debug("Disconnecting bad peer")
				defer func() {
					if err := r.p2p.Disconnect(stream.Conn().RemotePeer()); err != nil {
						log.WithError(err).Error("Failed to disconnect peer")
					}
				}()
			}
			r.writeErrorResponseToStream(responseCodeInvalidRequest, rateLimitedError, stream)
			return errors.New(rateLimitedError)
		}
		r.blocksRateLimiter.Add(stream.Conn().RemotePeer().String(), int64(allowedBlocksPerSecond))

		// TODO(3147): Update this with reasonable constraints.
		if endSlot-startSlot > rangeLimit || m.Step == 0 {
			r.writeErrorResponseToStream(responseCodeInvalidRequest, stepError, stream)
			err := errors.New(stepError)
			traceutil.AnnotateError(span, err)
			return err
		}

		if err := r.writeBlockRangeToStream(ctx, startSlot, endSlot, m.Step, stream); err != nil {
			return err
		}

		// Recalculate start and end slots for the next batch to be returned to the remote peer.
		startSlot = endSlot + m.Step
		endSlot = startSlot + (m.Step * (uint64(allowedBlocksPerSecond) - 1))
		if endSlot > endReqSlot {
			endSlot = endReqSlot
		}

		// do not wait if all blocks have already been sent.
		if startSlot > endReqSlot {
			break
		}

		// wait for ticker before resuming streaming blocks to remote peer.
		<-ticker.C
	}
	return nil
}

func (r *Service) writeBlockRangeToStream(ctx context.Context, startSlot, endSlot, step uint64, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.WriteBlockRangeToStream")
	defer span.End()

	filter := filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot).SetSlotStep(step)
	blks, err := r.db.Blocks(ctx, filter)
	if err != nil {
		log.WithError(err).Error("Failed to retrieve blocks")
		r.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
		traceutil.AnnotateError(span, err)
		return err
	}
	roots, err := r.db.BlockRoots(ctx, filter)
	if err != nil {
		log.WithError(err).Error("Failed to retrieve block roots")
		r.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
		traceutil.AnnotateError(span, err)
		return err
	}
	// handle genesis case
	if startSlot == 0 {
		genBlock, genRoot, err := r.retrieveGenesisBlock(ctx)
		if err != nil {
			log.WithError(err).Error("Failed to retrieve genesis block")
			r.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
			traceutil.AnnotateError(span, err)
			return err
		}
		blks = append([]*ethpb.SignedBeaconBlock{genBlock}, blks...)
		roots = append([][32]byte{genRoot}, roots...)
	}
	checkpoint, err := r.db.FinalizedCheckpoint(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to retrieve finalized checkpoint")
		r.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
		traceutil.AnnotateError(span, err)
		return err
	}
	for i, b := range blks {
		if b == nil || b.Block == nil {
			continue
		}
		blk := b.Block

		isRequestedSlotStep := (blk.Slot-startSlot)%step == 0
		isRecentUnfinalizedSlot := blk.Slot >= helpers.StartSlot(checkpoint.Epoch+1) || checkpoint.Epoch == 0
		if isRequestedSlotStep && (isRecentUnfinalizedSlot || r.db.IsFinalizedBlock(ctx, roots[i])) {
			if err := r.chunkWriter(stream, b); err != nil {
				log.WithError(err).Error("Failed to send a chunked response")
				return err
			}
		}
	}
	return nil
}

func (r *Service) writeErrorResponseToStream(responseCode byte, reason string, stream libp2pcore.Stream) {
	resp, err := r.generateErrorResponse(responseCode, reason)
	if err != nil {
		log.WithError(err).Error("Failed to generate a response error")
	} else {
		if _, err := stream.Write(resp); err != nil {
			log.WithError(err).Errorf("Failed to write to stream")
		}
	}
}

func (r *Service) retrieveGenesisBlock(ctx context.Context) (*ethpb.SignedBeaconBlock, [32]byte, error) {
	genBlock, err := r.db.GenesisBlock(ctx)
	if err != nil {
		return nil, [32]byte{}, err
	}
	genRoot, err := stateutil.BlockRoot(genBlock.Block)
	if err != nil {
		return nil, [32]byte{}, err
	}
	return genBlock, genRoot, nil
}
