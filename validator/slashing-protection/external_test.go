package slashingprotection

import (
	"context"
	"errors"
	"testing"

	"github.com/gogo/protobuf/proto"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"google.golang.org/grpc"
)

type mockSlasher struct {
	slashAttestation bool
	slashBlock       bool
}

func (ms mockSlasher) IsSlashableAttestation(ctx context.Context, in *eth.IndexedAttestation, opts ...grpc.CallOption) (*slashpb.AttesterSlashingResponse, error) {
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

func (ms mockSlasher) IsSlashableAttestationNoUpdate(ctx context.Context, in *eth.IndexedAttestation, opts ...grpc.CallOption) (*slashpb.Slashable, error) {
	return &slashpb.Slashable{
		Slashable: ms.slashAttestation,
	}, nil

}

func (ms mockSlasher) IsSlashableBlock(ctx context.Context, in *eth.SignedBeaconBlockHeader, opts ...grpc.CallOption) (*slashpb.ProposerSlashingResponse, error) {
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

func (ms mockSlasher) IsSlashableBlockNoUpdate(ctx context.Context, in *eth.BeaconBlockHeader, opts ...grpc.CallOption) (*slashpb.Slashable, error) {
	return &slashpb.Slashable{
		Slashable: ms.slashBlock,
	}, nil
}

func TestService_VerifyAttestation(t *testing.T) {
	s := &Service{slasherClient: mockSlasher{slashAttestation: true}}
	att := &eth.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &eth.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: []byte("great block"),
			Source: &eth.Checkpoint{
				Epoch: 4,
				Root:  []byte("good source"),
			},
			Target: &eth.Checkpoint{
				Epoch: 10,
				Root:  []byte("good target"),
			},
		},
	}
	if s.VerifyAttestation(context.Background(), att) {
		t.Error("Expected verify attestation to fail verification")
	}
	s = &Service{slasherClient: mockSlasher{slashAttestation: false}}
	if !s.VerifyAttestation(context.Background(), att) {
		t.Error("Expected verify attestation to pass verification")
	}
}

func TestService_VerifyBlock(t *testing.T) {
	s := &Service{slasherClient: mockSlasher{slashBlock: true}}
	blk := &eth.SignedBeaconBlockHeader{
		Header: &eth.BeaconBlockHeader{
			Slot:          0,
			ProposerIndex: 0,
			ParentRoot:    []byte("parent"),
			StateRoot:     []byte("state"),
			BodyRoot:      []byte("body"),
		},
	}
	if s.VerifyBlock(context.Background(), blk) {
		t.Error("Expected verify attestation to fail verification")
	}
	s = &Service{slasherClient: mockSlasher{slashBlock: false}}
	if !s.VerifyBlock(context.Background(), blk) {
		t.Error("Expected verify attestation to pass verification")
	}
}
