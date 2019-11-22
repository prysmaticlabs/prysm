package client

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSubmitAggregateAndProof_AssignmentRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, _, finish := setup(t)
	validator.assignments = &pb.AssignmentResponse{ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{}}
	defer finish()

	validator.SubmitAggregateAndProof(context.Background(), 0, validatorPubKey)

	testutil.AssertLogsContain(t, hook, "Could not fetch validator assignment")
}

func TestSubmitAggregateAndProof_Ok(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()
	validator.assignments = &pb.AssignmentResponse{ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{
		{
			PublicKey: validatorKey.PublicKey.Marshal(),
		},
	}}

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&pb.DomainResponse{}, nil /*err*/)

	m.aggregatorClient.EXPECT().SubmitAggregateAndProof(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.AggregationRequest{}),
	).Return(&pb.AggregationResponse{}, nil)

	validator.SubmitAggregateAndProof(context.Background(), 0, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Assigned and submitted aggregation and proof request")
}
