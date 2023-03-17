package mock

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type MockSlashingChecker struct {
	AttesterSlashingFound bool
	ProposerSlashingFound bool
	HighestAtts           map[primitives.ValidatorIndex]*ethpb.HighestAttestation
}

func (s *MockSlashingChecker) HighestAttestations(
	_ context.Context, indices []primitives.ValidatorIndex,
) ([]*ethpb.HighestAttestation, error) {
	atts := make([]*ethpb.HighestAttestation, 0, len(indices))
	for _, valIdx := range indices {
		att, ok := s.HighestAtts[valIdx]
		if !ok {
			continue
		}
		atts = append(atts, att)
	}
	return atts, nil
}

func (s *MockSlashingChecker) IsSlashableBlock(_ context.Context, _ *ethpb.SignedBeaconBlockHeader) (*ethpb.ProposerSlashing, error) {
	if s.ProposerSlashingFound {
		return &ethpb.ProposerSlashing{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					Slot:          0,
					ProposerIndex: 0,
					ParentRoot:    params.BeaconConfig().ZeroHash[:],
					StateRoot:     params.BeaconConfig().ZeroHash[:],
					BodyRoot:      params.BeaconConfig().ZeroHash[:],
				},
				Signature: params.BeaconConfig().EmptySignature[:],
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					Slot:          0,
					ProposerIndex: 0,
					ParentRoot:    params.BeaconConfig().ZeroHash[:],
					StateRoot:     params.BeaconConfig().ZeroHash[:],
					BodyRoot:      params.BeaconConfig().ZeroHash[:],
				},
				Signature: params.BeaconConfig().EmptySignature[:],
			},
		}, nil
	}
	return nil, nil
}

func (s *MockSlashingChecker) IsSlashableAttestation(_ context.Context, _ *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error) {
	if s.AttesterSlashingFound {
		return []*ethpb.AttesterSlashing{
			{
				Attestation_1: &ethpb.IndexedAttestation{
					Data: &ethpb.AttestationData{},
				},
				Attestation_2: &ethpb.IndexedAttestation{
					Data: &ethpb.AttestationData{},
				},
			},
		}, nil
	}
	return nil, nil
}
