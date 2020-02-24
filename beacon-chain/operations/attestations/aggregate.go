package attestations

import (
	"context"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// Define time to aggregate the unaggregated attestations at 2 times per slot, this gives
// enough confidence all the unaggregated attestations will be aggregated as aggregator requests.
var timeToAggregate = time.Duration(params.BeaconConfig().SecondsPerSlot/2) * time.Second

// This kicks off a routine to aggregate the unaggregated attestations from pool.
func (s *Service) aggregateRoutine() {
	ticker := time.NewTicker(timeToAggregate)
	ctx := context.TODO()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			attsToBeAggregated := append(s.pool.UnaggregatedAttestations(), s.pool.AggregatedAttestations()...)
			if err := s.aggregateAttestations(ctx, attsToBeAggregated); err != nil {
				log.WithError(err).Error("Could not aggregate attestation")
			}

			// Update metrics for aggregated and unaggregated attestations count.
			s.updateMetrics()
		}
	}
}

// This aggregates the input attestations via AggregateAttestations helper
// function.
func (s *Service) aggregateAttestations(ctx context.Context, attsToBeAggregated []*ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "Operations.attestations.aggregateAttestations")
	defer span.End()

	attsByRoot := make(map[[32]byte][]*ethpb.Attestation)

	for _, att := range attsToBeAggregated {
		attDataRoot, err := ssz.HashTreeRoot(att.Data)
		if err != nil {
			return err
		}
		attsByRoot[attDataRoot] = append(attsByRoot[attDataRoot], att)
	}

	for _, atts := range attsByRoot {
		for _, att := range atts {
			if !helpers.IsAggregated(att) && len(atts) > 1 {
				if err := s.pool.DeleteUnaggregatedAttestation(att); err != nil {
					return err
				}
			}
		}
	}

	for _, atts := range attsByRoot {
		aggregatedAtts, err := helpers.AggregateAttestations(atts)
		if err != nil {
			return err
		}
		for _, att := range aggregatedAtts {
			if helpers.IsAggregated(att) {
				if err := s.pool.SaveAggregatedAttestation(att); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
