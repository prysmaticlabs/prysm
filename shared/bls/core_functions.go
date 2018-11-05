package bls

// @file core.go - full credit goes to @herumi for this Go port of the
// C functions defined in the MCL, C/C++ crypto library
//
// @author MITSUNARI Shigeo(@herumi)
// @license modified new BSD license
// http://opensource.org/licenses/BSD-3-Clause

// #cgo LDFLAGS: -lstdc++ -lgmp
// #cgo CFLAGS:-DMCLBN_FP_UNIT_SIZE=6 -DMCL_DONT_USE_OPENSSL
// #include "external/herumi_mcl/include/mcl/bls.h"
import "C"
import "fmt"
import "unsafe"

// CurveFp254BNb -- 254 bit curve
const CurveFp254BNb = C.mclBn_CurveFp254BNb

// CurveFp382_1 -- 382 bit curve 1
const CurveFp382_1 = C.mclBn_CurveFp382_1

// CurveFp382_2 -- 382 bit curve 2
const CurveFp382_2 = C.mclBn_CurveFp382_2

// BLS12381 -- 381 curve spec.
const BLS12381 = C.MCL_BLS12_381

// IoSerializeHexStr helper.
const IoSerializeHexStr = C.MCLBN_IO_SERIALIZE_HEX_STR

// Init --
// call this function before calling all the other operations
// this function is not thread safe.
func Init(curve int) error {
	err := C.mclBn_init(C.int(curve), C.MCLBN_COMPILED_TIME_VAR)
	if err != 0 {
		return fmt.Errorf("ERR mclBn_init curve=%d", curve)
	}
	return nil
}

// GetMaxOpUnitSize --
func GetMaxOpUnitSize() int {
	return int(C.MCLBN_FP_UNIT_SIZE)
}

// GetOpUnitSize --
// the length of Fr is GetOpUnitSize() * 8 bytes
func GetOpUnitSize() int {
	return int(C.mclBn_getOpUnitSize())
}

// GetCurveOrder --
// return the order of G1
func GetCurveOrder() string {
	buf := make([]byte, 1024)
	// #nosec
	n := C.mclBn_getCurveOrder((*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)))
	if n == 0 {
		panic("implementation err. size of buf is small")
	}
	return string(buf[:n])
}

// GetFieldOrder --
// return the characteristic of the field where a curve is defined
func GetFieldOrder() string {
	buf := make([]byte, 1024)
	// #nosec
	n := C.mclBn_getFieldOrder((*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)))
	if n == 0 {
		panic("implementation err. size of buf is small")
	}
	return string(buf[:n])
}

// ID --
type ID struct {
	v Fr
}

// getPointer --
func (id *ID) getPointer() (p *C.blsId) {
	// #nosec
	return (*C.blsId)(unsafe.Pointer(id))
}

// GetLittleEndian --
func (id *ID) GetLittleEndian() []byte {
	return id.v.Serialize()
}

// SetLittleEndian --
func (id *ID) SetLittleEndian(buf []byte) error {
	return id.v.SetLittleEndian(buf)
}

// GetHexString --
func (id *ID) GetHexString() string {
	return id.v.GetString(16)
}

// GetDecString --
func (id *ID) GetDecString() string {
	return id.v.GetString(10)
}

// SetHexString --
func (id *ID) SetHexString(s string) error {
	return id.v.SetString(s, 16)
}

// SetDecString --
func (id *ID) SetDecString(s string) error {
	return id.v.SetString(s, 10)
}

// IsEqual --
func (id *ID) IsEqual(rhs *ID) bool {
	return id.v.IsEqual(&rhs.v)
}

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

// GetMasterPublicKey --
func GetMasterPublicKey(msk []SecretKey) (mpk []PublicKey) {
	n := len(msk)
	mpk = make([]PublicKey, n)
	for i := 0; i < n; i++ {
		mpk[i] = *msk[i].GetPublicKey()
	}
	return mpk
}

// Set --
func (sec *SecretKey) Set(msk []SecretKey, id *ID) error {
	// #nosec
	return FrEvaluatePolynomial(&sec.v, *(*[]Fr)(unsafe.Pointer(&msk)), &id.v)
}

// Recover --
func (sec *SecretKey) Recover(secVec []SecretKey, idVec []ID) error {
	// #nosec
	return FrLagrangeInterpolation(&sec.v, *(*[]Fr)(unsafe.Pointer(&idVec)), *(*[]Fr)(unsafe.Pointer(&secVec)))
}

// GetPop --
func (sec *SecretKey) GetPop() (sign *Sign) {
	sign = new(Sign)
	C.blsGetPop(sign.getPointer(), sec.getPointer())
	return sign
}

// PublicKey --
type PublicKey struct {
	v G2
}

// getPointer --
func (pub *PublicKey) getPointer() (p *C.blsPublicKey) {
	// #nosec
	return (*C.blsPublicKey)(unsafe.Pointer(pub))
}

// Serialize --
func (pub *PublicKey) Serialize() []byte {
	return pub.v.Serialize()
}

// Deserialize --
func (pub *PublicKey) Deserialize(buf []byte) error {
	return pub.v.Deserialize(buf)
}

// SerializeToHexStr --
func (pub *PublicKey) SerializeToHexStr() string {
	return pub.v.GetString(IoSerializeHexStr)
}

// DeserializeHexStr --
func (pub *PublicKey) DeserializeHexStr(s string) error {
	return pub.v.SetString(s, IoSerializeHexStr)
}

// GetHexString --
func (pub *PublicKey) GetHexString() string {
	return pub.v.GetString(16)
}

// SetHexString --
func (pub *PublicKey) SetHexString(s string) error {
	return pub.v.SetString(s, 16)
}

// IsEqual --
func (pub *PublicKey) IsEqual(rhs *PublicKey) bool {
	return pub.v.IsEqual(&rhs.v)
}

// Add --
func (pub *PublicKey) Add(rhs *PublicKey) {
	G2Add(&pub.v, &pub.v, &rhs.v)
}

// Set --
func (pub *PublicKey) Set(mpk []PublicKey, id *ID) error {
	// #nosec
	return G2EvaluatePolynomial(&pub.v, *(*[]G2)(unsafe.Pointer(&mpk)), &id.v)
}

// Recover --
func (pub *PublicKey) Recover(pubVec []PublicKey, idVec []ID) error {
	// #nosec
	return G2LagrangeInterpolation(&pub.v, *(*[]Fr)(unsafe.Pointer(&idVec)), *(*[]G2)(unsafe.Pointer(&pubVec)))
}

// Sign  --
type Sign struct {
	v G1
}

// getPointer --
func (sign *Sign) getPointer() (p *C.blsSignature) {
	// #nosec
	return (*C.blsSignature)(unsafe.Pointer(sign))
}

// Serialize --
func (sign *Sign) Serialize() []byte {
	return sign.v.Serialize()
}

// Deserialize --
func (sign *Sign) Deserialize(buf []byte) error {
	return sign.v.Deserialize(buf)
}

// SerializeToHexStr --
func (sign *Sign) SerializeToHexStr() string {
	return sign.v.GetString(IoSerializeHexStr)
}

// DeserializeHexStr --
func (sign *Sign) DeserializeHexStr(s string) error {
	return sign.v.SetString(s, IoSerializeHexStr)
}

// GetHexString --
func (sign *Sign) GetHexString() string {
	return sign.v.GetString(16)
}

// SetHexString --
func (sign *Sign) SetHexString(s string) error {
	return sign.v.SetString(s, 16)
}

// IsEqual --
func (sign *Sign) IsEqual(rhs *Sign) bool {
	return sign.v.IsEqual(&rhs.v)
}

// GetPublicKey --
func (sec *SecretKey) GetPublicKey() (pub *PublicKey) {
	pub = new(PublicKey)
	C.blsGetPublicKey(pub.getPointer(), sec.getPointer())
	return pub
}

// Sign -- Constant Time version
func (sec *SecretKey) Sign(m string) (sign *Sign) {
	sign = new(Sign)
	buf := []byte(m)
	// #nosec
	C.blsSign(sign.getPointer(), sec.getPointer(), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	return sign
}

// Add --
func (sign *Sign) Add(rhs *Sign) {
	C.blsSignatureAdd(sign.getPointer(), rhs.getPointer())
}

// Recover --
func (sign *Sign) Recover(signVec []Sign, idVec []ID) error {
	// #nosec
	return G1LagrangeInterpolation(&sign.v, *(*[]Fr)(unsafe.Pointer(&idVec)), *(*[]G1)(unsafe.Pointer(&signVec)))
}

// Verify --
func (sign *Sign) Verify(pub *PublicKey, m string) bool {
	buf := []byte(m)
	// #nosec
	return C.blsVerify(sign.getPointer(), pub.getPointer(), unsafe.Pointer(&buf[0]), C.size_t(len(buf))) == 1
}

// VerifyPop --
func (sign *Sign) VerifyPop(pub *PublicKey) bool {
	return C.blsVerifyPop(sign.getPointer(), pub.getPointer()) == 1
}

// DHKeyExchange --
func DHKeyExchange(sec *SecretKey, pub *PublicKey) (out PublicKey) {
	C.blsDHKeyExchange(out.getPointer(), sec.getPointer(), pub.getPointer())
	return out
}

// Fr --
type Fr struct {
	v C.mclBnFr
}

// getPointer --
func (x *Fr) getPointer() (p *C.mclBnFr) {
	// #nosec
	return (*C.mclBnFr)(unsafe.Pointer(x))
}

// Clear --
func (x *Fr) Clear() {
	// #nosec
	C.mclBnFr_clear(x.getPointer())
}

// SetInt64 --
func (x *Fr) SetInt64(v int64) {
	// #nosec
	C.mclBnFr_setInt(x.getPointer(), C.int64_t(v))
}

// SetString --
func (x *Fr) SetString(s string, base int) error {
	buf := []byte(s)
	// #nosec
	err := C.mclBnFr_setStr(x.getPointer(), (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), C.int(base))
	if err != 0 {
		return fmt.Errorf("err mclBnFr_setStr %x", err)
	}
	return nil
}

// Deserialize --
func (x *Fr) Deserialize(buf []byte) error {
	// #nosec
	err := C.mclBnFr_deserialize(x.getPointer(), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	if err == 0 {
		return fmt.Errorf("err mclBnFr_deserialize %x", buf)
	}
	return nil
}

// SetLittleEndian --
func (x *Fr) SetLittleEndian(buf []byte) error {
	// #nosec
	err := C.mclBnFr_setLittleEndian(x.getPointer(), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	if err != 0 {
		return fmt.Errorf("err mclBnFr_setLittleEndian %x", err)
	}
	return nil
}

// IsEqual --
func (x *Fr) IsEqual(rhs *Fr) bool {
	return C.mclBnFr_isEqual(x.getPointer(), rhs.getPointer()) == 1
}

// IsZero --
func (x *Fr) IsZero() bool {
	return C.mclBnFr_isZero(x.getPointer()) == 1
}

// IsOne --
func (x *Fr) IsOne() bool {
	return C.mclBnFr_isOne(x.getPointer()) == 1
}

// SetByCSPRNG --
func (x *Fr) SetByCSPRNG() {
	err := C.mclBnFr_setByCSPRNG(x.getPointer())
	if err != 0 {
		panic("err mclBnFr_setByCSPRNG")
	}
}

// SetHashOf --
func (x *Fr) SetHashOf(buf []byte) bool {
	// #nosec
	return C.mclBnFr_setHashOf(x.getPointer(), unsafe.Pointer(&buf[0]), C.size_t(len(buf))) == 0
}

// GetString --
func (x *Fr) GetString(base int) string {
	buf := make([]byte, 2048)
	// #nosec
	n := C.mclBnFr_getStr((*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), x.getPointer(), C.int(base))
	if n == 0 {
		panic("err mclBnFr_getStr")
	}
	return string(buf[:n])
}

// Serialize --
func (x *Fr) Serialize() []byte {
	buf := make([]byte, 2048)
	// #nosec
	n := C.mclBnFr_serialize(unsafe.Pointer(&buf[0]), C.size_t(len(buf)), x.getPointer())
	if n == 0 {
		panic("err mclBnFr_serialize")
	}
	return buf[:n]
}

// FrNeg --
func FrNeg(out *Fr, x *Fr) {
	C.mclBnFr_neg(out.getPointer(), x.getPointer())
}

// FrInv --
func FrInv(out *Fr, x *Fr) {
	C.mclBnFr_inv(out.getPointer(), x.getPointer())
}

// FrAdd --
func FrAdd(out *Fr, x *Fr, y *Fr) {
	C.mclBnFr_add(out.getPointer(), x.getPointer(), y.getPointer())
}

// FrSub --
func FrSub(out *Fr, x *Fr, y *Fr) {
	C.mclBnFr_sub(out.getPointer(), x.getPointer(), y.getPointer())
}

// FrMul --
func FrMul(out *Fr, x *Fr, y *Fr) {
	C.mclBnFr_mul(out.getPointer(), x.getPointer(), y.getPointer())
}

// FrDiv --
func FrDiv(out *Fr, x *Fr, y *Fr) {
	C.mclBnFr_div(out.getPointer(), x.getPointer(), y.getPointer())
}

// G1 --
type G1 struct {
	v C.mclBnG1
}

// getPointer --
func (x *G1) getPointer() (p *C.mclBnG1) {
	// #nosec
	return (*C.mclBnG1)(unsafe.Pointer(x))
}

// Clear --
func (x *G1) Clear() {
	// #nosec
	C.mclBnG1_clear(x.getPointer())
}

// SetString --
func (x *G1) SetString(s string, base int) error {
	buf := []byte(s)
	// #nosec
	err := C.mclBnG1_setStr(x.getPointer(), (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), C.int(base))
	if err != 0 {
		return fmt.Errorf("err mclBnG1_setStr %x", err)
	}
	return nil
}

// Deserialize --
func (x *G1) Deserialize(buf []byte) error {
	// #nosec
	err := C.mclBnG1_deserialize(x.getPointer(), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	if err == 0 {
		return fmt.Errorf("err mclBnG1_deserialize %x", buf)
	}
	return nil
}

// IsEqual --
func (x *G1) IsEqual(rhs *G1) bool {
	return C.mclBnG1_isEqual(x.getPointer(), rhs.getPointer()) == 1
}

// IsZero --
func (x *G1) IsZero() bool {
	return C.mclBnG1_isZero(x.getPointer()) == 1
}

// HashAndMapTo --
func (x *G1) HashAndMapTo(buf []byte) error {
	// #nosec
	err := C.mclBnG1_hashAndMapTo(x.getPointer(), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	if err != 0 {
		return fmt.Errorf("err mclBnG1_hashAndMapTo %x", err)
	}
	return nil
}

// GetString --
func (x *G1) GetString(base int) string {
	buf := make([]byte, 2048)
	// #nosec
	n := C.mclBnG1_getStr((*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), x.getPointer(), C.int(base))
	if n == 0 {
		panic("err mclBnG1_getStr")
	}
	return string(buf[:n])
}

// Serialize --
func (x *G1) Serialize() []byte {
	buf := make([]byte, 2048)
	// #nosec
	n := C.mclBnG1_serialize(unsafe.Pointer(&buf[0]), C.size_t(len(buf)), x.getPointer())
	if n == 0 {
		panic("err mclBnG1_serialize")
	}
	return buf[:n]
}

// G1Neg --
func G1Neg(out *G1, x *G1) {
	C.mclBnG1_neg(out.getPointer(), x.getPointer())
}

// G1Dbl --
func G1Dbl(out *G1, x *G1) {
	C.mclBnG1_dbl(out.getPointer(), x.getPointer())
}

// G1Add --
func G1Add(out *G1, x *G1, y *G1) {
	C.mclBnG1_add(out.getPointer(), x.getPointer(), y.getPointer())
}

// G1Sub --
func G1Sub(out *G1, x *G1, y *G1) {
	C.mclBnG1_sub(out.getPointer(), x.getPointer(), y.getPointer())
}

// G1Mul --
func G1Mul(out *G1, x *G1, y *Fr) {
	C.mclBnG1_mul(out.getPointer(), x.getPointer(), y.getPointer())
}

// G1MulCT -- constant time (depending on bit lengh of y)
func G1MulCT(out *G1, x *G1, y *Fr) {
	C.mclBnG1_mulCT(out.getPointer(), x.getPointer(), y.getPointer())
}

// G2 --
type G2 struct {
	v C.mclBnG2
}

// getPointer --
func (x *G2) getPointer() (p *C.mclBnG2) {
	// #nosec
	return (*C.mclBnG2)(unsafe.Pointer(x))
}

// Clear --
func (x *G2) Clear() {
	// #nosec
	C.mclBnG2_clear(x.getPointer())
}

// SetString --
func (x *G2) SetString(s string, base int) error {
	buf := []byte(s)
	// #nosec
	err := C.mclBnG2_setStr(x.getPointer(), (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), C.int(base))
	if err != 0 {
		return fmt.Errorf("err mclBnG2_setStr %x", err)
	}
	return nil
}

// Deserialize --
func (x *G2) Deserialize(buf []byte) error {
	// #nosec
	err := C.mclBnG2_deserialize(x.getPointer(), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	if err == 0 {
		return fmt.Errorf("err mclBnG2_deserialize %x", buf)
	}
	return nil
}

// IsEqual --
func (x *G2) IsEqual(rhs *G2) bool {
	return C.mclBnG2_isEqual(x.getPointer(), rhs.getPointer()) == 1
}

// IsZero --
func (x *G2) IsZero() bool {
	return C.mclBnG2_isZero(x.getPointer()) == 1
}

// HashAndMapTo --
func (x *G2) HashAndMapTo(buf []byte) error {
	// #nosec
	err := C.mclBnG2_hashAndMapTo(x.getPointer(), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	if err != 0 {
		return fmt.Errorf("err mclBnG2_hashAndMapTo %x", err)
	}
	return nil
}

// GetString --
func (x *G2) GetString(base int) string {
	buf := make([]byte, 2048)
	// #nosec
	n := C.mclBnG2_getStr((*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), x.getPointer(), C.int(base))
	if n == 0 {
		panic("err mclBnG2_getStr")
	}
	return string(buf[:n])
}

// Serialize --
func (x *G2) Serialize() []byte {
	buf := make([]byte, 2048)
	// #nosec
	n := C.mclBnG2_serialize(unsafe.Pointer(&buf[0]), C.size_t(len(buf)), x.getPointer())
	if n == 0 {
		panic("err mclBnG2_serialize")
	}
	return buf[:n]
}

// G2Neg --
func G2Neg(out *G2, x *G2) {
	C.mclBnG2_neg(out.getPointer(), x.getPointer())
}

// G2Dbl --
func G2Dbl(out *G2, x *G2) {
	C.mclBnG2_dbl(out.getPointer(), x.getPointer())
}

// G2Add --
func G2Add(out *G2, x *G2, y *G2) {
	C.mclBnG2_add(out.getPointer(), x.getPointer(), y.getPointer())
}

// G2Sub --
func G2Sub(out *G2, x *G2, y *G2) {
	C.mclBnG2_sub(out.getPointer(), x.getPointer(), y.getPointer())
}

// G2Mul --
func G2Mul(out *G2, x *G2, y *Fr) {
	C.mclBnG2_mul(out.getPointer(), x.getPointer(), y.getPointer())
}

// GT --
type GT struct {
	v C.mclBnGT
}

// getPointer --
func (x *GT) getPointer() (p *C.mclBnGT) {
	// #nosec
	return (*C.mclBnGT)(unsafe.Pointer(x))
}

// Clear --
func (x *GT) Clear() {
	// #nosec
	C.mclBnGT_clear(x.getPointer())
}

// SetInt64 --
func (x *GT) SetInt64(v int64) {
	// #nosec
	C.mclBnGT_setInt(x.getPointer(), C.int64_t(v))
}

// SetString --
func (x *GT) SetString(s string, base int) error {
	buf := []byte(s)
	// #nosec
	err := C.mclBnGT_setStr(x.getPointer(), (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), C.int(base))
	if err != 0 {
		return fmt.Errorf("err mclBnGT_setStr %x", err)
	}
	return nil
}

// Deserialize --
func (x *GT) Deserialize(buf []byte) error {
	// #nosec
	err := C.mclBnGT_deserialize(x.getPointer(), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	if err == 0 {
		return fmt.Errorf("err mclBnGT_deserialize %x", buf)
	}
	return nil
}

// IsEqual --
func (x *GT) IsEqual(rhs *GT) bool {
	return C.mclBnGT_isEqual(x.getPointer(), rhs.getPointer()) == 1
}

// IsZero --
func (x *GT) IsZero() bool {
	return C.mclBnGT_isZero(x.getPointer()) == 1
}

// IsOne --
func (x *GT) IsOne() bool {
	return C.mclBnGT_isOne(x.getPointer()) == 1
}

// GetString --
func (x *GT) GetString(base int) string {
	buf := make([]byte, 2048)
	// #nosec
	n := C.mclBnGT_getStr((*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), x.getPointer(), C.int(base))
	if n == 0 {
		panic("err mclBnGT_getStr")
	}
	return string(buf[:n])
}

// Serialize --
func (x *GT) Serialize() []byte {
	buf := make([]byte, 2048)
	// #nosec
	n := C.mclBnGT_serialize(unsafe.Pointer(&buf[0]), C.size_t(len(buf)), x.getPointer())
	if n == 0 {
		panic("err mclBnGT_serialize")
	}
	return buf[:n]
}

// GTNeg --
func GTNeg(out *GT, x *GT) {
	C.mclBnGT_neg(out.getPointer(), x.getPointer())
}

// GTInv --
func GTInv(out *GT, x *GT) {
	C.mclBnGT_inv(out.getPointer(), x.getPointer())
}

// GTAdd --
func GTAdd(out *GT, x *GT, y *GT) {
	C.mclBnGT_add(out.getPointer(), x.getPointer(), y.getPointer())
}

// GTSub --
func GTSub(out *GT, x *GT, y *GT) {
	C.mclBnGT_sub(out.getPointer(), x.getPointer(), y.getPointer())
}

// GTMul --
func GTMul(out *GT, x *GT, y *GT) {
	C.mclBnGT_mul(out.getPointer(), x.getPointer(), y.getPointer())
}

// GTDiv --
func GTDiv(out *GT, x *GT, y *GT) {
	C.mclBnGT_div(out.getPointer(), x.getPointer(), y.getPointer())
}

// GTPow --
func GTPow(out *GT, x *GT, y *Fr) {
	C.mclBnGT_pow(out.getPointer(), x.getPointer(), y.getPointer())
}

// Pairing --
func Pairing(out *GT, x *G1, y *G2) {
	C.mclBn_pairing(out.getPointer(), x.getPointer(), y.getPointer())
}

// FinalExp --
func FinalExp(out *GT, x *GT) {
	C.mclBn_finalExp(out.getPointer(), x.getPointer())
}

// MillerLoop --
func MillerLoop(out *GT, x *G1, y *G2) {
	C.mclBn_millerLoop(out.getPointer(), x.getPointer(), y.getPointer())
}

// GetUint64NumToPrecompute --
func GetUint64NumToPrecompute() int {
	return int(C.mclBn_getUint64NumToPrecompute())
}

// PrecomputeG2 --
func PrecomputeG2(Qbuf []uint64, Q *G2) {
	// #nosec
	C.mclBn_precomputeG2((*C.uint64_t)(unsafe.Pointer(&Qbuf[0])), Q.getPointer())
}

// PrecomputedMillerLoop --
func PrecomputedMillerLoop(out *GT, P *G1, Qbuf []uint64) {
	// #nosec
	C.mclBn_precomputedMillerLoop(out.getPointer(), P.getPointer(), (*C.uint64_t)(unsafe.Pointer(&Qbuf[0])))
}

// PrecomputedMillerLoop2 --
func PrecomputedMillerLoop2(out *GT, P1 *G1, Q1buf []uint64, P2 *G1, Q2buf []uint64) {
	// #nosec
	C.mclBn_precomputedMillerLoop2(out.getPointer(), P1.getPointer(), (*C.uint64_t)(unsafe.Pointer(&Q1buf[0])), P1.getPointer(), (*C.uint64_t)(unsafe.Pointer(&Q1buf[0])))
}

// FrEvaluatePolynomial -- y = c[0] + c[1] * x + c[2] * x^2 + ...
func FrEvaluatePolynomial(y *Fr, c []Fr, x *Fr) error {
	// #nosec
	err := C.mclBn_FrEvaluatePolynomial(y.getPointer(), (*C.mclBnFr)(unsafe.Pointer(&c[0])), (C.size_t)(len(c)), x.getPointer())
	if err != 0 {
		return fmt.Errorf("err mclBn_FrEvaluatePolynomial")
	}
	return nil
}

// G1EvaluatePolynomial -- y = c[0] + c[1] * x + c[2] * x^2 + ...
func G1EvaluatePolynomial(y *G1, c []G1, x *Fr) error {
	// #nosec
	err := C.mclBn_G1EvaluatePolynomial(y.getPointer(), (*C.mclBnG1)(unsafe.Pointer(&c[0])), (C.size_t)(len(c)), x.getPointer())
	if err != 0 {
		return fmt.Errorf("err mclBn_G1EvaluatePolynomial")
	}
	return nil
}

// G2EvaluatePolynomial -- y = c[0] + c[1] * x + c[2] * x^2 + ...
func G2EvaluatePolynomial(y *G2, c []G2, x *Fr) error {
	// #nosec
	err := C.mclBn_G2EvaluatePolynomial(y.getPointer(), (*C.mclBnG2)(unsafe.Pointer(&c[0])), (C.size_t)(len(c)), x.getPointer())
	if err != 0 {
		return fmt.Errorf("err mclBn_G2EvaluatePolynomial")
	}
	return nil
}

// FrLagrangeInterpolation --
func FrLagrangeInterpolation(out *Fr, xVec []Fr, yVec []Fr) error {
	if len(xVec) != len(yVec) {
		return fmt.Errorf("err FrLagrangeInterpolation:bad size")
	}
	// #nosec
	err := C.mclBn_FrLagrangeInterpolation(out.getPointer(), (*C.mclBnFr)(unsafe.Pointer(&xVec[0])), (*C.mclBnFr)(unsafe.Pointer(&yVec[0])), (C.size_t)(len(xVec)))
	if err != 0 {
		return fmt.Errorf("err FrLagrangeInterpolation")
	}
	return nil
}

// G1LagrangeInterpolation --
func G1LagrangeInterpolation(out *G1, xVec []Fr, yVec []G1) error {
	if len(xVec) != len(yVec) {
		return fmt.Errorf("err G1LagrangeInterpolation:bad size")
	}
	// #nosec
	err := C.mclBn_G1LagrangeInterpolation(out.getPointer(), (*C.mclBnFr)(unsafe.Pointer(&xVec[0])), (*C.mclBnG1)(unsafe.Pointer(&yVec[0])), (C.size_t)(len(xVec)))
	if err != 0 {
		return fmt.Errorf("err G1LagrangeInterpolation")
	}
	return nil
}

// G2LagrangeInterpolation --
func G2LagrangeInterpolation(out *G2, xVec []Fr, yVec []G2) error {
	if len(xVec) != len(yVec) {
		return fmt.Errorf("err G2LagrangeInterpolation:bad size")
	}
	// #nosec
	err := C.mclBn_G2LagrangeInterpolation(out.getPointer(), (*C.mclBnFr)(unsafe.Pointer(&xVec[0])), (*C.mclBnG2)(unsafe.Pointer(&yVec[0])), (C.size_t)(len(xVec)))
	if err != 0 {
		return fmt.Errorf("err G2LagrangeInterpolation")
	}
	return nil
}
