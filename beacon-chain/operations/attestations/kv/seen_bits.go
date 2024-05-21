package kv

import (
	"fmt"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
)

func (c *AttCaches) insertSeenBit(att interfaces.Attestation) error {
	r, err := hashFn(att.GetData())
	if err != nil {
		return err
	}

	v, ok := c.seenAtt.Get(string(r[:]))
	if ok {
		seenBits, ok := v.([]bitfield.Bitlist)
		if !ok {
			return errors.New("could not convert to bitlist type")
		}
		alreadyExists := false
		for _, bit := range seenBits {
			if c, err := bit.Contains(att.GetAggregationBits()); err != nil {
				return fmt.Errorf("failed to check seen bits on attestation when inserting bit: %w", err)
			} else if c {
				alreadyExists = true
				break
			}
		}
		if !alreadyExists {
			seenBits = append(seenBits, att.GetAggregationBits())
		}
		c.seenAtt.Set(string(r[:]), seenBits, cache.DefaultExpiration /* one epoch */)
		return nil
	}

	c.seenAtt.Set(string(r[:]), []bitfield.Bitlist{att.GetAggregationBits()}, cache.DefaultExpiration /* one epoch */)
	return nil
}

func (c *AttCaches) hasSeenBit(att interfaces.Attestation) (bool, error) {
	r, err := hashFn(att.GetData())
	if err != nil {
		return false, err
	}

	v, ok := c.seenAtt.Get(string(r[:]))
	if ok {
		seenBits, ok := v.([]bitfield.Bitlist)
		if !ok {
			return false, errors.New("could not convert to bitlist type")
		}
		for _, bit := range seenBits {
			if c, err := bit.Contains(att.GetAggregationBits()); err != nil {
				return false, fmt.Errorf("failed to check seen bits on attestation when reading bit: %w", err)
			} else if c {
				return true, nil
			}
		}
	}
	return false, nil
}
