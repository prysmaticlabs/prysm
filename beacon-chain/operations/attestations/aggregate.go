package attestations

import (
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Define time to aggregate the unaggregated attestations at 3 times per slot, this gives
// enough confidence all the unaggregated attestations will be aggregated as aggregator requests.
var timeToAggregate = time.Duration(params.BeaconConfig().SecondsPerSlot/3) * time.Second

// This kicks off a routine to aggregate the unaggregated attestations from pool.
func (s *Service) aggregateRoutine() {
	ticker := time.NewTicker(timeToAggregate)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			unaggregatedAtts := s.pool.UnaggregatedAttestations()
			if err := s.aggregateAttestations(unaggregatedAtts); err != nil {
				log.Errorf("Could not aggregate attestation: %v", err)
			}
		}
	}
}

// This aggregates the input attestations via AggregateAttestations helper
// function.
func (s *Service) aggregateAttestations(unaggregatedAtts []*ethpb.Attestation) error {
	unaggregatedAttsByRoot := make(map[[32]byte][]*ethpb.Attestation)

	for _, att := range unaggregatedAtts {
		attDataRoot, err := ssz.HashTreeRoot(att.Data)
		if err != nil {
			return err
		}
		unaggregatedAttsByRoot[attDataRoot] = append(unaggregatedAttsByRoot[attDataRoot], att)

		if err := s.pool.DeleteUnaggregatedAttestation(att); err != nil {
			return err
		}
	}

	for _, atts := range unaggregatedAttsByRoot {
		aggregatedAtts, err := helpers.AggregateAttestations(atts)
		if err != nil {
			return err
		}
		for _, att := range aggregatedAtts {
			// In case of aggregation bit overlaps, not every attestations will
			// be aggregated.
			if helpers.IsAggregated(att) {
				if err := s.pool.SaveAggregatedAttestation(att); err != nil {
					return err
				}
			} else {
				if err := s.pool.SaveUnaggregatedAttestation(att); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
