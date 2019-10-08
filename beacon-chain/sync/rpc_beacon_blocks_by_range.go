package sync

import (
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// beaconBlocksByRangeRPCHandler looks up the request blocks from the database from a given start block.
func (r *RegularSync) beaconBlocksByRangeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)
	log := log.WithField("handler", "beacon_blocks_by_range")

	m := msg.(*pb.BeaconBlocksByRangeRequest)

	startSlot := m.StartSlot
	endSlot := startSlot + (m.Step * m.Count)

	// TODO(3147): Update this with reasonable constraints.
	if endSlot-startSlot > 1000 || m.Step == 0 {
		resp, err := r.generateErrorResponse(responseCodeInvalidRequest, "invalid range or step")
		if err != nil {
			log.WithError(err).Error("Failed to generate a response error")
		} else {
			if _, err := stream.Write(resp); err != nil {
				log.WithError(err).Errorf("Failed to write to stream")
			}
		}
		return errors.New("invalid range or step")
	}

	// TODO(3147): Only return blocks on the chain of the head root.
	blks, err := r.db.Blocks(ctx, filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot))
	if err != nil {
		log.WithError(err).Error("Failed to retrieve blocks")
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

	for _, blk := range blks {
		if blk != nil && (blk.Slot-startSlot)%m.Step == 0 {
			if err := r.chunkWriter(stream, blk); err != nil {
				log.WithError(err).Error("Failed to send a chunked response")
				return err
			}
		}
	}
	return err
}
