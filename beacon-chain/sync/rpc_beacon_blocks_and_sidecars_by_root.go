package sync

import (
	"context"

	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// beaconBlockAndBlobsSidecarByRootRPCHandler looks up the request beacon block and blobs from the database from a given root
func (s *Service) beaconBlockAndBlobsSidecarByRootRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.beaconBlockAndBlobsSidecarByRootRPCHandler")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, ttfbTimeout)
	defer cancel()
	SetRPCStreamDeadlines(stream)

	log := log.WithField("handler", "beacon_block_and_blobs_sidecar_by_root")

	rawMsg, ok := msg.(*types.BeaconBlockByRootsReq)
	if !ok {
		return errors.New("message is not type BeaconBlockByRootsReq")
	}
	blockRoots := *rawMsg
	if err := s.rateLimiter.validateRequest(stream, uint64(len(blockRoots))); err != nil {
		return err
	}
	if len(blockRoots) == 0 {
		// Add to rate limiter in the event no
		// roots are requested.
		s.rateLimiter.add(stream, 1)
		s.writeErrorResponseToStream(responseCodeInvalidRequest, "no block roots provided in request", stream)
		return errors.New("no block roots provided")
	}

	if uint64(len(blockRoots)) > params.BeaconNetworkConfig().MaxRequestBlocks {
		s.cfg.p2p.Peers().Scorers().BadResponsesScorer().Increment(stream.Conn().RemotePeer())
		s.writeErrorResponseToStream(responseCodeInvalidRequest, "requested more than the max block limit", stream)
		return errors.New("requested more than the max block limit")
	}
	s.rateLimiter.add(stream, int64(len(blockRoots)))

	for _, root := range blockRoots {
		blk, err := s.cfg.beaconDB.Block(ctx, root)
		if err != nil {
			log.WithError(err).Debug("Could not fetch block")
			s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream)
			return err
		}

		sidecar, err := s.cfg.beaconDB.BlobsSidecar(ctx, root)
		if err != nil {
			log.WithError(err).Debug("Could not fetch sidecar")
			s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream) // TODO(4844): Return unavailable?
			return err
		}

		// TODO(4844): Reconstruct blind block
		pb, err := blk.Pb4844Block()
		if err != nil {
			log.WithError(err).Debug("Could not fetch block")
			s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream)
			return err
		}
		if err := WriteBlockAndBlobsSidecarChunk(stream, s.cfg.chain, s.cfg.p2p.Encoding(), &ethpb.SignedBeaconBlockAndBlobsSidecar{BeaconBlock: pb, BlobsSidecar: sidecar}); err != nil {
			log.WithError(err).Debug("Could not send block and sidecar")
			s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream)
			return err
		}
	}
	closeStream(stream, log)
	return nil
}

func (s *Service) sendBlocksAndSidecarsRequest(ctx context.Context, blockRoots *types.BeaconBlockByRootsReq, id peer.ID) error {
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()

	_, err := SendBlocksAndSidecarsByRootRequest(ctx, s.cfg.chain, s.cfg.p2p, id, blockRoots, func(blkAndSidecar *ethpb.SignedBeaconBlockAndBlobsSidecar) error {
		blk, err := blocks.NewSignedBeaconBlock(blkAndSidecar.BeaconBlock)
		if err != nil {
			return err
		}
		blkRoot, err := blk.Block().HashTreeRoot()
		if err != nil {
			return err
		}
		s.pendingQueueLock.Lock()
		defer s.pendingQueueLock.Unlock()
		if err := s.insertBlkAndBlobToQueue(blk.Block().Slot(), blk, blkRoot, blkAndSidecar.BlobsSidecar); err != nil {
			return err
		}
		return nil
	})
	return err
}
