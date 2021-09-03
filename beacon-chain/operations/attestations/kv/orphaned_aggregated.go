package kv

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	attaggregation "github.com/prysmaticlabs/prysm/shared/aggregation/attestations"
	"github.com/prysmaticlabs/prysm/shared/copyutil"
)

// SaveOrphanedAggregatedAttestation saves an orphaned aggregated attestation in cache.
func (c *AttCaches) SaveOrphanedAggregatedAttestation(att *ethpb.Attestation) error {
	if err := helpers.ValidateNilAttestation(att); err != nil {
		return err
	}
	if !helpers.IsAggregated(att) {
		return errors.New("orphaned attestation is not aggregated")
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
	copiedAtt := copyutil.CopyAttestation(att)
	c.orphanedAggregatedAttLock.Lock()
	defer c.orphanedAggregatedAttLock.Unlock()
	atts, ok := c.orphanedAggregatedAtt[r]
	if !ok {
		atts := []*ethpb.Attestation{copiedAtt}
		c.orphanedAggregatedAtt[r] = atts
		return nil
	}

	atts, err = attaggregation.Aggregate(append(atts, copiedAtt))
	if err != nil {
		return err
	}
	c.orphanedAggregatedAtt[r] = atts

	return nil
}

// DeleteOrphanedAggregatedAttestation deletes the orphaned aggregated attestations in cache.
func (c *AttCaches) DeleteOrphanedAggregatedAttestation(att *ethpb.Attestation) error {
	if err := helpers.ValidateNilAttestation(att); err != nil {
		return err
	}
	if !helpers.IsAggregated(att) {
		return errors.New("orphaned attestation is not aggregated")
	}
	r, err := hashFn(att.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation data")
	}

	if err := c.insertSeenBit(att); err != nil {
		return err
	}

	c.orphanedAggregatedAttLock.Lock()
	defer c.orphanedAggregatedAttLock.Unlock()
	attList, ok := c.orphanedAggregatedAtt[r]
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
		delete(c.orphanedAggregatedAtt, r)
	} else {
		c.orphanedAggregatedAtt[r] = filtered
	}

	return nil
}

// OrphanedAggregatedAttestations returns the orphaned aggregated attestations in cache.
func (c *AttCaches) OrphanedAggregatedAttestations() []*ethpb.Attestation {
	c.orphanedAggregatedAttLock.RLock()
	defer c.orphanedAggregatedAttLock.RUnlock()

	atts := make([]*ethpb.Attestation, 0)

	for _, a := range c.orphanedAggregatedAtt {
		atts = append(atts, a...)
	}

	return atts
}
