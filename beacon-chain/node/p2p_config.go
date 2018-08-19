package node

import (
	"github.com/prysmaticlabs/prysm/shared/p2p"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func configureP2P() (*p2p.Server, error) {
	s, err := p2p.NewServer()
	if err != nil {
		return nil, err
	}

	var adapters []p2p.Adapter

	s.RegisterTopic(v1.Topic_BEACON_BLOCK_HASH_ANNOUNCE.String(), pb.BeaconBlockHashAnnounce{}, adapters...)

	return s, nil
}