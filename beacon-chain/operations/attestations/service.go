package attestations

import (
	"context"

	"github.com/dgraph-io/ristretto"
)

var forkChoiceProcessedRootsSize = int64(1 << 16)

// Service of attestation pool operations.
type Service struct {
	ctx                      context.Context
	cancel                   context.CancelFunc
	pool                     Pool
	err                      error
	forkChoiceProcessedRoots *ristretto.Cache
}

// Config options for the service.
type Config struct {
	Pool Pool
}

// NewService instantiates a new attestation pool service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: forkChoiceProcessedRootsSize,
		MaxCost:     forkChoiceProcessedRootsSize,
		BufferItems: 64,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                      ctx,
		cancel:                   cancel,
		pool:                     cfg.Pool,
		forkChoiceProcessedRoots: cache,
	}, nil
}

// Start an attestation pool service's main event loop.
func (s *Service) Start() {
	go s.prepareForkChoiceAtts()
	go s.aggregateRoutine()
}

// Stop the beacon block attestation pool service's main event loop
// and associated goroutines.
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
