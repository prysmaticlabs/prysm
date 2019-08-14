package sync

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

// Time to first byte timeout. The maximum time to wait for first byte of
// request response (time-to-first-byte).
var ttfbTimeout = 5 * time.Second

func (r *RegularSync) registerRPC(topic string, base proto.Message, h rpcHandler) {
	r.p2p.SetStreamHandler(topic+r.p2p.Encoding().ProtocolSuffix(), func(stream network.Stream) {
		ctx, cancel := context.WithTimeout(r.ctx, ttfbTimeout)
		defer cancel()
		defer stream.Close()

		if err := stream.SetReadDeadline(roughtime.Now().Add(ttfbTimeout)); err != nil {
			// TODO
			return
		}

		n := proto.Clone(base)
		if err := r.p2p.Encoding().Decode(stream, n); err != nil {
			// TODO
			return
		}
		if err := h(ctx, n, stream); err != nil {
			// TODO
			return
		}
	})
}
