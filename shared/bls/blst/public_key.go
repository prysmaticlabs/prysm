package blst

import (
	"fmt"

	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bls/iface"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	blst "github.com/supranational/blst/bindings/go"
)

var maxKeys = int64(100000)
var pubkeyCache, _ = ristretto.NewCache(&ristretto.Config{
	NumCounters: maxKeys,
	MaxCost:     1 << 22, // ~4mb is cache max size
	BufferItems: 64,
})

// PublicKey used in the BLS signature scheme.
type PublicKey struct {
	p *blstPublicKey
}

// PublicKeyFromBytes creates a BLS public key from a  BigEndian byte slice.
func PublicKeyFromBytes(pubKey []byte) (iface.PublicKey, error) {
	if featureconfig.Get().SkipBLSVerify {
		return &PublicKey{}, nil
	}
	if len(pubKey) != params.BeaconConfig().BLSPubkeyLength {
		return nil, fmt.Errorf("public key must be %d bytes", params.BeaconConfig().BLSPubkeyLength)
	}
	if cv, ok := pubkeyCache.Get(string(pubKey)); ok {
		return cv.(*PublicKey).Copy(), nil
	}

	p := new(blstPublicKey).Uncompress(pubKey)
	if p == nil {
		return nil, errors.New("could not unmarshal bytes into public key")
	}
	pubKeyObj := &PublicKey{p: p}
	copiedKey := pubKeyObj.Copy()
	pubkeyCache.Set(string(pubKey), copiedKey, 48)
	return pubKeyObj, nil
}

// AggregatePublicKeys aggregates the provided raw public keys into a single key.
func AggregatePublicKeys(pubs [][]byte) (iface.PublicKey, error) {
	if featureconfig.Get().SkipBLSVerify {
		return &PublicKey{}, nil
	}
	agg := new(blst.P1Aggregate)
	mulP1 := make([]*blst.P1Affine, 0, len(pubs))
	for _, pubkey := range pubs {
		if len(pubkey) != params.BeaconConfig().BLSPubkeyLength {
			return nil, fmt.Errorf("public key must be %d bytes", params.BeaconConfig().BLSPubkeyLength)
		}
		if cv, ok := pubkeyCache.Get(string(pubkey)); ok {
			mulP1 = append(mulP1, cv.(*PublicKey).Copy().(*PublicKey).p)
			continue
		}
		p := new(blstPublicKey).Uncompress(pubkey)
		if p == nil {
			return nil, errors.New("could not unmarshal bytes into public key")
		}
		pubKeyObj := &PublicKey{p: p}
		pubkeyCache.Set(string(pubkey), pubKeyObj.Copy(), 48)
		mulP1 = append(mulP1, p)
	}
	agg.Aggregate(mulP1)
	return &PublicKey{p: agg.ToAffine()}, nil
}

// Marshal a public key into a LittleEndian byte slice.
func (p *PublicKey) Marshal() []byte {
	return p.p.Compress()
}

// Copy the public key to a new pointer reference.
func (p *PublicKey) Copy() iface.PublicKey {
	np := *p.p
	return &PublicKey{p: &np}
}

// Aggregate two public keys.
func (p *PublicKey) Aggregate(p2 iface.PublicKey) iface.PublicKey {
	if featureconfig.Get().SkipBLSVerify {
		return p
	}

	agg := new(blstAggregatePublicKey)
	agg.Add(p.p)
	agg.Add(p2.(*PublicKey).p)
	p.p = agg.ToAffine()

	return p
}
