package kv

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
)

// SaveBlockAttestation saves an block attestation in cache.
func (p *AttCaches) SaveBlockAttestation(att *ethpb.Attestation) error {
	if att == nil {
		return nil
	}
	r, err := hashFn(att.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	p.blockAttLock.Lock()
	defer p.blockAttLock.Unlock()
	atts, ok := p.blockAtt[r]
	if !ok {
		atts = make([]*ethpb.Attestation, 0)
	}

	// Ensure that this attestation is not already fully contained in an existing attestation.
	for _, a := range atts {
		if a.AggregationBits.Contains(att.AggregationBits) {
			return nil
		}
	}

	p.blockAtt[r] = append(atts, stateTrie.CopyAttestation(att))

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
	atts := make([]*ethpb.Attestation, 0)

	p.blockAttLock.RLock()
	defer p.blockAttLock.RUnlock()
	for _, att := range p.blockAtt {
		atts = append(atts, att...)
	}

	return atts
}

// DeleteBlockAttestation deletes a block attestation in cache.
func (p *AttCaches) DeleteBlockAttestation(att *ethpb.Attestation) error {
	if att == nil {
		return nil
	}
	r, err := hashFn(att.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	p.blockAttLock.Lock()
	defer p.blockAttLock.Unlock()
	delete(p.blockAtt, r)

	return nil
}
