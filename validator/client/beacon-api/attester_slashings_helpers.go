package beacon_api

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func convertAttesterSlashingsToProto(jsonAttesterSlashings []*apimiddleware.AttesterSlashingJson) ([]*ethpb.AttesterSlashing, error) {
	attesterSlashings := make([]*ethpb.AttesterSlashing, len(jsonAttesterSlashings))

	for index, jsonAttesterSlashing := range jsonAttesterSlashings {
		if jsonAttesterSlashing == nil {
			return nil, errors.Errorf("attester slashing at index `%d` is nil", index)
		}

		attestation1, err := convertIndexedAttestationToProto(jsonAttesterSlashing.Attestation_1)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get attestation 1")
		}

		attestation2, err := convertIndexedAttestationToProto(jsonAttesterSlashing.Attestation_2)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get attestation 2")
		}

		attesterSlashings[index] = &ethpb.AttesterSlashing{
			Attestation_1: attestation1,
			Attestation_2: attestation2,
		}
	}

	return attesterSlashings, nil
}

func convertIndexedAttestationToProto(jsonAttestation *apimiddleware.IndexedAttestationJson) (*ethpb.IndexedAttestation, error) {
	if jsonAttestation == nil {
		return nil, errors.New("indexed attestation is nil")
	}

	attestingIndices := make([]uint64, len(jsonAttestation.AttestingIndices))

	for index, jsonAttestingIndex := range jsonAttestation.AttestingIndices {
		attestingIndex, err := strconv.ParseUint(jsonAttestingIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse attesting index `%s`", jsonAttestingIndex)
		}

		attestingIndices[index] = attestingIndex
	}

	signature, err := hexutil.Decode(jsonAttestation.Signature)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode attestation signature `%s`", jsonAttestation.Signature)
	}

	attestationData, err := convertAttestationDataToProto(jsonAttestation.Data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get attestation data")
	}

	return &ethpb.IndexedAttestation{
		AttestingIndices: attestingIndices,
		Data:             attestationData,
		Signature:        signature,
	}, nil
}
