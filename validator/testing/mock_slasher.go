package testing

import (
	"context"
	"errors"

	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// MockSlasher mocks the slasher rpc server.
type MockSlasher struct {
	SlashAttestation             bool
	SlashBlock                   bool
	IsSlashableAttestationCalled bool
	IsSlashableBlockCalled       bool
}

// HighestAttestations will return an empty array of attestations.
func (MockSlasher) HighestAttestations(_ context.Context, _ *eth.HighestAttestationRequest, _ ...grpc.CallOption) (*eth.HighestAttestationResponse, error) {
	return &eth.HighestAttestationResponse{
		Attestations: nil,
	}, nil
}

// IsSlashableAttestation returns slashbale attestation if slash attestation is set to true.
func (ms MockSlasher) IsSlashableAttestation(_ context.Context, in *eth.IndexedAttestation, _ ...grpc.CallOption) (*eth.AttesterSlashingResponse, error) {
	ms.IsSlashableAttestationCalled = true // skipcq: RVV-B0006
	if ms.SlashAttestation {

		slashingAtt, ok := proto.Clone(in).(*eth.IndexedAttestation)
		if !ok {
			return nil, errors.New("object is not of type *eth.IndexedAttestation")
		}
		slashingAtt.Data.BeaconBlockRoot = []byte("slashing")
		slashings := []*eth.AttesterSlashing{{
			Attestation_1: in,
			Attestation_2: slashingAtt,
		},
		}
		return &eth.AttesterSlashingResponse{
			AttesterSlashings: slashings,
		}, nil
	}
	return nil, nil
}

// IsSlashableBlock returns proposer slashing if slash block is set to true.
func (ms MockSlasher) IsSlashableBlock(_ context.Context, in *eth.SignedBeaconBlockHeader, _ ...grpc.CallOption) (*eth.ProposerSlashingResponse, error) {
	ms.IsSlashableBlockCalled = true // skipcq: RVV-B0006
	if ms.SlashBlock {
		slashingBlk, ok := proto.Clone(in).(*eth.SignedBeaconBlockHeader)
		if !ok {
			return nil, errors.New("object is not of type *eth.SignedBeaconBlockHeader")
		}
		slashingBlk.Header.BodyRoot = []byte("slashing")
		slashings := []*eth.ProposerSlashing{{
			Header_1: in,
			Header_2: slashingBlk,
		},
		}
		return &eth.ProposerSlashingResponse{
			ProposerSlashings: slashings,
		}, nil
	}
	return nil, nil
}
