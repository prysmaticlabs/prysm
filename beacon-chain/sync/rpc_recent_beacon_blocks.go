package sync

import (
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// sendRecentBeaconBlocksRequest sends a recent beacon blocks request to a peer to get
// those corresponding blocks from that peer.
func (r *RegularSync) sendRecentBeaconBlocksRequest(ctx context.Context, blockRoots [][32]byte, id peer.ID) error {
	log := log.WithField("rpc", "recent_beacon_blocks")

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	stream, err := r.p2p.Send(ctx, blockRoots, id)
	if err != nil {
		return err
	}

	code, errMsg, err := ReadStatusCode(stream, r.p2p.Encoding())
	if err != nil {
		return err
	}

	if code != 0 {
		return errors.New(errMsg)
	}

	resp := make([]*eth.BeaconBlock, 0)
	if err := r.p2p.Encoding().DecodeWithLength(stream, &resp); err != nil {
		return err
	}
	for _, blk := range resp {
		if err := r.chain.ReceiveBlock(ctx, blk); err != nil {
			log.WithError(err).Error("Unable to process block")
			return nil
		}
	}

	return nil
}

// recentBeaconBlocksRPCHandler looks up the request blocks from the database from the given block roots.
func (r *RegularSync) recentBeaconBlocksRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)
	log := log.WithField("handler", "recent_beacon_blocks")

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
	ret := make([]*eth.BeaconBlock, 0)
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
		// if block returned is nil, it appends nil to the slice
		ret = append(ret, blk)
	}

	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		log.WithError(err).Error("Failed to write to stream")
	}
	_, err := r.p2p.Encoding().EncodeWithLength(stream, ret)
	return err
}
