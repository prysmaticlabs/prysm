package kv

import (
	"time"

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

	d, expTime, ok := p.aggregatedAtt.GetWithExpiration(string(r[:]))
	// If we have not seen the attestation data before, store in in cache with
	// the default expiration time out.
	if !ok {
		atts := []*ethpb.Attestation{att}
		p.aggregatedAtt.Set(string(r[:]), atts, cache.DefaultExpiration)
		return nil
	}

	atts, ok := d.([]*ethpb.Attestation)
	if !ok {
		return errors.New("cached value is not of type []*ethpb.Attestation")
	}

	atts, err = helpers.AggregateAttestations(append(atts, att))
	if err != nil {
		return err
	}

	// Delete attestation if the current time has passed the expiration time.
	if time.Now().Unix() >= expTime.Unix() {
		p.aggregatedAtt.Delete(string(r[:]))
		return nil
	}
	// Reset expiration time given how much time has passed.
	expDuration := time.Duration(expTime.Unix() - time.Now().Unix())
	p.aggregatedAtt.Set(string(r[:]), atts, expDuration*time.Second)

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
	// Delete all expired aggregated attestations before returning them.
	p.aggregatedAtt.DeleteExpired()

	atts := make([]*ethpb.Attestation, 0, p.aggregatedAtt.ItemCount())
	for s, i := range p.aggregatedAtt.Items() {
		// Type assertion for the worst case. This shouldn't happen.
		a, ok := i.Object.([]*ethpb.Attestation)
		if !ok {
			p.aggregatedAtt.Delete(s)
			continue
		}
		atts = append(atts, a...)
	}

	return atts
}

// AggregatedAttestationsBySlotIndex returns the aggregated attestations in cache,
// filtered by committee index and slot.
func (p *AttCaches) AggregatedAttestationsBySlotIndex(slot uint64, committeeIndex uint64) []*ethpb.Attestation {
	// Delete all expired aggregated attestations before returning them.
	p.aggregatedAtt.DeleteExpired()

	atts := make([]*ethpb.Attestation, 0, p.aggregatedAtt.ItemCount())
	for s, i := range p.aggregatedAtt.Items() {

		// Type assertion for the worst case. This shouldn't happen.
		a, ok := i.Object.([]*ethpb.Attestation)
		if !ok {
			p.aggregatedAtt.Delete(s)
			continue
		}

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
	a, expTime, ok := p.aggregatedAtt.GetWithExpiration(string(r[:]))
	if !ok {
		return nil
	}
	atts, ok := a.([]*ethpb.Attestation)
	if !ok {
		return errors.New("cached value is not of type []*ethpb.Attestation")
	}
	filtered := make([]*ethpb.Attestation, 0)
	for _, a := range atts {
		if !att.AggregationBits.Contains(a.AggregationBits) {
			filtered = append(filtered, a)
		}
	}
	if len(filtered) == 0 {
		p.aggregatedAtt.Delete(string(r[:]))
	} else {
		// Delete attestation if the current time has passed the expiration time.
		if time.Now().Unix() >= expTime.Unix() {
			p.aggregatedAtt.Delete(string(r[:]))
			return nil
		}
		// Reset expiration time given how much time has passed.
		expDuration := time.Duration(expTime.Unix() - time.Now().Unix())
		p.aggregatedAtt.Set(string(r[:]), filtered, expDuration*time.Second)
	}

	return nil
}

// HasAggregatedAttestation checks if the input attestations has already existed in cache.
func (p *AttCaches) HasAggregatedAttestation(att *ethpb.Attestation) (bool, error) {
	r, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		return false, errors.Wrap(err, "could not tree hash attestation")
	}

	if atts, ok := p.aggregatedAtt.Get(string(r[:])); ok {
		for _, a := range atts.([]*ethpb.Attestation) {
			if a.AggregationBits.Contains(att.AggregationBits) {
				return true, nil
			}
		}
	}

	if atts, ok := p.blockAtt.Get(string(r[:])); ok {
		for _, a := range atts.([]*ethpb.Attestation) {
			if a.AggregationBits.Contains(att.AggregationBits) {
				return true, nil
			}
		}
	}

	return false, nil
}

// AggregatedAttestationCount returns the number of aggregated attestations key in the pool.
func (p *AttCaches) AggregatedAttestationCount() int {
	return p.aggregatedAtt.ItemCount()
}
