package client

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mockSlasher "github.com/prysmaticlabs/prysm/validator/testing"
)

func TestPreBlockSignLocalValidation_PreventsLowerThanMinProposal(t *testing.T) {
	ctx := context.Background()
	validator, _, validatorKey, finish := setup(t)
	defer finish()
	lowestSignedSlot := uint64(10)
	pubKeyBytes := [48]byte{}
	copy(pubKeyBytes[:], validatorKey.PublicKey().Marshal())

	// We save a proposal at the lowest signed slot in the DB.
	err := validator.db.SaveProposalHistoryForSlot(ctx, pubKeyBytes, lowestSignedSlot, []byte{1})
	require.NoError(t, err)
	require.NoError(t, err)

	// We expect the same block with a slot lower than the lowest
	// signed slot to fail validation.
	block := &ethpb.BeaconBlock{
		Slot:          lowestSignedSlot - 1,
		ProposerIndex: 0,
	}
	err = validator.preBlockSignValidations(context.Background(), pubKeyBytes, block)
	require.ErrorContains(t, "could not sign block with slot <= lowest signed", err)

	// We expect the same block with a slot equal to the lowest
	// signed slot to fail validation.
	block = &ethpb.BeaconBlock{
		Slot:          lowestSignedSlot,
		ProposerIndex: 0,
	}
	err = validator.preBlockSignValidations(context.Background(), pubKeyBytes, block)
	require.ErrorContains(t, "could not sign block with slot <= lowest signed", err)

	// We expect the same block with a slot > than the lowest
	// signed slot to pass validation.
	block = &ethpb.BeaconBlock{
		Slot:          lowestSignedSlot + 1,
		ProposerIndex: 0,
	}
	err = validator.preBlockSignValidations(context.Background(), pubKeyBytes, block)
	require.NoError(t, err)
}

func TestPreBlockSignLocalValidation(t *testing.T) {
	ctx := context.Background()
	config := &featureconfig.Flags{
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
	pubKeyBytes := [48]byte{}
	copy(pubKeyBytes[:], validatorKey.PublicKey().Marshal())

	// We save a proposal at slot 10 with a dummy signing root.
	err := validator.db.SaveProposalHistoryForSlot(ctx, pubKeyBytes, 10, []byte{1})
	require.NoError(t, err)
	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	// We expect the same block sent out should return slashable error.
	err = validator.preBlockSignValidations(context.Background(), pubKey, block)
	require.ErrorContains(t, failedPreBlockSignLocalErr, err)

	// We save a proposal at slot 11 with a nil signing root.
	block.Slot = 11
	err = validator.db.SaveProposalHistoryForSlot(ctx, pubKeyBytes, block.Slot, nil)
	require.NoError(t, err)

	// We expect the same block sent out should return slashable error even
	// if we had a nil signing root stored in the database.
	err = validator.preBlockSignValidations(context.Background(), pubKey, block)
	require.ErrorContains(t, failedPreBlockSignLocalErr, err)

	// A block with a different slot for which we do not have a proposing history
	// should not be failing validation.
	block.Slot = 9
	err = validator.preBlockSignValidations(context.Background(), pubKey, block)
	require.NoError(t, err, "Expected allowed block not to throw error")
}

func TestPreBlockSignValidation(t *testing.T) {
	config := &featureconfig.Flags{
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
		SlasherProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, _, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	emptyBlock := testutil.NewBeaconBlock()
	emptyBlock.Block.Slot = 10
	emptyBlock.Block.ProposerIndex = 0
	mockProtector := &mockSlasher.MockProtector{AllowBlock: false}
	validator.protector = mockProtector
	err := validator.postBlockSignUpdate(context.Background(), pubKey, emptyBlock, nil)
	require.ErrorContains(t, failedPostBlockSignErr, err, "Expected error when post signature update is detected as slashable")
	mockProtector.AllowBlock = true
	err = validator.postBlockSignUpdate(context.Background(), pubKey, emptyBlock, &ethpb.DomainResponse{SignatureDomain: make([]byte, 32)})
	require.NoError(t, err, "Expected allowed attestation not to throw error")
}
