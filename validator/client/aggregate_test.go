package client

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
	testing2 "github.com/prysmaticlabs/prysm/testing"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSubmitAggregateAndProof_GetDutiesRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, _, validatorKey, finish := setup(t)
	validator.duties = &ethpb.DutiesResponse{Duties: []*ethpb.DutiesResponse_Duty{}}
	defer finish()

	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	validator.SubmitAggregateAndProof(context.Background(), 0, pubKey)

	require.LogsContain(t, hook, "Could not fetch validator assignment")
}

func TestSubmitAggregateAndProof_SignFails(t *testing.T) {
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	validator.duties = &ethpb.DutiesResponse{
		Duties: []*ethpb.DutiesResponse_Duty{
			{
				PublicKey: validatorKey.PublicKey().Marshal(),
			},
		},
	}

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().SubmitAggregateSelectionProof(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AggregateSelectionRequest{}),
	).Return(&ethpb.AggregateSelectionResponse{
		AggregateAndProof: &ethpb.AggregateAttestationAndProof{
			AggregatorIndex: 0,
			Aggregate: testing2.HydrateAttestation(&ethpb.Attestation{
				AggregationBits: make([]byte, 1),
			}),
			SelectionProof: make([]byte, 96),
		},
	}, nil)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: nil}, errors.New("bad domain root"))

	validator.SubmitAggregateAndProof(context.Background(), 0, pubKey)
}

func TestSubmitAggregateAndProof_Ok(t *testing.T) {
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	validator.duties = &ethpb.DutiesResponse{
		Duties: []*ethpb.DutiesResponse_Duty{
			{
				PublicKey: validatorKey.PublicKey().Marshal(),
			},
		},
	}

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().SubmitAggregateSelectionProof(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AggregateSelectionRequest{}),
	).Return(&ethpb.AggregateSelectionResponse{
		AggregateAndProof: &ethpb.AggregateAttestationAndProof{
			AggregatorIndex: 0,
			Aggregate: testing2.HydrateAttestation(&ethpb.Attestation{
				AggregationBits: make([]byte, 1),
			}),
			SelectionProof: make([]byte, 96),
		},
	}, nil)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	m.validatorClient.EXPECT().SubmitSignedAggregateSelectionProof(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedAggregateSubmitRequest{}),
	).Return(&ethpb.SignedAggregateSubmitResponse{AttestationDataRoot: make([]byte, 32)}, nil)

	validator.SubmitAggregateAndProof(context.Background(), 0, pubKey)
}

func TestWaitForSlotTwoThird_WaitCorrectly(t *testing.T) {
	validator, _, _, finish := setup(t)
	defer finish()
	currentTime := timeutils.Now()
	numOfSlots := types.Slot(4)
	validator.genesisTime = uint64(currentTime.Unix()) - uint64(numOfSlots.Mul(params.BeaconConfig().SecondsPerSlot))
	oneThird := slotutil.DivideSlotBy(3 /* one third of slot duration */)
	timeToSleep := oneThird + oneThird

	twoThirdTime := currentTime.Add(timeToSleep)
	validator.waitToSlotTwoThirds(context.Background(), numOfSlots)
	currentTime = timeutils.Now()
	assert.Equal(t, twoThirdTime.Unix(), currentTime.Unix())
}

func TestWaitForSlotTwoThird_DoneContext_ReturnsImmediately(t *testing.T) {
	validator, _, _, finish := setup(t)
	defer finish()
	currentTime := timeutils.Now()
	numOfSlots := types.Slot(4)
	validator.genesisTime = uint64(currentTime.Unix()) - uint64(numOfSlots.Mul(params.BeaconConfig().SecondsPerSlot))

	expectedTime := timeutils.Now()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	validator.waitToSlotTwoThirds(ctx, numOfSlots)
	currentTime = timeutils.Now()
	assert.Equal(t, expectedTime.Unix(), currentTime.Unix())
}

func TestAggregateAndProofSignature_CanSignValidSignature(t *testing.T) {
	validator, m, validatorKey, finish := setup(t)
	defer finish()

	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		&ethpb.DomainRequest{Epoch: 0, Domain: params.BeaconConfig().DomainAggregateAndProof[:]},
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)

	agg := &ethpb.AggregateAttestationAndProof{
		AggregatorIndex: 0,
		Aggregate: testing2.HydrateAttestation(&ethpb.Attestation{
			AggregationBits: bitfield.NewBitlist(1),
		}),
		SelectionProof: make([]byte, 96),
	}
	sig, err := validator.aggregateAndProofSig(context.Background(), pubKey, agg)
	require.NoError(t, err)
	_, err = bls.SignatureFromBytes(sig)
	require.NoError(t, err)
}
