package kv

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// SaveBlockAttestation saves an block attestation in cache.
func (c *AttCaches) SaveBlockAttestation(att ethpb.Att) error {
	if att == nil {
		return nil
	}
	r, err := hashFn(att.GetData())
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}
	key := versionAndDataRoot{att.Version(), r}

	c.blockAttLock.Lock()
	defer c.blockAttLock.Unlock()
	atts, ok := c.blockAtt[key]
	if !ok {
		atts = make([]ethpb.Att, 0, 1)
	}

	// Ensure that this attestation is not already fully contained in an existing attestation.
	for _, a := range atts {
		if c, err := a.GetAggregationBits().Contains(att.GetAggregationBits()); err != nil {
			return err
		} else if c {
			return nil
		}
	}

	c.blockAtt[key] = append(atts, att.Copy())

	return nil
}

// BlockAttestations returns the block attestations in cache.
func (c *AttCaches) BlockAttestations() []ethpb.Att {
	atts := make([]ethpb.Att, 0)

	c.blockAttLock.RLock()
	defer c.blockAttLock.RUnlock()
	for _, att := range c.blockAtt {
		atts = append(atts, att...)
	}

	return atts
}

// DeleteBlockAttestation deletes a block attestation in cache.
func (c *AttCaches) DeleteBlockAttestation(att ethpb.Att) error {
	if att == nil {
		return nil
	}
	r, err := hashFn(att.GetData())
	if err != nil {
		return errors.Wrap(err, "could not tree hash attestation")
	}

	c.blockAttLock.Lock()
	defer c.blockAttLock.Unlock()
	delete(c.blockAtt, versionAndDataRoot{att.Version(), r})

	return nil
}
