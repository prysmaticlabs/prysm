package sync

import (
	"bytes"
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	libp2pcore "github.com/libp2p/go-libp2p-core"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

	"github.com/prysmaticlabs/prysm/shared/params"
)

const genericError = "internal service error"

func (r *RegularSync) helloRPCHandler(ctx context.Context, msg proto.Message, stream libp2pcore.Stream) error {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)

	m := msg.(*pb.Hello)

	if !bytes.Equal(params.BeaconConfig().GenesisForkVersion, m.ForkVersion) {
		resp, err := r.generateErrorResponse(responseCodeInvalidRequest, errWrongForkVersion.Error())
		if err != nil {
			log.WithError(err).Error("Failed to generate a response error")
		} else {
			if _, err := stream.Write(resp); err != nil {
				log.WithError(err).Errorf("Failed to write to whatever")
			}
		}
		stream.Close()
		// Add a short delay to allow the stream to flush before closing the connection.
		// There is still a chance that the peer won't receive the message.
		time.Sleep(50 * time.Millisecond)
		if err := r.p2p.Disconnect(stream.Conn().RemotePeer()); err != nil {
			log.WithError(err).Error("Failed to disconnect from peer")
		}
		return errWrongForkVersion
	}

	state, err := r.db.HeadState(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get head state")
		resp, err := r.generateErrorResponse(responseCodeServerError, genericError)
		stream.Write(resp)
		return err
	}

	resp := &pb.Hello{
		ForkVersion:    params.BeaconConfig().GenesisForkVersion,
		FinalizedRoot:  state.FinalizedCheckpoint.Root,
		FinalizedEpoch: state.FinalizedCheckpoint.Epoch,
		HeadRoot:       state.BlockRoots[state.Slot%params.BeaconConfig().SlotsPerHistoricalRoot],
		HeadSlot:       state.Slot,
	}

	stream.Write([]byte{responseCodeSuccess})
	_, err = r.p2p.Encoding().Encode(stream, resp)

	return err
}
