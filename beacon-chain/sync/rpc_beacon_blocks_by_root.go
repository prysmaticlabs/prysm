package sync

import (
	"context"
	"io"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/mux"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// sendRecentBeaconBlocksRequest sends a recent beacon blocks request to a peer to get
// those corresponding blocks from that peer.
func (s *Service) sendRecentBeaconBlocksRequest(ctx context.Context, blockRoots [][32]byte, id peer.ID) error {
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()

	stream, err := s.p2p.Send(ctx, blockRoots, p2p.RPCBlocksByRootTopic, id)
	if err != nil {
		return err
	}
	defer func() {
		if err := helpers.FullClose(stream); err != nil {
			log.WithError(err).Debugf("Failed to reset stream with protocol %s", stream.Protocol())
		}
	}()
	for i := 0; i < len(blockRoots); i++ {
		isFirstChunk := i == 0
		blk, err := ReadChunkedBlock(stream, s.p2p, isFirstChunk)
		// Return error until #6408 is resolved.
		if err == io.EOF {
			return err
		}
		// Exit if peer sends more than max request blocks.
		if uint64(i) >= params.BeaconNetworkConfig().MaxRequestBlocks {
			break
		}
		if err != nil {
			log.WithError(err).Error("Unable to retrieve block from stream")
			return err
		}

		blkRoot, err := stateutil.BlockRoot(blk.Block)
		if err != nil {
			return err
		}
		s.pendingQueueLock.Lock()
		s.slotToPendingBlocks[blk.Block.Slot] = blk
		s.seenPendingBlocks[blkRoot] = true
		s.pendingQueueLock.Unlock()

	}
	return nil
}

// Deprecated: sendRecentBeaconBlocksRequestFallback sends a recent beacon blocks request to a peer to get
// those corresponding blocks from that peer. This is a method implemented so that we are eventually
// backward compatible with old Onyx nodes.
// TODO(#6408)
func (s *Service) sendRecentBeaconBlocksRequestFallback(ctx context.Context, blockRoots [][32]byte, id peer.ID) error {
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()
	req := &pbp2p.BeaconBlocksByRootRequest{}
	for _, root := range blockRoots {
		req.BlockRoots = append(req.BlockRoots, root[:])
	}
	stream, err := s.p2p.Send(ctx, req, p2p.RPCBlocksByRootTopic, id)
	if err != nil {
		return err
	}
	defer func() {
		if err := helpers.FullClose(stream); err != nil && err.Error() != mux.ErrReset.Error() {
			log.WithError(err).Debugf("Failed to reset stream with protocol %s", stream.Protocol())
		}
	}()
	for i := 0; i < len(blockRoots); i++ {
		isFirstChunk := i == 0
		blk, err := ReadChunkedBlock(stream, s.p2p, isFirstChunk)
		if err == io.EOF {
			break
		}
		// Exit if peer sends more than max request blocks.
		if uint64(i) >= params.BeaconNetworkConfig().MaxRequestBlocks {
			break
		}
		if err != nil {
			log.WithError(err).Error("Unable to retrieve block from stream")
			return err
		}

		blkRoot, err := stateutil.BlockRoot(blk.Block)
		if err != nil {
			return err
		}
		s.pendingQueueLock.Lock()
		s.slotToPendingBlocks[blk.Block.Slot] = blk
		s.seenPendingBlocks[blkRoot] = true
		s.pendingQueueLock.Unlock()

	}
	return nil
}

// beaconBlocksRootRPCHandler looks up the request blocks from the database from the given block roots.
func (s *Service) beaconBlocksRootRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer func() {
		if err := stream.Close(); err != nil {
			log.WithError(err).Error("Failed to close stream")
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, ttfbTimeout)
	defer cancel()
	SetRPCStreamDeadlines(stream)
	log := log.WithField("handler", "beacon_blocks_by_root")

	blockRoots, ok := msg.([][32]byte)
	if !ok {
		return errors.New("message is not type BeaconBlocksByRootRequest")
	}
	if len(blockRoots) == 0 {
		resp, err := s.generateErrorResponse(responseCodeInvalidRequest, "no block roots provided in request")
		if err != nil {
			log.WithError(err).Error("Failed to generate a response error")
		} else {
			if _, err := stream.Write(resp); err != nil {
				log.WithError(err).Errorf("Failed to write to stream")
			}
		}
		return errors.New("no block roots provided")
	}

	if int64(len(blockRoots)) > s.blocksRateLimiter.Remaining(stream.Conn().RemotePeer().String()) {
		s.p2p.Peers().IncrementBadResponses(stream.Conn().RemotePeer())
		if s.p2p.Peers().IsBad(stream.Conn().RemotePeer()) {
			log.Debug("Disconnecting bad peer")
			defer func() {
				if err := s.p2p.Disconnect(stream.Conn().RemotePeer()); err != nil {
					log.WithError(err).Error("Failed to disconnect peer")
				}
			}()
		}
		resp, err := s.generateErrorResponse(responseCodeInvalidRequest, rateLimitedError)
		if err != nil {
			log.WithError(err).Error("Failed to generate a response error")
		} else {
			if _, err := stream.Write(resp); err != nil {
				log.WithError(err).Errorf("Failed to write to stream")
			}
		}
		return errors.New(rateLimitedError)
	}

	if uint64(len(blockRoots)) > params.BeaconNetworkConfig().MaxRequestBlocks {
		resp, err := s.generateErrorResponse(responseCodeInvalidRequest, "requested more than the max block limit")
		if err != nil {
			log.WithError(err).Error("Failed to generate a response error")
		} else {
			if _, err := stream.Write(resp); err != nil {
				log.WithError(err).Errorf("Failed to write to stream")
			}
		}
		return errors.New("requested more than the max block limit")
	}

	s.blocksRateLimiter.Add(stream.Conn().RemotePeer().String(), int64(len(blockRoots)))

	for _, root := range blockRoots {
		blk, err := s.db.Block(ctx, root)
		if err != nil {
			log.WithError(err).Error("Failed to fetch block")
			resp, err := s.generateErrorResponse(responseCodeServerError, genericError)
			if err != nil {
				log.WithError(err).Error("Failed to generate a response error")
			} else {
				if _, err := stream.Write(resp); err != nil {
					log.WithError(err).Errorf("Failed to write to stream")
				}
			}
			return err
		}
		if blk == nil {
			continue
		}
		if err := s.chunkWriter(stream, blk); err != nil {
			return err
		}
	}
	return nil
}
