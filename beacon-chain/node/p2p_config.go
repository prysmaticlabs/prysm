package node

import (
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

func configureP2P() (*p2p.Server, error) {
	s, err := p2p.NewServer()
	if err != nil {
		return nil, err
	}

	// Configure adapters
	var adapters []p2p.Adapter
	type TestInterface struct {
	}

	s.RegisterTopic("test_topic", TestInterface{}, adapters...)

	return s, nil
}
