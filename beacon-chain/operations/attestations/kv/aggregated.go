package kv

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
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

	copiedAtt := stateTrie.CopyAttestation(att)
	atts, ok := p.aggregatedAtt[r]
	if !ok {
		atts := []*ethpb.Attestation{copiedAtt}
		p.aggregatedAtt[r] = atts
		return nil
	}

	atts, err = helpers.AggregateAttestations(append(atts, copiedAtt))
	if err != nil {
		return err
	}
	p.aggregatedAtt[r] = atts

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

// AggregatedAttestations returns the aggregated attestations in cache.
func (p *AttCaches) AggregatedAttestations() []*ethpb.Attestation {
	atts := make([]*ethpb.Attestation, 0)
	for _, a := range p.aggregatedAtt {
		atts = append(atts, a...)
	}

	return atts
}

// AggregatedAttestationsBySlotIndex returns the aggregated attestations in cache,
// filtered by committee index and slot.
func (p *AttCaches) AggregatedAttestationsBySlotIndex(slot uint64, committeeIndex uint64) []*ethpb.Attestation {
	atts := make([]*ethpb.Attestation, 0)
	for _, a := range p.aggregatedAtt {
		if slot == a[0].Data.Slot && committeeIndex == a[0].Data.CommitteeIndex {
			atts = append(atts, a...)
		}
	}

	return atts
}

// DeleteAggregatedAttestation deletes the aggregated attestations in cache.
func (p *AttCaches) DeleteAggregatedAttestation(att *ethpb.Attestation) error {
	if !helpers.IsAggregated(att) {
		return errors.New("attestation is not aggregated")
	}
	r, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation data")
	}
	attList, ok := p.aggregatedAtt[r]
	if !ok {
		return nil
	}

	filtered := make([]*ethpb.Attestation, 0)
	for _, a := range attList {
		if !att.AggregationBits.Contains(a.AggregationBits) {
			filtered = append(filtered, a)
		}
	}
	if len(filtered) == 0 {
		delete(p.aggregatedAtt, r)
	} else {
		p.aggregatedAtt[r] = filtered
	}

	return nil
}

// HasAggregatedAttestation checks if the input attestations has already existed in cache.
func (p *AttCaches) HasAggregatedAttestation(att *ethpb.Attestation) (bool, error) {
	r, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		return false, errors.Wrap(err, "could not tree hash attestation")
	}

	if atts, ok := p.aggregatedAtt[r]; ok {
		for _, a := range atts {
			if a.AggregationBits.Contains(att.AggregationBits) {
				return true, nil
			}
		}
	}

	return false, nil
}

// AggregatedAttestationCount returns the number of aggregated attestations key in the pool.
func (p *AttCaches) AggregatedAttestationCount() int {
	return len(p.aggregatedAtt)
}
