package sync

import (
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/monitoring/tracing"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// txBlobsByRangeRPCHandler --
func (s *Service) txBlobsByRangeRPCHandler(
	ctx context.Context, msg interface{}, stream libp2pcore.Stream,
) error {
	ctx, span := trace.StartSpan(ctx, "sync.txBlobsByRangeRPCHandler")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()
	SetRPCStreamDeadlines(stream)

	// Ticker to stagger out large requests.
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	m, ok := msg.(*pb.BeaconBlocksByRangeRequest)
	if !ok {
		return errors.New("message is not type *pb.TxBlobsByRangeRequest")
	}
	if err := s.validateRangeRequest(m); err != nil {
		s.writeErrorResponseToStream(
			responseCodeInvalidRequest,
			err.Error(),
			stream,
		)
		s.cfg.p2p.Peers().Scorers().BadResponsesScorer().Increment(
			stream.Conn().RemotePeer(),
		)
		tracing.AnnotateError(span, err)
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
	endSlot := startSlot.Add(m.Step * (count - 1))

	// The final requested slot from remote peer.
	endReqSlot := startSlot.Add(m.Step * (m.Count - 1))
}

func (s *Service) validateTxBlobRangeRequest(r *pb.BeaconBlocksByRangeRequest) error {
	startSlot := r.StartSlot
	count := r.Count
	step := r.Step

	maxRequestBlocks := params.BeaconNetworkConfig().MaxRequestBlocks
	// Add a buffer for possible large range requests from nodes syncing close to the
	// head of the chain.
	buffer := rangeLimit * 2
	highestExpectedSlot := s.cfg.chain.CurrentSlot().Add(uint64(buffer))

	// Ensure all request params are within appropriate bounds
	if count == 0 || count > maxRequestBlocks {
		return p2ptypes.ErrInvalidRequest
	}

	if step == 0 || step > rangeLimit {
		return p2ptypes.ErrInvalidRequest
	}

	if startSlot > highestExpectedSlot {
		return p2ptypes.ErrInvalidRequest
	}

	endSlot := startSlot.Add(step * (count - 1))
	if endSlot-startSlot > rangeLimit {
		return p2ptypes.ErrInvalidRequest
	}
	return nil
}

func (s *Service) writeTxBlobRangeToStream(ctx context.Context, startSlot, endSlot types.Slot, step uint64,
	prevRoot *[32]byte, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.writeTxBlobRangeToStream")
	defer span.End()

	// Get blobs and write them to the stream...
	//for _, b := range blks {
	//	if b == nil || b.IsNil() || b.Block().IsNil() {
	//		continue
	//	}
	//	if chunkErr := s.chunkBlockWriter(stream, b); chunkErr != nil {
	//		log.WithError(chunkErr).Debug("Could not send a chunked response")
	//		s.writeErrorResponseToStream(responseCodeServerError, p2ptypes.ErrGeneric.Error(), stream)
	//		tracing.AnnotateError(span, chunkErr)
	//		return chunkErr
	//	}
	//
	//}
	//// Return error in the event we have an invalid parent.
	//return err
	return nil
}
