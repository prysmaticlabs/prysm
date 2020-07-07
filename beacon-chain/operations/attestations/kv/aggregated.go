package kv

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	attaggregation "github.com/prysmaticlabs/prysm/shared/aggregation/attestations"
)

// AggregateUnaggregatedAttestations aggregates the unaggregated attestations and saves the
// newly aggregated attestations in the pool.
// It tracks the unaggregated attestations that weren't able to aggregate to prevent
// the deletion of unaggregated attestations in the pool.
func (p *AttCaches) AggregateUnaggregatedAttestations() error {
	unaggregatedAtts := p.UnaggregatedAttestations()
	attsByDataRoot := make(map[[32]byte][]*ethpb.Attestation, len(unaggregatedAtts))
	for _, att := range unaggregatedAtts {
		attDataRoot, err := stateutil.AttestationDataRoot(att.Data)
		if err != nil {
			return err
		}
		attsByDataRoot[attDataRoot] = append(attsByDataRoot[attDataRoot], att)
	}

	// Aggregate unaggregated attestations from the pool and save them in the pool.
	// Track the unaggregated attestations that aren't able to aggregate.
	leftOverUnaggregatedAtt := make(map[[32]byte]bool)
	for _, atts := range attsByDataRoot {
		aggregatedAtts := make([]*ethpb.Attestation, 0, len(atts))
		processedAtts, err := attaggregation.Aggregate(atts)
		if err != nil {
			return err
		}
		for _, att := range processedAtts {
			if helpers.IsAggregated(att) {
				aggregatedAtts = append(aggregatedAtts, att)
			} else {
				h, err := ssz.HashTreeRoot(att)
				if err != nil {
					return err
				}
				leftOverUnaggregatedAtt[h] = true
			}
		}
		if err := p.SaveAggregatedAttestations(aggregatedAtts); err != nil {
			return err
		}
	}

	// Remove the unaggregated attestations from the pool that were successfully aggregated.
	for _, att := range unaggregatedAtts {
		h, err := ssz.HashTreeRoot(att)
		if err != nil {
			return err
		}
		if leftOverUnaggregatedAtt[h] {
			continue
		}
		if err := p.DeleteUnaggregatedAttestation(att); err != nil {
			return err
		}
	}

	return nil
}

// SaveAggregatedAttestation saves an aggregated attestation in cache.
func (p *AttCaches) SaveAggregatedAttestation(att *ethpb.Attestation) error {
	if att == nil || att.Data == nil {
		return nil
	}
	if !helpers.IsAggregated(att) {
		return errors.New("attestation is not aggregated")
	}
	r, err := hashFn(att.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	copiedAtt := stateTrie.CopyAttestation(att)
	p.aggregatedAttLock.Lock()
	defer p.aggregatedAttLock.Unlock()
	atts, ok := p.aggregatedAtt[r]
	if !ok {
		atts := []*ethpb.Attestation{copiedAtt}
		p.aggregatedAtt[r] = atts
		return nil
	}

	atts, err = attaggregation.Aggregate(append(atts, copiedAtt))
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
	p.aggregatedAttLock.RLock()
	defer p.aggregatedAttLock.RUnlock()

	atts := make([]*ethpb.Attestation, 0, len(p.aggregatedAtt))
	for _, a := range p.aggregatedAtt {
		atts = append(atts, a...)
	}

	return atts
}

// AggregatedAttestationsBySlotIndex returns the aggregated attestations in cache,
// filtered by committee index and slot.
func (p *AttCaches) AggregatedAttestationsBySlotIndex(slot uint64, committeeIndex uint64) []*ethpb.Attestation {
	atts := make([]*ethpb.Attestation, 0)

	p.aggregatedAttLock.RLock()
	defer p.aggregatedAttLock.RUnlock()
	for _, a := range p.aggregatedAtt {
		if slot == a[0].Data.Slot && committeeIndex == a[0].Data.CommitteeIndex {
			atts = append(atts, a...)
		}
	}

	return atts
}

// DeleteAggregatedAttestation deletes the aggregated attestations in cache.
func (p *AttCaches) DeleteAggregatedAttestation(att *ethpb.Attestation) error {
	if att == nil || att.Data == nil {
		return nil
	}
	if !helpers.IsAggregated(att) {
		return errors.New("attestation is not aggregated")
	}
	r, err := hashFn(att.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation data")
	}

	p.aggregatedAttLock.Lock()
	defer p.aggregatedAttLock.Unlock()
	attList, ok := p.aggregatedAtt[r]
	if !ok {
		return nil
	}

	filtered := make([]*ethpb.Attestation, 0)
	for _, a := range attList {
		if att.AggregationBits.Len() == a.AggregationBits.Len() && !att.AggregationBits.Contains(a.AggregationBits) {
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
	if att == nil || att.Data == nil {
		return false, nil
	}
	r, err := hashFn(att.Data)
	if err != nil {
		return false, errors.Wrap(err, "could not tree hash attestation")
	}

	p.aggregatedAttLock.RLock()
	defer p.aggregatedAttLock.RUnlock()
	if atts, ok := p.aggregatedAtt[r]; ok {
		for _, a := range atts {
			if a.AggregationBits.Contains(att.AggregationBits) {
				return true, nil
			}
		}
	}

	p.blockAttLock.RLock()
	defer p.blockAttLock.RUnlock()
	if atts, ok := p.blockAtt[r]; ok {
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
	p.aggregatedAttLock.RLock()
	defer p.aggregatedAttLock.RUnlock()
	return len(p.aggregatedAtt)
}
