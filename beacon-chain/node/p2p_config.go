package node

import (
	"github.com/prysmaticlabs/prysm/shared/p2p"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var topicMappings = map[pb.Topic]interface{}{
	pb.Topic_BEACON_BLOCK_HASH_ANNOUNCE:          pb.BeaconBlockHashAnnounce{},
	pb.Topic_BEACON_BLOCK_REQUEST:                pb.BeaconBlockRequest{},
	pb.Topic_BEACON_BLOCK_REQUEST_BY_SLOT_NUMBER: pb.BeaconBlockRequestBySlotNumber{},
	pb.Topic_BEACON_BLOCK_RESPONSE:               pb.BeaconBlockResponse{},
	pb.Topic_CRYSTALLIZED_STATE_HASH_ANNOUNCE:    pb.CrystallizedStateHashAnnounce{},
	pb.Topic_CRYSTALLIZED_STATE_REQUEST:          pb.CrystallizedStateRequest{},
	pb.Topic_CRYSTALLIZED_STATE_RESPONSE:         pb.CrystallizedStateResponse{},
	pb.Topic_ACTIVE_STATE_HASH_ANNOUNCE:          pb.ActiveStateHashAnnounce{},
	pb.Topic_ACTIVE_STATE_REQUEST:                pb.ActiveStateRequest{},
	pb.Topic_ACTIVE_STATE_RESPONSE:               pb.ActiveStateResponse{},
}

func configureP2P() (*p2p.Server, error) {
	s, err := p2p.NewServer()
	if err != nil {
		return nil, err
	}

	// TODO(437, 438): Define default adapters for logging, monitoring, etc.
	var adapters []p2p.Adapter
	for k, v := range topicMappings {
		s.RegisterTopic(k.String(), v, adapters...)
	}

	return s, nil
}
