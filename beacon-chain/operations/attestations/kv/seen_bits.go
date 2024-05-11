package kv

import (
	"strconv"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

func (c *AttCaches) insertSeenBit(att ethpb.Att) error {
	var h [32]byte
	var err error
	if att.Version() == version.Phase0 {
		h, err = hashFn(att.GetData())
		if err != nil {
			return err
		}
	} else {
		data := ethpb.CopyAttestationData(att.GetData())
		data.CommitteeIndex = primitives.CommitteeIndex(att.GetCommitteeBitsVal().BitIndices()[0])
		h, err = hashFn(data)
		if err != nil {
			return err
		}
	}
	r := h

	v, ok := c.seenAtt.Get(string(r[:]) + strconv.Itoa(att.Version()))
	if ok {
		seenBits, ok := v.([]bitfield.Bitlist)
		if !ok {
			return errors.New("could not convert to bitlist type")
		}
		alreadyExists := false
		for _, bit := range seenBits {
			if c, err := bit.Contains(att.GetAggregationBits()); err != nil {
				return err
			} else if c {
				alreadyExists = true
				break
			}
		}
		if !alreadyExists {
			seenBits = append(seenBits, att.GetAggregationBits())
		}
		c.seenAtt.Set(string(r[:])+strconv.Itoa(att.Version()), seenBits, cache.DefaultExpiration /* one epoch */)
		return nil
	}

	c.seenAtt.Set(string(r[:])+strconv.Itoa(att.Version()), []bitfield.Bitlist{att.GetAggregationBits()}, cache.DefaultExpiration /* one epoch */)
	return nil
}

func (c *AttCaches) hasSeenBit(att ethpb.Att) (bool, error) {
	var h [32]byte
	var err error
	if att.Version() == version.Phase0 {
		h, err = hashFn(att.GetData())
		if err != nil {
			return false, err
		}
	} else {
		data := ethpb.CopyAttestationData(att.GetData())
		data.CommitteeIndex = primitives.CommitteeIndex(att.GetCommitteeBitsVal().BitIndices()[0])
		h, err = hashFn(data)
		if err != nil {
			return false, err
		}
	}
	r := h

	v, ok := c.seenAtt.Get(string(r[:]) + strconv.Itoa(att.Version()))
	if ok {
		seenBits, ok := v.([]bitfield.Bitlist)
		if !ok {
			return false, errors.New("could not convert to bitlist type")
		}
		for _, bit := range seenBits {
			if c, err := bit.Contains(att.GetAggregationBits()); err != nil {
				return false, err
			} else if c {
				return true, nil
			}
		}
	}
	return false, nil
}
