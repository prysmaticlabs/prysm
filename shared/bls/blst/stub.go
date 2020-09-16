// +build darwin,amd64 windows,amd64 linux,amd64,!blst_enabled linux,arm64,!blst_enabled

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

// SecretKeyFromBytes -- stub
func SecretKeyFromBytes(privKey []byte) (SecretKey, error) {
	panic("blst is only supported on linux amd64")
}

// PublicKeyFromBytes -- stub
func PublicKeyFromBytes(pubKey []byte) (PublicKey, error) {
	panic("blst is only supported on linux amd64")
}

// SignatureFromBytes -- stub
func SignatureFromBytes(sig []byte) (Signature, error) {
	panic("blst is only supported on linux amd64")
}

// AggregatePublicKeys -- stub
func AggregatePublicKeys(pubs [][]byte) (PublicKey, error) {
	panic("blst is only supported on linux amd64")
}

// AggregateSignatures -- stub
func AggregateSignatures(sigs []iface.Signature) iface.Signature {
	panic("blst is only supported on linux amd64")
}

// VerifyMultipleSignatures -- stub
func VerifyMultipleSignatures(sigs [][]byte, msgs [][32]byte, pubKeys []iface.PublicKey) (bool, error) {
	panic("blst is only supported on linux amd64")
}

// NewAggregateSignature -- stub
func NewAggregateSignature() iface.Signature {
	panic("blst is only supported on linux amd64")
}

// RandKey -- stub
func RandKey() iface.SecretKey {
	panic("blst is only supported on linux amd64")
}

// VerifyCompressed -- stub
func VerifyCompressed(signature []byte, pub []byte, msg []byte) bool {
	panic("blst is only supported on linux amd64")
}
