package p2p

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
)

// Send a message to a specific peer.
// TODO(3147): Implement.
func (s *Service) Send(ctx context.Context, message proto.Message, pid peer.ID) error {
	return nil
}
