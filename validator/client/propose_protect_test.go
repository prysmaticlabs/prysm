package client

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mockSlasher "github.com/prysmaticlabs/prysm/validator/testing"
)

func TestPreBlockSignValidation(t *testing.T) {
	config := &featureconfig.Flags{
		LocalProtection:   false,
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
	require.ErrorContains(t, failedPreBlockSignExternalErr, err)
	mockProtector.AllowBlock = true
	err = validator.preBlockSignValidations(context.Background(), validatorPubKey, block)
	require.NoError(t, err, "Expected allowed attestation not to throw error")
}

func TestPostBlockSignUpdate(t *testing.T) {
	config := &featureconfig.Flags{
		LocalProtection:   false,
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
	require.ErrorContains(t, failedPostBlockSignErr, err, "Expected error when post signature update is detected as slashable")
	mockProtector.AllowBlock = true
	err = validator.postBlockSignUpdate(context.Background(), validatorPubKey, block)
	require.NoError(t, err, "Expected allowed attestation not to throw error")
}
