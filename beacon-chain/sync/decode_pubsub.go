package sync

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
)

func (s *Service) decodePubsubMessage(msg *pubsub.Message) (proto.Message, error) {
	if msg == nil || msg.TopicIDs == nil || len(msg.TopicIDs) == 0 {
		return nil, errors.New("nil pubsub message")
	}
	topic := msg.TopicIDs[0]
	topic = strings.TrimSuffix(topic, s.p2p.Encoding().ProtocolSuffix())
	topic = s.replaceForkDigest(topic)
	base, ok := p2p.GossipTopicMappings[topic]
	if !ok {
		return nil, fmt.Errorf("no message mapped for topic %s", topic)
	}
	m := proto.Clone(base)
	if err := s.p2p.Encoding().DecodeGossip(msg.Data, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Replaces our fork digest with the formatter.
func (s *Service) replaceForkDigest(topic string) string {
	subStrings := strings.Split(topic, "/")
	subStrings[2] = "%x"
	return strings.Join(subStrings, "/")
}
