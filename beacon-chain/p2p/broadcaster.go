package p2p

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// ErrMessageNotMapped occurs on a Broadcast attempt when a message has not been defined in the
// GossipTypeMapping.
var ErrMessageNotMapped = errors.New("message type is not mapped to a PubSub topic")

// Broadcast a message to the p2p network.
func (s *Service) Broadcast(ctx context.Context, msg proto.Message) error {
	ctx, span := trace.StartSpan(ctx, "p2p.Broadcast")
	defer span.End()

	var topic string
	switch msg.(type) {
	case *eth.Attestation:
		topic = attestationToTopic(msg.(*eth.Attestation))
	default:
		var ok bool
		topic, ok = GossipTypeMapping[reflect.TypeOf(msg)]
		if !ok {
			traceutil.AnnotateError(span, ErrMessageNotMapped)
			return ErrMessageNotMapped
		}
	}

	span.AddAttributes(trace.StringAttribute("topic", topic))

	buf := new(bytes.Buffer)
	if _, err := s.Encoding().Encode(buf, msg); err != nil {
		err := errors.Wrap(err, "could not encode message")
		traceutil.AnnotateError(span, err)
		return err
	}

	if span.IsRecordingEvents() {
		id := hashutil.FastSum64(buf.Bytes())
		messageLen := int64(buf.Len())
		span.AddMessageSendEvent(int64(id), messageLen /*uncompressed*/, messageLen /*compressed*/)
	}

	if err := s.pubsub.Publish(topic+s.Encoding().ProtocolSuffix(), buf.Bytes()); err != nil {
		err := errors.Wrap(err, "could not publish message")
		traceutil.AnnotateError(span, err)
		return err
	}
	return nil
}

const attestationSubnetTopicFormat = "/eth2/committee_index%d_beacon_attestation"

func attestationToTopic(att *eth.Attestation) string {
	if att == nil || att.Data == nil {
		return ""
	}
	return fmt.Sprintf(attestationSubnetTopicFormat, att.Data.CommitteeIndex)
}
