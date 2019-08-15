package sync

import (
	"bytes"
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	libp2pcore "github.com/libp2p/go-libp2p-core"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var errWrongForkVersion = errors.New("wrong fork version")

// forkVersion is a 4 byte value representing what network this client is running. The current implementation
// uses the first 4 bytes of the contract address.
func forkVersion() []byte {
	// TODO
	return []byte("foo")
}

func (r *RegularSync) helloRPCHandler(ctx context.Context, msg proto.Message, stream libp2pcore.Stream) error {
	m := msg.(*pb.Hello)

	if !bytes.Equal(forkVersion(), m.ForkVersion) {
		if err := r.p2p.Disconnect(stream.Conn().RemotePeer()); err != nil {
			log.WithError(err).Error("Failed to disconnect from peer")
		}
		return errWrongForkVersion
	}

	return nil
}
