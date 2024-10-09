package cache

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/operations/attestations/forkchoice"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation"
	log "github.com/sirupsen/logrus"
)

type attGroup struct {
	slot primitives.Slot
	atts []ethpb.Att
}

type AttestationCache struct {
	atts map[attestation.Id]*attGroup
	sync.RWMutex
	forkchoiceAtts *forkchoice.Attestations
}

func NewAttestationCache() *AttestationCache {
	return &AttestationCache{
		atts:           make(map[attestation.Id]*attGroup),
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

	group := c.atts[id]
	if group == nil {
		group = &attGroup{
			slot: att.GetData().Slot,
			atts: []ethpb.Att{att},
		}
		c.atts[id] = group
		return nil
	}

	if att.IsAggregated() {
		group.atts = append(group.atts, att.Clone())
		return nil
	}

	// This should never happen because we return early for a new group.
	if len(group.atts) == 0 {
		log.Error("Attestation group contains no attestations, skipping insertion")
		return nil
	}

	a := group.atts[0]
	bit := att.GetAggregationBits().BitIndices()[0]
	if a.GetAggregationBits().BitAt(uint64(bit)) {
		return nil
	}
	sig, err := aggregateSig(a, att)
	if err != nil {
		return errors.Wrapf(err, "could not aggregate signatures")
	}

	a.GetAggregationBits().SetBitAt(uint64(bit), true)
	a.SetSignature(sig)

	return nil
}

func (c *AttestationCache) GetAll() []ethpb.Att {
	c.RLock()
	defer c.RUnlock()

	var result []ethpb.Att
	for _, group := range c.atts {
		result = append(result, group.atts...)
	}
	return result
}

func (c *AttestationCache) Count() int {
	c.RLock()
	defer c.RUnlock()

	count := 0
	for _, group := range c.atts {
		count += len(group.atts)
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

	group := c.atts[id]
	if group == nil {
		return nil
	}

	idx := 0
	for _, a := range group.atts {
		if covered, err := att.GetAggregationBits().Contains(a.GetAggregationBits()); err != nil {
			return err
		} else if !covered {
			group.atts[idx] = a
			idx++
		}
	}
	group.atts = group.atts[:idx]

	if len(group.atts) == 0 {
		delete(c.atts, id)
	}

	return nil
}

func (c *AttestationCache) PruneBefore(slot primitives.Slot) uint64 {
	c.Lock()
	defer c.Unlock()

	var pruneCount int
	for id, group := range c.atts {
		if group.slot < slot {
			pruneCount += len(group.atts)
			delete(c.atts, id)
		}
	}
	return uint64(pruneCount)
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

	group := c.atts[id]
	if group == nil {
		return false, nil
	}

	for _, a := range group.atts {
		if redundant, err := a.GetAggregationBits().Contains(att.GetAggregationBits()); err != nil {
			return true, err
		} else if redundant {
			return true, nil
		}
	}

	return false, nil
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

	for _, group := range c.atts {
		if len(group.atts) > 0 {
			// We can safely compare the first attestation because all attestations in a group
			// must have the same slot and committee index, since they are under the same key.
			a, ok := group.atts[0].(T)
			if ok && a.GetData().Slot == slot && a.CommitteeBitsVal().BitAt(uint64(committeeIndex)) {
				for _, a := range group.atts {
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
