package client

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestRequestAttestation_ValidatorAssignmentRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, _, finish := setup(t)
	validator.assignments = &pb.AssignmentResponse{ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{}}
	defer finish()

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Could not fetch validator assignment")
}

func TestAttestToBlockHead_RequestAttestationFailure(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	validator.assignments = &pb.AssignmentResponse{ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
		},
	}}
	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.ValidatorIndexRequest{}),
	).Return(&pb.ValidatorIndexResponse{Index: 5}, nil)

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Could not get validator index in assignment")
}

func TestAttestToBlockHead_SubmitAttestationRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	validator.assignments = &pb.AssignmentResponse{ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      make([]uint64, 111),
		}}}
	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.ValidatorIndexRequest{}),
	).Return(&pb.ValidatorIndexResponse{
		Index: 0,
	}, nil)
	m.attesterClient.EXPECT().RequestAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.AttestationRequest{}),
	).Return(&ethpb.AttestationData{
		BeaconBlockRoot: []byte{},
		Target:          &ethpb.Checkpoint{},
		Source:          &ethpb.Checkpoint{},
	}, nil)
	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch2
	).Return(&pb.DomainResponse{}, nil /*err*/)
	m.attesterClient.EXPECT().SubmitAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.Attestation{}),
	).Return(nil, errors.New("something went wrong"))

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Could not submit attestation to beacon node")
}

func TestAttestToBlockHead_AttestsCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	validatorIndex := uint64(7)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.assignments = &pb.AssignmentResponse{ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
		}}}
	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.ValidatorIndexRequest{}),
	).Return(&pb.ValidatorIndexResponse{
		Index: uint64(validatorIndex),
	}, nil)
	m.attesterClient.EXPECT().RequestAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.AttestationRequest{}),
	).Return(&ethpb.AttestationData{
		BeaconBlockRoot: []byte("A"),
		Target:          &ethpb.Checkpoint{Root: []byte("B")},
		Source:          &ethpb.Checkpoint{Root: []byte("C"), Epoch: 3},
	}, nil)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&pb.DomainResponse{}, nil /*err*/)

	var generatedAttestation *ethpb.Attestation
	m.attesterClient.EXPECT().SubmitAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.Attestation{}),
	).Do(func(_ context.Context, att *ethpb.Attestation) {
		generatedAttestation = att
	}).Return(&pb.AttestResponse{}, nil /* error */)

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)

	aggregationBitfield := bitfield.NewBitlist(uint64(len(committee)))
	aggregationBitfield.SetBitAt(4, true)
	custodyBitfield := bitfield.NewBitlist(uint64(len(committee)))
	expectedAttestation := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: []byte("A"),
			Target:          &ethpb.Checkpoint{Root: []byte("B")},
			Source:          &ethpb.Checkpoint{Root: []byte("C"), Epoch: 3},
		},
		AggregationBits: aggregationBitfield,
		CustodyBits:     custodyBitfield,
	}

	attDataAndCustodyBit := &pbp2p.AttestationDataAndCustodyBit{
		Data:       expectedAttestation.Data,
		CustodyBit: false,
	}
	root, err := ssz.HashTreeRoot(attDataAndCustodyBit)
	if err != nil {
		t.Fatal(err)
	}

	sig := validator.keys[validatorPubKey].SecretKey.Sign(root[:], 0).Marshal()
	expectedAttestation.Signature = sig
	if !reflect.DeepEqual(generatedAttestation, expectedAttestation) {
		t.Errorf("Incorrectly attested head, wanted %v, received %v", expectedAttestation, generatedAttestation)
	}
	testutil.AssertLogsContain(t, hook, "Submitted new attestation")
}

func TestAttestToBlockHead_DoesNotAttestBeforeDelay(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	validator.genesisTime = uint64(roughtime.Now().Unix())
	m.validatorClient.EXPECT().CommitteeAssignment(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.AssignmentRequest{}),
		gomock.Any(),
	).Times(0)

	m.attesterClient.EXPECT().SubmitAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.AttestationRequest{}),
	).Times(0)

	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.ValidatorIndexRequest{}),
	).Times(0)

	m.attesterClient.EXPECT().SubmitAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.Attestation{}),
	).Return(&pb.AttestResponse{}, nil /* error */).Times(0)

	timer := time.NewTimer(time.Duration(1 * time.Second))
	go validator.SubmitAttestation(context.Background(), 0, validatorPubKey)
	<-timer.C
}

func TestAttestToBlockHead_DoesAttestAfterDelay(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	var wg sync.WaitGroup
	wg.Add(2)
	defer wg.Wait()

	validator.genesisTime = uint64(roughtime.Now().Unix())
	validatorIndex := uint64(5)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.assignments = &pb.AssignmentResponse{ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
		}}}

	m.attesterClient.EXPECT().RequestAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.AttestationRequest{}),
	).Return(&ethpb.AttestationData{
		BeaconBlockRoot: []byte("A"),
		Target:          &ethpb.Checkpoint{Root: []byte("B")},
		Source:          &ethpb.Checkpoint{Root: []byte("C"), Epoch: 3},
	}, nil).Do(func(arg0, arg1 interface{}) {
		wg.Done()
	})

	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.ValidatorIndexRequest{}),
	).Return(&pb.ValidatorIndexResponse{
		Index: uint64(validatorIndex),
	}, nil).Do(func(arg0, arg1 interface{}) {
		wg.Done()
	})

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&pb.DomainResponse{}, nil /*err*/)

	m.attesterClient.EXPECT().SubmitAttestation(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&pb.AttestResponse{}, nil).Times(1)

	validator.SubmitAttestation(context.Background(), 0, validatorPubKey)
}

func TestAttestToBlockHead_CorrectBitfieldLength(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()
	validatorIndex := uint64(2)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.assignments = &pb.AssignmentResponse{ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
		}}}
	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.ValidatorIndexRequest{}),
	).Return(&pb.ValidatorIndexResponse{
		Index: uint64(validatorIndex),
	}, nil)
	m.attesterClient.EXPECT().RequestAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.AttestationRequest{}),
	).Return(&ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Root: []byte("B")},
		Source: &ethpb.Checkpoint{Root: []byte("C"), Epoch: 3},
	}, nil)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&pb.DomainResponse{}, nil /*err*/)

	var generatedAttestation *ethpb.Attestation
	m.attesterClient.EXPECT().SubmitAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.Attestation{}),
	).Do(func(_ context.Context, att *ethpb.Attestation) {
		generatedAttestation = att
	}).Return(&pb.AttestResponse{}, nil /* error */)

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)

	if len(generatedAttestation.AggregationBits) != 2 {
		t.Errorf("Wanted length %d, received %d", 2, len(generatedAttestation.AggregationBits))
	}
}

func TestWaitForSlotOneThird_WaitCorrectly(t *testing.T) {
	validator, _, finish := setup(t)
	defer finish()
	currentTime := uint64(time.Now().Unix())
	numOfSlots := uint64(4)
	validator.genesisTime = currentTime - (numOfSlots * params.BeaconConfig().SecondsPerSlot)
	timeToSleep := params.BeaconConfig().SecondsPerSlot / 3
	oneThird := currentTime + timeToSleep
	validator.waitToOneThird(context.Background(), numOfSlots)

	currentTime = uint64(time.Now().Unix())
	if currentTime != oneThird {
		t.Errorf("Wanted %d time for slot one-third but got %d", oneThird, currentTime)
	}
}
