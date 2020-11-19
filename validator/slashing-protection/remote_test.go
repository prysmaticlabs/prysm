package slashingprotection

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Protector(&RemoteProtector{})

type mockSlasher struct {
	slashAttestation                     bool
	slashBlock                           bool
	isSlashableAttestationCalled         bool
	isSlashableAttestationNoUpdateCalled bool
	isSlashableBlockCalled               bool
	isSlashableBlockNoUpdateCalled       bool
}

// IsSlashableAttestation returns slashbale attestation if slash attestation is set to true.
func (ms mockSlasher) IsSlashableAttestation(_ context.Context, in *eth.IndexedAttestation, _ ...grpc.CallOption) (*slashpb.AttesterSlashingResponse, error) {
	ms.isSlashableAttestationCalled = true
	if ms.slashAttestation {
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
func (ms mockSlasher) IsSlashableAttestationNoUpdate(_ context.Context, _ *eth.IndexedAttestation, _ ...grpc.CallOption) (*slashpb.Slashable, error) {
	ms.isSlashableAttestationNoUpdateCalled = true
	return &slashpb.Slashable{
		Slashable: ms.slashAttestation,
	}, nil

}

// IsSlashableBlock returns proposer slashing if slash block is set to true.
func (ms mockSlasher) IsSlashableBlock(_ context.Context, in *eth.SignedBeaconBlockHeader, _ ...grpc.CallOption) (*slashpb.ProposerSlashingResponse, error) {
	ms.isSlashableBlockCalled = true
	if ms.slashBlock {
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
func (ms mockSlasher) IsSlashableBlockNoUpdate(_ context.Context, _ *eth.BeaconBlockHeader, _ ...grpc.CallOption) (*slashpb.Slashable, error) {
	ms.isSlashableBlockNoUpdateCalled = true
	return &slashpb.Slashable{
		Slashable: ms.slashBlock,
	}, nil
}

func Test_parseSlasherError(t *testing.T) {
	tests := []struct {
		name    string
		errs    []error
		wantErr error
	}{
		{
			name: "Unavailable",
			errs: []error{
				status.Error(codes.Unavailable, "failed to reach server"),
				status.Error(codes.Canceled, "canceled"),
				status.Error(codes.ResourceExhausted, "exhausted"),
			},
			wantErr: ErrSlasherUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, e := range tt.errs {
				err := parseSlasherError(e)
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("parseSlasherError() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}
