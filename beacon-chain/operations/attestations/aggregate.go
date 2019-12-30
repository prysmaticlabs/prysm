package attestations

import (
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var timeToAggregate = time.Duration(params.BeaconConfig().SecondsPerSlot/3) * time.Second

func (s *Service) aggregateAttestations() {
	ticker := time.NewTicker(timeToAggregate)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.aggregateAttestation(); err != nil {
				log.Error(err)
			}
		}
	}
}

func (s *Service) aggregateAttestation() error {
	unaggregatedAtts := s.pool.UnaggregatedAttestations()
	attsByDataRoot := make(map[[32]byte][]*ethpb.Attestation)

	for _, att := range unaggregatedAtts {
		attDataRoot, err := ssz.HashTreeRoot(att.Data)
		if err != nil {
			return err
		}
		attsByDataRoot[attDataRoot] = append(attsByDataRoot[attDataRoot], att)

		if err := s.pool.DeleteUnaggregatedAttestation(att); err != nil {
			return err
		}
	}

	for _, atts := range attsByDataRoot {
		aggregatedAtts, err := helpers.AggregateAttestations(atts)
		if err != nil {
			return err
		}
		for _, att := range aggregatedAtts {
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
