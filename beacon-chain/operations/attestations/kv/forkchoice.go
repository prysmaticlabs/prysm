package kv

import (
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
)

// SaveForkchoiceAttestation saves an forkchoice attestation in cache.
func (p *AttCaches) SaveForkchoiceAttestation(att *ethpb.Attestation) error {
	r, err := ssz.HashTreeRoot(att)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	// DefaultExpiration is set to what was given to New(). In this case
	// it's one epoch.
	p.forkchoiceAtt.Set(string(r[:]), att, cache.DefaultExpiration)

	return nil
}

// SaveForkchoiceAttestations saves a list of forkchoice attestations in cache.
func (p *AttCaches) SaveForkchoiceAttestations(atts []*ethpb.Attestation) error {
	for _, att := range atts {
		if err := p.SaveForkchoiceAttestation(att); err != nil {
			return err
		}
	}

	return nil
}

// ForkchoiceAttestations returns the forkchoice attestations in cache.
func (p *AttCaches) ForkchoiceAttestations() []*ethpb.Attestation {
	atts := make([]*ethpb.Attestation, 0, p.forkchoiceAtt.ItemCount())
	for s, i := range p.forkchoiceAtt.Items() {
		// Type assertion for the worst case. This shouldn't happen.
		att, ok := i.Object.(*ethpb.Attestation)
		if !ok {
			p.forkchoiceAtt.Delete(s)
			continue
		}
		atts = append(atts, att)
	}

	return atts
}

// DeleteForkchoiceAttestation deletes a forkchoice attestation in cache.
func (p *AttCaches) DeleteForkchoiceAttestation(att *ethpb.Attestation) error {
	r, err := ssz.HashTreeRoot(att)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	p.forkchoiceAtt.Delete(string(r[:]))

	return nil
}
