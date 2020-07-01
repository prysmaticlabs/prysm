package herumi

import (
	"fmt"
	"math"

	"github.com/prysmaticlabs/prysm/shared/rand"

	bls12 "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bls/iface"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Signature used in the BLS signature scheme.
type Signature struct {
	s *bls12.Sign
}

// SignatureFromBytes creates a BLS signature from a LittleEndian byte slice.
func SignatureFromBytes(sig []byte) (iface.Signature, error) {
	if featureconfig.Get().SkipBLSVerify {
		return &Signature{}, nil
	}
	if len(sig) != params.BeaconConfig().BLSSignatureLength {
		return nil, fmt.Errorf("signature must be %d bytes", params.BeaconConfig().BLSSignatureLength)
	}
	signature := &bls12.Sign{}
	err := signature.Deserialize(sig)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal bytes into signature")
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
	return s.s.VerifyByte(pubKey.(*PublicKey).p, msg)
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
	msgSlices := make([]byte, 0, 32*len(msgs))
	rawKeys := make([]bls12.PublicKey, 0, len(pubKeys))
	for i := 0; i < size; i++ {
		msgSlices = append(msgSlices, msgs[i][:]...)
		rawKeys = append(rawKeys, *(pubKeys[i].(*PublicKey).p))
	}
	// Use "NoCheck" because we do not care if the messages are unique or not.
	return s.s.AggregateVerifyNoCheck(rawKeys, msgSlices)
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
	rawKeys := make([]bls12.PublicKey, len(pubKeys))
	for i := 0; i < len(pubKeys); i++ {
		rawKeys[i] = *(pubKeys[i].(*PublicKey).p)
	}

	return s.s.FastAggregateVerify(rawKeys, msg[:])
}

// NewAggregateSignature creates a blank aggregate signature.
func NewAggregateSignature() iface.Signature {
	return &Signature{s: bls12.HashAndMapToSignature([]byte{'m', 'o', 'c', 'k'})}
}

// AggregateSignatures converts a list of signatures into a single, aggregated sig.
func AggregateSignatures(sigs []iface.Signature) iface.Signature {
	if len(sigs) == 0 {
		return nil
	}
	if featureconfig.Get().SkipBLSVerify {
		return sigs[0]
	}

	signature := *sigs[0].Copy().(*Signature).s
	for i := 1; i < len(sigs); i++ {
		signature.Add(sigs[i].(*Signature).s)
	}
	return &Signature{s: &signature}
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

func VerifyMultipleSignatures(sigs []iface.Signature, msgs [][32]byte, pubKeys []iface.PublicKey) (bool, error) {
	if featureconfig.Get().SkipBLSVerify {
		return true, nil
	}
	if len(sigs) == 0 || len(pubKeys) == 0 {
		return false, nil
	}
	newGen := rand.NewGenerator()
	randNums := make([]bls12.Fr, len(sigs))
	for i := 0; i < len(sigs); i++ {
		if err := randNums[i].SetLittleEndian(bytesutil.Bytes8(uint64(newGen.Int63n(math.MaxInt64)))); err != nil {
			return false, err
		}
	}
	signatures := make([]bls12.G2, len(sigs))
	for i := 0; i < len(sigs); i++ {
		signatures[i] = *bls12.CastFromSign(sigs[i].(*Signature).s)
	}
	finalSig := new(bls12.G2)
	bls12.G2MulVec(finalSig, signatures, randNums)
	multiKeys := make([]bls12.PublicKey, len(pubKeys))
	for i := 0; i < len(pubKeys); i++ {
		g1 := new(bls12.G1)
		bls12.G1Mul(g1, bls12.CastFromPublicKey(pubKeys[i].(*PublicKey).p), &randNums[i])
		multiKeys[i] = *bls12.CastToPublicKey(g1)

	}
	msgSlices := make([]byte, 0, 32*len(msgs))
	for i := 0; i < len(msgs); i++ {
		msgSlices = append(msgSlices, msgs[i][:]...)

	}
	return bls12.CastToSign(finalSig).AggregateVerifyNoCheck(multiKeys, msgSlices), nil
}

func VerifyStuff() bool {
	secKey := &bls12.SecretKey{}
	secKey2 := &bls12.SecretKey{}

	secKey.SetByCSPRNG()
	secKey2.SetByCSPRNG()
	msg := bytesutil.ToBytes32([]byte("hello"))
	msg2 := bytesutil.ToBytes32([]byte("hello2"))
	sig := secKey.SignByte(msg[:])
	sig2 := secKey2.SignByte(msg2[:])
	rd1 := new(bls12.Fr)
	rd2 := new(bls12.Fr)
	rd1.SetInt64(127627632)
	rd2.SetInt64(9289382392)
	g2 := bls12.CastFromSign(sig)
	g22 := bls12.CastFromSign(sig2)
	newg2 := new(bls12.G2)
	newg22 := new(bls12.G2)
	newg1 := new(bls12.G1)
	newg12 := new(bls12.G1)
	bls12.G2Mul(newg2, g2, rd1)
	bls12.G1Mul(newg1, bls12.CastFromPublicKey(secKey.GetPublicKey()), rd1)
	bls12.G2Mul(newg22, g22, rd2)
	bls12.G1Mul(newg12, bls12.CastFromPublicKey(secKey2.GetPublicKey()), rd2)
	yes := bls12.CastToSign(newg2).VerifyByte(bls12.CastToPublicKey(newg1), msg[:])
	if !yes {
		return false
	}
	yes = bls12.CastToSign(newg22).VerifyByte(bls12.CastToPublicKey(newg12), msg2[:])
	if !yes {
		return false
	}
	add3 := new(bls12.G2)
	bls12.G2Add(add3, newg2, newg22)
	return bls12.CastToSign(add3).AggregateVerifyNoCheck([]bls12.PublicKey{*bls12.CastToPublicKey(newg1), *bls12.CastToPublicKey(newg12)},
		append(msg[:], msg2[:]...))
}

// Marshal a signature into a LittleEndian byte slice.
func (s *Signature) Marshal() []byte {
	if featureconfig.Get().SkipBLSVerify {
		return make([]byte, params.BeaconConfig().BLSSignatureLength)
	}

	return s.s.Serialize()
}

// Copy returns a full deep copy of a signature.
func (s *Signature) Copy() iface.Signature {
	return &Signature{s: &*s.s}
}
