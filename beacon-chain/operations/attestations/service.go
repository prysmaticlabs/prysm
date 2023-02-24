// Package attestations defines an attestation pool
// service implementation which is used to manage the lifecycle
// of aggregated, unaggregated, and fork-choice attestations.
package attestations

import (
	"context"
	"time"

	lru "github.com/hashicorp/golang-lru"
	lruwrpr "github.com/prysmaticlabs/prysm/v3/cache/lru"
	"github.com/prysmaticlabs/prysm/v3/config/params"
)

var forkChoiceProcessedRootsSize = 1 << 16

// Service of attestation pool operations.
type Service struct {
	cfg                      *Config
	ctx                      context.Context
	cancel                   context.CancelFunc
	err                      error
	forkChoiceProcessedRoots *lru.Cache
	genesisTime              uint64
}

// Config options for the service.
type Config struct {
	Pool          Pool
	pruneInterval time.Duration
}

// NewService instantiates a new attestation pool service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	cache := lruwrpr.New(forkChoiceProcessedRootsSize)

	if cfg.pruneInterval == 0 {
		// Prune expired attestations from the pool every slot interval.
		cfg.pruneInterval = time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second
	}

	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		cfg:                      cfg,
		ctx:                      ctx,
		cancel:                   cancel,
		forkChoiceProcessedRoots: cache,
	}, nil
}

// Start an attestation pool service's main event loop.
func (s *Service) Start() {
	go s.prepareForkChoiceAtts()
	go s.pruneAttsPool()
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

// SetGenesisTime sets genesis time for operation service to use.
func (s *Service) SetGenesisTime(t uint64) {
	s.genesisTime = t
}
