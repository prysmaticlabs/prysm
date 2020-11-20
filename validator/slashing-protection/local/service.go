package local

import (
	"context"

	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "local-slashing-protection")

// Service to manage validator slashing protection. Local slashing
// protection is mandatory at runtime.
type Service struct {
	ctx         context.Context
	cancel      context.CancelFunc
	validatorDB db.Database
}

// Config for the slashing protection service.
type Config struct {
	ValidatorDB db.Database
}

// NewService creates a new validator service for the service registry.
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	srv := &Service{
		ctx:         ctx,
		cancel:      cancel,
		validatorDB: cfg.ValidatorDB,
	}
	return srv, nil
}

// Start the slashing protection service.
func (s *Service) Start() {
}

// Stop the slashing protection service.
func (s *Service) Stop() error {
	s.cancel()
	return nil
}

// Status of the slashing protection service.
func (s *Service) Status() error {
	return nil
}
