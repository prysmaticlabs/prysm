// +build !linux,amd64

package blst

import "github.com/prysmaticlabs/prysm/shared/bls/iface"

type SecretKey struct{}

func (s SecretKey) PublicKey() iface.PublicKey {
	panic("blst is only supported on linux amd64")
}

func (s SecretKey) Sign(msg []byte) iface.Signature {
	panic("blst is only supported on linux amd64")
}

func (s SecretKey) Marshal() []byte {
	panic("blst is only supported on linux amd64")
}

type PublicKey struct{}

func (p PublicKey) Marshal() []byte {
	panic("blst is only supported on linux amd64")
}

func (p PublicKey) Copy() iface.PublicKey {
	panic("blst is only supported on linux amd64")
}

func (p PublicKey) Aggregate(p2 iface.PublicKey) iface.PublicKey {
	panic("blst is only supported on linux amd64")
}

type Signature struct{}

func (s Signature) Verify(pubKey iface.PublicKey, msg []byte) bool {
	panic("blst is only supported on linux amd64")
}

func (s Signature) AggregateVerify(pubKeys []iface.PublicKey, msgs [][32]byte) bool {
	panic("blst is only supported on linux amd64")
}

func (s Signature) FastAggregateVerify(pubKeys []iface.PublicKey, msg [32]byte) bool {
	panic("blst is only supported on linux amd64")
}

func (s Signature) Marshal() []byte {
	panic("blst is only supported on linux amd64")
}

func (s Signature) Copy() iface.Signature {
	panic("blst is only supported on linux amd64")
}
