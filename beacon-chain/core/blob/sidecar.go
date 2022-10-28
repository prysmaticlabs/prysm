package blob

import (
	"math/big"
	"math/bits"

	"github.com/ethereum/go-ethereum/crypto/kzg"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
	"github.com/protolambda/go-kzg/bls"
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

var (
	blsModulus   big.Int
	rootsOfUnity []bls.Fr
)

func init() {
	blsModulus.SetString(bls.ModulusStr, 10)

	// Initialize rootsOfUnity which are used by EvaluatePolyInEvaluationForm
	var one big.Int
	one.SetInt64(1)
	var length big.Int
	length.SetInt64(params.FieldElementsPerBlob)

	var divisor big.Int
	divisor.Sub(&blsModulus, &one)
	if new(big.Int).Mod(&divisor, &length).Int64() != 0 {
		panic("MODULUS-1 % FieldElementsPerBlob should equal 0")
	}
	divisor.Div(&divisor, &length) // divisor == MODULUS - 1 / length

	var rootOfUnity big.Int
	rootOfUnity.SetInt64(7) // PRIMITIVE_ROOT_OF_UNITY
	rootOfUnity.Exp(&rootOfUnity, &divisor, &blsModulus)

	current := one
	rootsOfUnity = make([]bls.Fr, params.FieldElementsPerBlob)
	for i := 0; i < params.FieldElementsPerBlob; i++ {
		bigToFr(&rootsOfUnity[i], &current)
		current.Mul(&current, &rootOfUnity).
			Mod(&current, &blsModulus)
	}

	rootsOfUnity = bitReversalPermutation(rootsOfUnity)
}

// Return a copy with bit-reversed permutation. This operation is idempotent.
// l is the array of roots of unity
func bitReversalPermutation(l []bls.Fr) []bls.Fr {
	if !isPowerOfTwo(params.FieldElementsPerBlob) {
		panic("params.FieldElementsPerBlob must be a power of two")
	}

	out := make([]bls.Fr, params.FieldElementsPerBlob)
	for i := range l {
		j := bits.Reverse64(uint64(i)) >> (65 - bits.Len64(params.FieldElementsPerBlob))
		out[i] = l[j]
	}

	return out
}

func isPowerOfTwo(value uint64) bool {
	return value > 0 && (value&(value-1) == 0)
}

// ValidateBlobsSidecar verifies the integrity of a sidecar, returning nil if the blob is valid.
// It implements validate_blob_transaction_wrapper in the EIP-4844 spec.
func ValidateBlobsSidecar(slot types.Slot, root [32]byte, commitments [][]byte, sidecar *eth.BlobsSidecar) error {
	if slot != sidecar.BeaconBlockSlot {
		return errors.New("invalid blob slot")
	}
	if root != bytesutil.ToBytes32(sidecar.BeaconBlockRoot) {
		return errors.New("invalid blob beacon block root")
	}
	if len(commitments) != len(sidecar.Blobs) {
		return errors.New("invalid blobs length")
	}

	rData := eth.BlobsAndCommitments{
		Blobs:       sidecar.Blobs,
		Commitments: commitments,
	}
	var r bls.Fr
	err := hashToBLSField(&r, &rData)
	if err != nil {
		return err
	}

	numberOfBlobs := len(sidecar.Blobs)
	if numberOfBlobs == 0 {
		return errors.New("no blobs found in sidecar")
	}
	rPowers := computePowers(&r, numberOfBlobs)

	aggregatedPolyCommitment, err := linComb(commitments, rPowers)
	if err != nil {
		return err
	}

	aggregatedPoly, err := vectorLinComb(sidecar.Blobs, rPowers)
	if err != nil {
		return err
	}

	xData := eth.PolynomialAndCommitment{
		Polynomial: make([][]byte, params.FieldElementsPerBlob),
		Commitment: aggregatedPolyCommitment,
	}
	for i := 0; i < params.FieldElementsPerBlob; i++ {
		v := bls.FrTo32(&aggregatedPoly[i])
		xData.Polynomial[i] = v[:]
	}
	var x bls.Fr
	err = hashToBLSField(&x, &xData)
	if err != nil {
		return err
	}

	var y bls.Fr
	EvaluatePolyInEvaluationForm(&y, aggregatedPoly, &x)

	b, err := verifyKZGProof(aggregatedPolyCommitment, &x, &y, sidecar.AggregatedProof)
	if err != nil {
		return err
	}
	if !b {
		return errors.New("couldn't verify proof")
	}
	return nil
}

func BlockContainsKZGs(b interfaces.BeaconBlock) bool {
	if blocks.IsPreEIP4844Version(b.Version()) {
		return false
	}
	blobKzgs, err := b.Body().BlobKzgs()
	if err != nil {
		// cannot happen!
		return false
	}
	return len(blobKzgs) != 0
}

// hashToBLSField implements hash_to_bls_field in the EIP-4844 spec, placing the computed field
// element in r and returning error if there was a failure computing the hash.
func hashToBLSField(r *bls.Fr, container ssz.Marshaler) error {
	ssz, err := container.MarshalSSZ()
	if err != nil {
		return err
	}

	hashToFr(r, hash.Hash(ssz))
	return nil
}

// computePowers implements compute_powers from the EIP-4844 spec
func computePowers(x *bls.Fr, n int) []bls.Fr {
	currentPower := bls.ONE
	powers := make([]bls.Fr, n)
	for i := 0; i < n; i++ {
		powers[i] = currentPower
		bls.MulModFr(&currentPower, &currentPower, x)
	}
	return powers
}

// linComb implements the function lincomb from the EIP-4844 spec
func linComb(commitments [][]byte, scalars []bls.Fr) ([]byte, error) {
	n := len(scalars)
	g1s := make([]bls.G1Point, n)
	for i := 0; i < n; i++ {
		g1, err := bls.FromCompressedG1(commitments[i])
		if err != nil {
			return nil, err
		}
		g1s[i] = *g1
	}
	r := bls.LinCombG1(g1s, scalars)
	return bls.ToCompressedG1(r), nil
}

// vectorLinComb implements the function vector_lincomb from the EIP-4844 spec
func vectorLinComb(blobs []*v1.Blob, scalars []bls.Fr) ([]bls.Fr, error) {
	r := make([]bls.Fr, params.FieldElementsPerBlob)
	x := bls.Fr{}
	var fe [32]byte
	feSlice := fe[:]       // create a slice that is backed by a tmp [32]byte array
	for v := range blobs { // iterate over blobs
		blob := blobs[v].Blob
		if len(blob) != params.FieldElementsPerBlob {
			return nil, errors.New("blob is the wrong size")
		}
		for i := 0; i < params.FieldElementsPerBlob; i++ { // iterate over a blob's field elements
			// copy the []byte field element from the blob into the tmp [32]byte array and then
			// convert to bls.Fr.
			copy(feSlice, blob[i])
			ok := bls.FrFrom32(&x, fe)
			if !ok {
				return nil, errors.New("couldn't convert blob data to field element")
			}
			bls.MulModFr(&x, &scalars[v], &x)
			bls.AddModFr(&r[i], &r[i], &x)
		}
	}
	return r, nil
}

func blsModInv(out *big.Int, x *big.Int) {
	if len(x.Bits()) != 0 { // if non-zero
		out.ModInverse(x, &blsModulus)
	}
}

// Evaluate a polynomial (in evaluation form) at an arbitrary point `x`
// Uses the barycentric formula.
func EvaluatePolyInEvaluationForm(yFr *bls.Fr, poly []bls.Fr, x *bls.Fr) {
	bls.EvaluatePolyInEvaluationForm(yFr, poly, x, rootsOfUnity, 0)
}

// verifyKZGProof implements verify_kzg_proof from the EIP-4844 spec
func verifyKZGProof(polynomialKZG []byte, x *bls.Fr, y *bls.Fr, quotientKZG []byte) (bool, error) {
	commitment, err := bls.FromCompressedG1(polynomialKZG)
	if err != nil {
		return false, err
	}
	proof, err := bls.FromCompressedG1(quotientKZG)
	if err != nil {
		return false, err
	}
	// TODO: have an implementation independent of geth's
	return kzg.VerifyKzgProof(commitment, x, y, proof), nil
}

// hashToFr interprets hash value v as a little endian integer, and converts it to a BLS field
// element after modding it with the BLS modulus.
func hashToFr(fr *bls.Fr, v [32]byte) *bls.Fr {
	var b big.Int
	hashToBig(&b, v).
		Mod(&b, &blsModulus)
	return bigToFr(fr, &b)
}

// hashToBig interprets the hash value v as a little-endian integer, puts the result in b,
// and returns it.
func hashToBig(b *big.Int, v [32]byte) *big.Int {
	// big.Int takes big-endian bytes but v is little endian so we byte swap
	for i := 0; i < 16; i++ {
		v[i], v[31-i] = v[31-i], v[i]
	}
	return b.SetBytes(v[:])
}

// bigToFr converts the big.Int represented BLS field element b to the go-kzg library
// representation bls.Fr, putting its value in fr and returning it.
func bigToFr(fr *bls.Fr, b *big.Int) *bls.Fr {
	// TODO: Conversion currently relies the string representation as an intermediary.  Submit a PR
	// to protolambda/go-kzg enabling something more efficient.
	bls.SetFr(fr, b.String())
	return fr
}

// frToBig converts BLS field element fr into a bignum, storing it in b and returning it.
func frToBig(b *big.Int, fr *bls.Fr) *big.Int {
	// TODO: Conversion currently relies the string representation as an intermediary.  Submit a PR
	// to protolambda/go-kzg enabling something more efficient.
	b.SetString(bls.FrStr(fr), 10)
	return b
}
