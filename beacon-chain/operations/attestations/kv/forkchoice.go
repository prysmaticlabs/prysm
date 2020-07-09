package kv

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
)

// SaveForkchoiceAttestation saves an forkchoice attestation in cache.
func (p *AttCaches) SaveForkchoiceAttestation(att *ethpb.Attestation) error {
	if att == nil {
		return nil
	}
	r, err := hashFn(att)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	p.forkchoiceAttLock.Lock()
	defer p.forkchoiceAttLock.Unlock()
	p.forkchoiceAtt[r] = stateTrie.CopyAttestation(att) // Copied.

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
	p.forkchoiceAttLock.RLock()
	defer p.forkchoiceAttLock.RUnlock()

	atts := make([]*ethpb.Attestation, 0, len(p.forkchoiceAtt))
	for _, att := range p.forkchoiceAtt {
		atts = append(atts, stateTrie.CopyAttestation(att) /* Copied */)
	}

	return atts
}

// DeleteForkchoiceAttestation deletes a forkchoice attestation in cache.
func (p *AttCaches) DeleteForkchoiceAttestation(att *ethpb.Attestation) error {
	if att == nil {
		return nil
	}
	r, err := hashFn(att)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	p.forkchoiceAttLock.Lock()
	defer p.forkchoiceAttLock.Unlock()
	delete(p.forkchoiceAtt, r)

	return nil
}
