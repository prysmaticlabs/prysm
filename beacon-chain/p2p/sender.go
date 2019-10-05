package p2p

import (
	"context"
	"reflect"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Send a message to a specific peer. The returned stream may be used for reading, but has been
// closed for writing.
func (s *Service) Send(ctx context.Context, message interface{}, pid peer.ID) (network.Stream, error) {
	ctx, span := trace.StartSpan(ctx, "p2p.Send")
	defer span.End()
	topic := RPCTypeMapping[reflect.TypeOf(message)] + s.Encoding().ProtocolSuffix()
	span.AddAttributes(trace.StringAttribute("topic", topic))

	// TTFB_TIME (5s) + RESP_TIMEOUT (10s).
	const deadline = 15 * time.Second
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	stream, err := s.host.NewStream(ctx, pid, protocol.ID(topic))
	if err != nil {
		traceutil.AnnotateError(span, err)
		return nil, err
	}
	if err := stream.SetReadDeadline(time.Now().Add(deadline)); err != nil {
		traceutil.AnnotateError(span, err)
		return nil, err
	}
	if err := stream.SetWriteDeadline(time.Now().Add(deadline)); err != nil {
		traceutil.AnnotateError(span, err)
		return nil, err
	}
	if _, err := s.Encoding().EncodeWithLength(stream, message); err != nil {
		traceutil.AnnotateError(span, err)
		return nil, err
	}

	// Close stream for writing.
	if err := stream.Close(); err != nil {
		traceutil.AnnotateError(span, err)
		return nil, err
	}

	return stream, nil
}
