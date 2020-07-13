package sync

import (
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// beaconBlocksByRangeRPCHandler looks up the request blocks from the database from a given start block.
func (s *Service) beaconBlocksByRangeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.BeaconBlocksByRangeHandler")
	defer span.End()
	defer func() {
		if err := stream.Close(); err != nil {
			log.WithError(err).Error("Failed to close stream")
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()
	SetRPCStreamDeadlines(stream)
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
	allowedBlocksPerSecond := uint64(flags.Get().BlockBatchLimit)
	if count > allowedBlocksPerSecond {
		count = allowedBlocksPerSecond
	}
	// initial batch start and end slots to be returned to remote peer.
	startSlot := m.StartSlot
	endSlot := startSlot + (m.Step * (count - 1))

	// The final requested slot from remote peer.
	endReqSlot := startSlot + (m.Step * (m.Count - 1))

	remainingBucketCapacity := s.blocksRateLimiter.Remaining(stream.Conn().RemotePeer().String())
	span.AddAttributes(
		trace.Int64Attribute("start", int64(startSlot)),
		trace.Int64Attribute("end", int64(endReqSlot)),
		trace.Int64Attribute("step", int64(m.Step)),
		trace.Int64Attribute("count", int64(m.Count)),
		trace.StringAttribute("peer", stream.Conn().RemotePeer().Pretty()),
		trace.Int64Attribute("remaining_capacity", remainingBucketCapacity),
	)
	maxRequestBlocks := params.BeaconNetworkConfig().MaxRequestBlocks
	for startSlot <= endReqSlot {
		remainingBucketCapacity = s.blocksRateLimiter.Remaining(stream.Conn().RemotePeer().String())
		if int64(allowedBlocksPerSecond) > remainingBucketCapacity {
			s.p2p.Peers().Scorer().IncrementBadResponses(stream.Conn().RemotePeer())
			if s.p2p.Peers().IsBad(stream.Conn().RemotePeer()) {
				log.Debug("Disconnecting bad peer")
				defer func() {
					if err := s.p2p.Disconnect(stream.Conn().RemotePeer()); err != nil {
						log.WithError(err).Error("Failed to disconnect peer")
					}
				}()
			}
			s.writeErrorResponseToStream(responseCodeInvalidRequest, rateLimitedError, stream)
			return errors.New(rateLimitedError)
		}

		if endSlot-startSlot > rangeLimit || m.Step == 0 || m.Count > maxRequestBlocks {
			s.writeErrorResponseToStream(responseCodeInvalidRequest, stepError, stream)
			err := errors.New(stepError)
			traceutil.AnnotateError(span, err)
			return err
		}

		if err := s.writeBlockRangeToStream(ctx, startSlot, endSlot, m.Step, stream); err != nil {
			return err
		}

		// Decrease allowed blocks capacity by the number of streamed blocks.
		if startSlot <= endSlot {
			s.blocksRateLimiter.Add(
				stream.Conn().RemotePeer().String(), int64(1+(endSlot-startSlot)/m.Step))
		}

		// Recalculate start and end slots for the next batch to be returned to the remote peer.
		startSlot = endSlot + m.Step
		endSlot = startSlot + (m.Step * (allowedBlocksPerSecond - 1))
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

func (s *Service) writeBlockRangeToStream(ctx context.Context, startSlot, endSlot, step uint64, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.WriteBlockRangeToStream")
	defer span.End()

	filter := filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot).SetSlotStep(step)
	blks, err := s.db.Blocks(ctx, filter)
	if err != nil {
		log.WithError(err).Error("Failed to retrieve blocks")
		s.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
		traceutil.AnnotateError(span, err)
		return err
	}
	roots, err := s.db.BlockRoots(ctx, filter)
	if err != nil {
		log.WithError(err).Error("Failed to retrieve block roots")
		s.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
		traceutil.AnnotateError(span, err)
		return err
	}
	// handle genesis case
	if startSlot == 0 {
		genBlock, genRoot, err := s.retrieveGenesisBlock(ctx)
		if err != nil {
			log.WithError(err).Error("Failed to retrieve genesis block")
			s.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
			traceutil.AnnotateError(span, err)
			return err
		}
		blks = append([]*ethpb.SignedBeaconBlock{genBlock}, blks...)
		roots = append([][32]byte{genRoot}, roots...)
	}
	// Filter and sort our retrieved blocks, so that
	// we only return valid sets of blocks.
	blks, roots = s.dedupBlocksAndRoots(blks, roots)
	blks, roots = s.sortBlocksAndRoots(blks, roots)
	for i, b := range blks {
		if b == nil || b.Block == nil {
			continue
		}
		blk := b.Block

		// Check that the block is valid according to the request and part of the canonical chain.
		isRequestedSlotStep := (blk.Slot-startSlot)%step == 0
		if isRequestedSlotStep {
			canonical, err := s.chain.IsCanonical(ctx, roots[i])
			if err != nil {
				log.WithError(err).Error("Failed to determine canonical block")
				s.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
				traceutil.AnnotateError(span, err)
				return err
			}
			if !canonical {
				continue
			}
			if err := s.chunkWriter(stream, b); err != nil {
				log.WithError(err).Error("Failed to send a chunked response")
				s.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
				traceutil.AnnotateError(span, err)
				return err
			}
		}
	}
	return nil
}

func (s *Service) writeErrorResponseToStream(responseCode byte, reason string, stream libp2pcore.Stream) {
	resp, err := s.generateErrorResponse(responseCode, reason)
	if err != nil {
		log.WithError(err).Error("Failed to generate a response error")
	} else {
		if _, err := stream.Write(resp); err != nil {
			log.WithError(err).Errorf("Failed to write to stream")
		}
	}
}

func (s *Service) retrieveGenesisBlock(ctx context.Context) (*ethpb.SignedBeaconBlock, [32]byte, error) {
	genBlock, err := s.db.GenesisBlock(ctx)
	if err != nil {
		return nil, [32]byte{}, err
	}
	genRoot, err := stateutil.BlockRoot(genBlock.Block)
	if err != nil {
		return nil, [32]byte{}, err
	}
	return genBlock, genRoot, nil
}
