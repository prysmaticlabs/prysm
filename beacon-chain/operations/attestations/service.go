package attestations

import (
	"context"
)

// Service represents a service that handles the internal
// logic of attestation pool operations
type Service struct {
	ctx    context.Context
	cancel context.CancelFunc
	pool   Pool
	error  error
}

// Config options for the service.
type Config struct {
	Pool Pool
}

// NewService instantiates a new attestation pool service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start an attestation pool service's main event loop.
func (s *Service) Start() {
}

// Stop the beacon block attestation pool service's main event loop
// and associated goroutines.
func (s *Service) Stop() error {
	defer s.cancel()
	return nil
}

// Status returns the current service error if there's any.
func (s *Service) Status() error {
	if s.error != nil {
		return s.error
	}
	return nil
}
