package kv

import (
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
)

// SaveBlockAttestation saves an block attestation in cache.
func (p *AttCaches) SaveBlockAttestation(att *ethpb.Attestation) error {
	r, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	var atts []*ethpb.Attestation
	d, ok := p.blockAtt.Get(string(r[:]))
	if !ok {
		atts = make([]*ethpb.Attestation, 0)
	} else {
		atts, ok = d.([]*ethpb.Attestation)
		if !ok {
			return errors.New("cached value is not of type []*ethpb.Attestation")
		}
	}

	// Ensure that this attestation is not already fully contained in an existing attestation.
	for _, a := range atts {
		if a.AggregationBits.Contains(att.AggregationBits) {
			return nil
		}
	}
	atts = append(atts, att)

	// DefaultExpiration is set to what was given to New(). In this case
	// it's one epoch.
	p.blockAtt.Set(string(r[:]), atts, cache.DefaultExpiration)

	return nil
}

// SaveBlockAttestations saves a list of block attestations in cache.
func (p *AttCaches) SaveBlockAttestations(atts []*ethpb.Attestation) error {
	for _, att := range atts {
		if err := p.SaveBlockAttestation(att); err != nil {
			return err
		}
	}

	return nil
}

// BlockAttestations returns the block attestations in cache.
func (p *AttCaches) BlockAttestations() []*ethpb.Attestation {
	atts := make([]*ethpb.Attestation, 0, p.blockAtt.ItemCount())
	for s, i := range p.blockAtt.Items() {
		// Type assertion for the worst case. This shouldn't happen.
		att, ok := i.Object.([]*ethpb.Attestation)
		if !ok {
			p.blockAtt.Delete(s)
			continue
		}
		atts = append(atts, att...)
	}

	return atts
}

// DeleteBlockAttestation deletes a block attestation in cache.
func (p *AttCaches) DeleteBlockAttestation(att *ethpb.Attestation) error {
	r, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	p.blockAtt.Delete(string(r[:]))

	return nil
}
