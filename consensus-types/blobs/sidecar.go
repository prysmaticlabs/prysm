package blobs

import (
	"math/big"

	gethType "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
	"github.com/protolambda/go-kzg/bls"
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

var (
	ErrInvalidBlobSlot            = errors.New("invalid blob slot")
	ErrInvalidBlobBeaconBlockRoot = errors.New("invalid blob beacon block root")
	ErrInvalidBlobsLength         = errors.New("invalid blobs length")
	ErrCouldNotComputeCommitment  = errors.New("could not compute commitment")
	ErrMissmatchKzgs              = errors.New("missmatch kzgs")

	blsModulus   big.Int
	rootsOfUnity [params.FieldElementsPerBlob]bls.Fr
)

func init() {
	blsModulus.SetString("52435875175126190479447740508185965837690552500527637822603658699938581184513", 10)
}

// VerifyBlobsSidecar verifies the integrity of a sidecar.
// def verify_blobs_sidecar(slot: Slot, beacon_block_root: Root,
//                         expected_kzgs: Sequence[KZGCommitment], blobs_sidecar: BlobsSidecar):
//    assert slot == blobs_sidecar.beacon_block_slot
//    assert beacon_block_root == blobs_sidecar.beacon_block_root
//    blobs = blobs_sidecar.blobs
//    assert len(expected_kzgs) == len(blobs)
//    for kzg, blob in zip(expected_kzgs, blobs):
//        assert blob_to_kzg(blob) == kzg
func VerifyBlobsSidecar(slot types.Slot, beaconBlockRoot [32]byte, expectedKZGs [][48]byte, blobsSidecar *eth.BlobsSidecar) error {
	// TODO(EIP-4844): Apply optimization - https://github.com/ethereum/consensus-specs/blob/0ba5b3b5c5bb58fbe0f094dcd02dedc4ff1c6f7c/specs/eip4844/validator.md#verify_blobs_sidecar
	if slot != blobsSidecar.BeaconBlockSlot {
		return ErrInvalidBlobSlot
	}
	if beaconBlockRoot != bytesutil.ToBytes32(blobsSidecar.BeaconBlockRoot) {
		return ErrInvalidBlobBeaconBlockRoot
	}
	if len(expectedKZGs) != len(blobsSidecar.Blobs) {
		return ErrInvalidBlobsLength
	}
	for i, expectedKzg := range expectedKZGs {
		var blob gethType.Blob
		for i, b := range blobsSidecar.Blobs[i].Blob {
			var f gethType.BLSFieldElement
			copy(f[:], b)
			blob[i] = f
		}
		kzg, ok := blob.ComputeCommitment()
		if !ok {
			return ErrCouldNotComputeCommitment
		}
		if kzg != expectedKzg {
			return ErrMissmatchKzgs
		}
	}
	return nil
}

// BlockContainsSidecar returns true if the block contains an external sidecar and internal kzgs
func BlockContainsSidecar(b interfaces.SignedBeaconBlock) (bool, error) {
	hasKzg, err := BlockContainsKZGs(b.Block())
	if err != nil {
		return false, err
	}
	if !hasKzg {
		return false, nil
	}
	_, err = b.SideCar()
	switch {
	case errors.Is(err, blocks.ErrNilSidecar):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}

func BlockContainsKZGs(b interfaces.BeaconBlock) (bool, error) {
	if blocks.IsPreEIP4844Version(b.Version()) {
		return false, nil
	}
	blobKzgs, err := b.Body().BlobKzgs()
	if err != nil {
		return false, err
	}
	return len(blobKzgs) != 0, nil
}

// computeAggregatedPolyAndCommitment computes and returns the aggregated polynomial and aggregated kzg commitment.
//
// Spec code:
// def compute_aggregated_poly_and_commitment(
//        blobs: Sequence[Sequence[BLSFieldElement]],
//        kzg_commitments: Sequence[KZGCommitment]) -> Tuple[Polynomial, KZGCommitment]:
//    # Generate random linear combination challenges
//    r = hash_to_bls_field(BlobsAndCommitments(blobs=blobs, kzg_commitments=kzg_commitments))
//    r_powers = compute_powers(r, len(kzg_commitments))
//
//    # Create aggregated polynomial in evaluation form
//    aggregated_poly = Polynomial(vector_lincomb(blobs, r_powers))
//
//    # Compute commitment to aggregated polynomial
//    aggregated_poly_commitment = KZGCommitment(g1_lincomb(kzg_commitments, r_powers))
//
//    return aggregated_poly, aggregated_poly_commitment
func computeAggregatedPolyAndCommitment(b []*v1.Blob, c [][]byte) ([]bls.Fr, []byte, error) {
	bc := &eth.BlobsAndCommitments{
		Blobs:       b,
		Commitments: c,
	}

	// Generate random linear combination challenges.
	r, err := hashToBlsField(bc)
	if err != nil {
		return nil, nil, err
	}
	rPowers := computePowers(r, len(c))

	// Create aggregated polynomial in evaluation form.
	ap, err := vectorLinComb(b, rPowers)
	if err != nil {
		return nil, nil, err
	}

	// Create aggregated commitment to aggregated polynomial.
	ac, err := g1LinComb(c, rPowers)
	if err != nil {
		return nil, nil, err
	}

	return ap, ac, nil
}

// hashToBlsField computes the 32-byte hash of serialized container and convert it to BLS field.
// The output is not uniform over the BLS field.
//
// Spec code:
// def hash_to_bls_field(x: Container) -> BLSFieldElement:
//    return bytes_to_bls_field(hash(ssz_serialize(x)))
func hashToBlsField(x ssz.Marshaler) (*bls.Fr, error) {
	m, err := x.MarshalSSZ()
	if err != nil {
		return nil, err
	}

	h := hash.Hash(m)

	var b big.Int
	// Reverse the byte to interpret hash `h` as a little-endian integer then
	// mod it with the BLS modulus.
	b.SetBytes(bytesutil.ReverseByteOrder(h[:])).Mod(&b, &blsModulus)

	// Convert big int from above to field element.
	var f *bls.Fr
	bls.SetFr(f, b.String())
	return f, nil
}

// computePowers returns the power of field element input `x` to power of [0, n-1].
//
// spec code:
// def compute_powers(x: BLSFieldElement, n: uint64) -> Sequence[BLSFieldElement]:
//    current_power = 1
//    powers = []
//    for _ in range(n):
//        powers.append(BLSFieldElement(current_power))
//        current_power = current_power * int(x) % BLS_MODULUS
//    return powers
func computePowers(x *bls.Fr, n int) []bls.Fr {
	currentPower := bls.ONE
	powers := make([]bls.Fr, n)
	for i := range powers {
		powers[i] = currentPower
		bls.MulModFr(&currentPower, &currentPower, x)
	}
	return powers
}

// vectorLinComb interpret the input `vectors` as a 2D matrix and compute the linear combination
// of each column with the input `scalars`.
//
// spec code:
// def vector_lincomb(vectors: Sequence[Sequence[BLSFieldElement]],
//                   scalars: Sequence[BLSFieldElement]) -> Sequence[BLSFieldElement]:
//    result = [0] * len(vectors[0])
//    for v, s in zip(vectors, scalars):
//        for i, x in enumerate(v):
//            result[i] = (result[i] + int(s) * int(x)) % BLS_MODULUS
//    return [BLSFieldElement(x) for x in result]
func vectorLinComb(vectors []*v1.Blob, scalars []bls.Fr) ([]bls.Fr, error) {
	if len(vectors) != len(scalars) {
		return nil, errors.New("vectors and scalars are not the same length")
	}

	results := make([]bls.Fr, params.FieldElementsPerBlob)
	x := bls.Fr{}
	for i, v := range vectors {
		b := v.Blob
		if len(b) != params.FieldElementsPerBlob {
			return nil, errors.New("blob is the wrong size")
		}
		s := scalars[i]
		for j := 0; j < params.FieldElementsPerBlob; j++ { // iterate over a blob's field elements
			ok := bls.FrFrom32(&x, bytesutil.ToBytes32(b[j]))
			if !ok {
				return nil, errors.New("could not convert blob data to field element")
			}
			bls.MulModFr(&x, &x, &s)
			bls.AddModFr(&results[i], &results[i], &x)
		}
	}
	return results, nil
}

// g1LinComb performs BLS multi-scalar multiplication for input `points` to `scalars`.
//
// spec code:
// def g1_lincomb(points: Sequence[KZGCommitment], scalars: Sequence[BLSFieldElement]) -> KZGCommitment:
//    """
//    BLS multiscalar multiplication. This function can be optimized using Pippenger's algorithm and variants.
//    """
//    assert len(points) == len(scalars)
//    result = bls.Z1
//    for x, a in zip(points, scalars):
//        result = bls.add(result, bls.multiply(bls.bytes48_to_G1(x), a))
//    return KZGCommitment(bls.G1_to_bytes48(result))
func g1LinComb(points [][]byte, scalars []bls.Fr) ([]byte, error) {
	if len(points) != len(scalars) {
		return nil, errors.New("points and scalars have to be the same length")
	}
	g1s := make([]bls.G1Point, len(points))
	for i := range g1s {
		g1, err := bls.FromCompressedG1(points[i])
		if err != nil {
			return nil, err
		}
		g1s[i] = *g1
	}
	return bls.ToCompressedG1(bls.LinCombG1(g1s, scalars)), nil
}
