package verification

import (
	"context"

	lru "github.com/hashicorp/golang-lru"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	lruwrpr "github.com/prysmaticlabs/prysm/v4/cache/lru"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/network/forks"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

func NewSignatureCache(sg *stategen.State) *SignatureCache {
	return &SignatureCache{verified: lruwrpr.New(256)}
}

type SignatureCache struct {
	sg       *stategen.State
	verified *lru.Cache
}

type sigCacheKey struct {
	root     [32]byte
	parent   [32]byte
	sig      [96]byte
	proposer primitives.ValidatorIndex
	slot     primitives.Slot
}

func (c *SignatureCache) VerifySignature(ctx context.Context, sig [96]byte, root, parent [32]byte, proposer primitives.ValidatorIndex, slot primitives.Slot) error {
	key := sigCacheKey{root: root, parent: parent, sig: sig, proposer: proposer, slot: slot}
	_, verified := c.verified.Get(key)
	if verified {
		return nil
	}
	ps, err := c.sg.StateByRoot(ctx, parent)
	e := slots.ToEpoch(slot)
	fork, err := forks.Fork(e)
	if err != nil {
		return err
	}
	domain, err := signing.Domain(fork, e, params.BeaconConfig().DomainBlobSidecar, ps.GenesisValidatorsRoot())
	if err != nil {
		return err
	}
	pv, err := ps.ValidatorAtIndex(proposer)
	if err != nil {
		return err
	}
	pb, err := bls.PublicKeyFromBytes(pv.PublicKey)
	if err != nil {
		return err
	}
	s, err := bls.SignatureFromBytes(sig[:])
	if err != nil {
		return err
	}
	sr, err := signing.ComputeSigningRootWithRoot(root, domain)
	if err != nil {
		return err
	}
	if !s.Verify(pb, sr[:]) {
		return signing.ErrSigFailedToVerify
	}

	c.verified.Add(key, struct{}{})
	return nil
}
