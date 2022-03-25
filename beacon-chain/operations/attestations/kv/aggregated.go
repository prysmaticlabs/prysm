package kv

import (
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	attaggregation "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation/aggregation/attestations"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// AggregateUnaggregatedAttestations aggregates the unaggregated attestations and saves the
// newly aggregated attestations in the pool.
// It tracks the unaggregated attestations that weren't able to aggregate to prevent
// the deletion of unaggregated attestations in the pool.
func (c *AttCaches) AggregateUnaggregatedAttestations(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "operations.attestations.kv.AggregateUnaggregatedAttestations")
	defer span.End()
	unaggregatedAtts, err := c.UnaggregatedAttestations()
	if err != nil {
		return err
	}
	return c.aggregateUnaggregatedAttestations(ctx, unaggregatedAtts)
}

// AggregateUnaggregatedAttestationsBySlotIndex aggregates the unaggregated attestations and saves
// newly aggregated attestations in the pool. Unaggregated attestations are filtered by slot and
// committee index.
func (c *AttCaches) AggregateUnaggregatedAttestationsBySlotIndex(ctx context.Context, slot types.Slot, committeeIndex types.CommitteeIndex) error {
	ctx, span := trace.StartSpan(ctx, "operations.attestations.kv.AggregateUnaggregatedAttestationsBySlotIndex")
	defer span.End()
	unaggregatedAtts := c.UnaggregatedAttestationsBySlotIndex(ctx, slot, committeeIndex)
	return c.aggregateUnaggregatedAttestations(ctx, unaggregatedAtts)
}

func (c *AttCaches) aggregateUnaggregatedAttestations(ctx context.Context, unaggregatedAtts []*ethpb.Attestation) error {
	_, span := trace.StartSpan(ctx, "operations.attestations.kv.aggregateUnaggregatedAttestations")
	defer span.End()

	attsByDataRoot := make(map[[32]byte][]*ethpb.Attestation, len(unaggregatedAtts))
	for _, att := range unaggregatedAtts {
		attDataRoot, err := att.Data.HashTreeRoot()
		if err != nil {
			return err
		}
		attsByDataRoot[attDataRoot] = append(attsByDataRoot[attDataRoot], att)
	}

	// Aggregate unaggregated attestations from the pool and save them in the pool.
	// Track the unaggregated attestations that aren't able to aggregate.
	leftOverUnaggregatedAtt := make(map[[32]byte]bool)
	for _, atts := range attsByDataRoot {
		aggregatedAtts := make([]*ethpb.Attestation, 0, len(atts))
		processedAtts, err := attaggregation.Aggregate(atts)
		if err != nil {
			return err
		}
		for _, att := range processedAtts {
			if helpers.IsAggregated(att) {
				aggregatedAtts = append(aggregatedAtts, att)
			} else {
				h, err := hashFn(att)
				if err != nil {
					return err
				}
				leftOverUnaggregatedAtt[h] = true
			}
		}
		if err := c.SaveAggregatedAttestations(aggregatedAtts); err != nil {
			return err
		}
	}

	// Remove the unaggregated attestations from the pool that were successfully aggregated.
	for _, att := range unaggregatedAtts {
		h, err := hashFn(att)
		if err != nil {
			return err
		}
		if leftOverUnaggregatedAtt[h] {
			continue
		}
		if err := c.DeleteUnaggregatedAttestation(att); err != nil {
			return err
		}
	}

	return nil
}

// SaveAggregatedAttestation saves an aggregated attestation in cache.
func (c *AttCaches) SaveAggregatedAttestation(att *ethpb.Attestation) error {
	if err := helpers.ValidateNilAttestation(att); err != nil {
		return err
	}
	if !helpers.IsAggregated(att) {
		return errors.New("attestation is not aggregated")
	}
	has, err := c.HasAggregatedAttestation(att)
	if err != nil {
		return err
	}
	if has {
		return nil
	}

	seen, err := c.hasSeenBit(att)
	if err != nil {
		return err
	}
	if seen {
		return nil
	}

	r, err := hashFn(att.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}
	copiedAtt := ethpb.CopyAttestation(att)
	c.aggregatedAttLock.Lock()
	defer c.aggregatedAttLock.Unlock()
	atts, ok := c.aggregatedAtt[r]
	if !ok {
		atts := []*ethpb.Attestation{copiedAtt}
		c.aggregatedAtt[r] = atts
		return nil
	}

	atts, err = attaggregation.Aggregate(append(atts, copiedAtt))
	if err != nil {
		return err
	}
	c.aggregatedAtt[r] = atts

	return nil
}

// SaveAggregatedAttestations saves a list of aggregated attestations in cache.
func (c *AttCaches) SaveAggregatedAttestations(atts []*ethpb.Attestation) error {
	for _, att := range atts {
		if err := c.SaveAggregatedAttestation(att); err != nil {
			log.WithError(err).Debug("Could not save aggregated attestation")
			if err := c.DeleteAggregatedAttestation(att); err != nil {
				log.WithError(err).Debug("Could not delete aggregated attestation")
			}
		}
	}
	return nil
}

// AggregatedAttestations returns the aggregated attestations in cache.
func (c *AttCaches) AggregatedAttestations() []*ethpb.Attestation {
	c.aggregatedAttLock.RLock()
	defer c.aggregatedAttLock.RUnlock()

	atts := make([]*ethpb.Attestation, 0)

	for _, a := range c.aggregatedAtt {
		atts = append(atts, a...)
	}

	return atts
}

// AggregatedAttestationsBySlotIndex returns the aggregated attestations in cache,
// filtered by committee index and slot.
func (c *AttCaches) AggregatedAttestationsBySlotIndex(ctx context.Context, slot types.Slot, committeeIndex types.CommitteeIndex) []*ethpb.Attestation {
	_, span := trace.StartSpan(ctx, "operations.attestations.kv.AggregatedAttestationsBySlotIndex")
	defer span.End()

	atts := make([]*ethpb.Attestation, 0)

	c.aggregatedAttLock.RLock()
	defer c.aggregatedAttLock.RUnlock()
	for _, a := range c.aggregatedAtt {
		if slot == a[0].Data.Slot && committeeIndex == a[0].Data.CommitteeIndex {
			atts = append(atts, a...)
		}
	}

	return atts
}

// DeleteAggregatedAttestation deletes the aggregated attestations in cache.
func (c *AttCaches) DeleteAggregatedAttestation(att *ethpb.Attestation) error {
	if err := helpers.ValidateNilAttestation(att); err != nil {
		return err
	}
	if !helpers.IsAggregated(att) {
		return errors.New("attestation is not aggregated")
	}
	r, err := hashFn(att.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation data")
	}

	if err := c.insertSeenBit(att); err != nil {
		return err
	}

	c.aggregatedAttLock.Lock()
	defer c.aggregatedAttLock.Unlock()
	attList, ok := c.aggregatedAtt[r]
	if !ok {
		return nil
	}

	filtered := make([]*ethpb.Attestation, 0)
	for _, a := range attList {
		if c, err := att.AggregationBits.Contains(a.AggregationBits); err != nil {
			return err
		} else if !c {
			filtered = append(filtered, a)
		}
	}
	if len(filtered) == 0 {
		delete(c.aggregatedAtt, r)
	} else {
		c.aggregatedAtt[r] = filtered
	}

	return nil
}

// HasAggregatedAttestation checks if the input attestations has already existed in cache.
func (c *AttCaches) HasAggregatedAttestation(att *ethpb.Attestation) (bool, error) {
	if err := helpers.ValidateNilAttestation(att); err != nil {
		return false, err
	}
	r, err := hashFn(att.Data)
	if err != nil {
		return false, errors.Wrap(err, "could not tree hash attestation")
	}

	c.aggregatedAttLock.RLock()
	defer c.aggregatedAttLock.RUnlock()
	if atts, ok := c.aggregatedAtt[r]; ok {
		for _, a := range atts {
			if c, err := a.AggregationBits.Contains(att.AggregationBits); err != nil {
				return false, err
			} else if c {
				return true, nil
			}
		}
	}

	c.blockAttLock.RLock()
	defer c.blockAttLock.RUnlock()
	if atts, ok := c.blockAtt[r]; ok {
		for _, a := range atts {
			if c, err := a.AggregationBits.Contains(att.AggregationBits); err != nil {
				return false, err
			} else if c {
				return true, nil
			}
		}
	}

	return false, nil
}

// AggregatedAttestationCount returns the number of aggregated attestations key in the pool.
func (c *AttCaches) AggregatedAttestationCount() int {
	c.aggregatedAttLock.RLock()
	defer c.aggregatedAttLock.RUnlock()
	return len(c.aggregatedAtt)
}
