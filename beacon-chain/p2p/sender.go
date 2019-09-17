package p2p

import (
	"context"
	"reflect"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// Send a message to a specific peer. The returned stream may be used for reading, but has been
// closed for writing.
func (s *Service) Send(ctx context.Context, message interface{}, pid peer.ID) (network.Stream, error) {
	topic := RPCTypeMapping[reflect.TypeOf(message)] + s.Encoding().ProtocolSuffix()

	// TTFB_TIME (5s) + RESP_TIMEOUT (10s).
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	stream, err := s.host.NewStream(ctx, pid, protocol.ID(topic))
	if err != nil {
		return nil, err
	}

	if _, err := s.Encoding().EncodeWithLength(stream, message); err != nil {
		return nil, err
	}

	// Close stream for writing.
	if err := stream.Close(); err != nil {
		return nil, err
	}

	return stream, nil
}
