package sync

import (
	"context"
	"io"
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

		// TODO: This should be handled by the encoding. Perhaps the encoding can accept io.Reader?
		// Read the varint prefix
		msgLen, err := readVarint(stream)
		if err != nil {
			panic(err)
			// TODO
		}
		b := make([]byte, msgLen)
		i, err := stream.Read(b)
		if err != nil {
			panic(err)
			// TODO
		}

		log.Printf("Copied %d bytes", i)
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

// TODO: Move this stuff into the encoder layer for ssz.

const maxVarintLength = 10

func readVarint(r io.Reader) (uint64, error) {
	b := make([]byte, 0, maxVarintLength)
	for i := 0; i < maxVarintLength; i++ {
		b1 := make([]byte, 1)
		n, err := r.Read(b1)
		if err != nil {
			return 0, err
		}
		if n != 1 {
			panic("n isn't 1") // TODO: do something different than panic.
		}
		b = append(b, b1[0])

		// If most signficant bit is not set, we have reached the end of the Varint.
		if b1[0]&0x80 == 0 {
			break
		}
	}

	vi, n := proto.DecodeVarint(b)
	if n != len(b) {
		panic("n != len(b)") // TODO: do something different than panic.
	}
	return vi, nil
}
