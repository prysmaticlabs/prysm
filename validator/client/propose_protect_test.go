package client

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mockSlasher "github.com/prysmaticlabs/prysm/validator/testing"
)

func TestPreBlockSignLocalValidation(t *testing.T) {
	ctx := context.Background()
	config := &featureconfig.Flags{
		LocalProtection:   true,
		SlasherProtection: false,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, _, validatorKey, finish := setup(t)
	defer finish()

	block := &ethpb.BeaconBlock{
		Slot:          10,
		ProposerIndex: 0,
	}
	err := validator.db.SaveProposalHistoryForSlot(ctx, validatorKey.PublicKey().Marshal(), 10, []byte{1})
	require.NoError(t, err)
	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	err = validator.preBlockSignValidations(context.Background(), pubKey, block)
	require.ErrorContains(t, failedPreBlockSignLocalErr, err)
	block.Slot = 9
	err = validator.preBlockSignValidations(context.Background(), pubKey, block)
	require.NoError(t, err, "Expected allowed attestation not to throw error")
}

func TestPreBlockSignValidation(t *testing.T) {
	config := &featureconfig.Flags{
		LocalProtection:   false,
		SlasherProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, _, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	block := &ethpb.BeaconBlock{
		Slot:          10,
		ProposerIndex: 0,
	}
	mockProtector := &mockSlasher.MockProtector{AllowBlock: false}
	validator.protector = mockProtector
	err := validator.preBlockSignValidations(context.Background(), pubKey, block)
	require.ErrorContains(t, failedPreBlockSignExternalErr, err)
	mockProtector.AllowBlock = true
	err = validator.preBlockSignValidations(context.Background(), pubKey, block)
	require.NoError(t, err, "Expected allowed attestation not to throw error")
}

func TestPostBlockSignUpdate(t *testing.T) {
	config := &featureconfig.Flags{
		LocalProtection:   false,
		SlasherProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, _, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:          10,
			ProposerIndex: 0,
		},
	}
	mockProtector := &mockSlasher.MockProtector{AllowBlock: false}
	validator.protector = mockProtector
	err := validator.postBlockSignUpdate(context.Background(), pubKey, block, nil)
	require.ErrorContains(t, failedPostBlockSignErr, err, "Expected error when post signature update is detected as slashable")
	mockProtector.AllowBlock = true
	err = validator.postBlockSignUpdate(context.Background(), pubKey, block, nil)
	require.NoError(t, err, "Expected allowed attestation not to throw error")
}
