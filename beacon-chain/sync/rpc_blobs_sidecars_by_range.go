package sync

import (
	"context"
	"io"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p"
	p2ptypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/monitoring/tracing"
	pb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"go.opencensus.io/trace"
)

// We assume a cost of 1 MiB per sidecar responded to a range request.
const avgSidecarBlobsTransferBytes = 1 << 10

type BlobsSidecarProcessor func(sidecar *pb.BlobsSidecar) error

// blobsSidecarsByRangeRPCHandler looks up the request blobs from the database from a given start slot index
func (s *Service) blobsSidecarsByRangeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.BlobsSidecarsByRangeHandler")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()
	SetRPCStreamDeadlines(stream)

	r, ok := msg.(*pb.BlobsSidecarsByRangeRequest)
	if !ok {
		return errors.New("message is not type *pb.BlobsSidecarsByRangeRequest")
	}

	startSlot := r.StartSlot
	count := r.Count
	endSlot := startSlot.Add(count)

	var numBlobs uint64
	maxRequestBlobsSidecars := params.BeaconNetworkConfig().MaxRequestBlobsSidecars
	for slot := startSlot; slot < endSlot && numBlobs < maxRequestBlobsSidecars; slot = slot.Add(1) {
		// TODO(XXX): With danksharding, there could be multiple sidecars per slot. As such, the cost requirement will be dynamic
		if err := s.rateLimiter.validateRequest(stream, uint64(avgSidecarBlobsTransferBytes)); err != nil {
			return err
		}

		sidecars, err := s.cfg.beaconDB.BlobsSidecarsBySlot(ctx, slot)
		if err != nil {
			s.writeErrorResponseToStream(responseCodeServerError, p2ptypes.ErrGeneric.Error(), stream)
			tracing.AnnotateError(span, err)
			return err
		}
		if len(sidecars) == 0 {
			continue
		}

		var outLen int
		for _, sidecar := range sidecars {
			outLen += estimateBlobsSidecarCost(sidecar)
			SetStreamWriteDeadline(stream, defaultWriteDuration)
			if chunkErr := WriteBlobsSidecarChunk(stream, s.cfg.chain, s.cfg.p2p.Encoding(), sidecar); chunkErr != nil {
				log.WithError(chunkErr).Debug("Could not send a chunked response")
				s.writeErrorResponseToStream(responseCodeServerError, p2ptypes.ErrGeneric.Error(), stream)
				tracing.AnnotateError(span, chunkErr)
				return chunkErr
			}
		}
		numBlobs++
		s.rateLimiter.add(stream, int64(outLen))

		// Short-circuit immediately once we've sent the last blob.
		if slot.Add(1) >= endSlot {
			break
		}

		key := stream.Conn().RemotePeer().String()
		sidecarLimiter, err := s.rateLimiter.topicCollector(string(stream.Protocol()))
		if err != nil {
			return err
		}
		// Throttling - wait until we have enough tokens to send the next blobs
		if sidecarLimiter.Remaining(key) < avgSidecarBlobsTransferBytes {
			timer := time.NewTimer(sidecarLimiter.TillEmpty(key))
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
				timer.Stop()
			}
		}
	}

	closeStream(stream, log)
	return nil
}

// sendRecentBlobsSidecarsRequest retrieves sidecars and inserts them to the pending queue
func (s *Service) sendRecentBlobSidecarsRequest(ctx context.Context, req *pb.BlobsSidecarsByRangeRequest, pid peer.ID) error {
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()

	_, err := SendBlobsSidecarsByRangeRequest(ctx, s.cfg.chain, s.cfg.p2p, pid, req, func(sc *pb.BlobsSidecar) error {
		s.pendingQueueLock.Lock()
		s.insertSidecarToPendingQueue(&queuedBlobsSidecar{s: sc})
		s.pendingQueueLock.Unlock()
		return nil
	})
	return err
}

func SendBlobsSidecarsByRangeRequest(
	ctx context.Context, chain blockchain.ChainInfoFetcher, p2pProvider p2p.P2P, pid peer.ID,
	req *pb.BlobsSidecarsByRangeRequest, sidecarProcessor BlobsSidecarProcessor) ([]*pb.BlobsSidecar, error) {
	topic, err := p2p.TopicFromMessage(p2p.BlobsSidecarsByRangeMessageName, slots.ToEpoch(chain.CurrentSlot()))
	if err != nil {
		return nil, err
	}
	stream, err := p2pProvider.Send(ctx, req, topic, pid)
	if err != nil {
		return nil, err
	}
	defer closeStream(stream, log)

	var sidecars []*pb.BlobsSidecar
	process := func(sidecar *pb.BlobsSidecar) error {
		sidecars = append(sidecars, sidecar)
		if sidecarProcessor != nil {
			return sidecarProcessor(sidecar)
		}
		return nil
	}

	var prevSlot types.Slot
	for i := uint64(0); ; i++ {
		isFirstChunk := len(sidecars) == 0
		sidecar, err := ReadChunkedBlobsSidecar(stream, chain, p2pProvider, isFirstChunk)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		if i >= req.Count || i >= params.BeaconNetworkConfig().MaxRequestBlobsSidecars {
			return nil, ErrInvalidFetchedData
		}
		if sidecar.BeaconBlockSlot < req.StartSlot || sidecar.BeaconBlockSlot >= req.StartSlot.Add(req.Count) {
			return nil, ErrInvalidFetchedData
		}
		// assert slots aren't out of order and always increasing
		if prevSlot >= sidecar.BeaconBlockSlot {
			return nil, ErrInvalidFetchedData
		}
		prevSlot = sidecar.BeaconBlockSlot

		if err := process(sidecar); err != nil {
			return nil, err
		}
	}
	return sidecars, nil
}

func estimateBlobsSidecarCost(sidecar *pb.BlobsSidecar) int {
	// This represents the fixed cost (in bytes) of the beacon_block_root and beacon_block_slot fields in the sidecar
	const overheadCost = 32 + 8
	cost := overheadCost
	for _, blob := range sidecar.Blobs {
		for _, b := range blob.Blob {
			cost += len(b)
		}
	}
	return cost
}
