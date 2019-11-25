package client

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
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

func TestWaitForSlotTwoThird_WaitCorrectly(t *testing.T) {
	validator, _, finish := setup(t)
	defer finish()
	currentTime := uint64(time.Now().Unix())
	numOfSlots := uint64(4)
	validator.genesisTime = currentTime - (numOfSlots * params.BeaconConfig().SecondsPerSlot)
	timeToSleep := params.BeaconConfig().SecondsPerSlot * 2 / 3
	twoThirdTime := currentTime + timeToSleep
	validator.waitToSlotTwoThirds(context.Background(), numOfSlots)

	currentTime = uint64(time.Now().Unix())
	if currentTime != twoThirdTime {
		t.Errorf("Wanted %d time for slot two third but got %d", twoThirdTime, currentTime)
	}
}
