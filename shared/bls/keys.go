package bls

// #cgo LDFLAGS: -lstdc++ -lgmp
// #cgo CFLAGS:-DMCLBN_FP_UNIT_SIZE=6 -DMCL_DONT_USE_OPENSSL
// #include "external/herumi_mcl/include/mcl/bls.h"
import "C"
import (
	"unsafe"
)

// SecretKey --
type SecretKey struct {
	v Fr
}

// getPointer --
func (sec *SecretKey) getPointer() (p *C.blsSecretKey) {
	// #nosec
	return (*C.blsSecretKey)(unsafe.Pointer(sec))
}

// GetLittleEndian --
func (sec *SecretKey) GetLittleEndian() []byte {
	return sec.v.Serialize()
}

// SetLittleEndian --
func (sec *SecretKey) SetLittleEndian(buf []byte) error {
	return sec.v.SetLittleEndian(buf)
}

// SerializeToHexStr --
func (sec *SecretKey) SerializeToHexStr() string {
	return sec.v.GetString(IoSerializeHexStr)
}

// DeserializeHexStr --
func (sec *SecretKey) DeserializeHexStr(s string) error {
	return sec.v.SetString(s, IoSerializeHexStr)
}

// GetHexString --
func (sec *SecretKey) GetHexString() string {
	return sec.v.GetString(16)
}

// GetDecString --
func (sec *SecretKey) GetDecString() string {
	return sec.v.GetString(10)
}

// SetHexString --
func (sec *SecretKey) SetHexString(s string) error {
	return sec.v.SetString(s, 16)
}

// SetDecString --
func (sec *SecretKey) SetDecString(s string) error {
	return sec.v.SetString(s, 10)
}

// IsEqual --
func (sec *SecretKey) IsEqual(rhs *SecretKey) bool {
	return sec.v.IsEqual(&rhs.v)
}

// SetByCSPRNG --
func (sec *SecretKey) SetByCSPRNG() {
	sec.v.SetByCSPRNG()
}

// Add --
func (sec *SecretKey) Add(rhs *SecretKey) {
	FrAdd(&sec.v, &sec.v, &rhs.v)
}

// GetMasterSecretKey --
func (sec *SecretKey) GetMasterSecretKey(k int) (msk []SecretKey) {
	msk = make([]SecretKey, k)
	msk[0] = *sec
	for i := 1; i < k; i++ {
		msk[i].SetByCSPRNG()
	}
	return msk
}
