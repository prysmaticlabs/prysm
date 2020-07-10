package blst

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bls/iface"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
	blst "github.com/supranational/blst/bindings/go"
)

var dst = []byte("BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_")

const scalarBytes = 32
const randBitsEntropy = 64

// Signature used in the BLS signature scheme.
type Signature struct {
	s *blstSignature
}

// SignatureFromBytes creates a BLS signature from a LittleEndian byte slice.
func SignatureFromBytes(sig []byte) (iface.Signature, error) {
	if featureconfig.Get().SkipBLSVerify {
		return &Signature{}, nil
	}
	if len(sig) != params.BeaconConfig().BLSSignatureLength {
		return nil, fmt.Errorf("signature must be %d bytes", params.BeaconConfig().BLSSignatureLength)
	}
	signature := new(blstSignature).Uncompress(sig)
	if signature == nil {
		return nil, errors.New("could not unmarshal bytes into signature")
	}
	return &Signature{s: signature}, nil
}

// Verify a bls signature given a public key, a message.
//
// In IETF draft BLS specification:
// Verify(PK, message, signature) -> VALID or INVALID: a verification
//      algorithm that outputs VALID if signature is a valid signature of
//      message under public key PK, and INVALID otherwise.
//
// In ETH2.0 specification:
// def Verify(PK: BLSPubkey, message: Bytes, signature: BLSSignature) -> bool
func (s *Signature) Verify(pubKey iface.PublicKey, msg []byte) bool {
	if featureconfig.Get().SkipBLSVerify {
		return true
	}
	return s.s.Verify(pubKey.(*PublicKey).p, msg, dst)
}

// AggregateVerify verifies each public key against its respective message.
// This is vulnerable to rogue public-key attack. Each user must
// provide a proof-of-knowledge of the public key.
//
// In IETF draft BLS specification:
// AggregateVerify((PK_1, message_1), ..., (PK_n, message_n),
//      signature) -> VALID or INVALID: an aggregate verification
//      algorithm that outputs VALID if signature is a valid aggregated
//      signature for a collection of public keys and messages, and
//      outputs INVALID otherwise.
//
// In ETH2.0 specification:
// def AggregateVerify(pairs: Sequence[PK: BLSPubkey, message: Bytes], signature: BLSSignature) -> boo
func (s *Signature) AggregateVerify(pubKeys []iface.PublicKey, msgs [][32]byte) bool {
	if featureconfig.Get().SkipBLSVerify {
		return true
	}
	size := len(pubKeys)
	if size == 0 {
		return false
	}
	if size != len(msgs) {
		return false
	}
	msgSlices := make([][]byte, len(msgs))
	rawKeys := make([]*blstPublicKey, len(msgs))
	for i := 0; i < size; i++ {
		msgSlices[i] = msgs[i][:]
		rawKeys[i] = pubKeys[i].(*PublicKey).p
	}
	return s.s.AggregateVerify(rawKeys, msgSlices, dst)
}

// FastAggregateVerify verifies all the provided public keys with their aggregated signature.
//
// In IETF draft BLS specification:
// FastAggregateVerify(PK_1, ..., PK_n, message, signature) -> VALID
//      or INVALID: a verification algorithm for the aggregate of multiple
//      signatures on the same message.  This function is faster than
//      AggregateVerify.
//
// In ETH2.0 specification:
// def FastAggregateVerify(PKs: Sequence[BLSPubkey], message: Bytes, signature: BLSSignature) -> bool
func (s *Signature) FastAggregateVerify(pubKeys []iface.PublicKey, msg [32]byte) bool {
	if featureconfig.Get().SkipBLSVerify {
		return true
	}
	if len(pubKeys) == 0 {
		return false
	}
	rawKeys := make([]*blstPublicKey, len(pubKeys))
	for i := 0; i < len(pubKeys); i++ {
		rawKeys[i] = pubKeys[i].(*PublicKey).p
	}

	return s.s.FastAggregateVerify(rawKeys, msg[:], dst)
}

// NewAggregateSignature creates a blank aggregate signature.
func NewAggregateSignature() iface.Signature {
	sig := blst.HashToG2([]byte{'m', 'o', 'c', 'k'}, dst).ToAffine()
	return &Signature{s: sig}
}

// AggregateSignatures converts a list of signatures into a single, aggregated sig.
func AggregateSignatures(sigs []iface.Signature) iface.Signature {
	if len(sigs) == 0 {
		return nil
	}
	if featureconfig.Get().SkipBLSVerify {
		return sigs[0]
	}

	rawSigs := make([]*blstSignature, len(sigs))
	for i := 0; i < len(sigs); i++ {
		rawSigs[i] = sigs[i].(*Signature).s
	}

	signature := new(blstAggregateSignature).Aggregate(rawSigs)
	if signature == nil {
		return nil
	}
	return &Signature{s: signature.ToAffine()}
}

// Aggregate is an alias for AggregateSignatures, defined to conform to BLS specification.
//
// In IETF draft BLS specification:
// Aggregate(signature_1, ..., signature_n) -> signature: an
//      aggregation algorithm that compresses a collection of signatures
//      into a single signature.
//
// In ETH2.0 specification:
// def Aggregate(signatures: Sequence[BLSSignature]) -> BLSSignature
//
// Deprecated: Use AggregateSignatures.
func Aggregate(sigs []iface.Signature) iface.Signature {
	return AggregateSignatures(sigs)
}

// VerifyMultipleSignatures verifies a non-singular set of signatures and its respective pubkeys and messages.
// This method provides a safe way to verify multiple signatures at once. We pick a number randomly from 1 to max
// uint64 and then multiply the signature by it. We continue doing this for all signatures and its respective pubkeys.
// S* = S_1 * r_1 + S_2 * r_2 + ... + S_n * r_n
// P'_{i,j} = P_{i,j} * r_i
// e(S*, G) = \prod_{i=1}^n \prod_{j=1}^{m_i} e(P'_{i,j}, M_{i,j})
// Using this we can verify multiple signatures safely.
func VerifyMultipleSignatures(sigs []iface.Signature, msgs [][32]byte, pubKeys []iface.PublicKey) (bool, error) {
	if featureconfig.Get().SkipBLSVerify {
		return true, nil
	}
	if len(sigs) == 0 || len(pubKeys) == 0 {
		return false, nil
	}
	length := len(sigs)
	if length != len(pubKeys) || length != len(msgs) {
		return false, errors.Errorf("provided signatures, pubkeys and messages have differing lengths. S: %d, P: %d,M %d",
			length, len(pubKeys), len(msgs))
	}
	mulP2Aff := make([]*blst.P2Affine, length)
	mulP1Aff := make([]*blst.P1Affine, length)
	rawMsgs := make([]blst.Message, length)

	for i := 0; i < length; i++ {
		mulP2Aff[i] = sigs[i].(*Signature).s
		mulP1Aff[i] = pubKeys[i].(*PublicKey).p
		rawMsgs[i] = msgs[i][:]
	}
	// Secure source of RNG
	randGen := rand.NewGenerator()

	randFunc := func(scalar *blst.Scalar) {
		var rbytes [scalarBytes]byte
		randGen.Read(rbytes[:])
		scalar.FromBEndian(rbytes[:])
	}
	dummySig := new(blst.P2Affine)
	return dummySig.MultipleAggregateVerify(mulP2Aff, mulP1Aff, rawMsgs, dst, randFunc, randBitsEntropy), nil
}

// Marshal a signature into a LittleEndian byte slice.
func (s *Signature) Marshal() []byte {
	if featureconfig.Get().SkipBLSVerify {
		return make([]byte, params.BeaconConfig().BLSSignatureLength)
	}

	return s.s.Compress()
}

// Copy returns a full deep copy of a signature.
func (s *Signature) Copy() iface.Signature {
	return &Signature{s: &*s.s}
}

// TODO: Add tests...
func VerifyCompressed(signature []byte, pub []byte, msg []byte) bool {
	return new(blstSignature).VerifyCompressed(signature, pub, msg, dst)
}
