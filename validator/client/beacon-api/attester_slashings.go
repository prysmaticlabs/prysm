package beacon_api

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (c beaconApiSlasherClient) getSlashableAttestations(ctx context.Context, in *ethpb.IndexedAttestation) (*ethpb.AttesterSlashingResponse, error) {
	attesterSlashingsPoolResponse := apimiddleware.AttesterSlashingsPoolResponseJson{}
	if _, err := c.jsonRestHandler.GetRestJsonResponse(ctx, "/eth/v1/beacon/pool/attester_slashings", &attesterSlashingsPoolResponse); err != nil {
		return nil, errors.Wrap(err, "failed to get attester slashings")
	}

	attesterSlashings, err := convertAttesterSlashingsToProto(attesterSlashingsPoolResponse.Data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert attester slashings")
	}

	type mapKey struct {
		targetEpoch    primitives.Epoch
		attestingIndex uint64
	}

	// Make a map of the requested attesting indices to easily filter them
	attestingIndicesMap := make(map[mapKey]bool, len(in.AttestingIndices))
	for _, attestingIndex := range in.AttestingIndices {
		attestingIndicesMap[mapKey{
			targetEpoch:    in.Data.Target.Epoch,
			attestingIndex: attestingIndex,
		}] = true
	}

	filteredAttesterSlashings := make([]*ethpb.AttesterSlashing, 0)
	for _, attesterSlashing := range attesterSlashings {
		// Keep the intersection of the Attestation_1 and Attestion_2 attesting indices
		attestingIndices1Map := make(map[uint64]bool, len(attesterSlashing.Attestation_1.AttestingIndices))
		for _, attestingIndex := range attesterSlashing.Attestation_1.AttestingIndices {
			attestingIndices1Map[attestingIndex] = true
		}

		attestingIndicesIntersection := make([]uint64, 0, len(attesterSlashing.Attestation_1.AttestingIndices))
		for _, attestingIndex := range attesterSlashing.Attestation_2.AttestingIndices {
			if _, ok := attestingIndices1Map[attestingIndex]; ok {
				attestingIndicesIntersection = append(attestingIndicesIntersection, attestingIndex)
			}
		}

		for _, attestingIndex := range attestingIndicesIntersection {
			_, foundAttestation1Match := attestingIndicesMap[mapKey{
				targetEpoch:    attesterSlashing.Attestation_1.Data.Target.Epoch,
				attestingIndex: attestingIndex,
			}]

			_, foundAttestation2Match := attestingIndicesMap[mapKey{
				targetEpoch:    attesterSlashing.Attestation_2.Data.Target.Epoch,
				attestingIndex: attestingIndex,
			}]

			if foundAttestation1Match || foundAttestation2Match {
				filteredAttesterSlashings = append(filteredAttesterSlashings, attesterSlashing)
				break
			}
		}
	}

	return &ethpb.AttesterSlashingResponse{
		AttesterSlashings: filteredAttesterSlashings,
	}, nil
}
