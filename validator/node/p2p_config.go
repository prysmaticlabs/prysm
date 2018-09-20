package node

import (
	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/shared/p2p"

	pb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
)

var topicMappings = map[pb.Topic]proto.Message{
	pb.Topic_COLLATION_BODY_REQUEST:  &pb.CollationBodyRequest{},
	pb.Topic_COLLATION_BODY_RESPONSE: &pb.CollationBodyResponse{},
	pb.Topic_TRANSACTIONS:            &pb.Transaction{},
}

func configureP2P() (*p2p.Server, error) {
	s, err := p2p.NewServer()
	if err != nil {
		return nil, err
	}

	// TODO(437): Define default adapters for logging, monitoring, etc.
	// TODO(438): Define default adapters for logging, monitoring, etc.
	var adapters []p2p.Adapter
	for k, v := range topicMappings {
		s.RegisterTopic(k.String(), v, adapters...)
	}

	return s, nil
}
