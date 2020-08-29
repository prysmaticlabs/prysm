// +build !linux,amd64

package blst

import "github.com/prysmaticlabs/prysm/shared/bls/iface"

// SecretKey -- stub
type SecretKey struct{}

// PublicKey -- stub
func (s SecretKey) PublicKey() iface.PublicKey {
	panic("blst is only supported on linux amd64")
}

// Sign -- stub
func (s SecretKey) Sign(msg []byte) iface.Signature {
	panic("blst is only supported on linux amd64")
}

// Marshal -- stub
func (s SecretKey) Marshal() []byte {
	panic("blst is only supported on linux amd64")
}

// PublicKey -- stub
type PublicKey struct{}

// Marshal -- stub
func (p PublicKey) Marshal() []byte {
	panic("blst is only supported on linux amd64")
}

// Copy -- stub
func (p PublicKey) Copy() iface.PublicKey {
	panic("blst is only supported on linux amd64")
}

// Aggregate -- stub
func (p PublicKey) Aggregate(p2 iface.PublicKey) iface.PublicKey {
	panic("blst is only supported on linux amd64")
}

// Signature -- stub
type Signature struct{}

// Verify -- stub
func (s Signature) Verify(pubKey iface.PublicKey, msg []byte) bool {
	panic("blst is only supported on linux amd64")
}

// AggregateVerify -- stub
func (s Signature) AggregateVerify(pubKeys []iface.PublicKey, msgs [][32]byte) bool {
	panic("blst is only supported on linux amd64")
}

// FastAggregateVerify -- stub
func (s Signature) FastAggregateVerify(pubKeys []iface.PublicKey, msg [32]byte) bool {
	panic("blst is only supported on linux amd64")
}

// Marshal -- stub
func (s Signature) Marshal() []byte {
	panic("blst is only supported on linux amd64")
}

// Copy -- stub
func (s Signature) Copy() iface.Signature {
	panic("blst is only supported on linux amd64")
}
