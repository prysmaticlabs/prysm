package sync

import (
	"context"
	"reflect"
	"strings"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
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
	topic := baseTopic + s.cfg.P2P.Encoding().ProtocolSuffix()
	log := log.WithField("topic", topic)
	s.cfg.P2P.SetStreamHandler(topic, func(stream network.Stream) {
		ctx, cancel := context.WithTimeout(s.ctx, ttfbTimeout)
		defer cancel()

		// Resetting after closing is a no-op so defer a reset in case something goes wrong.
		// It's up to the handler to Close the stream (send an EOF) if
		// it successfully writes a response. We don't blindly call
		// Close here because we may have only written a partial
		// response.
		defer func() {
			_err := stream.Reset()
			_ = _err
		}()

		ctx, span := trace.StartSpan(ctx, "sync.rpc")
		defer span.End()
		span.AddAttributes(trace.StringAttribute("topic", topic))
		span.AddAttributes(trace.StringAttribute("peer", stream.Conn().RemotePeer().Pretty()))
		log := log.WithField("peer", stream.Conn().RemotePeer().Pretty()).WithField("topic", string(stream.Protocol()))

		// Check before hand that peer is valid.
		if s.cfg.P2P.Peers().IsBad(stream.Conn().RemotePeer()) {
			if err := s.sendGoodByeAndDisconnect(ctx, p2ptypes.GoodbyeCodeBanned, stream.Conn().RemotePeer()); err != nil {
				log.Debugf("Could not disconnect from peer: %v", err)
			}
			return
		}
		// Validate request according to peer limits.
		if err := s.rateLimiter.validateRawRpcRequest(stream); err != nil {
			log.Debugf("Could not validate rpc request from peer: %v", err)
			return
		}
		s.rateLimiter.addRawStream(stream)

		if err := stream.SetReadDeadline(timeutils.Now().Add(ttfbTimeout)); err != nil {
			log.WithError(err).Debug("Could not set stream read deadline")
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
				if err != p2ptypes.ErrWrongForkDigestVersion {
					log.WithError(err).Debug("Could not handle p2p RPC")
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
			if err := s.cfg.P2P.Encoding().DecodeWithMaxLength(stream, msg.Interface()); err != nil {
				// Debug logs for goodbye/status errors
				if strings.Contains(topic, p2p.RPCGoodByeTopic) || strings.Contains(topic, p2p.RPCStatusTopic) {
					log.WithError(err).Debug("Could not decode goodbye stream message")
					traceutil.AnnotateError(span, err)
					return
				}
				log.WithError(err).Debug("Could not decode stream message")
				traceutil.AnnotateError(span, err)
				return
			}
			if err := handle(ctx, msg.Interface(), stream); err != nil {
				messageFailedProcessingCounter.WithLabelValues(topic).Inc()
				if err != p2ptypes.ErrWrongForkDigestVersion {
					log.WithError(err).Debug("Could not handle p2p RPC")
				}
				traceutil.AnnotateError(span, err)
			}
		} else {
			msg := reflect.New(t)
			if err := s.cfg.P2P.Encoding().DecodeWithMaxLength(stream, msg.Interface()); err != nil {
				log.WithError(err).Debug("Could not decode stream message")
				traceutil.AnnotateError(span, err)
				return
			}
			if err := handle(ctx, msg.Elem().Interface(), stream); err != nil {
				messageFailedProcessingCounter.WithLabelValues(topic).Inc()
				if err != p2ptypes.ErrWrongForkDigestVersion {
					log.WithError(err).Debug("Could not handle p2p RPC")
				}
				traceutil.AnnotateError(span, err)
			}
		}
	})
}
