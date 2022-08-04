package blob

import (
	"math/big"

	"github.com/ethereum/go-ethereum/crypto/kzg"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
	"github.com/protolambda/go-kzg/bls"
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

var (
	blsModulus   big.Int
	rootsOfUnity [params.FieldElementsPerBlob]bls.Fr
)

func init() {
	// blsModulus value copied from protolambda/go-kzg. TODO: submit a PR to request this value be
	// publicly exposed there.
	blsModulus.SetString("52435875175126190479447740508185965837690552500527637822603658699938581184513", 10)

	// Initialize rootsOfUnity which are used by evaluatePolynomialInEvaluationForm
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
	for i := 0; i < params.FieldElementsPerBlob; i++ {
		bigToFr(&rootsOfUnity[i], &current)
		current.Mul(&current, &rootOfUnity).
			Mod(&current, &blsModulus)
	}
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
		Polynomial: make([][]byte, params.FieldElementsPerBlob, params.FieldElementsPerBlob),
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

	y := evaluatePolynomialInEvaluationForm(aggregatedPoly, &x)

	var yFr bls.Fr
	b, err := verifyKZGProof(aggregatedPolyCommitment, &x, bigToFr(&yFr, y), sidecar.AggregatedProof)
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
func hashToBLSField(r *bls.Fr, container ssz.HashRoot) error {
	h, err := container.HashTreeRoot()
	if err != nil {
		return err
	}
	hashToFr(r, h)
	return nil
}

// computePowers implements compute_powers from the EIP-4844 spec
func computePowers(x *bls.Fr, n int) []bls.Fr {
	currentPower := bls.ONE
	powers := make([]bls.Fr, n, n)
	for i := 0; i < n; i++ {
		powers[i] = currentPower
		bls.MulModFr(&currentPower, &currentPower, x)
	}
	return powers
}

// linComb implements the function lincomb from the EIP-4844 spec
func linComb(commitments [][]byte, scalars []bls.Fr) ([]byte, error) {
	n := len(scalars)
	g1s := make([]*bls.G1Point, n, n)
	var err error
	for i := 0; i < n; i++ {
		g1s[i], err = bls.FromCompressedG1(commitments[i])
		if err != nil {
			return nil, err
		}
	}
	r := bls.ZeroG1
	// Can theoretically make this faster using a multi-exponential algo but since n is small it
	// may not matter.
	for i := 0; i < n; i++ {
		bls.MulG1(g1s[i], g1s[i], &scalars[i])
		bls.AddG1(&r, &r, g1s[i])
	}
	return bls.ToCompressedG1(&r), nil
}

// vectorLinComb implements the function vector_lincomb from the EIP-4844 spec
func vectorLinComb(blobs []*v1.Blob, scalars []bls.Fr) ([]bls.Fr, error) {
	r := make([]bls.Fr, params.FieldElementsPerBlob, params.FieldElementsPerBlob)
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

// evaluatePolynomialInEvaluationForm implement the function evaluate_polynomial_in_evaluation_form
// from the EIP-4844 spec
func evaluatePolynomialInEvaluationForm(poly []bls.Fr, x *bls.Fr) *big.Int {
	var tmp, tmpSub, r bls.Fr
	for i := 0; i < params.FieldElementsPerBlob; i++ {
		bls.SubModFr(&tmpSub, x, &rootsOfUnity[i])
		bls.MulModFr(&tmp, &poly[i], &rootsOfUnity[i])
		bls.DivModFr(&tmp, &tmp, &tmpSub)
		bls.AddModFr(&r, &r, &tmp)
	}
	// seems PowModInv() isn't implemented in go-kzg except for the pure bignum impl so we have
	// to convert to bigint here for the final computation. TODO: add this to go-kzg
	var yChallenge, width, inv, t, xc big.Int
	width.SetInt64(params.FieldElementsPerBlob)
	inv.ModInverse(&width, &blsModulus)
	frToBig(&xc, x)
	frToBig(&yChallenge, &r)
	t.Exp(&xc, &width, &blsModulus).
		Sub(&t, big.NewInt(1)).
		Mul(&t, &inv)
	yChallenge.Mul(&yChallenge, &t).
		Mod(&yChallenge, &blsModulus)
	return &yChallenge
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
