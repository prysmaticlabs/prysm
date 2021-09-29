package client

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	mockSlasher "github.com/prysmaticlabs/prysm/validator/testing"
)

func Test_slashableProposalCheck_PreventsLowerThanMinProposal(t *testing.T) {
	ctx := context.Background()
	validator, _, validatorKey, finish := setup(t)
	defer finish()
	lowestSignedSlot := types.Slot(10)
	pubKeyBytes := [48]byte{}
	copy(pubKeyBytes[:], validatorKey.PublicKey().Marshal())

	// We save a proposal at the lowest signed slot in the DB.
	err := validator.db.SaveProposalHistoryForSlot(ctx, pubKeyBytes, lowestSignedSlot, []byte{1})
	require.NoError(t, err)
	require.NoError(t, err)

	// We expect the same block with a slot lower than the lowest
	// signed slot to fail validation.
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:          lowestSignedSlot - 1,
			ProposerIndex: 0,
		},
		Signature: params.BeaconConfig().EmptySignature[:],
	}
	err = validator.slashableProposalCheck(context.Background(), pubKeyBytes, wrapper.WrappedPhase0SignedBeaconBlock(block), [32]byte{4})
	require.ErrorContains(t, "could not sign block with slot <= lowest signed", err)

	// We expect the same block with a slot equal to the lowest
	// signed slot to pass validation if signing roots are equal.
	block = &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:          lowestSignedSlot,
			ProposerIndex: 0,
		},
		Signature: params.BeaconConfig().EmptySignature[:],
	}
	err = validator.slashableProposalCheck(context.Background(), pubKeyBytes, wrapper.WrappedPhase0SignedBeaconBlock(block), [32]byte{1})
	require.NoError(t, err)

	// We expect the same block with a slot equal to the lowest
	// signed slot to fail validation if signing roots are different.
	err = validator.slashableProposalCheck(context.Background(), pubKeyBytes, wrapper.WrappedPhase0SignedBeaconBlock(block), [32]byte{4})
	require.ErrorContains(t, failedBlockSignLocalErr, err)

	// We expect the same block with a slot > than the lowest
	// signed slot to pass validation.
	block = &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:          lowestSignedSlot + 1,
			ProposerIndex: 0,
		},
		Signature: params.BeaconConfig().EmptySignature[:],
	}
	err = validator.slashableProposalCheck(context.Background(), pubKeyBytes, wrapper.WrappedPhase0SignedBeaconBlock(block), [32]byte{3})
	require.NoError(t, err)
}

func Test_slashableProposalCheck(t *testing.T) {
	ctx := context.Background()
	config := &features.Flags{
		RemoteSlasherProtection: true,
	}
	reset := features.InitWithReset(config)
	defer reset()
	validator, mocks, validatorKey, finish := setup(t)
	defer finish()

	block := util.HydrateSignedBeaconBlock(&ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:          10,
			ProposerIndex: 0,
		},
		Signature: params.BeaconConfig().EmptySignature[:],
	})

	pubKeyBytes := [48]byte{}
	copy(pubKeyBytes[:], validatorKey.PublicKey().Marshal())

	// We save a proposal at slot 1 as our lowest proposal.
	err := validator.db.SaveProposalHistoryForSlot(ctx, pubKeyBytes, 1, []byte{1})
	require.NoError(t, err)

	// We save a proposal at slot 10 with a dummy signing root.
	dummySigningRoot := [32]byte{1}
	err = validator.db.SaveProposalHistoryForSlot(ctx, pubKeyBytes, 10, dummySigningRoot[:])
	require.NoError(t, err)
	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	mock.slasherClient.EXPECT().IsSlashableBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Times(2).Return(&ethpb.ProposerSlashingResponse{}, nil /*err*/)

	// We expect the same block sent out with the same root should not be slasahble.
	err = validator.slashableProposalCheck(context.Background(), pubKey, wrapper.WrappedPhase0SignedBeaconBlock(block), dummySigningRoot)
	require.NoError(t, err)

	// We expect the same block sent out with a different signing root should be slasahble.
	err = validator.slashableProposalCheck(context.Background(), pubKey, wrapper.WrappedPhase0SignedBeaconBlock(block), [32]byte{2})
	require.ErrorContains(t, failedBlockSignLocalErr, err)

	// We save a proposal at slot 11 with a nil signing root.
	block.Block.Slot = 11
	err = validator.db.SaveProposalHistoryForSlot(ctx, pubKeyBytes, block.Block.Slot, nil)
	require.NoError(t, err)

	// We expect the same block sent out should return slashable error even
	// if we had a nil signing root stored in the database.
	err = validator.slashableProposalCheck(context.Background(), pubKey, wrapper.WrappedPhase0SignedBeaconBlock(block), [32]byte{2})
	require.ErrorContains(t, failedBlockSignLocalErr, err)

	// A block with a different slot for which we do not have a proposing history
	// should not be failing validation.
	block.Block.Slot = 9
	err = validator.slashableProposalCheck(context.Background(), pubKey, wrapper.WrappedPhase0SignedBeaconBlock(block), [32]byte{3})
	require.NoError(t, err, "Expected allowed block not to throw error")
}

func Test_slashableProposalCheck_RemoteProtection(t *testing.T) {
	config := &features.Flags{
		RemoteSlasherProtection: true,
	}
	reset := features.InitWithReset(config)
	defer reset()
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())

	block := util.NewBeaconBlock()
	block.Block.Slot = 10

	m.nodeClient.EXPECT().IsSlashableBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.ProposerSlashingResponse{ProposerSlashings: []*ethpb.ProposerSlashing{{}}}, nil /*err*/)

	err := validator.slashableProposalCheck(context.Background(), pubKey, wrapper.WrappedPhase0SignedBeaconBlock(block), [32]byte{2})
	require.ErrorContains(t, failedBlockSignExternalErr, err)

	m.slasherClient.EXPECT().IsSlashableBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.ProposerSlashingResponse{}, nil /*err*/)

	err = validator.slashableProposalCheck(context.Background(), pubKey, wrapper.WrappedPhase0SignedBeaconBlock(block), [32]byte{2})
	require.NoError(t, err, "Expected allowed block not to throw error")
}
