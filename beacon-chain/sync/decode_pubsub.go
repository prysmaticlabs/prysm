package sync

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
)

func (r *Service) decodePubsubMessage(msg *pubsub.Message) (proto.Message, error) {
	if msg == nil || msg.TopicIDs == nil || len(msg.TopicIDs) == 0 {
		return nil, errors.New("nil pubsub message")
	}
	topic := msg.TopicIDs[0]
	topic = strings.TrimSuffix(topic, r.p2p.Encoding().ProtocolSuffix())
	base, ok := p2p.GossipTopicMappings[topic]
	if !ok {
		return nil, fmt.Errorf("no message mapped for topic %s", topic)
	}
	m := proto.Clone(base)
	if err := r.p2p.Encoding().Decode(msg.Data, m); err != nil {
		return nil, err
	}
	return m, nil
}
