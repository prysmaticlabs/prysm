package client

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSubmitValidatorRegistration(t *testing.T) {
	_, m, validatorKey, finish := setup(t)
	defer finish()

	ctx := context.Background()
	require.NoError(t, nil, SubmitValidatorRegistration(ctx, m.validatorClient, m.nodeClient, m.signfunc, []*ethpb.ValidatorRegistrationV1{}))

	reg := &ethpb.ValidatorRegistrationV1{
		FeeRecipient: bytesutil.PadTo([]byte("fee"), 20),
		GasLimit:     123456,
		Timestamp:    uint64(time.Now().Unix()),
		Pubkey:       validatorKey.PublicKey().Marshal(),
	}
	regs := []*ethpb.ValidatorRegistrationV1{reg}

	ti := &timestamppb.Timestamp{}
	m.nodeClient.EXPECT().
		GetGenesis(gomock.Any(), &emptypb.Empty{}).
		Return(&ethpb.Genesis{GenesisTime: ti}, nil)

	m.validatorClient.EXPECT().
		SubmitValidatorRegistration(gomock.Any(), &ethpb.SignedValidatorRegistrationsV1{
			Messages: []*ethpb.SignedValidatorRegistrationV1{
				{Message: reg,
					Signature: params.BeaconConfig().ZeroHash[:]},
			},
		}).
		Return(nil, nil)
	require.NoError(t, nil, SubmitValidatorRegistration(ctx, m.validatorClient, m.nodeClient, m.signfunc, regs))
}

func TestSubmitValidatorRegistration_CantSign(t *testing.T) {
	_, m, validatorKey, finish := setup(t)
	defer finish()

	ctx := context.Background()
	reg := &ethpb.ValidatorRegistrationV1{
		FeeRecipient: bytesutil.PadTo([]byte("fee"), 20),
		GasLimit:     123456,
		Timestamp:    uint64(time.Now().Unix()),
		Pubkey:       validatorKey.PublicKey().Marshal(),
	}
	regs := []*ethpb.ValidatorRegistrationV1{reg}

	genesisTime := &timestamppb.Timestamp{}
	m.nodeClient.EXPECT().
		GetGenesis(gomock.Any(), &emptypb.Empty{}).
		Return(&ethpb.Genesis{GenesisTime: genesisTime}, nil)

	m.validatorClient.EXPECT().
		SubmitValidatorRegistration(gomock.Any(), &ethpb.SignedValidatorRegistrationsV1{
			Messages: []*ethpb.SignedValidatorRegistrationV1{
				{Message: reg,
					Signature: params.BeaconConfig().ZeroHash[:]},
			},
		}).
		Return(nil, errors.New("could not sign"))
	require.ErrorContains(t, "could not sign", SubmitValidatorRegistration(ctx, m.validatorClient, m.nodeClient, m.signfunc, regs))
}

func Test_signValidatorRegistration(t *testing.T) {
	_, m, validatorKey, finish := setup(t)
	defer finish()

	ctx := context.Background()
	reg := &ethpb.ValidatorRegistrationV1{
		FeeRecipient: bytesutil.PadTo([]byte("fee"), 20),
		GasLimit:     123456,
		Timestamp:    uint64(time.Now().Unix()),
		Pubkey:       validatorKey.PublicKey().Marshal(),
	}
	_, err := signValidatorRegistration(
		ctx,
		1,
		m.validatorClient, m.signfunc, reg)
	require.NoError(t, err)

}
