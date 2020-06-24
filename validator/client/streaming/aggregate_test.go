package streaming

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	logTest "github.com/sirupsen/logrus/hooks/test"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestSubmitAggregateAndProof_GetDutiesRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, _, finish := setup(t)
	validator.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty)
	validator.dutiesByEpoch[0] = []*ethpb.DutiesResponse_Duty{}
	defer finish()

	validator.SubmitAggregateAndProof(context.Background(), 0, validatorPubKey)

	testutil.AssertLogsContain(t, hook, "Could not fetch validator assignment")
}

func TestSubmitAggregateAndProof_Ok(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()
	validator.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty)
	validator.dutiesByEpoch[0] = []*ethpb.DutiesResponse_Duty{
		{
			PublicKey: validatorKey.PublicKey.Marshal(),
		},
	}

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().SubmitAggregateSelectionProof(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AggregateSelectionRequest{}),
	).Return(&ethpb.AggregateSelectionResponse{
		AggregateAndProof: &ethpb.AggregateAttestationAndProof{
			AggregatorIndex: 0,
			Aggregate:       &ethpb.Attestation{Data: &ethpb.AttestationData{}},
			SelectionProof:  nil,
		},
	}, nil)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().SubmitSignedAggregateSelectionProof(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedAggregateSubmitRequest{}),
	).Return(&ethpb.SignedAggregateSubmitResponse{}, nil)

	validator.SubmitAggregateAndProof(context.Background(), 0, validatorPubKey)
}

func TestWaitForSlotTwoThird_WaitCorrectly(t *testing.T) {
	validator, _, finish := setup(t)
	defer finish()
	currentTime := roughtime.Now()
	numOfSlots := uint64(4)
	validator.genesisTime = uint64(currentTime.Unix()) - (numOfSlots * params.BeaconConfig().SecondsPerSlot)
	oneThird := slotutil.DivideSlotBy(3 /* one third of slot duration */)
	timeToSleep := oneThird + oneThird

	twoThirdTime := currentTime.Add(timeToSleep)
	validator.waitToSlotTwoThirds(context.Background(), numOfSlots)
	currentTime = roughtime.Now()
	if currentTime.Unix() != twoThirdTime.Unix() {
		t.Errorf("Wanted %v time for slot two third but got %v", twoThirdTime, currentTime)
	}
}

func TestAggregateAndProofSignature_CanSignValidSignature(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		&ethpb.DomainRequest{Epoch: 0, Domain: params.BeaconConfig().DomainAggregateAndProof[:]},
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	agg := &ethpb.AggregateAttestationAndProof{
		AggregatorIndex: 0,
		Aggregate:       &ethpb.Attestation{Data: &ethpb.AttestationData{}},
		SelectionProof:  nil,
	}
	sig, err := validator.aggregateAndProofSig(context.Background(), validatorPubKey, agg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bls.SignatureFromBytes(sig); err != nil {
		t.Fatal(err)
	}
}
