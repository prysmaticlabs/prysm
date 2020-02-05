package sync

import (
	"context"
	"reflect"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Time to first byte timeout. The maximum time to wait for first byte of
// request response (time-to-first-byte). The client is expected to give up if
// they don't receive the first byte within 5 seconds.
const ttfbTimeout = 5 * time.Second

// maxChunkSize would be the maximum allowed size that a request/response chunk can be.
// any size beyond that would be rejected and the corresponding stream reset. This would
// be 1048576 bytes or 1 MiB.
const maxChunkSize = 1 << 20

// rpcHandler is responsible for handling and responding to any incoming message.
// This method may return an error to internal monitoring, but the error will
// not be relayed to the peer.
type rpcHandler func(context.Context, interface{}, libp2pcore.Stream) error

// registerRPCHandlers for p2p RPC.
func (r *Service) registerRPCHandlers() {
	r.registerRPC(
		"/eth2/beacon_chain/req/status/1",
		&pb.Status{},
		r.statusRPCHandler,
	)
	r.registerRPC(
		"/eth2/beacon_chain/req/goodbye/1",
		new(uint64),
		r.goodbyeRPCHandler,
	)
	r.registerRPC(
		"/eth2/beacon_chain/req/beacon_blocks_by_range/1",
		&pb.BeaconBlocksByRangeRequest{},
		r.beaconBlocksByRangeRPCHandler,
	)
	r.registerRPC(
		"/eth2/beacon_chain/req/beacon_blocks_by_root/1",
		[][32]byte{},
		r.beaconBlocksRootRPCHandler,
	)
}

// registerRPC for a given topic with an expected protobuf message type.
func (r *Service) registerRPC(topic string, base interface{}, handle rpcHandler) {
	topic += r.p2p.Encoding().ProtocolSuffix()
	log := log.WithField("topic", topic)
	r.p2p.SetStreamHandler(topic, func(stream network.Stream) {
		ctx, cancel := context.WithTimeout(context.Background(), ttfbTimeout)
		defer cancel()
		defer stream.Close()
		ctx, span := trace.StartSpan(ctx, "sync.rpc")
		defer span.End()
		span.AddAttributes(trace.StringAttribute("topic", topic))
		span.AddAttributes(trace.StringAttribute("peer", stream.Conn().RemotePeer().Pretty()))
		log := log.WithField("peer", stream.Conn().RemotePeer().Pretty())

		if err := stream.SetReadDeadline(roughtime.Now().Add(ttfbTimeout)); err != nil {
			log.WithError(err).Error("Could not set stream read deadline")
			return
		}

		// Increment message received counter.
		messageReceivedCounter.WithLabelValues(topic).Inc()

		// Given we have an input argument that can be pointer or [][32]byte, this gives us
		// a way to check for its reflect.Kind and based on the result, we can decode
		// accordingly.
		t := reflect.TypeOf(base)
		if t.Kind() == reflect.Ptr {
			msg := reflect.New(t.Elem())
			if err := r.p2p.Encoding().DecodeWithLength(stream, msg.Interface()); err != nil {
				log.WithError(err).Warn("Failed to decode stream message")
				traceutil.AnnotateError(span, err)
				return
			}
			if err := handle(ctx, msg.Interface(), stream); err != nil {
				messageFailedProcessingCounter.WithLabelValues(topic).Inc()
				if err != errWrongForkVersion {
					log.WithError(err).Warn("Failed to handle p2p RPC")
				}
				traceutil.AnnotateError(span, err)
			}
		} else {
			msg := reflect.New(t)
			if err := r.p2p.Encoding().DecodeWithLength(stream, msg.Interface()); err != nil {
				log.WithError(err).Warn("Failed to decode stream message")
				traceutil.AnnotateError(span, err)
				return
			}
			if err := handle(ctx, msg.Elem().Interface(), stream); err != nil {
				messageFailedProcessingCounter.WithLabelValues(topic).Inc()
				if err != errWrongForkVersion {
					log.WithError(err).Warn("Failed to handle p2p RPC")
				}
				traceutil.AnnotateError(span, err)
			}
		}

	})
}
