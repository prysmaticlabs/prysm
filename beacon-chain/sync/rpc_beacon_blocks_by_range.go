package sync

import (
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
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
			log.WithError(err).Debug("Failed to close stream")
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()
	SetRPCStreamDeadlines(stream)

	// Ticker to stagger out large requests.
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	m, ok := msg.(*pb.BeaconBlocksByRangeRequest)
	if !ok {
		return errors.New("message is not type *pb.BeaconBlockByRangeRequest")
	}
	if err := s.validateRangeRequest(m); err != nil {
		s.writeErrorResponseToStream(responseCodeInvalidRequest, err.Error(), stream)
		s.p2p.Peers().Scorers().BadResponsesScorer().Increment(stream.Conn().RemotePeer())
		traceutil.AnnotateError(span, err)
		return err
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

	blockLimiter, err := s.rateLimiter.topicCollector(string(stream.Protocol()))
	if err != nil {
		return err
	}
	remainingBucketCapacity := blockLimiter.Remaining(stream.Conn().RemotePeer().String())
	span.AddAttributes(
		trace.Int64Attribute("start", int64(startSlot)),
		trace.Int64Attribute("end", int64(endReqSlot)),
		trace.Int64Attribute("step", int64(m.Step)),
		trace.Int64Attribute("count", int64(m.Count)),
		trace.StringAttribute("peer", stream.Conn().RemotePeer().Pretty()),
		trace.Int64Attribute("remaining_capacity", remainingBucketCapacity),
	)
	for startSlot <= endReqSlot {
		if err := s.rateLimiter.validateRequest(stream, allowedBlocksPerSecond); err != nil {
			traceutil.AnnotateError(span, err)
			return err
		}

		if endSlot-startSlot > rangeLimit {
			s.writeErrorResponseToStream(responseCodeInvalidRequest, reqError, stream)
			err := errors.New(reqError)
			traceutil.AnnotateError(span, err)
			return err
		}

		if err := s.writeBlockRangeToStream(ctx, startSlot, endSlot, m.Step, stream); err != nil {
			return err
		}

		// Decrease allowed blocks capacity by the number of streamed blocks.
		if startSlot <= endSlot {
			s.rateLimiter.add(stream, int64(1+(endSlot-startSlot)/m.Step))
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
		log.WithError(err).Debug("Failed to retrieve blocks")
		s.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
		traceutil.AnnotateError(span, err)
		return err
	}
	roots := make([][32]byte, 0, len(blks))
	for _, b := range blks {
		root, err := b.Block.HashTreeRoot()
		if err != nil {
			log.WithError(err).Debug("Failed to retrieve block root")
			s.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
			traceutil.AnnotateError(span, err)
			return err
		}
		roots = append(roots, root)
	}
	// handle genesis case
	if startSlot == 0 {
		genBlock, genRoot, err := s.retrieveGenesisBlock(ctx)
		if err != nil {
			log.WithError(err).Debug("Failed to retrieve genesis block")
			s.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
			traceutil.AnnotateError(span, err)
			return err
		}
		blks = append([]*ethpb.SignedBeaconBlock{genBlock}, blks...)
		roots = append([][32]byte{genRoot}, roots...)
	}
	// Filter and sort our retrieved blocks, so that
	// we only return valid sets of blocks.
	blks, roots, err = s.dedupBlocksAndRoots(blks, roots)
	if err != nil {
		s.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
		traceutil.AnnotateError(span, err)
		return err
	}
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
				log.WithError(err).Debug("Failed to determine canonical block")
				s.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
				traceutil.AnnotateError(span, err)
				return err
			}
			if !canonical {
				continue
			}
			if err := s.chunkWriter(stream, b); err != nil {
				log.WithError(err).Debug("Failed to send a chunked response")
				s.writeErrorResponseToStream(responseCodeServerError, genericError, stream)
				traceutil.AnnotateError(span, err)
				return err
			}
		}
	}
	return nil
}

func (s *Service) validateRangeRequest(r *pb.BeaconBlocksByRangeRequest) error {
	startSlot := r.StartSlot
	count := r.Count
	step := r.Step

	maxRequestBlocks := params.BeaconNetworkConfig().MaxRequestBlocks
	// Add a buffer for possible large range requests from nodes syncing close to the
	// head of the chain.
	buffer := rangeLimit * 2
	highestExpectedSlot := s.chain.CurrentSlot() + uint64(buffer)

	// Ensure all request params are within appropriate bounds
	if count == 0 || count > maxRequestBlocks {
		return errors.New(reqError)
	}

	if step == 0 || step > rangeLimit {
		return errors.New(reqError)
	}

	if startSlot > highestExpectedSlot {
		return errors.New(reqError)
	}

	endSlot := startSlot + (step * (count - 1))
	if endSlot-startSlot > rangeLimit {
		return errors.New(reqError)
	}
	return nil
}

func (s *Service) writeErrorResponseToStream(responseCode byte, reason string, stream libp2pcore.Stream) {
	writeErrorResponseToStream(responseCode, reason, stream, s.p2p)
}

func (s *Service) retrieveGenesisBlock(ctx context.Context) (*ethpb.SignedBeaconBlock, [32]byte, error) {
	genBlock, err := s.db.GenesisBlock(ctx)
	if err != nil {
		return nil, [32]byte{}, err
	}
	genRoot, err := genBlock.Block.HashTreeRoot()
	if err != nil {
		return nil, [32]byte{}, err
	}
	return genBlock, genRoot, nil
}
