package kv

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
)

// SaveForkchoiceAttestation saves an forkchoice attestation in cache.
func (c *AttCaches) SaveForkchoiceAttestation(att interfaces.Attestation) error {
	if att == nil {
		return nil
	}
	r, err := hashFn(att)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	att = interfaces.CopyAttestation(att)
	c.forkchoiceAttLock.Lock()
	defer c.forkchoiceAttLock.Unlock()
	c.forkchoiceAtt[r] = att

	return nil
}

// SaveForkchoiceAttestations saves a list of forkchoice attestations in cache.
func (c *AttCaches) SaveForkchoiceAttestations(atts []interfaces.Attestation) error {
	for _, att := range atts {
		if err := c.SaveForkchoiceAttestation(att); err != nil {
			return err
		}
	}

	return nil
}

// ForkchoiceAttestations returns the forkchoice attestations in cache.
func (c *AttCaches) ForkchoiceAttestations() []interfaces.Attestation {
	c.forkchoiceAttLock.RLock()
	defer c.forkchoiceAttLock.RUnlock()

	atts := make([]interfaces.Attestation, 0, len(c.forkchoiceAtt))
	for _, att := range c.forkchoiceAtt {
		atts = append(atts, interfaces.CopyAttestation(att) /* Copied */)
	}

	return atts
}

// DeleteForkchoiceAttestation deletes a forkchoice attestation in cache.
func (c *AttCaches) DeleteForkchoiceAttestation(att interfaces.Attestation) error {
	if att == nil {
		return nil
	}
	r, err := hashFn(att)
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	c.forkchoiceAttLock.Lock()
	defer c.forkchoiceAttLock.Unlock()
	delete(c.forkchoiceAtt, r)

	return nil
}

// ForkchoiceAttestationCount returns the number of fork choice attestations key in the pool.
func (c *AttCaches) ForkchoiceAttestationCount() int {
	c.forkchoiceAttLock.RLock()
	defer c.forkchoiceAttLock.RUnlock()
	return len(c.forkchoiceAtt)
}
