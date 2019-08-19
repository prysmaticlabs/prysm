package p2p

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	network "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	"github.com/prysmaticlabs/prysm/shared"
	deprecatedp2p "github.com/prysmaticlabs/prysm/shared/deprecated-p2p"
	"github.com/prysmaticlabs/prysm/shared/event"
)

var _ = shared.Service(&Service{})

// Service for managing peer to peer (p2p) networking.
type Service struct {
	ctx    context.Context
	cancel context.CancelFunc

	started    bool
	cfg        *Config
	startupErr error

	host   host.Host
	pubsub *pubsub.PubSub
}

// NewService initializes a new p2p service compatible with shared.Service interface. No
// connections are made until the Start function is called during the service registry startup.
func NewService(cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
	}, nil
}

// Start the p2p service.
func (s *Service) Start() {
	if s.started {
		log.Error("Attempted to start p2p service when it was already started")
		return
	}
	s.started = true

	// TODO(3147): Add host options
	h, err := libp2p.New(s.ctx)
	if err != nil {
		s.startupErr = err
		return
	}
	s.host = h

	// TODO(3147): Add gossip sub options
	gs, err := pubsub.NewGossipSub(s.ctx, s.host)
	if err != nil {
		s.startupErr = err
		return
	}
	s.pubsub = gs
}

// Stop the p2p service and terminate all peer connections.
func (s *Service) Stop() error {
	s.started = false
	return nil
}

// Status of the p2p service. Will return an error if the service is considered unhealthy to
// indicate that this node should not serve traffic until the issue has been resolved.
func (s *Service) Status() error {
	if !s.started {
		return errors.New("not running")
	}
	return nil
}

// Encoding returns the configured networking encoding.
func (s *Service) Encoding() encoder.NetworkEncoding {
	// TODO(3147): Return based on flag value
	return &encoder.SszNetworkEncoder{}
}

// PubSub returns the p2p pubsub framework.
func (s *Service) PubSub() *pubsub.PubSub {
	return s.pubsub
}

// SetStreamHandler sets the protocol handler on the p2p host multiplexer.
// This method is a pass through to libp2pcore.Host.SetStreamHandler.
func (s *Service) SetStreamHandler(topic string, handler network.StreamHandler) {
	s.host.SetStreamHandler(protocol.ID(topic), handler)
}

// Disconnect from a peer.
func (s *Service) Disconnect(pid peer.ID) error {
	// TODO(3147): Implement disconnect
	return nil
}

// Subscribe to some topic.
// TODO(3147): Remove
// DEPRECATED: Do not use.
func (s *Service) Subscribe(_ proto.Message, _ chan deprecatedp2p.Message) event.Subscription {
	return nil
}
