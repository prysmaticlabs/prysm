package sync

import (
	"context"
	"runtime/debug"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Pipeline decodes the incoming subscription data, runs the validation, and handles the
// message.
type pipeline struct {
	ctx         context.Context
	topic       string
	base        proto.Message
	validate    validator
	handle      subHandler
	encoding    encoder.NetworkEncoding
	self        peer.ID
	sub         *pubsub.Subscription
	broadcaster p2p.Broadcaster
}

func (p *pipeline) process(data []byte, fromSelf bool) {
	ctx, _ := context.WithTimeout(context.Background(), pubsubMessageTimeout)
	ctx, span := trace.StartSpan(ctx, "sync.pubsub")
	defer span.End()

	log := log.WithField("topic", p.topic)

	defer func() {
		if r := recover(); r != nil {
			traceutil.AnnotateError(span, fmt.Errorf("panic occurred: %v", r))
			log.WithField("error", r).Error("Panic occurred")
			debug.PrintStack()
		}
	}()

	span.AddAttributes(trace.StringAttribute("topic", p.topic))
	span.AddAttributes(trace.BoolAttribute("fromSelf", fromSelf))

	if data == nil {
		log.Warn("Received nil message on pubsub")
		return
	}

	if span.IsRecordingEvents() {
		id := hashutil.FastSum64(data)
		messageLen := int64(len(data))
		span.AddMessageReceiveEvent(int64(id), messageLen /*uncompressed*/, messageLen /*compressed*/)
	}

	msg := proto.Clone(p.base)
	if err := p.encoding.Decode(data, msg); err != nil {
		traceutil.AnnotateError(span, err)
		log.WithError(err).Warn("Failed to decode pubsub message")
		return
	}

	valid, err := p.validate(ctx, msg, p.broadcaster, fromSelf)
	if err != nil {
		if !fromSelf {
			log.WithError(err).Error("Message failed to verify")
			messageFailedValidationCounter.WithLabelValues(p.topic).Inc()
		}
		return
	}
	if !valid {
		return
	}

	if err := p.handle(ctx, msg); err != nil {
		traceutil.AnnotateError(span, err)
		log.WithError(err).Error("Failed to handle p2p pubsub")
		messageFailedProcessingCounter.WithLabelValues(p.topic).Inc()
		return
	}
}

func (p *pipeline) messageLoop() {
	log := log.WithField("topic", p.topic)

	for {
		msg, err := p.sub.Next(p.ctx)
		if err != nil && err.Error() != "subscription cancelled by calling sub.Cancel()" {
			log.WithError(err).Error("Subscription next failed")
			return
		}
		// Special validation occurs on messages received from ourselves.
		fromSelf := msg.GetFrom() == p.self

		messageReceivedCounter.WithLabelValues(p.topic + p.encoding.ProtocolSuffix()).Inc()

		go p.process(msg.Data, fromSelf)
	}
}