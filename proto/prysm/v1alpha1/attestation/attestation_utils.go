// Package attestationutil contains useful helpers for converting
// attestations into indexed form.
package attestation

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"go.opencensus.io/trace"
)

// ConvertToIndexed converts attestation to (almost) indexed-verifiable form.
//
// Note about spec pseudocode definition. The state was used by get_attesting_indices to determine
// the attestation committee. Now that we provide this as an argument, we no longer need to provide
// a state.
//
// Spec pseudocode definition:
//
//	def get_indexed_attestation(state: BeaconState, attestation: Attestation) -> IndexedAttestation:
//	 """
//	 Return the indexed attestation corresponding to ``attestation``.
//	 """
//	 attesting_indices = get_attesting_indices(state, attestation.data, attestation.aggregation_bits)
//
//	 return IndexedAttestation(
//	     attesting_indices=sorted(attesting_indices),
//	     data=attestation.data,
//	     signature=attestation.signature,
//	 )
func ConvertToIndexed(ctx context.Context, attestation interfaces.Attestation, committees [][]primitives.ValidatorIndex) (ethpb.IndexedAtt, error) {
	attIndices, err := AttestingIndices(attestation, committees)
	if err != nil {
		return nil, err
	}

	sort.Slice(attIndices, func(i, j int) bool {
		return attIndices[i] < attIndices[j]
	})

	if attestation.Version() < version.Electra {
		return &ethpb.IndexedAttestation{
			Data:             attestation.GetData(),
			Signature:        attestation.GetSignature(),
			AttestingIndices: attIndices,
		}, nil
	}
	return &ethpb.IndexedAttestationElectra{
		Data:             attestation.GetData(),
		Signature:        attestation.GetSignature(),
		AttestingIndices: attIndices,
	}, nil
}

// AttestingIndices returns the attesting participants indices from the attestation data. The
// committee is provided as an argument rather than a imported implementation from the spec definition.
// Having the committee as an argument allows for re-use of beacon committees when possible.
//
// Spec pseudocode definition:
//
//	def get_attesting_indices(state: BeaconState,
//	                       data: AttestationData,
//	                       bits: Bitlist[MAX_VALIDATORS_PER_COMMITTEE]) -> Set[ValidatorIndex]:
//	 """
//	 Return the set of attesting indices corresponding to ``data`` and ``bits``.
//	 """
//	 committee = get_beacon_committee(state, data.slot, data.index)
//	 return set(index for i, index in enumerate(committee) if bits[i])
func AttestingIndices(att interfaces.Attestation, committees [][]primitives.ValidatorIndex) ([]uint64, error) {
	if len(committees) == 0 {
		return []uint64{}, nil
	}

	aggBits := att.GetAggregationBits()

	if att.Version() < version.Electra {
		committee := committees[0]
		if aggBits.Len() != uint64(len(committee)) {
			return nil, fmt.Errorf("bitfield length %d is not equal to committee length %d", aggBits.Len(), len(committee))
		}
		indices := make([]uint64, 0, aggBits.Count())
		p := aggBits.BitIndices()
		for _, idx := range p {
			if idx < len(committee) {
				indices = append(indices, uint64(committee[idx]))
			}
		}
		return indices, nil
	}

	attesters := make([]uint64, 0, len(aggBits))
	committeeOffset := 0
	for _, c := range committees {
		committeeAttesters := make([]uint64, 0, len(c))
		for i, vi := range c {
			if aggBits[committeeOffset+i] == 1 {
				committeeAttesters = append(committeeAttesters, uint64(c[vi]))
			}
		}
		attesters = append(attesters, committeeAttesters...)
		committeeOffset += len(c)
	}

	return attesters, nil
}

// VerifyIndexedAttestationSig this helper function performs the last part of the
// spec indexed attestation validation starting at Verify aggregate signature
// comment.
//
// Spec pseudocode definition:
//
//	def is_valid_indexed_attestation(state: BeaconState, indexed_attestation: IndexedAttestation) -> bool:
//	 """
//	 Check if ``indexed_attestation`` is not empty, has sorted and unique indices and has a valid aggregate signature.
//	 """
//	 # Verify indices are sorted and unique
//	 indices = indexed_attestation.attesting_indices
//	 if len(indices) == 0 or not indices == sorted(set(indices)):
//	     return False
//	 # Verify aggregate signature
//	 pubkeys = [state.validators[i].pubkey for i in indices]
//	 domain = get_domain(state, DOMAIN_BEACON_ATTESTER, indexed_attestation.data.target.epoch)
//	 signing_root = compute_signing_root(indexed_attestation.data, domain)
//	 return bls.FastAggregateVerify(pubkeys, signing_root, indexed_attestation.signature)
func VerifyIndexedAttestationSig(ctx context.Context, indexedAtt ethpb.IndexedAtt, pubKeys []bls.PublicKey, domain []byte) error {
	_, span := trace.StartSpan(ctx, "attestationutil.VerifyIndexedAttestationSig")
	defer span.End()
	indices := indexedAtt.GetAttestingIndices()

	messageHash, err := signing.ComputeSigningRoot(indexedAtt.GetData(), domain)
	if err != nil {
		return errors.Wrap(err, "could not get signing root of object")
	}

	sig, err := bls.SignatureFromBytes(indexedAtt.GetSignature())
	if err != nil {
		return errors.Wrap(err, "could not convert bytes to signature")
	}

	voted := len(indices) > 0
	if voted && !sig.FastAggregateVerify(pubKeys, messageHash) {
		return signing.ErrSigFailedToVerify
	}
	return nil
}

// IsValidAttestationIndices this helper function performs the first part of the
// spec indexed attestation validation starting at Check if “indexed_attestation“
// comment and ends at Verify aggregate signature comment.
//
// Spec pseudocode definition:
//
//	def is_valid_indexed_attestation(state: BeaconState, indexed_attestation: IndexedAttestation) -> bool:
//	  """
//	  Check if ``indexed_attestation`` is not empty, has sorted and unique indices and has a valid aggregate signature.
//	  """
//	  # Verify indices are sorted and unique
//	  indices = indexed_attestation.attesting_indices
//	  if len(indices) == 0 or not indices == sorted(set(indices)):
//	      return False
//	  # Verify aggregate signature
//	  pubkeys = [state.validators[i].pubkey for i in indices]
//	  domain = get_domain(state, DOMAIN_BEACON_ATTESTER, indexed_attestation.data.target.epoch)
//	  signing_root = compute_signing_root(indexed_attestation.data, domain)
//	  return bls.FastAggregateVerify(pubkeys, signing_root, indexed_attestation.signature)
func IsValidAttestationIndices(ctx context.Context, indexedAttestation ethpb.IndexedAtt) error {
	_, span := trace.StartSpan(ctx, "attestationutil.IsValidAttestationIndices")
	defer span.End()

	if indexedAttestation == nil ||
		indexedAttestation.GetData() == nil ||
		indexedAttestation.GetData().Target == nil ||
		indexedAttestation.GetAttestingIndices() == nil {
		return errors.New("nil or missing indexed attestation data")
	}
	indices := indexedAttestation.GetAttestingIndices()
	if len(indices) == 0 {
		return errors.New("expected non-empty attesting indices")
	}
	maxLength := params.BeaconConfig().MaxValidatorsPerCommittee * params.BeaconConfig().MaxCommitteesPerSlot
	if uint64(len(indices)) > maxLength {
		return fmt.Errorf("validator indices count exceeds MAX_VALIDATORS_PER_COMMITTEE * MAX_COMMITTEES_PER_SLOT, %d > %d", len(indices), maxLength)
	}
	for i := 1; i < len(indices); i++ {
		if indices[i-1] >= indices[i] {
			return errors.New("attesting indices is not uniquely sorted")
		}
	}
	return nil
}

// AttDataIsEqual this function performs an equality check between 2 attestation data, if they're unequal, it will return false.
func AttDataIsEqual(attData1, attData2 *ethpb.AttestationData) bool {
	if attData1.Slot != attData2.Slot {
		return false
	}
	if attData1.CommitteeIndex != attData2.CommitteeIndex {
		return false
	}
	if !bytes.Equal(attData1.BeaconBlockRoot, attData2.BeaconBlockRoot) {
		return false
	}
	if attData1.Source.Epoch != attData2.Source.Epoch {
		return false
	}
	if !bytes.Equal(attData1.Source.Root, attData2.Source.Root) {
		return false
	}
	if attData1.Target.Epoch != attData2.Target.Epoch {
		return false
	}
	if !bytes.Equal(attData1.Target.Root, attData2.Target.Root) {
		return false
	}
	return true
}

// CheckPointIsEqual performs an equality check between 2 check points, returns false if unequal.
func CheckPointIsEqual(checkPt1, checkPt2 *ethpb.Checkpoint) bool {
	if checkPt1.Epoch != checkPt2.Epoch {
		return false
	}
	if !bytes.Equal(checkPt1.Root, checkPt2.Root) {
		return false
	}
	return true
}
