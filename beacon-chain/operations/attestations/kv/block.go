package kv

import (
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
)

// SaveBlockAttestation saves an block attestation in cache.
func (c *AttCaches) SaveBlockAttestation(att blocks.ROAttestation) error {
	c.blockAttLock.Lock()
	defer c.blockAttLock.Unlock()
	atts, ok := c.blockAtt[att.DataId()]
	if !ok {
		atts = make([]blocks.ROAttestation, 0, 1)
	}

	// Ensure that this attestation is not already fully contained in an existing attestation.
	for _, a := range atts {
		if c, err := a.GetAggregationBits().Contains(att.GetAggregationBits()); err != nil {
			return err
		} else if c {
			return nil
		}
	}

	c.blockAtt[att.DataId()] = append(atts, att.Copy())

	return nil
}

// BlockAttestations returns the block attestations in cache.
func (c *AttCaches) BlockAttestations() []blocks.ROAttestation {
	atts := make([]blocks.ROAttestation, 0)

	c.blockAttLock.RLock()
	defer c.blockAttLock.RUnlock()
	for _, att := range c.blockAtt {
		atts = append(atts, att...)
	}

	return atts
}

// DeleteBlockAttestation deletes a block attestation in cache.
func (c *AttCaches) DeleteBlockAttestation(att blocks.ROAttestation) error {
	c.blockAttLock.Lock()
	defer c.blockAttLock.Unlock()
	delete(c.blockAtt, att.DataId())

	return nil
}
