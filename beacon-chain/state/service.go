package state

import (
	"context"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "state")

// Service defining a beacon state cache that manages the entire lifecycle of
// the beacon state in the eth2 runtime.
type Service struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// Start the service event loop.
func (s *Service) Start() {
	// TODO(Raul): Need to wait for state initialized event...
}

// Stop the service event loop.
func (s *Service) Stop() error {
	defer s.cancel()
	return nil
}

// Status reports the healthy status of the service. Returning nil means service
// is correctly running without error.
func (s *Service) Status() error {
	return nil
}
