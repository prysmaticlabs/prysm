package eth_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	upstreampb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// Test that Prysm copied protobufs have the same wire type and tag number.
func TestProtoCompatability(t *testing.T) {
	tests := []struct {
		a proto.Message
		b proto.Message
	}{
		// attestation.proto
		{
			a: &pb.Attestation{},
			b: &upstreampb.Attestation{},
		},
		{
			a: &pb.AttestationData{},
			b: &upstreampb.AttestationData{},
		},
		{
			a: &pb.Checkpoint{},
			b: &upstreampb.Checkpoint{},
		},
		{
			a: &pb.Crosslink{},
			b: &upstreampb.Crosslink{},
		},
		// beacon_block.proto
		{
			a: &pb.BeaconBlock{},
			b: &upstreampb.BeaconBlock{},
		},
		{
			a: &pb.BeaconBlockBody{},
			b: &upstreampb.BeaconBlockBody{},
		},
		{
			a: &pb.ProposerSlashing{},
			b: &upstreampb.ProposerSlashing{},
		},
		{
			a: &pb.AttesterSlashing{},
			b: &upstreampb.AttesterSlashing{},
		},
		{
			a: &pb.Deposit{},
			b: &upstreampb.Deposit{},
		},
		{
			a: &pb.Deposit_Data{},
			b: &upstreampb.Deposit_Data{},
		},
		{
			a: &pb.VoluntaryExit{},
			b: &upstreampb.VoluntaryExit{},
		},
		{
			a: &pb.Transfer{},
			b: &upstreampb.Transfer{},
		},
		{
			a: &pb.Eth1Data{},
			b: &upstreampb.Eth1Data{},
		},
		{
			a: &pb.BeaconBlockHeader{},
			b: &upstreampb.BeaconBlockHeader{},
		},
		{
			a: &pb.IndexedAttestation{},
			b: &upstreampb.IndexedAttestation{},
		},
		// beacon_chain.proto
		{
			a: &pb.ListAttestationsRequest{},
			b: &upstreampb.ListAttestationsRequest{},
		},
		{
			a: &pb.ListAttestationsResponse{},
			b: &upstreampb.ListAttestationsResponse{},
		},
		{
			a: &pb.ListBlocksRequest{},
			b: &upstreampb.ListBlocksRequest{},
		},
		{
			a: &pb.ListBlocksResponse{},
			b: &upstreampb.ListBlocksResponse{},
		},
		{
			a: &pb.ChainHead{},
			b: &upstreampb.ChainHead{},
		},
		{
			a: &pb.GetValidatorBalancesRequest{},
			b: &upstreampb.GetValidatorBalancesRequest{},
		},
		{
			a: &pb.ValidatorBalances{},
			b: &upstreampb.ValidatorBalances{},
		},
		{
			a: &pb.GetValidatorsRequest{},
			b: &upstreampb.GetValidatorsRequest{},
		},
		{
			a: &pb.Validators{},
			b: &upstreampb.Validators{},
		},
		{
			a: &pb.GetValidatorActiveSetChangesRequest{},
			b: &upstreampb.GetValidatorActiveSetChangesRequest{},
		},
		{
			a: &pb.ActiveSetChanges{},
			b: &upstreampb.ActiveSetChanges{},
		},
		{
			a: &pb.ValidatorQueue{},
			b: &upstreampb.ValidatorQueue{},
		},
		{
			a: &pb.ListValidatorAssignmentsRequest{},
			b: &upstreampb.ListValidatorAssignmentsRequest{},
		},
		{
			a: &pb.ValidatorAssignments{},
			b: &upstreampb.ValidatorAssignments{},
		},
		{
			a: &pb.ValidatorAssignments_CommitteeAssignment{},
			b: &upstreampb.ValidatorAssignments_CommitteeAssignment{},
		},
		{
			a: &pb.GetValidatorParticipationRequest{},
			b: &upstreampb.GetValidatorParticipationRequest{},
		},
		{
			a: &pb.ValidatorParticipation{},
			b: &upstreampb.ValidatorParticipation{},
		},
		{
			a: &pb.AttestationPoolResponse{},
			b: &upstreampb.AttestationPoolResponse{},
		},
		// node.proto
		{
			a: &pb.SyncStatus{},
			b: &upstreampb.SyncStatus{},
		},
		{
			a: &pb.Genesis{},
			b: &upstreampb.Genesis{},
		},
		{
			a: &pb.Version{},
			b: &upstreampb.Version{},
		},
		{
			a: &pb.ImplementedServices{},
			b: &upstreampb.ImplementedServices{},
		},
		// validator.proto
		{
			a: &pb.DutiesRequest{},
			b: &upstreampb.DutiesRequest{},
		},
		{
			a: &pb.DutiesResponse{},
			b: &upstreampb.DutiesResponse{},
		},
		{
			a: &pb.DutiesResponse_Duty{},
			b: &upstreampb.DutiesResponse_Duty{},
		},
		{
			a: &pb.BlockRequest{},
			b: &upstreampb.BlockRequest{},
		},
		{
			a: &pb.AttestationDataRequest{},
			b: &upstreampb.AttestationDataRequest{},
		},
		{
			a: &pb.Validator{},
			b: &upstreampb.Validator{},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T", tt.a), func(t *testing.T) {

			propsA := proto.GetProperties(reflect.TypeOf(tt.a).Elem())
			propsB := proto.GetProperties(reflect.TypeOf(tt.b).Elem())

			if propsA.Len() != propsB.Len() {
				t.Fatalf(
					"%T does not have same number of properties (%d) as %T (%d)",
					tt.a,
					propsA.Len(),
					tt.b,
					propsB.Len(),
				)
			}

			for i, propA := range propsA.Prop {
				propB := propsB.Prop[i]

				if propA.Name != propB.Name {
					t.Errorf(
						"%T.%s field is named differently than %T.%s",
						tt.a,
						propA.Name,
						tt.b,
						propB.Name,
					)
				}

				if propA.Wire != propB.Wire {
					t.Errorf(
						"%T.%s has different wire (%s) than %T.%s (%s)",
						tt.a,
						propA.Name,
						propA.Wire,
						tt.b,
						propB.Name,
						propB.Wire,
					)
				}

				if propA.WireType != propB.WireType {
					t.Errorf(
						"%T.%s has different wiretype (%d) than %T.%s (%d)",
						tt.a,
						propA.Name,
						propA.WireType,
						tt.b,
						propB.Name,
						propB.WireType,
					)
				}

				if propA.Tag != propB.Tag {
					t.Errorf(
						"%T.%s has different tag value (%d) than %T.%s (%d)",
						tt.a,
						propA.Name,
						propA.Tag,
						tt.b,
						propB.Name,
						propB.Tag,
					)
				}
			}
		})
	}
}
