package sync

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	libp2pcore "github.com/libp2p/go-libp2p-core"
)

// rpcHandler is responsible for handling and responding to any incoming message.
// This method may return an error to internal monitoring, but the error will
// not be relayed to the peer.
type rpcHandler func(context.Context, proto.Message, libp2pcore.Stream) error

// TODO(3147): Delete after all handlers implemented.
func notImplementedRPCHandler(context.Context, proto.Message, libp2pcore.Stream) error {
	return errors.New("not implemented")
}
