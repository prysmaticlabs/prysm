package sync

import (
	"context"
	"io"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
)

// sendRecentBeaconBlocksRequest sends a recent beacon blocks request to a peer to get
// those corresponding blocks from that peer.
func (r *Service) sendRecentBeaconBlocksRequest(ctx context.Context, blockRoots [][32]byte, id peer.ID) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	stream, err := r.p2p.Send(ctx, blockRoots, id)
	if err != nil {
		return err
	}
	for i := 0; i < len(blockRoots); i++ {
		blk, err := ReadChunkedBlock(stream, r.p2p)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.WithError(err).Error("Unable to retrieve block from stream")
			return err
		}

		blkRoot, err := ssz.HashTreeRoot(blk.Block)
		if err != nil {
			return err
		}
		r.pendingQueueLock.Lock()
		r.slotToPendingBlocks[blk.Block.Slot] = blk
		r.seenPendingBlocks[blkRoot] = true
		r.pendingQueueLock.Unlock()

	}
	return nil
}

// beaconBlocksRootRPCHandler looks up the request blocks from the database from the given block roots.
func (r *Service) beaconBlocksRootRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)
	log := log.WithField("handler", "beacon_blocks_by_root")

	blockRoots := msg.([][32]byte)
	if len(blockRoots) == 0 {
		resp, err := r.generateErrorResponse(responseCodeInvalidRequest, "no block roots provided in request")
		if err != nil {
			log.WithError(err).Error("Failed to generate a response error")
		} else {
			if _, err := stream.Write(resp); err != nil {
				log.WithError(err).Errorf("Failed to write to stream")
			}
		}
		return errors.New("no block roots provided")
	}

	if int64(len(blockRoots)) > r.blocksRateLimiter.Remaining(stream.Conn().RemotePeer().String()) {
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

	r.blocksRateLimiter.Add(stream.Conn().RemotePeer().String(), int64(len(blockRoots)))

	for _, root := range blockRoots {
		blk, err := r.db.Block(ctx, root)
		if err != nil {
			log.WithError(err).Error("Failed to fetch block")
			resp, err := r.generateErrorResponse(responseCodeServerError, genericError)
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
		if err := r.chunkWriter(stream, blk); err != nil {
			return err
		}
	}
	return nil
}
