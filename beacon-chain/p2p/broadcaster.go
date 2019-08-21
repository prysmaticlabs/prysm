package p2p

import (
	"bytes"
	"context"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

// ErrMessageNotMapped occurs on a Broadcast attempt when a message has not been defined in the
// GossipTypeMapping.
var ErrMessageNotMapped = errors.New("message type is not mapped to a PubSub topic")

// Broadcast a message to the p2p network.
func (s *Service) Broadcast(ctx context.Context, msg proto.Message) error {
	topic, ok := GossipTypeMapping[reflect.TypeOf(msg)]
	if !ok {
		return ErrMessageNotMapped
	}

	buf := new(bytes.Buffer)
	if _, err := s.Encoding().Encode(buf, msg); err != nil {
		return errors.Wrap(err, "could not encode message")
	}

	if err := s.pubsub.Publish(topic+s.Encoding().ProtocolSuffix(), buf.Bytes()); err != nil {
		return errors.Wrap(err, "could not publish message")
	}
	return nil
}
