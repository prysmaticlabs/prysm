package sync

import (
	"context"
	"reflect"
	"strings"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/mux"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Time to first byte timeout. The maximum time to wait for first byte of
// request response (time-to-first-byte). The client is expected to give up if
// they don't receive the first byte within 5 seconds.
var ttfbTimeout = params.BeaconNetworkConfig().TtfbTimeout

// respTimeout is the maximum time for complete response transfer.
var respTimeout = params.BeaconNetworkConfig().RespTimeout

// rpcHandler is responsible for handling and responding to any incoming message.
// This method may return an error to internal monitoring, but the error will
// not be relayed to the peer.
type rpcHandler func(context.Context, interface{}, libp2pcore.Stream) error

// registerRPCHandlers for p2p RPC.
func (s *Service) registerRPCHandlers() {
	s.registerRPC(
		p2p.RPCStatusTopic,
		s.statusRPCHandler,
	)
	s.registerRPC(
		p2p.RPCGoodByeTopic,
		s.goodbyeRPCHandler,
	)
	s.registerRPC(
		p2p.RPCBlocksByRangeTopic,
		s.beaconBlocksByRangeRPCHandler,
	)
	s.registerRPC(
		p2p.RPCBlocksByRootTopic,
		s.beaconBlocksRootRPCHandler,
	)
	s.registerRPC(
		p2p.RPCPingTopic,
		s.pingHandler,
	)
	s.registerRPC(
		p2p.RPCMetaDataTopic,
		s.metaDataHandler,
	)
}

// registerRPC for a given topic with an expected protobuf message type.
func (s *Service) registerRPC(baseTopic string, handle rpcHandler) {
	topic := baseTopic + s.p2p.Encoding().ProtocolSuffix()
	log := log.WithField("topic", topic)
	s.p2p.SetStreamHandler(topic, func(stream network.Stream) {
		ctx, cancel := context.WithTimeout(context.Background(), ttfbTimeout)
		defer cancel()
		defer func() {
			if err := helpers.FullClose(stream); err != nil && err.Error() != mux.ErrReset.Error() {
				log.WithError(err).Debug("Failed to reset stream")
			}
		}()
		ctx, span := trace.StartSpan(ctx, "sync.rpc")
		defer span.End()
		span.AddAttributes(trace.StringAttribute("topic", topic))
		span.AddAttributes(trace.StringAttribute("peer", stream.Conn().RemotePeer().Pretty()))
		log := log.WithField("peer", stream.Conn().RemotePeer().Pretty())

		if err := stream.SetReadDeadline(roughtime.Now().Add(ttfbTimeout)); err != nil {
			log.WithError(err).Error("Could not set stream read deadline")
			return
		}

		base, ok := p2p.RPCTopicMappings[baseTopic]
		if !ok {
			log.Errorf("Could not retrieve base message for topic %s", baseTopic)
			return
		}
		t := reflect.TypeOf(base)
		// Copy Base
		base = reflect.New(t)

		// Increment message received counter.
		messageReceivedCounter.WithLabelValues(topic).Inc()

		// since metadata requests do not have any data in the payload, we
		// do not decode anything.
		if baseTopic == p2p.RPCMetaDataTopic {
			if err := handle(ctx, base, stream); err != nil {
				messageFailedProcessingCounter.WithLabelValues(topic).Inc()
				if err != errWrongForkDigestVersion {
					log.WithError(err).Warn("Failed to handle p2p RPC")
				}
				traceutil.AnnotateError(span, err)
			}
			return
		}

		// Given we have an input argument that can be pointer or the actual object, this gives us
		// a way to check for its reflect.Kind and based on the result, we can decode
		// accordingly.
		if t.Kind() == reflect.Ptr {
			msg := reflect.New(t.Elem())
			if err := s.p2p.Encoding().DecodeWithMaxLength(stream, msg.Interface()); err != nil {
				// Debug logs for goodbye/status errors
				if strings.Contains(topic, p2p.RPCGoodByeTopic) || strings.Contains(topic, p2p.RPCStatusTopic) {
					log.WithError(err).Debug("Failed to decode goodbye stream message")
					traceutil.AnnotateError(span, err)
					return
				}
				log.WithError(err).Warn("Failed to decode stream message")
				traceutil.AnnotateError(span, err)
				return
			}
			if err := handle(ctx, msg.Interface(), stream); err != nil {
				messageFailedProcessingCounter.WithLabelValues(topic).Inc()
				if err != errWrongForkDigestVersion {
					log.WithError(err).Warn("Failed to handle p2p RPC")
				}
				traceutil.AnnotateError(span, err)
			}
		} else {
			msg := reflect.New(t)
			if err := s.p2p.Encoding().DecodeWithMaxLength(stream, msg.Interface()); err != nil {
				log.WithError(err).Warn("Failed to decode stream message")
				traceutil.AnnotateError(span, err)
				return
			}
			if err := handle(ctx, msg.Elem().Interface(), stream); err != nil {
				messageFailedProcessingCounter.WithLabelValues(topic).Inc()
				if err != errWrongForkDigestVersion {
					log.WithError(err).Warn("Failed to handle p2p RPC")
				}
				traceutil.AnnotateError(span, err)
			}
		}

	})
}
