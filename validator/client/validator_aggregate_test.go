package client

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSubmitAggregateAndProof_GetDutiesRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, _, finish := setup(t)
	validator.duties = &ethpb.DutiesResponse{Duties: []*ethpb.DutiesResponse_Duty{}}
	defer finish()

	validator.SubmitAggregateAndProof(context.Background(), 0, validatorPubKey)

	testutil.AssertLogsContain(t, hook, "Could not fetch validator assignment")
}

func TestSubmitAggregateAndProof_Ok(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()
	validator.duties = &ethpb.DutiesResponse{
		Duties: []*ethpb.DutiesResponse_Duty{
			{
				PublicKey: validatorKey.PublicKey.Marshal(),
			},
		},
	}

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().SubmitAggregateAndProof(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AggregationRequest{}),
	).Return(&ethpb.AggregationResponse{}, nil)

	validator.SubmitAggregateAndProof(context.Background(), 0, validatorPubKey)
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
