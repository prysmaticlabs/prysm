package streaming

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	mockSlasher "github.com/prysmaticlabs/prysm/validator/testing"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestRequestAttestation_ValidatorDutiesRequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, _, finish := setup(t)
	validator.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty)
	validator.dutiesByEpoch[0] = []*ethpb.DutiesResponse_Duty{}
	defer finish()

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Could not fetch validator assignment")
}

func TestAttestToBlockHead_SubmitAttestation_EmptyCommittee(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, _, finish := setup(t)
	defer finish()
	validator.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty)
	validator.dutiesByEpoch[0] = []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 0,
			Committee:      make([]uint64, 0),
			ValidatorIndex: 0,
		},
	}
	validator.SubmitAttestation(context.Background(), 0, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Empty committee")
}

func TestAttestToBlockHead_SubmitAttestation_RequestFailure(t *testing.T) {
	hook := logTest.NewGlobal()

	validator, m, finish := setup(t)
	defer finish()
	validator.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty)
	validator.dutiesByEpoch[0] = []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      make([]uint64, 111),
			ValidatorIndex: 0,
		},
	}
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
		ProtectAttester: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, m, finish := setup(t)
	defer finish()
	hook := logTest.NewGlobal()
	validatorIndex := uint64(7)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty)
	validator.dutiesByEpoch[0] = []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		},
	}

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
	if err != nil {
		t.Fatal(err)
	}

	sig, err := validator.keyManager.Sign(validatorPubKey, root)
	if err != nil {
		t.Fatal(err)
	}
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
		ProtectAttester: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()
	validatorIndex := uint64(7)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty)
	validator.dutiesByEpoch[0] = []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		},
	}
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
	testutil.AssertLogsContain(t, hook, "Attempted to make a slashable attestation, rejected")
}

func TestPostSignatureUpdate(t *testing.T) {
	config := &featureconfig.Flags{
		ProtectAttester:   false,
		SlasherProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, _, finish := setup(t)
	defer finish()
	att := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &ethpb.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: []byte("great block"),
			Source: &ethpb.Checkpoint{
				Epoch: 4,
				Root:  []byte("good source"),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 10,
				Root:  []byte("good target"),
			},
		},
	}
	mockProtector := &mockSlasher.MockProtector{AllowAttestation: false}
	validator.protector = mockProtector
	err := validator.postSignatureUpdate(context.Background(), att, validatorPubKey)
	if err == nil || !strings.Contains(err.Error(), "made a slashable attestation,") {
		t.Fatalf("Expected error to be thrown when post signature update is detected as slashable. got: %v", err)
	}
	mockProtector.AllowAttestation = true
	err = validator.postSignatureUpdate(context.Background(), att, validatorPubKey)
	if err != nil {
		t.Fatalf("Expected allowed attestation not to throw error. got: %v", err)
	}
}

func TestPreSignatureValidation(t *testing.T) {
	config := &featureconfig.Flags{
		ProtectAttester:   false,
		SlasherProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, _, finish := setup(t)
	defer finish()
	hook := logTest.NewGlobal()
	att := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &ethpb.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: []byte("great block"),
			Source: &ethpb.Checkpoint{
				Epoch: 4,
				Root:  []byte("good source"),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 10,
				Root:  []byte("good target"),
			},
		},
	}
	mockProtector := &mockSlasher.MockProtector{AllowAttestation: false}
	validator.protector = mockProtector
	err := validator.preSigningValidations(context.Background(), att, validatorPubKey)
	if err == nil || !strings.Contains(err.Error(), "rejected by external slasher service") {
		t.Fatal(err)
	}
	testutil.AssertLogsContain(t, hook, "Attempted to make a slashable attestation, rejected by external slasher service")
	mockProtector.AllowAttestation = true
	err = validator.preSigningValidations(context.Background(), att, validatorPubKey)
	if err != nil {
		t.Fatalf("Expected allowed attestation not to throw error. got: %v", err)
	}
}

func TestAttestToBlockHead_BlocksSurroundAtt(t *testing.T) {
	config := &featureconfig.Flags{
		ProtectAttester: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()
	validatorIndex := uint64(7)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty)
	validator.dutiesByEpoch[0] = []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		},
	}
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
	testutil.AssertLogsContain(t, hook, "Attempted to make a slashable attestation, rejected")
}

func TestAttestToBlockHead_BlocksSurroundedAtt(t *testing.T) {
	config := &featureconfig.Flags{
		ProtectAttester: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()
	validatorIndex := uint64(7)
	committee := []uint64{0, 3, 4, 2, validatorIndex, 6, 8, 9, 10}
	validator.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty)
	validator.dutiesByEpoch[0] = []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		},
	}
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

	m.validatorClient.EXPECT().GetAttestationData(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.AttestationDataRequest{}),
	).Return(&ethpb.AttestationData{
		BeaconBlockRoot: []byte("A"),
		Target:          &ethpb.Checkpoint{Root: []byte("B"), Epoch: 2},
		Source:          &ethpb.Checkpoint{Root: []byte("C"), Epoch: 1},
	}, nil)

	validator.SubmitAttestation(context.Background(), 30, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Attempted to make a slashable attestation, rejected")
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
	validator.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty)
	validator.dutiesByEpoch[0] = []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		},
	}

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
	validator.dutiesByEpoch = make(map[uint64][]*ethpb.DutiesResponse_Duty)
	validator.dutiesByEpoch[0] = []*ethpb.DutiesResponse_Duty{
		{
			PublicKey:      validatorKey.PublicKey.Marshal(),
			CommitteeIndex: 5,
			Committee:      committee,
			ValidatorIndex: validatorIndex,
		},
	}
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

	if len(generatedAttestation.AggregationBits) != 2 {
		t.Errorf("Wanted length %d, received %d", 2, len(generatedAttestation.AggregationBits))
	}
}

func TestAttestationHistory_BlocksDoubleAttestation(t *testing.T) {
	newMap := make(map[uint64]uint64)
	newMap[0] = params.BeaconConfig().FarFutureEpoch
	attestations := &slashpb.AttestationHistory{
		TargetToSource:     newMap,
		LatestEpochWritten: 0,
	}

	// Mark an attestation spanning epochs 0 to 3.
	newAttSource := uint64(0)
	newAttTarget := uint64(3)
	attestations = markAttestationForTargetEpoch(attestations, newAttSource, newAttTarget)
	if attestations.LatestEpochWritten != newAttTarget {
		t.Fatalf("Expected latest epoch written to be %d, received %d", newAttTarget, attestations.LatestEpochWritten)
	}

	// Try an attestation that should be slashable (double att) spanning epochs 1 to 3.
	newAttSource = uint64(1)
	newAttTarget = uint64(3)
	if !isNewAttSlashable(attestations, newAttSource, newAttTarget) {
		t.Fatalf("Expected attestation of source %d and target %d to be considered slashable", newAttSource, newAttTarget)
	}
}

func TestAttestationHistory_Prunes(t *testing.T) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	newMap := make(map[uint64]uint64)
	newMap[0] = params.BeaconConfig().FarFutureEpoch
	attestations := &slashpb.AttestationHistory{
		TargetToSource:     newMap,
		LatestEpochWritten: 0,
	}

	// Try an attestation on totally unmarked history, should not be slashable.
	if isNewAttSlashable(attestations, 0, wsPeriod+5) {
		t.Fatalf("Expected attestation of source 0, target %d to be considered slashable", wsPeriod+5)
	}

	// Mark attestations spanning epochs 0 to 3 and 6 to 9.
	prunedNewAttSource := uint64(0)
	prunedNewAttTarget := uint64(3)
	attestations = markAttestationForTargetEpoch(attestations, prunedNewAttSource, prunedNewAttTarget)
	newAttSource := prunedNewAttSource + 6
	newAttTarget := prunedNewAttTarget + 6
	attestations = markAttestationForTargetEpoch(attestations, newAttSource, newAttTarget)
	if attestations.LatestEpochWritten != newAttTarget {
		t.Fatalf("Expected latest epoch written to be %d, received %d", newAttTarget, attestations.LatestEpochWritten)
	}

	// Mark an attestation spanning epochs 54000 to 54003.
	farNewAttSource := newAttSource + wsPeriod
	farNewAttTarget := newAttTarget + wsPeriod
	attestations = markAttestationForTargetEpoch(attestations, farNewAttSource, farNewAttTarget)
	if attestations.LatestEpochWritten != farNewAttTarget {
		t.Fatalf("Expected latest epoch written to be %d, received %d", newAttTarget, attestations.LatestEpochWritten)
	}

	if safeTargetToSource(attestations, prunedNewAttTarget) != params.BeaconConfig().FarFutureEpoch {
		t.Fatalf("Expected attestation at target epoch %d to not be marked", prunedNewAttTarget)
	}

	if safeTargetToSource(attestations, farNewAttTarget) != farNewAttSource {
		t.Fatalf("Expected attestation at target epoch %d to not be marked", farNewAttSource)
	}

	// Try an attestation from existing source to outside prune, should slash.
	if !isNewAttSlashable(attestations, newAttSource, farNewAttTarget) {
		t.Fatalf("Expected attestation of source %d, target %d to be considered slashable", newAttSource, farNewAttTarget)
	}
	// Try an attestation from before existing target to outside prune, should slash.
	if !isNewAttSlashable(attestations, newAttTarget-1, farNewAttTarget) {
		t.Fatalf("Expected attestation of source %d, target %d to be considered slashable", newAttTarget-1, farNewAttTarget)
	}
	// Try an attestation larger than pruning amount, should slash.
	if !isNewAttSlashable(attestations, 0, farNewAttTarget+5) {
		t.Fatalf("Expected attestation of source 0, target %d to be considered slashable", farNewAttTarget+5)
	}
}

func TestAttestationHistory_BlocksSurroundedAttestation(t *testing.T) {
	newMap := make(map[uint64]uint64)
	newMap[0] = params.BeaconConfig().FarFutureEpoch
	attestations := &slashpb.AttestationHistory{
		TargetToSource:     newMap,
		LatestEpochWritten: 0,
	}

	// Mark an attestation spanning epochs 0 to 3.
	newAttSource := uint64(0)
	newAttTarget := uint64(3)
	attestations = markAttestationForTargetEpoch(attestations, newAttSource, newAttTarget)
	if attestations.LatestEpochWritten != newAttTarget {
		t.Fatalf("Expected latest epoch written to be %d, received %d", newAttTarget, attestations.LatestEpochWritten)
	}

	// Try an attestation that should be slashable (being surrounded) spanning epochs 1 to 2.
	newAttSource = uint64(1)
	newAttTarget = uint64(2)
	if !isNewAttSlashable(attestations, newAttSource, newAttTarget) {
		t.Fatalf("Expected attestation of source %d and target %d to be considered slashable", newAttSource, newAttTarget)
	}
}

func TestAttestationHistory_BlocksSurroundingAttestation(t *testing.T) {
	newMap := make(map[uint64]uint64)
	newMap[0] = params.BeaconConfig().FarFutureEpoch
	attestations := &slashpb.AttestationHistory{
		TargetToSource:     newMap,
		LatestEpochWritten: 0,
	}

	// Mark an attestation spanning epochs 1 to 2.
	newAttSource := uint64(1)
	newAttTarget := uint64(2)
	attestations = markAttestationForTargetEpoch(attestations, newAttSource, newAttTarget)
	if attestations.LatestEpochWritten != newAttTarget {
		t.Fatalf("Expected latest epoch written to be %d, received %d", newAttTarget, attestations.LatestEpochWritten)
	}
	if attestations.TargetToSource[newAttTarget] != newAttSource {
		t.Fatalf("Expected source epoch to be %d, received %d", newAttSource, attestations.TargetToSource[newAttTarget])
	}

	// Try an attestation that should be slashable (surrounding) spanning epochs 0 to 3.
	newAttSource = uint64(0)
	newAttTarget = uint64(3)
	if !isNewAttSlashable(attestations, newAttSource, newAttTarget) {
		t.Fatalf("Expected attestation of source %d and target %d to be considered slashable", newAttSource, newAttTarget)
	}
}
