package sync

import (
	"context"
	"io/ioutil"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

// Time to first byte timeout. The maximum time to wait for first byte of
// request response (time-to-first-byte).
var ttfbTimeout = 5 * time.Second

func (r *RegularSync) registerRPC(topic string, base proto.Message, h rpcHandler) {
	r.p2p.SetStreamHandler(topic+"/ssz", func(stream network.Stream) {
		ctx, cancel := context.WithTimeout(r.ctx, ttfbTimeout)
		defer cancel()
		defer stream.Close()

		if err := stream.SetReadDeadline(roughtime.Now().Add(ttfbTimeout)); err != nil {
			// TODO
			return
		}
		b, err := ioutil.ReadAll(stream)
		if err != nil {
			// TODO
			return
		}
		log.Printf("stream data, len(b)=%d, %#x", len(b), b)
		n := proto.Clone(base)
		if err := r.p2p.Encoding().DecodeTo(b, n); err != nil {
			// TODO
			return
		}
		if err := h(ctx, n, stream); err != nil {
			// TODO
			return
		}
	})
}
