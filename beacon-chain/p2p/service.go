package p2p

import (
	"context"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared"
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
