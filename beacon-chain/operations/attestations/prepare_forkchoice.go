package attestations

import (
	"context"
	"errors"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// Prepare attestations for fork choice at every half of the slot.
var prepareForkChoiceAttsPeriod = time.Duration(params.BeaconConfig().SecondsPerSlot/3) * time.Second

// This prepares fork choice attestations by running batchForkChoiceAtts
// every prepareForkChoiceAttsPeriod.
func (s *Service) prepareForkChoiceAtts() {
	ticker := time.NewTicker(prepareForkChoiceAttsPeriod)
	for {
		ctx := context.Background()
		select {
		case <-ticker.C:
			if err := s.batchForkChoiceAtts(ctx); err != nil {
				log.WithError(err).Error("Could not prepare attestations for fork choice")
			}
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting routine")
			return
		}
	}
}

// This gets the attestations from the unaggregated, aggregated and block
// pool. Then finds the common data, aggregate and batch them for fork choice.
// The resulting attestations are saved in the fork choice pool.
func (s *Service) batchForkChoiceAtts(ctx context.Context) error {
	_, span := trace.StartSpan(ctx, "Operations.attestations.batchForkChoiceAtts")
	defer span.End()

	attsByDataRoot := make(map[[32]byte][]*ethpb.Attestation)

	atts := append(s.pool.UnaggregatedAttestations(), s.pool.AggregatedAttestations()...)
	atts = append(atts, s.pool.BlockAttestations()...)
	atts = append(atts, s.pool.ForkchoiceAttestations()...)

	for _, att := range atts {
		seen, err := s.seen(att)
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

	for _, a := range s.pool.BlockAttestations() {
		if err := s.pool.DeleteBlockAttestation(a); err != nil {
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

	if err := s.pool.SaveForkchoiceAttestations(aggregatedAtts); err != nil {
		return err
	}

	return nil
}

// This checks if the attestation has previously been aggregated for fork choice
// return true if yes, false if no.
func (s *Service) seen(att *ethpb.Attestation) (bool, error) {
	attRoot, err := hashutil.HashProto(att.Data)
	if err != nil {
		return false, err
	}
	savedBits, ok := s.forkChoiceProcessedRoots.Get(string(attRoot[:]))
	if ok {
		savedBitlist, ok := savedBits.(bitfield.Bitlist)
		if !ok {
			return false, errors.New("not a bit field")
		}
		if savedBitlist.Contains(att.AggregationBits) {
			return true, nil
		}
	}

	s.forkChoiceProcessedRoots.Set(string(attRoot[:]), att.AggregationBits, 1 /*cost*/)
	return false, nil
}
