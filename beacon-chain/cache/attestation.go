package cache

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation"
)

type AttestationCache struct {
	attestations map[attestation.Id]ethpb.Att
	sync.Mutex
}

func NewAttestationCache() *AttestationCache {
	return &AttestationCache{
		attestations: make(map[attestation.Id]ethpb.Att),
	}
}

func (c *AttestationCache) Add(att ethpb.Att) error {
	c.Lock()
	defer c.Unlock()

	id, err := attestation.NewId(att, attestation.Data)
	if err != nil {
		return errors.Wrapf(err, "could not create attestation ID")
	}
	agg := c.attestations[id]
	if agg == nil {
		c.attestations[id] = att
		return nil
	}
	bit := att.GetAggregationBits().BitIndices()[0]
	if agg.GetAggregationBits().BitAt(uint64(bit)) {
		return nil
	}
	sig, err := aggregateSig(agg, att)
	if err != nil {
		return errors.Wrapf(err, "could not aggregate signatures")
	}

	agg.GetAggregationBits().SetBitAt(uint64(bit), true)
	agg.SetSignature(sig)

	return nil
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
