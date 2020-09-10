package kv

import (
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
)

func (p *AttCaches) insertSeenBit(att *ethpb.Attestation) error {
	r, err := hashFn(att.Data)
	if err != nil {
		return err
	}

	v, ok := p.seenAtt.Get(string(r[:]))
	if ok {
		seenBits, ok := v.([]bitfield.Bitlist)
		if !ok {
			return errors.New("could not convert to bitlist type")
		}
		alreadyExists := false
		for _, bit := range seenBits {
			if bit.Len() == att.AggregationBits.Len() && bit.Contains(att.AggregationBits) {
				alreadyExists = true
				break
			}
		}
		if !alreadyExists {
			seenBits = append(seenBits, att.AggregationBits)
		}
		p.seenAtt.Set(string(r[:]), seenBits, cache.DefaultExpiration /* one epoch */)
		return nil
	}

	p.seenAtt.Set(string(r[:]), []bitfield.Bitlist{att.AggregationBits}, cache.DefaultExpiration /* one epoch */)
	return nil
}

func (p *AttCaches) hasSeenBit(att *ethpb.Attestation) (bool, error) {
	r, err := hashFn(att.Data)
	if err != nil {
		return false, err
	}

	v, ok := p.seenAtt.Get(string(r[:]))
	if ok {
		seenBits, ok := v.([]bitfield.Bitlist)
		if !ok {
			return false, errors.New("could not convert to bitlist type")
		}
		for _, bit := range seenBits {
			if bit.Len() == att.AggregationBits.Len() && bit.Contains(att.AggregationBits) {
				return true, nil
			}
		}
	}
	return false, nil
}
