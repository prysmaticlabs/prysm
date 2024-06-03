package kv

import (
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
)

// SaveForkchoiceAttestation saves an forkchoice attestation in cache.
func (c *AttCaches) SaveForkchoiceAttestation(att blocks.ROAttestation) error {
	c.forkchoiceAttLock.Lock()
	defer c.forkchoiceAttLock.Unlock()
	c.forkchoiceAtt[att.Id()] = att

	return nil
}

// SaveForkchoiceAttestations saves a list of forkchoice attestations in cache.
func (c *AttCaches) SaveForkchoiceAttestations(atts []blocks.ROAttestation) error {
	for _, att := range atts {
		if err := c.SaveForkchoiceAttestation(att); err != nil {
			return err
		}
	}

	return nil
}

// ForkchoiceAttestations returns the forkchoice attestations in cache.
func (c *AttCaches) ForkchoiceAttestations() []blocks.ROAttestation {
	c.forkchoiceAttLock.RLock()
	defer c.forkchoiceAttLock.RUnlock()

	atts := make([]blocks.ROAttestation, 0, len(c.forkchoiceAtt))
	for _, att := range c.forkchoiceAtt {
		atts = append(atts, att.Copy())
	}

	return atts
}

// DeleteForkchoiceAttestation deletes a forkchoice attestation in cache.
func (c *AttCaches) DeleteForkchoiceAttestation(att blocks.ROAttestation) error {
	c.forkchoiceAttLock.Lock()
	defer c.forkchoiceAttLock.Unlock()
	delete(c.forkchoiceAtt, att.Id())

	return nil
}

// ForkchoiceAttestationCount returns the number of fork choice attestations key in the pool.
func (c *AttCaches) ForkchoiceAttestationCount() int {
	c.forkchoiceAttLock.RLock()
	defer c.forkchoiceAttLock.RUnlock()
	return len(c.forkchoiceAtt)
}
