package polling

import (
	"context"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	mockSlasher "github.com/prysmaticlabs/prysm/validator/testing"
)

func TestPreBlockSignValidation(t *testing.T) {
	config := &featureconfig.Flags{
		ProtectAttester:   false,
		SlasherProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, _, finish := setup(t)
	defer finish()

	block := &ethpb.BeaconBlock{
		Slot:          10,
		ProposerIndex: 0,
	}
	mockProtector := &mockSlasher.MockProtector{AllowBlock: false}
	validator.protector = mockProtector
	err := validator.preBlockSignValidations(context.Background(), validatorPubKey, block)
	if err == nil || !strings.Contains(err.Error(), failedPreBlockSignExternalErr) {
		t.Fatal(err)
	}
	mockProtector.AllowBlock = true
	err = validator.preBlockSignValidations(context.Background(), validatorPubKey, block)
	if err != nil {
		t.Fatalf("Expected allowed attestation not to throw error. got: %v", err)
	}
}

func TestPostBlockSignUpdate(t *testing.T) {
	config := &featureconfig.Flags{
		ProtectAttester:   false,
		SlasherProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, _, finish := setup(t)
	defer finish()

	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:          10,
			ProposerIndex: 0,
		},
	}
	mockProtector := &mockSlasher.MockProtector{AllowBlock: false}
	validator.protector = mockProtector
	err := validator.postBlockSignUpdate(context.Background(), validatorPubKey, block)
	if err == nil || !strings.Contains(err.Error(), failedPostBlockSignErr) {
		t.Fatalf("Expected error to be thrown when post signature update is detected as slashable. got: %v", err)
	}
	mockProtector.AllowBlock = true
	err = validator.postBlockSignUpdate(context.Background(), validatorPubKey, block)
	if err != nil {
		t.Fatalf("Expected allowed attestation not to throw error. got: %v", err)
	}
}
