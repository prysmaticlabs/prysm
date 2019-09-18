package sync

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"go.opencensus.io/trace"
)

// Time to first byte timeout. The maximum time to wait for first byte of
// request response (time-to-first-byte). The client is expected to give up if
// they don't receive the first byte within 5 seconds.
var ttfbTimeout = 5 * time.Second

// rpcHandler is responsible for handling and responding to any incoming message.
// This method may return an error to internal monitoring, but the error will
// not be relayed to the peer.
type rpcHandler func(context.Context, interface{}, libp2pcore.Stream) error

// registerRPCHandlers for p2p RPC.
func (r *RegularSync) registerRPCHandlers() {
	r.registerRPC(
		"/eth2/beacon_chain/req/hello/1",
		&pb.Hello{},
		r.statusRPCHandler,
	)
	r.registerRPC(
		"/eth2/beacon_chain/req/goodbye/1",
		&pb.Goodbye{},
		r.goodbyeRPCHandler,
	)
	r.registerRPC(
		"/eth2/beacon_chain/req/beacon_blocks_by_range/1",
		&pb.BeaconBlocksRequest{},
		r.beaconBlocksByRangeRPCHandler,
	)
	r.registerRPC(
		"/eth2/beacon_chain/req/beacon_blocks_by_root/1",
		[][32]byte{},
		r.beaconBlocksRootRPCHandler,
	)
}

// registerRPC for a given topic with an expected protobuf message type.
func (r *RegularSync) registerRPC(topic string, base interface{}, handle rpcHandler) {
	topic += r.p2p.Encoding().ProtocolSuffix()
	log := log.WithField("topic", topic)
	r.p2p.SetStreamHandler(topic, func(stream network.Stream) {
		ctx, cancel := context.WithTimeout(context.Background(), ttfbTimeout)
		defer cancel()
		defer stream.Close()
		ctx, span := trace.StartSpan(ctx, "sync.rpc")
		defer span.End()
		span.AddAttributes(trace.StringAttribute("topic", topic))

		if err := stream.SetReadDeadline(roughtime.Now().Add(ttfbTimeout)); err != nil {
			log.WithError(err).Error("Could not set stream read deadline")
			return
		}

		// Clone the base message type so we have a newly initialized message as the decoding
		// destination.
		msg := proto.Clone(base)
		if err := r.p2p.Encoding().DecodeWithLength(stream, msg); err != nil {
			log.WithError(err).Error("Failed to decode stream message")
			return
		}
		if err := handle(ctx, msg, stream); err != nil {
			// TODO(3147): Update metrics
			log.WithError(err).Error("Failed to handle p2p RPC")
		}
	})
}
