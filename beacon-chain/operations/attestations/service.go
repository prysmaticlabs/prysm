// Package attestations defines an attestation pool
// service implementation which is used to manage the lifecycle
// of aggregated, unaggregated, and fork-choice attestations.
package attestations

import (
	"context"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var forkChoiceProcessedRootsSize = 1 << 16

// Service of attestation pool operations.
type Service struct {
	ctx                      context.Context
	cancel                   context.CancelFunc
	pool                     Pool
	err                      error
	forkChoiceProcessedRoots *lru.Cache
	genesisTime              uint64
	pruneInterval            time.Duration
}

// Config options for the service.
type Config struct {
	Pool          Pool
	pruneInterval time.Duration
}

// NewService instantiates a new attestation pool service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	cache, err := lru.New(forkChoiceProcessedRootsSize)
	if err != nil {
		return nil, err
	}

	pruneInterval := cfg.pruneInterval
	if pruneInterval == 0 {
		// Prune expired attestations from the pool every slot interval.
		pruneInterval = time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second
	}

	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                      ctx,
		cancel:                   cancel,
		pool:                     cfg.Pool,
		forkChoiceProcessedRoots: cache,
		pruneInterval:            pruneInterval,
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
