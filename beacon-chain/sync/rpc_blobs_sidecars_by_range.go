package sync

import (
	"context"
	"fmt"
	"io"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blob"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/monitoring/tracing"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
	"go.opencensus.io/trace"
)

// We assume a cost of 1 MiB per sidecar responded to a range request.
const avgSidecarBlobsTransferBytes = 1 << 10

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

		exists, sidecars, err := s.cfg.beaconDB.BlobsSidecarsBySlot(ctx, slot)
		if err != nil {
			s.writeErrorResponseToStream(responseCodeServerError, p2ptypes.ErrGeneric.Error(), stream)
			tracing.AnnotateError(span, err)
			return err
		}
		if !exists {
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

func SendBlobsSidecarsByRangeRequest(
	ctx context.Context, db db.NoHeadAccessDatabase, chain blockchain.ChainInfoFetcher, p2pProvider p2p.P2P, pid peer.ID,
	req *pb.BlobsSidecarsByRangeRequest) ([]*pb.BlobsSidecar, error) {
	topic, err := p2p.TopicFromMessage(p2p.BlobsSidecarsByRangeMessageName, slots.ToEpoch(chain.CurrentSlot()))
	if err != nil {
		return nil, err
	}
	stream, err := p2pProvider.Send(ctx, req, topic, pid)
	if err != nil {
		return nil, err
	}
	defer closeStream(stream, log)

	var blobsSidecars []*pb.BlobsSidecar
	for {
		blobs, err := ReadChunkedBlobsSidecar(stream, chain, p2pProvider, len(blobsSidecars) == 0)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		signed, err := db.Block(ctx, bytesutil.ToBytes32(blobs.BeaconBlockRoot))
		if err != nil {
			return nil, err
		}
		if signed == nil || signed.IsNil() || signed.Block().IsNil() {
			return nil, fmt.Errorf("unable to find block with block root for slot: %v", blobs.BeaconBlockSlot)
		}
		blk, err := signed.PbEip4844Block()
		if err != nil {
			return nil, err
		}
		blockKzgs := blk.Block.Body.BlobKzgs
		expectedKzgs := make([][48]byte, len(blockKzgs))
		for i := range blockKzgs {
			expectedKzgs[i] = bytesutil.ToBytes48(blockKzgs[i])
		}
		if err := blob.VerifyBlobsSidecar(blobs.BeaconBlockSlot, bytesutil.ToBytes32(blobs.BeaconBlockRoot), expectedKzgs, blobs); err != nil {
			return nil, errors.Wrap(err, "invalid blobs sidecar")
		}

		blobsSidecars = append(blobsSidecars, blobs)
		if len(blobsSidecars) >= int(params.BeaconNetworkConfig().MaxRequestBlobsSidecars) {
			return nil, ErrInvalidFetchedData
		}
	}
	return blobsSidecars, nil
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
