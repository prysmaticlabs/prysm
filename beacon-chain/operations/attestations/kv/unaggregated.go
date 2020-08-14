package kv

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
)

// SaveUnaggregatedAttestation saves an unaggregated attestation in cache.
func (p *AttCaches) SaveUnaggregatedAttestation(att *ethpb.Attestation) error {
	if att == nil {
		return nil
	}
	if helpers.IsAggregated(att) {
		return errors.New("attestation is aggregated")
	}

	r, err := hashFn(att.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	// Don't save the attestation if the bitfield has been contained in previous blocks.
	p.seenAggregatedAttLock.RLock()
	seenBits, ok := p.seenAggregatedAtt[r]
	p.seenAggregatedAttLock.RUnlock()
	if ok {
		for _, bit := range seenBits {
			if bit.Len() == att.AggregationBits.Len() && bit.Contains(att.AggregationBits) {
				return nil
			}
		}
	}

	r, err = hashFn(att)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}
	p.unAggregateAttLock.Lock()
	defer p.unAggregateAttLock.Unlock()
	p.unAggregatedAtt[r] = stateTrie.CopyAttestation(att) // Copied.

	return nil
}

// SaveUnaggregatedAttestations saves a list of unaggregated attestations in cache.
func (p *AttCaches) SaveUnaggregatedAttestations(atts []*ethpb.Attestation) error {
	for _, att := range atts {
		if err := p.SaveUnaggregatedAttestation(att); err != nil {
			return err
		}
	}

	return nil
}

// UnaggregatedAttestations returns all the unaggregated attestations in cache.
func (p *AttCaches) UnaggregatedAttestations() ([]*ethpb.Attestation, error) {
	p.unAggregateAttLock.RLock()
	unAggregatedAtts := p.unAggregatedAtt
	p.unAggregateAttLock.RUnlock()

	atts := make([]*ethpb.Attestation, 0, len(unAggregatedAtts))
	for _, att := range unAggregatedAtts {
		r, err := hashFn(att.Data)
		if err != nil {
			return nil, errors.Wrap(err, "could not tree hash attestation")
		}
		p.seenAggregatedAttLock.RLock()
		seenBits, ok := p.seenAggregatedAtt[r]
		p.seenAggregatedAttLock.RUnlock()
		if ok {
			for _, bit := range seenBits {
				if bit.Len() == att.AggregationBits.Len() && bit.Contains(att.AggregationBits) {
					if err := p.DeleteUnaggregatedAttestation(att); err != nil {
						return nil, err
					}
					continue
				}
			}
		}

		atts = append(atts, stateTrie.CopyAttestation(att) /* Copied */)
	}

	return atts, nil
}

// UnaggregatedAttestationsBySlotIndex returns the unaggregated attestations in cache,
// filtered by committee index and slot.
func (p *AttCaches) UnaggregatedAttestationsBySlotIndex(slot uint64, committeeIndex uint64) []*ethpb.Attestation {
	atts := make([]*ethpb.Attestation, 0)

	p.unAggregateAttLock.RLock()
	unAggregatedAtts := p.unAggregatedAtt
	defer p.unAggregateAttLock.RUnlock()
	for _, a := range unAggregatedAtts {
		if slot == a.Data.Slot && committeeIndex == a.Data.CommitteeIndex {
			atts = append(atts, a)
		}
	}

	return atts
}

// DeleteUnaggregatedAttestation deletes the unaggregated attestations in cache.
func (p *AttCaches) DeleteUnaggregatedAttestation(att *ethpb.Attestation) error {
	if att == nil {
		return nil
	}
	if helpers.IsAggregated(att) {
		return errors.New("attestation is aggregated")
	}

	r, err := hashFn(att)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	p.unAggregateAttLock.Lock()
	defer p.unAggregateAttLock.Unlock()
	delete(p.unAggregatedAtt, r)

	return nil
}

// UnaggregatedAttestationCount returns the number of unaggregated attestations key in the pool.
func (p *AttCaches) UnaggregatedAttestationCount() int {
	p.unAggregateAttLock.RLock()
	defer p.unAggregateAttLock.RUnlock()
	return len(p.unAggregatedAtt)
}
