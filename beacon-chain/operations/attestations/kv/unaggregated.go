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

	seen, err := p.hasSeenBit(att)
	if err != nil {
		return err
	}
	if seen {
		return nil
	}

	r, err := hashFn(att)
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
	p.unAggregateAttLock.Lock()
	defer p.unAggregateAttLock.Unlock()
	unAggregatedAtts := p.unAggregatedAtt
	atts := make([]*ethpb.Attestation, 0, len(unAggregatedAtts))
	for _, att := range unAggregatedAtts {
		seen, err := p.hasSeenBit(att)
		if err != nil {
			return nil, err
		}
		if !seen {
			atts = append(atts, stateTrie.CopyAttestation(att) /* Copied */)
		}
	}
	return atts, nil
}

// UnaggregatedAttestationsBySlotIndex returns the unaggregated attestations in cache,
// filtered by committee index and slot.
func (p *AttCaches) UnaggregatedAttestationsBySlotIndex(slot, committeeIndex uint64) []*ethpb.Attestation {
	atts := make([]*ethpb.Attestation, 0)

	p.unAggregateAttLock.RLock()
	defer p.unAggregateAttLock.RUnlock()

	unAggregatedAtts := p.unAggregatedAtt
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

	if err := p.insertSeenBit(att); err != nil {
		return err
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

// DeleteSeenUnaggregatedAttestations deletes the unaggregated attestations in cache
// that have been already processed once. Returns number of attestations deleted.
func (p *AttCaches) DeleteSeenUnaggregatedAttestations() (int, error) {
	p.unAggregateAttLock.Lock()
	defer p.unAggregateAttLock.Unlock()

	count := 0
	for _, att := range p.unAggregatedAtt {
		if att == nil || helpers.IsAggregated(att) {
			continue
		}
		if seen, err := p.hasSeenBit(att); err == nil && seen {
			r, err := hashFn(att)
			if err != nil {
				return count, errors.Wrap(err, "could not tree hash attestation")
			}
			delete(p.unAggregatedAtt, r)
			count++
		}
	}
	return count, nil
}

// UnaggregatedAttestationCount returns the number of unaggregated attestations key in the pool.
func (p *AttCaches) UnaggregatedAttestationCount() int {
	p.unAggregateAttLock.RLock()
	defer p.unAggregateAttLock.RUnlock()
	return len(p.unAggregatedAtt)
}
