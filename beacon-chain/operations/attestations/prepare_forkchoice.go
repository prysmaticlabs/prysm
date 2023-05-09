package attestations

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/crypto/hash"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	attaggregation "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1/attestation/aggregation/attestations"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
)

// This prepares fork choice attestations by running batchForkChoiceAtts
// every prepareForkChoiceAttsPeriod.
func (s *Service) prepareForkChoiceAtts() {
	ticker1 := slots.NewSlotTickerWithOffset(time.Unix(int64(s.genesisTime), 0), 7*time.Second, params.BeaconConfig().SecondsPerSlot)
	ticker2 := slots.NewSlotTickerWithOffset(time.Unix(int64(s.genesisTime), 0), 10*time.Second, params.BeaconConfig().SecondsPerSlot)
	ticker3 := slots.NewSlotTickerWithOffset(time.Unix(int64(s.genesisTime), 0), 11*time.Second, params.BeaconConfig().SecondsPerSlot)
	for {
		select {
		case <-ticker1.C():
			t := time.Now()
			if err := s.batchForkChoiceAtts(s.ctx); err != nil {
				log.WithError(err).Error("Could not prepare attestations for fork choice")
			}
			log.Debug("Batch forkchoice atts took: ", time.Since(t).Milliseconds())
		case <-ticker2.C():
			if err := s.batchForkChoiceAtts(s.ctx); err != nil {
				log.WithError(err).Error("Could not prepare attestations for fork choice")
			}
		case <-ticker3.C():
			if err := s.batchForkChoiceAtts(s.ctx); err != nil {
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
	ctx, span := trace.StartSpan(ctx, "Operations.attestations.batchForkChoiceAtts")
	defer span.End()

	if err := s.cfg.Pool.AggregateUnaggregatedAttestations(ctx); err != nil {
		return err
	}
	atts := append(s.cfg.Pool.AggregatedAttestations(), s.cfg.Pool.BlockAttestations()...)
	atts = append(atts, s.cfg.Pool.ForkchoiceAttestations()...)

	attsByDataRoot := make(map[[32]byte][]*ethpb.Attestation, len(atts))

	// Consolidate attestations by aggregating them by similar data root.
	for _, att := range atts {
		seen, err := s.seen(att)
		if err != nil {
			return err
		}
		if seen {
			continue
		}

		attDataRoot, err := att.Data.HashTreeRoot()
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

	for _, a := range s.cfg.Pool.BlockAttestations() {
		if err := s.cfg.Pool.DeleteBlockAttestation(a); err != nil {
			return err
		}
	}

	return nil
}

// This aggregates a list of attestations using the aggregation algorithm defined in AggregateAttestations
// and saves the attestations for fork choice.
func (s *Service) aggregateAndSaveForkChoiceAtts(atts []*ethpb.Attestation) error {
	clonedAtts := make([]*ethpb.Attestation, len(atts))
	for i, a := range atts {
		clonedAtts[i] = ethpb.CopyAttestation(a)
	}
	aggregatedAtts, err := attaggregation.Aggregate(clonedAtts)
	if err != nil {
		return err
	}

	return s.cfg.Pool.SaveForkchoiceAttestations(aggregatedAtts)
}

// This checks if the attestation has previously been aggregated for fork choice
// return true if yes, false if no.
func (s *Service) seen(att *ethpb.Attestation) (bool, error) {
	attRoot, err := hash.HashProto(att.Data)
	if err != nil {
		return false, err
	}
	incomingBits := att.AggregationBits
	savedBits, ok := s.forkChoiceProcessedRoots.Get(attRoot)
	if ok {
		savedBitlist, ok := savedBits.(bitfield.Bitlist)
		if !ok {
			return false, errors.New("not a bit field")
		}
		if savedBitlist.Len() == incomingBits.Len() {
			// Returns true if the node has seen all the bits in the new bit field of the incoming attestation.
			if bytes.Equal(savedBitlist, incomingBits) {
				return true, nil
			}
			if c, err := savedBitlist.Contains(incomingBits); err != nil {
				return false, err
			} else if c {
				return true, nil
			}
			var err error
			// Update the bit fields by Or'ing them with the new ones.
			incomingBits, err = incomingBits.Or(savedBitlist)
			if err != nil {
				return false, err
			}
		}
	}

	s.forkChoiceProcessedRoots.Add(attRoot, incomingBits)
	return false, nil
}
