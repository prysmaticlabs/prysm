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
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestRequestAttestation_ValidatorDutiesRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, _, finish := setup(t)
	validator.duties = &ethpb.DutiesResponse{Duties: []*ethpb.DutiesResponse_Duty{}}
	defer finish()

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Could not fetch validator assignment")
}

func TestAttestToBlockHead_SubmitAttestation_EmptyCommittee(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, _, finish := setup(t)
	defer finish()
	validator.duties = &ethpb.DutiesResponse{Duties: []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 0,
			Committee:      make([]uint64, 0),
			ValidatorIndex: 0,
		}}}
	validator.SubmitAttestation(context.Background(), 0, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Empty committee")
}

func TestAttestToBlockHead_SubmitAttestation_RequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	validator.duties = &ethpb.DutiesResponse{Duties: []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      make([]uint64, 111),
			ValidatorIndex: 0,
		}}}
	m.validatorClient.EXPECT().GetAttestationData(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AttestationDataRequest{}),
	).Return(&ethpb.AttestationData{
		BeaconBlockRoot: []byte{},
		Target:          &ethpb.Checkpoint{},
		Source:          &ethpb.Checkpoint{},
	}, nil)
	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch2
	).Return(&ethpb.DomainResponse{}, nil /*err*/)
	m.validatorClient.EXPECT().ProposeAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.Attestation{}),
	).Return(nil, errors.New("something went wrong"))

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Could not submit attestation to beacon node")
}

func TestAttestToBlockHead_AttestsCorrectly(t *testing.T) {
	config := &featureconfig.Flags{
		LocalProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, m, finish := setup(t)
	defer finish()
	hook := logTest.NewGlobal()
	validatorIndex := uint64(7)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.duties = &ethpb.DutiesResponse{Duties: []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		},
	}}

	beaconBlockRoot := bytesutil.ToBytes32([]byte("A"))
	targetRoot := bytesutil.ToBytes32([]byte("B"))
	sourceRoot := bytesutil.ToBytes32([]byte("C"))
	m.validatorClient.EXPECT().GetAttestationData(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AttestationDataRequest{}),
	).Return(&ethpb.AttestationData{
		BeaconBlockRoot: beaconBlockRoot[:],
		Target:          &ethpb.Checkpoint{Root: targetRoot[:]},
		Source:          &ethpb.Checkpoint{Root: sourceRoot[:], Epoch: 3},
	}, nil)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: []byte{}}, nil /*err*/)

	var generatedAttestation *ethpb.Attestation
	m.validatorClient.EXPECT().ProposeAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.Attestation{}),
	).Do(func(_ context.Context, att *ethpb.Attestation) {
		generatedAttestation = att
	}).Return(&ethpb.AttestResponse{}, nil /* error */)

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)

	aggregationBitfield := bitfield.NewBitlist(uint64(len(committee)))
	aggregationBitfield.SetBitAt(4, true)
	expectedAttestation := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: beaconBlockRoot[:],
			Target:          &ethpb.Checkpoint{Root: targetRoot[:]},
			Source:          &ethpb.Checkpoint{Root: sourceRoot[:], Epoch: 3},
		},
		AggregationBits: aggregationBitfield,
	}

	root, err := helpers.ComputeSigningRoot(expectedAttestation.Data, []byte{})
	require.NoError(t, err)

	sig, err := validator.keyManager.Sign(validatorPubKey, root)
	require.NoError(t, err)
	expectedAttestation.Signature = sig.Marshal()
	if !reflect.DeepEqual(generatedAttestation, expectedAttestation) {
		t.Errorf("Incorrectly attested head, wanted %v, received %v", expectedAttestation, generatedAttestation)
		diff, _ := messagediff.PrettyDiff(expectedAttestation, generatedAttestation)
		t.Log(diff)
	}
	testutil.AssertLogsDoNotContain(t, hook, "Could not")
}

func TestAttestToBlockHead_BlocksDoubleAtt(t *testing.T) {
	config := &featureconfig.Flags{
		LocalProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()
	validatorIndex := uint64(7)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.duties = &ethpb.DutiesResponse{Duties: []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		},
	}}
	beaconBlockRoot := bytesutil.ToBytes32([]byte("A"))
	targetRoot := bytesutil.ToBytes32([]byte("B"))
	sourceRoot := bytesutil.ToBytes32([]byte("C"))

	m.validatorClient.EXPECT().GetAttestationData(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AttestationDataRequest{}),
	).Times(2).Return(&ethpb.AttestationData{
		BeaconBlockRoot: beaconBlockRoot[:],
		Target:          &ethpb.Checkpoint{Root: targetRoot[:], Epoch: 4},
		Source:          &ethpb.Checkpoint{Root: sourceRoot[:], Epoch: 3},
	}, nil)
	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.Attestation{}),
	).Return(&ethpb.AttestResponse{}, nil /* error */)

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)
	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)
	testutil.AssertLogsContain(t, hook, failedPreAttSignLocalErr)
}

func TestAttestToBlockHead_BlocksSurroundAtt(t *testing.T) {
	config := &featureconfig.Flags{
		LocalProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()
	validatorIndex := uint64(7)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.duties = &ethpb.DutiesResponse{Duties: []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		},
	}}
	beaconBlockRoot := bytesutil.ToBytes32([]byte("A"))
	targetRoot := bytesutil.ToBytes32([]byte("B"))
	sourceRoot := bytesutil.ToBytes32([]byte("C"))

	m.validatorClient.EXPECT().GetAttestationData(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AttestationDataRequest{}),
	).Times(2).Return(&ethpb.AttestationData{
		BeaconBlockRoot: beaconBlockRoot[:],
		Target:          &ethpb.Checkpoint{Root: targetRoot[:], Epoch: 2},
		Source:          &ethpb.Checkpoint{Root: sourceRoot[:], Epoch: 1},
	}, nil)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.Attestation{}),
	).Return(&ethpb.AttestResponse{}, nil /* error */)

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)
	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)
	testutil.AssertLogsContain(t, hook, failedPreAttSignLocalErr)
}

func TestAttestToBlockHead_BlocksSurroundedAtt(t *testing.T) {
	config := &featureconfig.Flags{
		LocalProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()
	validatorIndex := uint64(7)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.duties = &ethpb.DutiesResponse{Duties: []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		},
	}}
	beaconBlockRoot := bytesutil.ToBytes32([]byte("A"))
	targetRoot := bytesutil.ToBytes32([]byte("B"))
	sourceRoot := bytesutil.ToBytes32([]byte("C"))

	m.validatorClient.EXPECT().GetAttestationData(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AttestationDataRequest{}),
	).Return(&ethpb.AttestationData{
		BeaconBlockRoot: beaconBlockRoot[:],
		Target:          &ethpb.Checkpoint{Root: targetRoot[:], Epoch: 3},
		Source:          &ethpb.Checkpoint{Root: sourceRoot[:], Epoch: 0},
	}, nil)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.Attestation{}),
	).Return(&ethpb.AttestResponse{}, nil /* error */)

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)
	testutil.AssertLogsDoNotContain(t, hook, failedPreAttSignLocalErr)

	m.validatorClient.EXPECT().GetAttestationData(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AttestationDataRequest{}),
	).Return(&ethpb.AttestationData{
		BeaconBlockRoot: []byte("A"),
		Target:          &ethpb.Checkpoint{Root: []byte("B"), Epoch: 2},
		Source:          &ethpb.Checkpoint{Root: []byte("C"), Epoch: 1},
	}, nil)

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)
	testutil.AssertLogsContain(t, hook, failedPreAttSignLocalErr)
}

func TestAttestToBlockHead_DoesNotAttestBeforeDelay(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	validator.genesisTime = uint64(roughtime.Now().Unix())
	m.validatorClient.EXPECT().GetDuties(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.DutiesRequest{}),
		gomock.Any(),
	).Times(0)

	m.validatorClient.EXPECT().GetAttestationData(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AttestationDataRequest{}),
	).Times(0)

	m.validatorClient.EXPECT().ProposeAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.Attestation{}),
	).Return(&ethpb.AttestResponse{}, nil /* error */).Times(0)

	timer := time.NewTimer(1 * time.Second)
	go validator.SubmitAttestation(context.Background(), 0, validatorPubKey)
	<-timer.C
}

func TestAttestToBlockHead_DoesAttestAfterDelay(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()

	validator.genesisTime = uint64(roughtime.Now().Unix())
	validatorIndex := uint64(5)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.duties = &ethpb.DutiesResponse{Duties: []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		}}}

	m.validatorClient.EXPECT().GetAttestationData(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AttestationDataRequest{}),
	).Return(&ethpb.AttestationData{
		BeaconBlockRoot: []byte("A"),
		Target:          &ethpb.Checkpoint{Root: []byte("B")},
		Source:          &ethpb.Checkpoint{Root: []byte("C"), Epoch: 3},
	}, nil).Do(func(arg0, arg1 interface{}) {
		wg.Done()
	})

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeAttestation(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.AttestResponse{}, nil).Times(1)

	validator.SubmitAttestation(context.Background(), 0, validatorPubKey)
}

func TestAttestToBlockHead_CorrectBitfieldLength(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()
	validatorIndex := uint64(2)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.duties = &ethpb.DutiesResponse{Duties: []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		}}}
	m.validatorClient.EXPECT().GetAttestationData(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AttestationDataRequest{}),
	).Return(&ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Root: []byte("B")},
		Source: &ethpb.Checkpoint{Root: []byte("C"), Epoch: 3},
	}, nil)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	var generatedAttestation *ethpb.Attestation
	m.validatorClient.EXPECT().ProposeAttestation(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.Attestation{}),
	).Do(func(_ context.Context, att *ethpb.Attestation) {
		generatedAttestation = att
	}).Return(&ethpb.AttestResponse{}, nil /* error */)

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)

	assert.Equal(t, 2, len(generatedAttestation.AggregationBits))
}
