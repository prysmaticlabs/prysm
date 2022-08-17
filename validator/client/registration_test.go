package client

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestSubmitValidatorRegistrations(t *testing.T) {
	_, m, validatorKey, finish := setup(t)
	defer finish()

	ctx := context.Background()
	require.NoError(t, nil, SubmitValidatorRegistrations(ctx, m.validatorClient, []*ethpb.SignedValidatorRegistrationV1{}))

	reg := &ethpb.ValidatorRegistrationV1{
		FeeRecipient: bytesutil.PadTo([]byte("fee"), 20),
		GasLimit:     123456,
		Timestamp:    uint64(time.Now().Unix()),
		Pubkey:       validatorKey.PublicKey().Marshal(),
	}

	m.validatorClient.EXPECT().
		SubmitValidatorRegistrations(gomock.Any(), &ethpb.SignedValidatorRegistrationsV1{
			Messages: []*ethpb.SignedValidatorRegistrationV1{
				{Message: reg,
					Signature: params.BeaconConfig().ZeroHash[:]},
			},
		}).
		Return(nil, nil)
	require.NoError(t, nil, SubmitValidatorRegistrations(ctx, m.validatorClient, []*ethpb.SignedValidatorRegistrationV1{
		{Message: reg,
			Signature: params.BeaconConfig().ZeroHash[:]},
	}))
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

	m.validatorClient.EXPECT().
		SubmitValidatorRegistrations(gomock.Any(), &ethpb.SignedValidatorRegistrationsV1{
			Messages: []*ethpb.SignedValidatorRegistrationV1{
				{Message: reg,
					Signature: params.BeaconConfig().ZeroHash[:]},
			},
		}).
		Return(nil, errors.New("could not sign"))
	require.ErrorContains(t, "could not sign", SubmitValidatorRegistrations(ctx, m.validatorClient, []*ethpb.SignedValidatorRegistrationV1{
		{Message: reg,
			Signature: params.BeaconConfig().ZeroHash[:]},
	}))
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
	_, err := signValidatorRegistration(ctx, m.signfunc, reg)
	require.NoError(t, err)

}

func TestValidator_SignValidatorRegistrationRequest(t *testing.T) {
	_, m, validatorKey, finish := setup(t)
	defer finish()
	ctx := context.Background()
	byteval, err := hexutil.Decode("0x878705ba3f8bc32fcf7f4caa1a35e72af65cf766")
	require.NoError(t, err)
	tests := []struct {
		name            string
		arg             *ethpb.ValidatorRegistrationV1
		validatorSetter func(t *testing.T) *validator
		isCached        bool
		err             string
	}{
		{
			name: " Happy Path cached",
			arg: &ethpb.ValidatorRegistrationV1{
				Pubkey:       validatorKey.PublicKey().Marshal(),
				FeeRecipient: make([]byte, fieldparams.FeeRecipientLength),
				GasLimit:     30000000,
				Timestamp:    uint64(time.Now().Unix()),
			},
			validatorSetter: func(t *testing.T) *validator {
				v := validator{
					pubkeyToValidatorIndex:       make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
					signedValidatorRegistrations: make(map[[fieldparams.BLSPubkeyLength]byte]*ethpb.SignedValidatorRegistrationV1),
					useWeb:                       false,
					genesisTime:                  0,
				}
				v.signedValidatorRegistrations[bytesutil.ToBytes48(validatorKey.PublicKey().Marshal())] = &ethpb.SignedValidatorRegistrationV1{
					Message: &ethpb.ValidatorRegistrationV1{
						Pubkey:       validatorKey.PublicKey().Marshal(),
						GasLimit:     30000000,
						FeeRecipient: make([]byte, fieldparams.FeeRecipientLength),
						Timestamp:    uint64(time.Now().Unix()),
					},
					Signature: make([]byte, 0),
				}
				return &v
			},
			isCached: true,
		},
		{
			name: " Happy Path not cached gas updated",
			arg: &ethpb.ValidatorRegistrationV1{
				Pubkey:       validatorKey.PublicKey().Marshal(),
				FeeRecipient: make([]byte, fieldparams.FeeRecipientLength),
				GasLimit:     30000000,
				Timestamp:    uint64(time.Now().Unix()),
			},
			validatorSetter: func(t *testing.T) *validator {
				v := validator{
					pubkeyToValidatorIndex:       make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
					signedValidatorRegistrations: make(map[[fieldparams.BLSPubkeyLength]byte]*ethpb.SignedValidatorRegistrationV1),
					useWeb:                       false,
					genesisTime:                  0,
				}
				v.signedValidatorRegistrations[bytesutil.ToBytes48(validatorKey.PublicKey().Marshal())] = &ethpb.SignedValidatorRegistrationV1{
					Message: &ethpb.ValidatorRegistrationV1{
						Pubkey:       validatorKey.PublicKey().Marshal(),
						GasLimit:     35000000,
						FeeRecipient: make([]byte, fieldparams.FeeRecipientLength),
						Timestamp:    uint64(time.Now().Unix() - 1),
					},
					Signature: make([]byte, 0),
				}
				return &v
			},
			isCached: false,
		},
		{
			name: " Happy Path not cached feerecipient updated",
			arg: &ethpb.ValidatorRegistrationV1{
				Pubkey:       validatorKey.PublicKey().Marshal(),
				FeeRecipient: byteval,
				GasLimit:     30000000,
				Timestamp:    uint64(time.Now().Unix()),
			},
			validatorSetter: func(t *testing.T) *validator {
				v := validator{
					pubkeyToValidatorIndex:       make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
					signedValidatorRegistrations: make(map[[fieldparams.BLSPubkeyLength]byte]*ethpb.SignedValidatorRegistrationV1),
					useWeb:                       false,
					genesisTime:                  0,
				}
				v.signedValidatorRegistrations[bytesutil.ToBytes48(validatorKey.PublicKey().Marshal())] = &ethpb.SignedValidatorRegistrationV1{
					Message: &ethpb.ValidatorRegistrationV1{
						Pubkey:       validatorKey.PublicKey().Marshal(),
						GasLimit:     30000000,
						FeeRecipient: make([]byte, fieldparams.FeeRecipientLength),
						Timestamp:    uint64(time.Now().Unix() - 1),
					},
					Signature: make([]byte, 0),
				}
				return &v
			},
			isCached: false,
		},
		{
			name: " Happy Path not cached first Entry",
			arg: &ethpb.ValidatorRegistrationV1{
				Pubkey:       validatorKey.PublicKey().Marshal(),
				FeeRecipient: byteval,
				GasLimit:     30000000,
				Timestamp:    uint64(time.Now().Unix()),
			},
			validatorSetter: func(t *testing.T) *validator {
				v := validator{
					pubkeyToValidatorIndex:       make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
					signedValidatorRegistrations: make(map[[fieldparams.BLSPubkeyLength]byte]*ethpb.SignedValidatorRegistrationV1),
					useWeb:                       false,
					genesisTime:                  0,
				}
				return &v
			},
			isCached: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.validatorSetter(t)

			startingReq, ok := v.signedValidatorRegistrations[bytesutil.ToBytes48(tt.arg.Pubkey)]

			got, err := v.SignValidatorRegistrationRequest(ctx, m.signfunc, tt.arg)
			require.NoError(t, err)
			if tt.isCached {
				require.DeepEqual(t, got, v.signedValidatorRegistrations[bytesutil.ToBytes48(tt.arg.Pubkey)])
			} else {
				if ok {
					require.NotEqual(t, got.Message.Timestamp, startingReq.Message.Timestamp)
				}
				require.Equal(t, got.Message.Timestamp, tt.arg.Timestamp)
				require.Equal(t, got.Message.GasLimit, tt.arg.GasLimit)
				require.Equal(t, hexutil.Encode(got.Message.FeeRecipient), hexutil.Encode(tt.arg.FeeRecipient))
				require.DeepEqual(t, got, v.signedValidatorRegistrations[bytesutil.ToBytes48(tt.arg.Pubkey)])
			}
		})
	}
}
