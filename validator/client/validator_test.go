package client

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/internal"
)

var _ = Validator(&validator{})

var fakePubKey = &pb.PublicKey{}

func TestUpdateAssignmentsDoesNothingWhenNotEpochStart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	slot := uint64(1)
	v := validator{
		pubKey:          fakePubKey,
		validatorClient: client,
	}
	client.EXPECT().ValidatorEpochAssignments(
		gomock.Any(),
		gomock.Any(),
	).Times(0)

	v.UpdateAssignments(context.Background(), slot)
}

func TestUpdateAssignmentsReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	v := validator{
		pubKey:          fakePubKey,
		validatorClient: client,
	}

	expected := errors.New("bad")

	client.EXPECT().ValidatorEpochAssignments(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, expected)

	err := v.UpdateAssignments(context.Background(), params.BeaconConfig().EpochLength)
	if err != expected {
		t.Errorf("Bad error; want=%v got=%v", expected, err)
	}
}

func TestUpdateAssignmentsDoesUpdateAssignments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := internal.NewMockValidatorServiceClient(ctrl)

	slot := params.BeaconConfig().EpochLength
	resp := &pb.ValidatorEpochAssignmentsResponse{
		EpochStart: slot,
		Assignments: []*pb.Assignment{
			&pb.Assignment{
				Role:         pb.ValidatorRole_PROPOSER,
				AssignedSlot: 67,
			},
			&pb.Assignment{
				Role:         pb.ValidatorRole_ATTESTER,
				AssignedSlot: 78,
			},
		},
	}
	v := validator{
		pubKey:          fakePubKey,
		validatorClient: client,
	}
	client.EXPECT().ValidatorEpochAssignments(
		gomock.Any(),
		gomock.Any(),
	).Return(resp, nil)

	v.UpdateAssignments(context.Background(), slot)

	expected := map[uint64]*pb.Assignment{
		67: resp.Assignments[0],
		78: resp.Assignments[1],
	}
	if !reflect.DeepEqual(v.assignments, expected) {
		t.Errorf("Unexpected validator assignments. want=%v got=%v", expected, v.assignments)
	}
}
