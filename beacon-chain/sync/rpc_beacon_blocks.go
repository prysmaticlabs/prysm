package sync

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// beaconBlocksRPCHandler looks up the request blocks from the database from a given start block
// and ensures that all blocks are a descendant of the message head block root.
func (r *RegularSync) beaconBlocksRPCHandler(ctx context.Context, msg proto.Message, stream libp2pcore.Stream) error {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)

	m := msg.(*pb.BeaconBlocksRequest)

	// TODO Only return canonical blocks.
	startSlot := m.HeadSlot
	endSlot := startSlot + (m.Step * m.Count)

	// TODO: Update this with reasonable constraints.
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

	blks, err := r.db.Blocks(ctx, filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot))
	if err != nil {
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
	ret := &pb.BeaconBlocksResponse{}

	for _, blk := range blks {
		if (blk.Slot-startSlot)%m.Step == 0 {
			ret.Blocks = append(ret.Blocks, blk)
		}
	}

	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		log.WithError(err).Error("Failed to write to stream")
	}
	_, err = r.p2p.Encoding().Encode(stream, ret)
	return err
}
