package kv

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// SaveUnaggregatedAttestation saves an unaggregated attestation in cache.
func (c *AttCaches) SaveUnaggregatedAttestation(att blocks.ROAttestation) error {
	if helpers.IsAggregated(att.Att) {
		return errors.New("attestation is aggregated")
	}

	seen, err := c.hasSeenBit(att)
	if err != nil {
		return err
	}
	if seen {
		return nil
	}

	c.unAggregateAttLock.Lock()
	defer c.unAggregateAttLock.Unlock()
	c.unAggregatedAtt[att.Id()] = att

	return nil
}

// SaveUnaggregatedAttestations saves a list of unaggregated attestations in cache.
func (c *AttCaches) SaveUnaggregatedAttestations(atts []blocks.ROAttestation) error {
	for _, att := range atts {
		if err := c.SaveUnaggregatedAttestation(att); err != nil {
			return err
		}
	}

	return nil
}

// UnaggregatedAttestations returns all the unaggregated attestations in cache.
func (c *AttCaches) UnaggregatedAttestations() ([]blocks.ROAttestation, error) {
	c.unAggregateAttLock.RLock()
	defer c.unAggregateAttLock.RUnlock()
	unAggregatedAtts := c.unAggregatedAtt
	atts := make([]blocks.ROAttestation, 0, len(unAggregatedAtts))
	for _, att := range unAggregatedAtts {
		seen, err := c.hasSeenBit(att)
		if err != nil {
			return nil, err
		}
		if !seen {
			atts = append(atts, att.Copy())
		}
	}
	return atts, nil
}

// UnaggregatedAttestationsBySlotIndex returns the unaggregated attestations in cache,
// filtered by committee index and slot.
func (c *AttCaches) UnaggregatedAttestationsBySlotIndex(ctx context.Context, slot primitives.Slot, committeeIndex primitives.CommitteeIndex) []ethpb.Att {
	_, span := trace.StartSpan(ctx, "operations.attestations.kv.UnaggregatedAttestationsBySlotIndex")
	defer span.End()

	atts := make([]ethpb.Att, 0)

	c.unAggregateAttLock.RLock()
	defer c.unAggregateAttLock.RUnlock()

	unAggregatedAtts := c.unAggregatedAtt
	for _, a := range unAggregatedAtts {
		if slot == a.GetData().Slot && committeeIndex == a.GetData().CommitteeIndex {
			att, ok := a.Att.(*ethpb.Attestation)
			if ok {
				atts = append(atts, att)
			}
		}
	}

	return atts
}

// DeleteUnaggregatedAttestation deletes the unaggregated attestations in cache.
func (c *AttCaches) DeleteUnaggregatedAttestation(att blocks.ROAttestation) error {
	if helpers.IsAggregated(att.Att) {
		return errors.New("attestation is aggregated")
	}

	if err := c.insertSeenBit(att); err != nil {
		return err
	}

	c.unAggregateAttLock.Lock()
	defer c.unAggregateAttLock.Unlock()
	delete(c.unAggregatedAtt, att.Id())

	return nil
}

// DeleteSeenUnaggregatedAttestations deletes the unaggregated attestations in cache
// that have been already processed once. Returns number of attestations deleted.
func (c *AttCaches) DeleteSeenUnaggregatedAttestations() (int, error) {
	c.unAggregateAttLock.Lock()
	defer c.unAggregateAttLock.Unlock()

	count := 0
	for id, att := range c.unAggregatedAtt {
		if helpers.IsAggregated(att.Att) {
			continue
		}
		if seen, err := c.hasSeenBit(att); err == nil && seen {
			delete(c.unAggregatedAtt, id)
			count++
		}
	}
	return count, nil
}

// UnaggregatedAttestationCount returns the number of unaggregated attestations key in the pool.
func (c *AttCaches) UnaggregatedAttestationCount() int {
	c.unAggregateAttLock.RLock()
	defer c.unAggregateAttLock.RUnlock()
	return len(c.unAggregatedAtt)
}
