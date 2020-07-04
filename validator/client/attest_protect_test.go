package client

import (
	"context"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	mockSlasher "github.com/prysmaticlabs/prysm/validator/testing"
)

func TestPreSignatureValidation(t *testing.T) {
	config := &featureconfig.Flags{
		LocalProtection:   false,
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
	err := validator.preAttSignValidations(context.Background(), att, validatorPubKey)
	if err == nil || !strings.Contains(err.Error(), failedPreAttSignExternalErr) {
		t.Fatal(err)
	}
	mockProtector.AllowAttestation = true
	err = validator.preAttSignValidations(context.Background(), att, validatorPubKey)
	if err != nil {
		t.Fatalf("Expected allowed attestation not to throw error. got: %v", err)
	}
}

func TestPostSignatureUpdate(t *testing.T) {
	config := &featureconfig.Flags{
		LocalProtection:   false,
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
	err := validator.postAttSignUpdate(context.Background(), att, validatorPubKey)
	if err == nil || !strings.Contains(err.Error(), failedPostAttSignExternalErr) {
		t.Fatalf("Expected error to be thrown when post signature update is detected as slashable. got: %v", err)
	}
	mockProtector.AllowAttestation = true
	err = validator.postAttSignUpdate(context.Background(), att, validatorPubKey)
	if err != nil {
		t.Fatalf("Expected allowed attestation not to throw error. got: %v", err)
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
