package cache

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/operations/attestations/forkchoice"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation"
)

type attGroup struct {
	slot     primitives.Slot
	local    ethpb.Att
	external []ethpb.Att
}

type AttestationCache struct {
	attestations map[attestation.Id]*attGroup
	sync.RWMutex
	forkchoiceAtts *forkchoice.Attestations
}

func NewAttestationCache() *AttestationCache {
	return &AttestationCache{
		attestations:   make(map[attestation.Id]*attGroup),
		forkchoiceAtts: forkchoice.New(),
	}
}

func (c *AttestationCache) Add(att ethpb.Att) error {
	if att.IsNil() {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	id, err := attestation.NewId(att, attestation.Data)
	if err != nil {
		return errors.Wrapf(err, "could not create attestation ID")
	}

	group := c.attestations[id]
	if group == nil {
		group = &attGroup{
			slot: att.GetData().Slot,
		}
		c.attestations[id] = group
	}

	if att.IsAggregated() {
		group.external = append(group.external, att.Clone())
		return nil
	}

	local := c.attestations[id].local
	if local == nil {
		local = att.Clone()
	}
	bit := att.GetAggregationBits().BitIndices()[0]
	if local.GetAggregationBits().BitAt(uint64(bit)) {
		return nil
	}
	sig, err := aggregateSig(local, att)
	if err != nil {
		return errors.Wrapf(err, "could not aggregate signatures")
	}

	local.GetAggregationBits().SetBitAt(uint64(bit), true)
	local.SetSignature(sig)

	return nil
}

func (c *AttestationCache) GetAll() []ethpb.Att {
	c.RLock()
	defer c.RUnlock()

	var result []ethpb.Att
	for _, group := range c.attestations {
		if group.local != nil {
			result = append(result, group.local)
		}
		for _, a := range group.external {
			result = append(result, a)
		}
	}
	return result
}

func (c *AttestationCache) Count() int {
	c.RLock()
	defer c.RUnlock()

	count := 0
	for _, group := range c.attestations {
		if group.local != nil {
			count++
		}
		count += len(group.external)
	}
	return count
}

func (c *AttestationCache) DeleteCovered(att ethpb.Att) error {
	if att.IsNil() {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	id, err := attestation.NewId(att, attestation.Data)
	if err != nil {
		return errors.Wrapf(err, "could not create attestation ID")
	}

	group := c.attestations[id]
	if group == nil {
		return nil
	}

	local := group.local
	if local != nil {
		if covered, err := att.GetAggregationBits().Contains(local.GetAggregationBits()); err != nil {
			return err
		} else if covered {
			group.local = nil
		}
	}

	attsToKeep := make([]ethpb.Att, 0, len(group.external))
	for _, a := range group.external {
		if covered, err := att.GetAggregationBits().Contains(a.GetAggregationBits()); err != nil {
			return err
		} else if !covered {
			attsToKeep = append(attsToKeep, a)
		}
	}
	group.external = attsToKeep

	if group.local == nil && len(group.external) == 0 {
		delete(c.attestations, id)
	}

	return nil
}

func (c *AttestationCache) PruneBefore(slot primitives.Slot) uint64 {
	c.Lock()
	defer c.Unlock()

	var pruneCount uint64
	for id, group := range c.attestations {
		if group.slot < slot {
			if group.local != nil {
				pruneCount++
			}
			pruneCount += uint64(len(group.external))
			delete(c.attestations, id)
		}
	}
	return pruneCount
}

func (c *AttestationCache) AggregateIsRedundant(att ethpb.Att) (bool, error) {
	if att.IsNil() {
		return true, nil
	}

	c.RLock()
	defer c.RUnlock()

	id, err := attestation.NewId(att, attestation.Data)
	if err != nil {
		return true, errors.Wrapf(err, "could not create attestation ID")
	}

	group := c.attestations[id]
	if group == nil {
		return false, nil
	}

	for _, a := range group.external {
		if redundant, err := a.GetAggregationBits().Contains(att.GetAggregationBits()); err != nil {
			return true, err
		} else if redundant {
			return true, nil
		}
	}

	return false, nil
}

// SaveForkchoiceAttestation saves a forkchoice attestation.
func (c *AttestationCache) SaveForkchoiceAttestation(att ethpb.Att) error {
	return c.forkchoiceAtts.SaveForkchoiceAttestation(att)
}

// SaveForkchoiceAttestations saves forkchoice attestations.
func (c *AttestationCache) SaveForkchoiceAttestations(att []ethpb.Att) error {
	return c.forkchoiceAtts.SaveForkchoiceAttestations(att)
}

// ForkchoiceAttestations returns all forkchoice attestations.
func (c *AttestationCache) ForkchoiceAttestations() []ethpb.Att {
	return c.forkchoiceAtts.ForkchoiceAttestations()
}

// DeleteForkchoiceAttestation deletes a forkchoice attestation.
func (c *AttestationCache) DeleteForkchoiceAttestation(att ethpb.Att) error {
	return c.forkchoiceAtts.DeleteForkchoiceAttestation(att)
}

func GetBySlotAndCommitteeIndex[T ethpb.Att](c *AttestationCache, slot primitives.Slot, committeeIndex primitives.CommitteeIndex) []T {
	c.RLock()
	defer c.RUnlock()

	var result []T

	for _, group := range c.attestations {
		local, ok := group.local.(T)
		if ok {
			if local.GetData().Slot == slot && local.CommitteeBitsVal().BitAt(uint64(committeeIndex)) {
				result = append(result, local)
				for _, a := range group.external {
					a, ok := a.(T)
					if ok {
						result = append(result, a)
					}
				}
			}
		} else if len(group.external) > 0 {
			a, ok := group.external[0].(T)
			if ok && a.GetData().Slot == slot && a.CommitteeBitsVal().BitAt(uint64(committeeIndex)) {
				for _, a := range group.external {
					a, ok := a.(T)
					if ok {
						result = append(result, a)
					}
				}
			}
		}
	}

	return result
}

func aggregateSig(agg ethpb.Att, att ethpb.Att) ([]byte, error) {
	aggSig, err := bls.SignatureFromBytesNoValidation(agg.GetSignature())
	if err != nil {
		return nil, err
	}
	attSig, err := bls.SignatureFromBytesNoValidation(att.GetSignature())
	if err != nil {
		return nil, err
	}
	return bls.AggregateSignatures([]bls.Signature{aggSig, attSig}).Marshal(), nil
}
