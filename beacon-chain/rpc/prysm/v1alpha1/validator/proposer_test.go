package validator

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/builder"
	builderTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/builder/testing"
	dbutil "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestProposer_GetFeeRecipientByPubKey(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()
	numDeposits := uint64(64)
	beaconState, _ := util.DeterministicGenesisState(t, numDeposits)
	bsRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	proposerServer := &Server{
		BeaconDB:    db,
		HeadFetcher: &mock.ChainService{Root: bsRoot[:], State: beaconState},
	}
	pubkey, err := hexutil.Decode("0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a")
	require.NoError(t, err)
	resp, err := proposerServer.GetFeeRecipientByPubKey(ctx, &ethpb.FeeRecipientByPubKeyRequest{
		PublicKey: pubkey,
	})
	require.NoError(t, err)

	require.Equal(t, params.BeaconConfig().DefaultFeeRecipient.Hex(), hexutil.Encode(resp.FeeRecipient))
	params.BeaconConfig().DefaultFeeRecipient = common.HexToAddress("0x046Fb65722E7b2455012BFEBf6177F1D2e9728D9")
	resp, err = proposerServer.GetFeeRecipientByPubKey(ctx, &ethpb.FeeRecipientByPubKeyRequest{
		PublicKey: beaconState.Validators()[0].PublicKey,
	})
	require.NoError(t, err)

	require.Equal(t, params.BeaconConfig().DefaultFeeRecipient.Hex(), common.BytesToAddress(resp.FeeRecipient).Hex())
	index, err := proposerServer.ValidatorIndex(ctx, &ethpb.ValidatorIndexRequest{
		PublicKey: beaconState.Validators()[0].PublicKey,
	})
	require.NoError(t, err)
	err = proposerServer.BeaconDB.SaveFeeRecipientsByValidatorIDs(ctx, []primitives.ValidatorIndex{index.Index}, []common.Address{common.HexToAddress("0x055Fb65722E7b2455012BFEBf6177F1D2e9728D8")})
	require.NoError(t, err)
	resp, err = proposerServer.GetFeeRecipientByPubKey(ctx, &ethpb.FeeRecipientByPubKeyRequest{
		PublicKey: beaconState.Validators()[0].PublicKey,
	})
	require.NoError(t, err)

	require.Equal(t, common.HexToAddress("0x055Fb65722E7b2455012BFEBf6177F1D2e9728D8").Hex(), common.BytesToAddress(resp.FeeRecipient).Hex())
}

func TestProposer_PrepareBeaconProposer(t *testing.T) {
	type args struct {
		request *ethpb.PrepareBeaconProposerRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name: "Happy Path",
			args: args{
				request: &ethpb.PrepareBeaconProposerRequest{
					Recipients: []*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{
						{
							FeeRecipient:   make([]byte, fieldparams.FeeRecipientLength),
							ValidatorIndex: 1,
						},
					},
				},
			},
			wantErr: "",
		},
		{
			name: "invalid fee recipient length",
			args: args{
				request: &ethpb.PrepareBeaconProposerRequest{
					Recipients: []*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{
						{
							FeeRecipient:   make([]byte, fieldparams.BLSPubkeyLength),
							ValidatorIndex: 1,
						},
					},
				},
			},
			wantErr: "Invalid fee recipient address",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := dbutil.SetupDB(t)
			ctx := context.Background()
			s := &Server{BeaconDB: db}
			_, err := s.PrepareBeaconProposer(ctx, tt.args.request)
			if tt.wantErr != "" {
				require.ErrorContains(t, tt.wantErr, err)
				return
			}
			require.NoError(t, err)
			address, err := s.BeaconDB.FeeRecipientByValidatorID(ctx, 1)
			require.NoError(t, err)
			require.Equal(t, common.BytesToAddress(tt.args.request.Recipients[0].FeeRecipient), address)

		})
	}
}

func TestProposer_PrepareBeaconProposerOverlapping(t *testing.T) {
	hook := logTest.NewGlobal()
	db := dbutil.SetupDB(t)
	ctx := context.Background()
	proposerServer := &Server{BeaconDB: db}

	// New validator
	f := bytesutil.PadTo([]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	req := &ethpb.PrepareBeaconProposerRequest{
		Recipients: []*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{
			{FeeRecipient: f, ValidatorIndex: 1},
		},
	}
	_, err := proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Updated fee recipient addresses for validator indices")

	// Same validator
	hook.Reset()
	_, err = proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsDoNotContain(t, hook, "Updated fee recipient addresses for validator indices")

	// Same validator with different fee recipient
	hook.Reset()
	f = bytesutil.PadTo([]byte{0x01, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	req = &ethpb.PrepareBeaconProposerRequest{
		Recipients: []*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{
			{FeeRecipient: f, ValidatorIndex: 1},
		},
	}
	_, err = proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Updated fee recipient addresses for validator indices")

	// More than one validator
	hook.Reset()
	f = bytesutil.PadTo([]byte{0x01, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	req = &ethpb.PrepareBeaconProposerRequest{
		Recipients: []*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{
			{FeeRecipient: f, ValidatorIndex: 1},
			{FeeRecipient: f, ValidatorIndex: 2},
		},
	}
	_, err = proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsContain(t, hook, "Updated fee recipient addresses for validator indices")

	// Same validators
	hook.Reset()
	_, err = proposerServer.PrepareBeaconProposer(ctx, req)
	require.NoError(t, err)
	require.LogsDoNotContain(t, hook, "Updated fee recipient addresses for validator indices")
}

func BenchmarkServer_PrepareBeaconProposer(b *testing.B) {
	db := dbutil.SetupDB(b)
	ctx := context.Background()
	proposerServer := &Server{BeaconDB: db}

	f := bytesutil.PadTo([]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}, fieldparams.FeeRecipientLength)
	recipients := make([]*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer, 0)
	for i := 0; i < 10000; i++ {
		recipients = append(recipients, &ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{FeeRecipient: f, ValidatorIndex: primitives.ValidatorIndex(i)})
	}

	req := &ethpb.PrepareBeaconProposerRequest{
		Recipients: recipients,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := proposerServer.PrepareBeaconProposer(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestProposer_SubmitValidatorRegistrations(t *testing.T) {
	ctx := context.Background()
	proposerServer := &Server{}
	reg := &ethpb.SignedValidatorRegistrationsV1{}
	_, err := proposerServer.SubmitValidatorRegistrations(ctx, reg)
	require.ErrorContains(t, builder.ErrNoBuilder.Error(), err)
	proposerServer = &Server{BlockBuilder: &builderTest.MockBuilderService{}}
	_, err = proposerServer.SubmitValidatorRegistrations(ctx, reg)
	require.ErrorContains(t, builder.ErrNoBuilder.Error(), err)
	proposerServer = &Server{BlockBuilder: &builderTest.MockBuilderService{HasConfigured: true}}
	_, err = proposerServer.SubmitValidatorRegistrations(ctx, reg)
	require.NoError(t, err)
	proposerServer = &Server{BlockBuilder: &builderTest.MockBuilderService{HasConfigured: true, ErrRegisterValidator: errors.New("bad")}}
	_, err = proposerServer.SubmitValidatorRegistrations(ctx, reg)
	require.ErrorContains(t, "bad", err)
}
