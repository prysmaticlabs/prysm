package p2p

import (
	"bytes"
	"reflect"

	"github.com/gogo/protobuf/proto"
)

// TODO: I think broadcast should return an error

// Broadcast a message to the p2p network.
func (s *Service) Broadcast(msg proto.Message) {
	topic, ok := GossipTypeMapping[reflect.TypeOf(msg)]
	if !ok {
		// TODO: complain
		panic("msg is not a registered topic")
	}

	buf := new(bytes.Buffer)
	if _, err := s.Encoding().Encode(buf, msg); err != nil {
		// TODO: complain
		panic(err)
	}

	log.Infof("Publishing to topic %s", topic+s.Encoding().ProtocolSuffix())
	if err := s.pubsub.Publish(topic+s.Encoding().ProtocolSuffix(), buf.Bytes()); err != nil {
		// TODO: complain
		panic(err)
	}
}
