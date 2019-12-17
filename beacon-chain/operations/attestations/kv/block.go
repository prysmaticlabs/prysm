package kv

import (
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
)

// SaveBlockAttestation saves an block attestation in cache.
func (p *AttCaches) SaveBlockAttestation(att *ethpb.Attestation) error {
	r, err := ssz.HashTreeRoot(att)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	// DefaultExpiration is set to what was given to New(). In this case
	// it's one epoch.
	p.blockAtt.Set(string(r[:]), att, cache.DefaultExpiration)

	return nil
}

// SaveBlockAttestations saves a list of block attestation in cache.
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
		att, ok := i.Object.(*ethpb.Attestation)
		if !ok {
			p.blockAtt.Delete(s)
		}
		atts = append(atts, att)
	}

	return atts
}

// DeleteBlockAttestation deletes a block attestation in cache.
func (p *AttCaches) DeleteBlockAttestation(att *ethpb.Attestation) error {
	r, err := ssz.HashTreeRoot(att)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	p.blockAtt.Delete(string(r[:]))

	return nil
}
