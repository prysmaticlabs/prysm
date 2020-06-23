package testing

import (
	"context"
	"errors"

	"github.com/golang/protobuf/proto"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"google.golang.org/grpc"
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

// IsSlashableAttestation returns slashbale attestation if slash attestation is set to true.
func (ms MockSlasher) IsSlashableAttestation(ctx context.Context, in *eth.IndexedAttestation, opts ...grpc.CallOption) (*slashpb.AttesterSlashingResponse, error) {
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
func (ms MockSlasher) IsSlashableAttestationNoUpdate(ctx context.Context, in *eth.IndexedAttestation, opts ...grpc.CallOption) (*slashpb.Slashable, error) {
	ms.IsSlashableAttestationNoUpdateCalled = true
	return &slashpb.Slashable{
		Slashable: ms.SlashAttestation,
	}, nil

}

// IsSlashableBlock returns proposer slashing if slash block is set to true.
func (ms MockSlasher) IsSlashableBlock(ctx context.Context, in *eth.SignedBeaconBlockHeader, opts ...grpc.CallOption) (*slashpb.ProposerSlashingResponse, error) {
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
func (ms MockSlasher) IsSlashableBlockNoUpdate(ctx context.Context, in *eth.BeaconBlockHeader, opts ...grpc.CallOption) (*slashpb.Slashable, error) {
	ms.IsSlashableBlockNoUpdateCalled = true
	return &slashpb.Slashable{
		Slashable: ms.SlashBlock,
	}, nil
}
