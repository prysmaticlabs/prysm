package attestations

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

var forkchoiceProcessedRootsSize = int64(1 << 16)

// forkchoiceProcessedAttRoots cache with max size of ~2Mib ( including keys)
var forkchoiceProcessedRoots, _ = ristretto.NewCache(&ristretto.Config{
	NumCounters: forkchoiceProcessedRootsSize,
	MaxCost:     forkchoiceProcessedRootsSize,
	BufferItems: 64,
})

// prepare attestations for fork choice for every half of the slot.
var prepareForkchoiceAttsPeriod = time.Duration(params.BeaconConfig().SecondsPerSlot/2) * time.Second

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
		pool:   cfg.Pool,
	}
}

// Start an attestation pool service's main event loop.
func (s *Service) Start() {
	go s.forkChoiceAtts()
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

// This gets the attestations from the unaggregated, aggregated and block
// pool. Then finds the common data and aggregate them for fork choice. The resulting attestations
// are saved in the fork choice pool.
func (s *Service) prepareForkChoiceAtts(ctx context.Context) error {
	_, span := trace.StartSpan(ctx, "Operations.attestations.prepareForkChoiceAtts")
	defer span.End()

	attsByDataRoot := make(map[[32]byte][]*ethpb.Attestation)

	atts := append(s.pool.UnaggregatedAttestations(), s.pool.AggregatedAttestations()...)
	atts = append(atts, s.pool.BlockAttestations()...)

	for _, att := range atts {
		seen, err := seen(att)
		if err != nil {
			return err
		}
		if seen {
			continue
		}

		attDataRoot, err := ssz.HashTreeRoot(att.Data)
		if err != nil {
			return err
		}
		attsByDataRoot[attDataRoot] = append(attsByDataRoot[attDataRoot], att)
	}

	for _, atts := range attsByDataRoot {
		if err := s.aggregateAndSaveForkChoiceAtts(atts); err != nil {
			return err
		}
	}

	return nil
}

// This aggregates a list of attestations using the aggregation algorithm defined in AggregateAttestations
// and saves the attestations for fork choice.
func (s *Service) aggregateAndSaveForkChoiceAtts(atts []*ethpb.Attestation) error {
	aggregatedAtts, err := helpers.AggregateAttestations(atts)
	if err != nil {
		return err
	}
	for _, att := range aggregatedAtts {
		if err := s.pool.SaveForkchoiceAttestation(att); err != nil {
			return err
		}
	}
	return nil
}

// This prepares fork choice attestations by running prepareForkChoiceAtts
// every prepareForkchoiceAttsPeriod.
func (s *Service) forkChoiceAtts() {
	ticker := time.NewTicker(prepareForkchoiceAttsPeriod)
	for {
		ctx := context.Background()
		select {
		case <-ticker.C:
			if err := s.prepareForkChoiceAtts(ctx); err != nil {
				log.WithError(err).Error("Could not save fork choice attestations")
			}
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting routine")
			return
		}
	}
}

// This checks if the attestation has previously been aggregated for fork choice
// return true if yes, false if no.
func seen(att *ethpb.Attestation) (bool, error) {
	attRoot, err := hashutil.HashProto(att)
	if err != nil {
		return false, err
	}
	if _, ok := forkchoiceProcessedRoots.Get(string(attRoot[:])); ok {
		return true, nil
	}
	forkchoiceProcessedRoots.Set(string(attRoot[:]), true /*value*/, 1 /*cost*/)

	return false, nil
}
