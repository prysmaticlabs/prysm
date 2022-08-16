package sync

import (
	"context"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
)

// sendRecentBeaconBlocksRequest sends a recent beacon blocks request to a peer to get
// those corresponding blocks from that peer.
func (s *Service) sendRecentBeaconBlocksRequest(ctx context.Context, blockRoots *types.BeaconBlockByRootsReq, id peer.ID) error {
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()

	_, err := SendBeaconBlocksByRootRequest(ctx, s.cfg.chain, s.cfg.p2p, id, blockRoots, func(blk interfaces.SignedBeaconBlock) error {
		blkRoot, err := blk.Block().HashTreeRoot()
		if err != nil {
			return err
		}
		s.pendingQueueLock.Lock()
		if err := s.insertBlockToPendingQueue(blk.Block().Slot(), blk, blkRoot); err != nil {
			return err
		}
		s.pendingQueueLock.Unlock()
		return nil
	})
	return err
}

// beaconBlocksRootRPCHandler looks up the request blocks from the database from the given block roots.
func (s *Service) beaconBlocksRootRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	ctx, cancel := context.WithTimeout(ctx, ttfbTimeout)
	defer cancel()
	SetRPCStreamDeadlines(stream)
	log := log.WithField("handler", "beacon_blocks_by_root")

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
		if err := blocks.BeaconBlockIsNil(blk); err != nil {
			continue
		}

		if blk.Block().IsBlinded() {
			blk, err = s.cfg.executionPayloadReconstructor.ReconstructFullBellatrixBlock(ctx, blk)
			if err != nil {
				log.WithError(err).Error("Could not get reconstruct full bellatrix block from blinded body")
				s.writeErrorResponseToStream(responseCodeServerError, types.ErrGeneric.Error(), stream)
				return err
			}
		}

		if err := s.chunkBlockWriter(stream, blk); err != nil {
			return err
		}
	}

	closeStream(stream, log)
	return nil
}
