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

// helloRPCHandler reads the incoming Hello RPC from the peer and responds with our version of a hello message.
// This handler will disconnect any peer that does not match our fork version.
func (r *RegularSync) helloRPCHandler(ctx context.Context, msg proto.Message, stream libp2pcore.Stream) error {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)

	log := log.WithField("rpc", "hello")
	m := msg.(*pb.Hello)

	if !bytes.Equal(params.BeaconConfig().GenesisForkVersion, m.ForkVersion) {
		resp, err := r.generateErrorResponse(responseCodeInvalidRequest, errWrongForkVersion.Error())
		if err != nil {
			log.WithError(err).Error("Failed to generate a response error")
		} else {
			if _, err := stream.Write(resp); err != nil {
				log.WithError(err).Errorf("Failed to write to stream")
			}
		}
		stream.Close() // Close before disconnecting.
		// Add a short delay to allow the stream to flush before closing the connection.
		// There is still a chance that the peer won't receive the message.
		time.Sleep(50 * time.Millisecond)
		if err := r.p2p.Disconnect(stream.Conn().RemotePeer()); err != nil {
			log.WithError(err).Error("Failed to disconnect from peer")
		}
		return errWrongForkVersion
	}

	r.p2p.AddHandshake(stream.Conn().RemotePeer(), m)

	resp := &pb.Hello{
		ForkVersion:    params.BeaconConfig().GenesisForkVersion,
		FinalizedRoot:  r.chain.FinalizedCheckpt().Root,
		FinalizedEpoch: r.chain.FinalizedCheckpt().Epoch,
		HeadRoot:       r.chain.HeadRoot(),
		HeadSlot:       r.chain.HeadSlot(),
	}

	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		log.WithError(err).Error("Failed to write to stream")
	}
	_, err := r.p2p.Encoding().Encode(stream, resp)

	return err
}
