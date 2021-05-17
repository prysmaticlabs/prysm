package sync

import (
	"errors"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
)

func (s *Service) rpcContext(stream network.Stream) ([]byte, error) {
	_, _, version, err := p2p.TopicDeconstructor(string(stream.Protocol()))
	if err != nil {
		return nil, err
	}
	switch version {
	case p2p.SchemaVersionV1:
		// Return empty context for a v1 method.
		return []byte{}, nil
	default:
		return nil, errors.New("invalid version of %s registered for topic: %s")
	}
}
