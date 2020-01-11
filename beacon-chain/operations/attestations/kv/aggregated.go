package kv

import (
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
)

// SaveAggregatedAttestation saves an aggregated attestation in cache.
func (p *AttCaches) SaveAggregatedAttestation(att *ethpb.Attestation) error {
	if !helpers.IsAggregated(att) {
		return errors.New("attestation is not aggregated")
	}

	r, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}
	key := string(r[:])

	var atts []*ethpb.Attestation
	obj, ok := p.aggregatedAtt.Get(key)
	if ok {
		atts, ok = obj.([]*ethpb.Attestation)
		if !ok {
			p.aggregatedAtt.Delete(key)
		}
	}
	atts = append(atts, att)

	// DefaultExpiration is set to what was given to New(). In this case
	// it's one epoch.
	p.aggregatedAtt.Set(key, atts, cache.DefaultExpiration)

	return nil
}

// SaveAggregatedAttestations saves a list of aggregated attestations in cache.
func (p *AttCaches) SaveAggregatedAttestations(atts []*ethpb.Attestation) error {
	for _, att := range atts {
		if err := p.SaveAggregatedAttestation(att); err != nil {
			return err
		}
	}
	return nil
}

// AggregatedAttestations returns all the aggregated attestations in cache.
func (p *AttCaches) AggregatedAttestations() []*ethpb.Attestation {
	var atts []*ethpb.Attestation
	for s, i := range p.aggregatedAtt.Items() {
		// Type assertion for the worst case. This shouldn't happen.
		att, ok := i.Object.([]*ethpb.Attestation)
		if !ok {
			p.aggregatedAtt.Delete(s)
		}
		atts = append(atts, att...)
	}

	return atts
}

// AggregatedAttestationsBySlotIndex returns the all aggregated attestations in cache,
// filtered by committee index and slot.
func (p *AttCaches) AggregatedAttestationsBySlotIndex(slot uint64, committeeIndex uint64) []*ethpb.Attestation {
	var aggregatedAttsBySlotIndex []*ethpb.Attestation
	for s, i := range p.aggregatedAtt.Items() {

		// Type assertion for the worst case. This shouldn't happen.
		atts, ok := i.Object.([]*ethpb.Attestation)
		if !ok {
			p.aggregatedAtt.Delete(s)
		}
		for _, att := range atts {
			if slot == att.Data.Slot && committeeIndex == att.Data.CommitteeIndex {
				aggregatedAttsBySlotIndex = append(aggregatedAttsBySlotIndex, att)
			}
		}
	}

	return aggregatedAttsBySlotIndex
}

// HasAggregatedAttestation checks if the input attestation has already existed in cache.
func (p *AttCaches) HasAggregatedAttestation(att *ethpb.Attestation) (bool, error) {
	r, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		return false, errors.Wrap(err, "could not tree hash attestation")
	}
	key := string(r[:])

	obj, ok := p.aggregatedAtt.Get(key)
	// Verify if we seen this attestation data at all.
	if !ok {
		return false, nil
	} else {
		atts, ok := obj.([]*ethpb.Attestation)
		if !ok {
			p.aggregatedAtt.Delete(key)
		}
		// Verify the bit field of the input attestation is fully contained by the pool attestations.
		for _, a := range atts {
			if a.AggregationBits.Contains(att.AggregationBits) {
				return true, nil
			}
		}
	}

	return false, nil
}
