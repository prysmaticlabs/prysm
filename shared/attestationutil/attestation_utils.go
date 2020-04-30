package attestationutil

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"go.opencensus.io/trace"
)

// ConvertToIndexed converts attestation to (almost) indexed-verifiable form.
//
// Note about spec pseudocode definition. The state was used by get_attesting_indices to determine
// the attestation committee. Now that we provide this as an argument, we no longer need to provide
// a state.
//
// Spec pseudocode definition:
//   def get_indexed_attestation(state: BeaconState, attestation: Attestation) -> IndexedAttestation:
//    """
//    Return the indexed attestation corresponding to ``attestation``.
//    """
//    attesting_indices = get_attesting_indices(state, attestation.data, attestation.aggregation_bits)
//
//    return IndexedAttestation(
//        attesting_indices=sorted(attesting_indices),
//        data=attestation.data,
//        signature=attestation.signature,
//    )
func ConvertToIndexed(ctx context.Context, attestation *ethpb.Attestation, committee []uint64) *ethpb.IndexedAttestation {
	ctx, span := trace.StartSpan(ctx, "attestationutil.ConvertToIndexed")
	defer span.End()

	attIndices := AttestingIndices(attestation.AggregationBits, committee)

	sort.Slice(attIndices, func(i, j int) bool {
		return attIndices[i] < attIndices[j]
	})
	inAtt := &ethpb.IndexedAttestation{
		Data:             attestation.Data,
		Signature:        attestation.Signature,
		AttestingIndices: attIndices,
	}
	return inAtt
}

// AttestingIndices returns the attesting participants indices from the attestation data. The
// committee is provided as an argument rather than a direct implementation from the spec definition.
// Having the committee as an argument allows for re-use of beacon committees when possible.
//
// Spec pseudocode definition:
//   def get_attesting_indices(state: BeaconState,
//                          data: AttestationData,
//                          bits: Bitlist[MAX_VALIDATORS_PER_COMMITTEE]) -> Set[ValidatorIndex]:
//    """
//    Return the set of attesting indices corresponding to ``data`` and ``bits``.
//    """
//    committee = get_beacon_committee(state, data.slot, data.index)
//    return set(index for i, index in enumerate(committee) if bits[i])
func AttestingIndices(bf bitfield.Bitfield, committee []uint64) []uint64 {
	indices := make([]uint64, 0, len(committee))
	for _, idx := range bf.BitIndices() {
		if idx < len(committee) {
			indices = append(indices, committee[idx])
		}
	}
	return indices
}

// VerifyIndexedAttestation this helper function performs the last part of the
// spec indexed attestation validation starting at Verify aggregate signature
// comment.
//
// Spec pseudocode definition:
//  def is_valid_indexed_attestation(state: BeaconState, indexed_attestation: IndexedAttestation) -> bool:
//    """
//    Check if ``indexed_attestation`` has valid indices and signature.
//    """
//    indices = indexed_attestation.attesting_indices
//
//    # Verify max number of indices
//    if not len(indices) <= MAX_VALIDATORS_PER_COMMITTEE:
//        return False
//    # Verify indices are sorted and unique
//        if not indices == sorted(set(indices)):
//    # Verify aggregate signature
//    if not bls_verify(
//        pubkey=bls_aggregate_pubkeys([state.validators[i].pubkey for i in indices]),
//        message_hash=hash_tree_root(indexed_attestation.data),
//        signature=indexed_attestation.signature,
//        domain=get_domain(state, DOMAIN_BEACON_ATTESTER, indexed_attestation.data.target.epoch),
//    ):
//        return False
//    return True
func VerifyIndexedAttestation(ctx context.Context, indexedAtt *ethpb.IndexedAttestation, pubKeys []*bls.PublicKey, domain []byte) error {
	ctx, span := trace.StartSpan(ctx, "attestationutil.VerifyIndexedAttestation")
	defer span.End()
	indices := indexedAtt.AttestingIndices
	messageHash, err := helpers.ComputeSigningRoot(indexedAtt.Data, domain)
	if err != nil {
		return errors.Wrap(err, "could not get signing root of object")
	}

	sig, err := bls.SignatureFromBytes(indexedAtt.Signature)
	if err != nil {
		return errors.Wrap(err, "could not convert bytes to signature")
	}

	voted := len(indices) > 0
	if voted && !sig.FastAggregateVerify(pubKeys, messageHash) {
		return helpers.ErrSigFailedToVerify
	}
	return nil
}
