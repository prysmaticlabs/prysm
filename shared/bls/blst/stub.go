// +build darwin windows !blst_enabled

package blst

import "github.com/prysmaticlabs/prysm/shared/bls/iface"

// This stub file exists until build issues can be resolved for darwin and windows.
const err = "blst is only supported on linux with blst_enabled gotag"

// SecretKey -- stub
type SecretKey struct{}

// PublicKey -- stub
func (s SecretKey) PublicKey() iface.PublicKey {
	panic(err)
}

// Sign -- stub
func (s SecretKey) Sign(_ []byte) iface.Signature {
	panic(err)
}

// Marshal -- stub
func (s SecretKey) Marshal() []byte {
	panic(err)
}

// PublicKey -- stub
type PublicKey struct{}

// Marshal -- stub
func (p PublicKey) Marshal() []byte {
	panic(err)
}

// Copy -- stub
func (p PublicKey) Copy() iface.PublicKey {
	panic(err)
}

// Aggregate -- stub
func (p PublicKey) Aggregate(_ iface.PublicKey) iface.PublicKey {
	panic(err)
}

// Signature -- stub
type Signature struct{}

// Verify -- stub
func (s Signature) Verify(_ iface.PublicKey, _ []byte) bool {
	panic(err)
}

// AggregateVerify -- stub
func (s Signature) AggregateVerify(_ []iface.PublicKey, _ [][32]byte) bool {
	panic(err)
}

// FastAggregateVerify -- stub
func (s Signature) FastAggregateVerify(_ []iface.PublicKey, _ [32]byte) bool {
	panic(err)
}

// Marshal -- stub
func (s Signature) Marshal() []byte {
	panic(err)
}

// Copy -- stub
func (s Signature) Copy() iface.Signature {
	panic(err)
}

// SecretKeyFromBytes -- stub
func SecretKeyFromBytes(_ []byte) (SecretKey, error) {
	panic(err)
}

// PublicKeyFromBytes -- stub
func PublicKeyFromBytes(_ []byte) (PublicKey, error) {
	panic(err)
}

// SignatureFromBytes -- stub
func SignatureFromBytes(_ []byte) (Signature, error) {
	panic(err)
}

// AggregatePublicKeys -- stub
func AggregatePublicKeys(_ [][]byte) (PublicKey, error) {
	panic(err)
}

// AggregateSignatures -- stub
func AggregateSignatures(_ []iface.Signature) iface.Signature {
	panic(err)
}

// VerifyMultipleSignatures -- stub
func VerifyMultipleSignatures(_ [][]byte, _ [][32]byte, _ []iface.PublicKey) (bool, error) {
	panic(err)
}

// NewAggregateSignature -- stub
func NewAggregateSignature() iface.Signature {
	panic(err)
}

// RandKey -- stub
func RandKey() iface.SecretKey {
	panic(err)
}

// VerifyCompressed -- stub
func VerifyCompressed(_, _, _ []byte) bool {
	panic(err)
}
