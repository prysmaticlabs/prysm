package testing

import (
	"context"
	"errors"

	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// MockSlasher mocks the slasher rpc server.
type MockSlasher struct {
	SlashAttestation                     bool
	SlashBlock                           bool
	IsSlashableAttestationCalled         bool
	IsSlashableAttestationNoUpdateCalled bool
	IsSlashableBlockCalled               bool
	IsSlashableBlockNoUpdateCalled       bool
}

// HighestAttestations will return an empty array of attestations.
func (ms MockSlasher) HighestAttestations(ctx context.Context, req *slashpb.HighestAttestationRequest, _ ...grpc.CallOption) (*slashpb.HighestAttestationResponse, error) {
	return &slashpb.HighestAttestationResponse{
		Attestations: nil,
	}, nil
}

// IsSlashableAttestation returns slashbale attestation if slash attestation is set to true.
func (ms MockSlasher) IsSlashableAttestation(_ context.Context, in *eth.IndexedAttestation, _ ...grpc.CallOption) (*slashpb.AttesterSlashingResponse, error) {
	ms.IsSlashableAttestationCalled = true
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
		return &slashpb.AttesterSlashingResponse{
			AttesterSlashing: slashings,
		}, nil
	}
	return nil, nil
}

// IsSlashableAttestationNoUpdate returns slashbale if slash attestation is set to true.
func (ms MockSlasher) IsSlashableAttestationNoUpdate(_ context.Context, _ *eth.IndexedAttestation, _ ...grpc.CallOption) (*slashpb.Slashable, error) {
	ms.IsSlashableAttestationNoUpdateCalled = true
	return &slashpb.Slashable{
		Slashable: ms.SlashAttestation,
	}, nil

}

// IsSlashableBlock returns proposer slashing if slash block is set to true.
func (ms MockSlasher) IsSlashableBlock(_ context.Context, in *eth.SignedBeaconBlockHeader, _ ...grpc.CallOption) (*slashpb.ProposerSlashingResponse, error) {
	ms.IsSlashableBlockCalled = true
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
		return &slashpb.ProposerSlashingResponse{
			ProposerSlashing: slashings,
		}, nil
	}
	return nil, nil
}

// IsSlashableBlockNoUpdate returns slashbale if slash block is set to true.
func (ms MockSlasher) IsSlashableBlockNoUpdate(_ context.Context, _ *eth.BeaconBlockHeader, _ ...grpc.CallOption) (*slashpb.Slashable, error) {
	ms.IsSlashableBlockNoUpdateCalled = true
	return &slashpb.Slashable{
		Slashable: ms.SlashBlock,
	}, nil
}
