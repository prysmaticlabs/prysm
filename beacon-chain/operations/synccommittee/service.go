package synccommittee

import (
	"context"
)

// Service of sync committee object store operations.
type Service struct {
	store       *Store
	ctx         context.Context
	cancel      context.CancelFunc
	err         error
	genesisTime uint64
}

// NewService instantiates a new sync committee object store service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, s *Store) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		store:  s,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start the sync committee store service's main event loop.
func (s *Service) Start() {
	go s.pruneSyncCommitteeStore()
}

// Stop the sync committee store service.
func (s *Service) Stop() error {
	defer s.cancel()
	return nil
}

// Status returns the current service err if there's any.
func (s *Service) Status() error {
	if s.err != nil {
		return s.err
	}
	return nil
}

// SetGenesisTime sets genesis time for service to use.
func (s *Service) SetGenesisTime(t uint64) {
	s.genesisTime = t
}
