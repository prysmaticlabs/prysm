package client

import (
	"context"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	bitfield "github.com/prysmaticlabs/go-bitfield"
	ssz "github.com/prysmaticlabs/go-ssz"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestRequestAttestation_ValidatorIndexRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	validator.assignments = &pb.AssignmentResponse{ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{}}
	defer finish()
	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.ValidatorIndexRequest{}),
	).Return(nil /* Validator Index Response*/, errors.New("something bad happened"))

	validator.AttestToBlockHead(context.Background(), 30, hex.EncodeToString(validatorKey.PublicKey.Marshal()))
	testutil.AssertLogsContain(t, hook, "Could not fetch validator index")
}

func TestAttestToBlockHead_RequestAttestationFailure(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	validator.assignments = &pb.AssignmentResponse{ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{
		{
			PublicKey: validatorKey.PublicKey.Marshal(),
			Shard:     5,
		},
	}}
	m.validatorClient.EXPECT().ValidatorIndex(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.ValidatorIndexRequest{}),
	).Return(&pb.ValidatorIndexResponse{Index: 5}, nil)
	m.attesterClient.EXPECT().RequestAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.AttestationRequest{}),
	).Return(nil, errors.New("something went wrong"))

	validator.AttestToBlockHead(context.Background(), 30, hex.EncodeToString(validatorKey.PublicKey.Marshal()))
	testutil.AssertLogsContain(t, hook, "Could not request attestation to sign at slot")
}

func TestAttestToBlockHead_SubmitAttestationRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	validator.assignments = &pb.AssignmentResponse{ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{
		{
			PublicKey: validatorKey.PublicKey.Marshal(),
			Shard:     5,
			Committee: make([]uint64, 111),
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
		Crosslink:       &ethpb.Crosslink{},
	}, nil)
	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch2
	).Return(&pb.DomainResponse{}, nil /*err*/)
	m.attesterClient.EXPECT().SubmitAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.Attestation{}),
	).Return(nil, errors.New("something went wrong"))

	validator.AttestToBlockHead(context.Background(), 30, hex.EncodeToString(validatorKey.PublicKey.Marshal()))
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
			PublicKey: validatorKey.PublicKey.Marshal(),
			Shard:     5,
			Committee: committee,
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
		Crosslink:       &ethpb.Crosslink{Shard: 5, DataRoot: []byte{'D'}},
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

	validator.AttestToBlockHead(context.Background(), 30, hex.EncodeToString(validatorKey.PublicKey.Marshal()))

	aggregationBitfield := bitfield.NewBitlist(uint64(len(committee)))
	aggregationBitfield.SetBitAt(4, true)
	custodyBitfield := bitfield.NewBitlist(uint64(len(committee)))
	expectedAttestation := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: []byte("A"),
			Target:          &ethpb.Checkpoint{Root: []byte("B")},
			Source:          &ethpb.Checkpoint{Root: []byte("C"), Epoch: 3},
			Crosslink:       &ethpb.Crosslink{Shard: 5, DataRoot: []byte{'D'}},
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

	k := hex.EncodeToString(validatorKey.PublicKey.Marshal())
	sig := validator.keys[k].SecretKey.Sign(root[:], 0).Marshal()
	expectedAttestation.Signature = sig

	if !proto.Equal(generatedAttestation, expectedAttestation) {
		t.Errorf("Incorrectly attested head, wanted %v, received %v", expectedAttestation, generatedAttestation)
	}
	testutil.AssertLogsContain(t, hook, "Attested latest head")
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

	delay = 3
	timer := time.NewTimer(time.Duration(1 * time.Second))
	go validator.AttestToBlockHead(context.Background(), 0, hex.EncodeToString(validatorKey.PublicKey.Marshal()))
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
			PublicKey: validatorKey.PublicKey.Marshal(),
			Shard:     5,
			Committee: committee,
		}}}

	m.attesterClient.EXPECT().RequestAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.AttestationRequest{}),
	).Return(&ethpb.AttestationData{
		BeaconBlockRoot: []byte("A"),
		Target:          &ethpb.Checkpoint{Root: []byte("B")},
		Source:          &ethpb.Checkpoint{Root: []byte("C"), Epoch: 3},
		Crosslink:       &ethpb.Crosslink{DataRoot: []byte{'D'}},
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

	delay = 0
	validator.AttestToBlockHead(context.Background(), 0, hex.EncodeToString(validatorKey.PublicKey.Marshal()))
}

func TestAttestToBlockHead_CorrectBitfieldLength(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()
	validatorIndex := uint64(2)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.assignments = &pb.AssignmentResponse{ValidatorAssignment: []*pb.AssignmentResponse_ValidatorAssignment{
		{
			PublicKey: validatorKey.PublicKey.Marshal(),
			Shard:     5,
			Committee: committee,
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
		Target:    &ethpb.Checkpoint{Root: []byte("B")},
		Source:    &ethpb.Checkpoint{Root: []byte("C"), Epoch: 3},
		Crosslink: &ethpb.Crosslink{DataRoot: []byte{'D'}},
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

	validator.AttestToBlockHead(context.Background(), 30, hex.EncodeToString(validatorKey.PublicKey.Marshal()))

	if len(generatedAttestation.AggregationBits) != 2 {
		t.Errorf("Wanted length %d, received %d", 2, len(generatedAttestation.AggregationBits))
	}
}
