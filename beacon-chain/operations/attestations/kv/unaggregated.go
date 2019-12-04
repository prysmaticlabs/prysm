package kv

import (
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
)

// SaveUnaggregatedAttestation saves an unaggregated attestation in cache.
func (p *AttCaches) SaveUnaggregatedAttestation(att *ethpb.Attestation) error {
	if aggregated(att.AggregationBits) {
		return errors.New("attestation is aggregated")
	}

	r, err := ssz.HashTreeRoot(att)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	// DefaultExpiration is set to what was given to New(). In this case
	// it's one epoch.
	p.unAggregatedAtt.Set(string(r[:]), att, cache.DefaultExpiration)

	return nil
}

// UnaggregatedAttestation returns the aggregated attestations in cache,
// filtered by committee index and slot.
func (p *AttCaches) UnaggregatedAttestation(slot uint64, committeeIndex uint64) []*ethpb.Attestation {
	atts := make([]*ethpb.Attestation, 0, p.unAggregatedAtt.ItemCount())
	for s, i := range p.unAggregatedAtt.Items() {

		// Type assertion for the worst case. This shouldn't happen.
		att, ok := i.Object.(*ethpb.Attestation)
		if !ok {
			p.unAggregatedAtt.Delete(s)
		}

		if slot == att.Data.Slot && committeeIndex == att.Data.CommitteeIndex {
			atts = append(atts, att)
		}
	}

	return atts
}

// DeleteUnaggregatedAttestation deletes the unaggregated attestations in cache.
func (p *AttCaches) DeleteUnaggregatedAttestation(att *ethpb.Attestation) error {
	if aggregated(att.AggregationBits) {
		return errors.New("attestation is not aggregated")
	}

	r, err := ssz.HashTreeRoot(att)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	p.unAggregatedAtt.Delete(string(r[:]))

	return nil
}
