package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	validatorserviceconfig "github.com/prysmaticlabs/prysm/v4/config/validator/service"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/validator"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	validatormock "github.com/prysmaticlabs/prysm/v4/testing/validator-mock"
	"github.com/prysmaticlabs/prysm/v4/validator/accounts"
	"github.com/prysmaticlabs/prysm/v4/validator/accounts/iface"
	mock "github.com/prysmaticlabs/prysm/v4/validator/accounts/testing"
	"github.com/prysmaticlabs/prysm/v4/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v4/validator/client"
	dbtest "github.com/prysmaticlabs/prysm/v4/validator/db/testing"
	"github.com/prysmaticlabs/prysm/v4/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v4/validator/keymanager/derived"
	remoteweb3signer "github.com/prysmaticlabs/prysm/v4/validator/keymanager/remote-web3signer"
	mocks "github.com/prysmaticlabs/prysm/v4/validator/testing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestServer_SetVoluntaryExit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})
	defaultWalletPath = setupWalletDir(t)
	opts := []accounts.Option{
		accounts.WithWalletDir(defaultWalletPath),
		accounts.WithKeymanagerType(keymanager.Derived),
		accounts.WithWalletPassword(strongPass),
		accounts.WithSkipMnemonicConfirm(true),
	}
	acc, err := accounts.NewCLIManager(opts...)
	require.NoError(t, err)
	w, err := acc.WalletCreate(ctx)
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)

	m := &mock.Validator{Km: km}
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Validator: m,
	})
	require.NoError(t, err)

	dr, ok := km.(*derived.Keymanager)
	require.Equal(t, true, ok)
	err = dr.RecoverAccountsFromMnemonic(ctx, mocks.TestMnemonic, derived.DefaultMnemonicLanguage, "", 1)
	require.NoError(t, err)
	pubKeys, err := dr.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	beaconClient := validatormock.NewMockValidatorClient(ctrl)
	mockNodeClient := validatormock.NewMockNodeClient(ctrl)
	// Any time in the past will suffice
	genesisTime := &timestamppb.Timestamp{
		Seconds: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}

	beaconClient.EXPECT().ValidatorIndex(gomock.Any(), &eth.ValidatorIndexRequest{PublicKey: pubKeys[0][:]}).
		Times(3).
		Return(&eth.ValidatorIndexResponse{Index: 2}, nil)

	beaconClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Times(3).
		Return(&eth.DomainResponse{SignatureDomain: make([]byte, common.HashLength)}, nil /*err*/)

	mockNodeClient.EXPECT().
		GetGenesis(gomock.Any(), gomock.Any()).
		Times(3).
		Return(&eth.Genesis{GenesisTime: genesisTime}, nil)

	s := &Server{
		validatorService:          vs,
		beaconNodeValidatorClient: beaconClient,
		wallet:                    w,
		beaconNodeClient:          mockNodeClient,
		walletInitialized:         w != nil,
	}

	type want struct {
		epoch          primitives.Epoch
		validatorIndex uint64
		signature      []byte
	}

	type wantError struct {
		expectedStatusCode int
		expectedErrorMsg   string
	}

	tests := []struct {
		name      string
		epoch     int
		pubkey    string
		w         want
		wError    *wantError
		mockSetup func(s *Server) error
	}{
		{
			name:   "Ok: with epoch",
			epoch:  30000000,
			pubkey: hexutil.Encode(pubKeys[0][:]),
			w: want{
				epoch:          30000000,
				validatorIndex: 2,
				signature:      []uint8{175, 157, 5, 134, 253, 2, 193, 35, 176, 43, 217, 36, 39, 240, 24, 79, 207, 133, 150, 7, 237, 16, 54, 244, 64, 27, 244, 17, 8, 225, 140, 1, 172, 24, 35, 95, 178, 116, 172, 213, 113, 182, 193, 61, 192, 65, 162, 253, 19, 202, 111, 164, 195, 215, 0, 205, 95, 7, 30, 251, 244, 157, 210, 155, 238, 30, 35, 219, 177, 232, 174, 62, 218, 69, 23, 249, 180, 140, 60, 29, 190, 249, 229, 95, 235, 236, 81, 33, 60, 4, 201, 227, 70, 239, 167, 2},
			},
		},
		{
			name:   "Ok: epoch not set",
			pubkey: hexutil.Encode(pubKeys[0][:]),
			w: want{
				epoch:          0,
				validatorIndex: 2,
				signature:      []uint8{},
			},
		},
		{
			name:  "Error: Missing Public Key in URL Params",
			epoch: 30000000,
			wError: &wantError{
				expectedStatusCode: http.StatusBadRequest,
				expectedErrorMsg:   "pubkey is required in URL params",
			},
		},
		{
			name:   "Error: Invalid Public Key Length",
			epoch:  30000000,
			pubkey: "0x1asd1231",
			wError: &wantError{
				expectedStatusCode: http.StatusBadRequest,
				expectedErrorMsg:   "pubkey is invalid: invalid hex string",
			},
		},
		{
			name:   "Error: No Wallet Found",
			epoch:  30000000,
			pubkey: hexutil.Encode(pubKeys[0][:]),
			wError: &wantError{
				expectedStatusCode: http.StatusServiceUnavailable,
				expectedErrorMsg:   "No wallet found",
			},
			mockSetup: func(s *Server) error {
				s.wallet = nil
				s.walletInitialized = false
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetup != nil {
				require.NoError(t, tt.mockSetup(s))
			}
			req := httptest.NewRequest("POST", fmt.Sprintf("/eth/v1/validator/{pubkey}/voluntary_exit?epoch=%d", tt.epoch), nil)
			req = mux.SetURLVars(req, map[string]string{"pubkey": tt.pubkey})
			w := httptest.NewRecorder()
			w.Body = &bytes.Buffer{}

			s.SetVoluntaryExit(w, req)
			if tt.wError != nil {
				assert.Equal(t, tt.wError.expectedStatusCode, w.Code)
				require.StringContains(t, tt.wError.expectedErrorMsg, w.Body.String())
				return
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
			}
			resp := &SetVoluntaryExitResponse{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), resp))
			if tt.w.epoch == 0 {
				genesisResponse, err := s.beaconNodeClient.GetGenesis(ctx, &emptypb.Empty{})
				require.NoError(t, err)
				tt.w.epoch, err = client.CurrentEpoch(genesisResponse.GenesisTime)
				require.NoError(t, err)
				req2 := httptest.NewRequest("POST", fmt.Sprintf("/eth/v1/validator/{pubkey}/voluntary_exit?epoch=%d", tt.epoch), nil)
				req2 = mux.SetURLVars(req2, map[string]string{"pubkey": hexutil.Encode(pubKeys[0][:])})
				w2 := httptest.NewRecorder()
				w2.Body = &bytes.Buffer{}
				s.SetVoluntaryExit(w2, req2)
				if tt.wError != nil {
					assert.Equal(t, tt.wError.expectedStatusCode, w2.Code)
					require.StringContains(t, tt.wError.expectedErrorMsg, w2.Body.String())
				} else {
					assert.Equal(t, http.StatusOK, w2.Code)
					resp2 := &SetVoluntaryExitResponse{}
					require.NoError(t, json.Unmarshal(w2.Body.Bytes(), resp2))
					tt.w.signature, err = hexutil.Decode(resp2.Data.Signature)
					require.NoError(t, err)
				}

			}
			if tt.wError == nil {
				require.Equal(t, fmt.Sprintf("%d", tt.w.epoch), resp.Data.Message.Epoch)
				require.Equal(t, fmt.Sprintf("%d", tt.w.validatorIndex), resp.Data.Message.ValidatorIndex)
				require.NotEmpty(t, resp.Data.Signature)
				bSig, err := hexutil.Decode(resp.Data.Signature)
				require.NoError(t, err)
				ok = bytes.Equal(tt.w.signature, bSig)
				require.Equal(t, true, ok)
			}
		})
	}
}

func TestServer_GetGasLimit(t *testing.T) {
	ctx := context.Background()
	byteval, err := hexutil.Decode("0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493")
	byteval2, err2 := hexutil.Decode("0x1234567878903438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493")
	require.NoError(t, err)
	require.NoError(t, err2)

	tests := []struct {
		name   string
		args   *validatorserviceconfig.ProposerSettings
		pubkey [48]byte
		want   uint64
	}{
		{
			name: "ProposerSetting for specific pubkey exists",
			args: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(byteval): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: 123456789},
					},
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: 987654321},
				},
			},
			pubkey: bytesutil.ToBytes48(byteval),
			want:   123456789,
		},
		{
			name: "ProposerSetting for specific pubkey does not exist",
			args: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(byteval): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: 123456789},
					},
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: 987654321},
				},
			},
			// no settings for the following validator, so the gaslimit returned is the default value.
			pubkey: bytesutil.ToBytes48(byteval2),
			want:   987654321,
		},
		{
			name:   "No proposerSetting at all",
			args:   nil,
			pubkey: bytesutil.ToBytes48(byteval),
			want:   params.BeaconConfig().DefaultBuilderGasLimit,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mock.Validator{}
			err := m.SetProposerSettings(ctx, tt.args)
			require.NoError(t, err)
			vs, err := client.NewValidatorService(ctx, &client.Config{
				Validator: m,
			})
			require.NoError(t, err)
			s := &Server{
				validatorService: vs,
			}
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/eth/v1/validator/{pubkey}/gas_limit"), nil)
			req = mux.SetURLVars(req, map[string]string{"pubkey": hexutil.Encode(tt.pubkey[:])})
			w := httptest.NewRecorder()
			w.Body = &bytes.Buffer{}
			s.GetGasLimit(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			resp := &GetGasLimitResponse{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), resp))
			assert.Equal(t, fmt.Sprintf("%d", tt.want), resp.Data.GasLimit)
		})
	}
}

func TestServer_SetGasLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	beaconClient := validatormock.NewMockValidatorClient(ctrl)
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})

	pubkey1, err := hexutil.Decode("0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493")
	pubkey2, err2 := hexutil.Decode("0xbedefeaa94e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2cdddddddddddddddddddddddd")
	require.NoError(t, err)
	require.NoError(t, err2)

	type beaconResp struct {
		resp  *eth.FeeRecipientByPubKeyResponse
		error error
	}

	type want struct {
		pubkey   []byte
		gaslimit uint64
	}

	tests := []struct {
		name             string
		pubkey           []byte
		newGasLimit      uint64
		proposerSettings *validatorserviceconfig.ProposerSettings
		w                []*want
		beaconReturn     *beaconResp
		wantErr          string
	}{
		{
			name:             "ProposerSettings is nil",
			pubkey:           pubkey1,
			newGasLimit:      9999,
			proposerSettings: nil,
			wantErr:          "No proposer settings were found to update",
		},
		{
			name:        "ProposerSettings.ProposeConfig is nil AND ProposerSettings.DefaultConfig is nil",
			pubkey:      pubkey1,
			newGasLimit: 9999,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: nil,
				DefaultConfig: nil,
			},
			wantErr: "Gas limit changes only apply when builder is enabled",
		},
		{
			name:        "ProposerSettings.ProposeConfig is nil AND ProposerSettings.DefaultConfig.BuilderConfig is nil",
			pubkey:      pubkey1,
			newGasLimit: 9999,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: nil,
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					BuilderConfig: nil,
				},
			},
			wantErr: "Gas limit changes only apply when builder is enabled",
		},
		{
			name:        "ProposerSettings.ProposeConfig is defined for pubkey, BuilderConfig is nil AND ProposerSettings.DefaultConfig is nil",
			pubkey:      pubkey1,
			newGasLimit: 9999,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(pubkey1): {
						BuilderConfig: nil,
					},
				},
				DefaultConfig: nil,
			},
			wantErr: "Gas limit changes only apply when builder is enabled",
		},
		{
			name:        "ProposerSettings.ProposeConfig is defined for pubkey, BuilderConfig is defined AND ProposerSettings.DefaultConfig is nil",
			pubkey:      pubkey1,
			newGasLimit: 9999,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(pubkey1): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{},
					},
				},
				DefaultConfig: nil,
			},
			wantErr: "Gas limit changes only apply when builder is enabled",
		},
		{
			name:        "ProposerSettings.ProposeConfig is NOT defined for pubkey, BuilderConfig is defined AND ProposerSettings.DefaultConfig is nil",
			pubkey:      pubkey2,
			newGasLimit: 9999,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(pubkey2): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{
							Enabled:  true,
							GasLimit: 12345,
						},
					},
				},
				DefaultConfig: nil,
			},
			w: []*want{{
				pubkey2,
				9999,
			},
			},
		},
		{
			name:        "ProposerSettings.ProposeConfig is defined for pubkey, BuilderConfig is nil AND ProposerSettings.DefaultConfig.BuilderConfig is defined",
			pubkey:      pubkey1,
			newGasLimit: 9999,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(pubkey2): {
						BuilderConfig: nil,
					},
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					BuilderConfig: &validatorserviceconfig.BuilderConfig{
						Enabled: true,
					},
				},
			},
			w: []*want{{
				pubkey1,
				9999,
			},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mock.Validator{}
			err := m.SetProposerSettings(ctx, tt.proposerSettings)
			require.NoError(t, err)
			validatorDB := dbtest.SetupDB(t, [][fieldparams.BLSPubkeyLength]byte{})
			vs, err := client.NewValidatorService(ctx, &client.Config{
				Validator: m,
				ValDB:     validatorDB,
			})
			require.NoError(t, err)

			s := &Server{
				validatorService:          vs,
				beaconNodeValidatorClient: beaconClient,
				valDB:                     validatorDB,
			}

			if tt.beaconReturn != nil {
				beaconClient.EXPECT().GetFeeRecipientByPubKey(
					gomock.Any(),
					gomock.Any(),
				).Return(tt.beaconReturn.resp, tt.beaconReturn.error)
			}

			request := &SetGasLimitRequest{
				GasLimit: fmt.Sprintf("%d", tt.newGasLimit),
			}

			var buf bytes.Buffer
			err = json.NewEncoder(&buf).Encode(request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/eth/v1/validator/{pubkey}/gas_limit"), &buf)
			req = mux.SetURLVars(req, map[string]string{"pubkey": hexutil.Encode(tt.pubkey)})
			w := httptest.NewRecorder()
			w.Body = &bytes.Buffer{}

			s.SetGasLimit(w, req)

			if tt.wantErr != "" {
				assert.NotEqual(t, http.StatusOK, w.Code)
				require.StringContains(t, tt.wantErr, w.Body.String())
			} else {
				assert.Equal(t, http.StatusAccepted, w.Code)
				for _, wantObj := range tt.w {
					assert.Equal(t, wantObj.gaslimit, uint64(s.validatorService.ProposerSettings().ProposeConfig[bytesutil.ToBytes48(wantObj.pubkey)].BuilderConfig.GasLimit))
				}
			}
		})
	}
}

func TestServer_SetGasLimit_ValidatorServiceNil(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/eth/v1/validator/{pubkey}/gas_limit"), nil)
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}

	s.SetGasLimit(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code)
	require.StringContains(t, "Validator service not ready", w.Body.String())
}

func TestServer_SetGasLimit_InvalidPubKey(t *testing.T) {
	s := &Server{
		validatorService: &client.ValidatorService{},
	}
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/eth/v1/validator/{pubkey}/gas_limit"), nil)
	req = mux.SetURLVars(req, map[string]string{"pubkey": "0x00"})
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}

	s.SetGasLimit(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code)
	require.StringContains(t, "Invalid pubkey", w.Body.String())
}

func TestServer_DeleteGasLimit(t *testing.T) {
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})
	pubkey1, err := hexutil.Decode("0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493")
	pubkey2, err2 := hexutil.Decode("0xbedefeaa94e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2cdddddddddddddddddddddddd")
	require.NoError(t, err)
	require.NoError(t, err2)

	// This test changes global default values, we do not want this to side-affect other
	// tests, so store the origin global default and then restore after tests are done.
	originBeaconChainGasLimit := params.BeaconConfig().DefaultBuilderGasLimit
	defer func() {
		params.BeaconConfig().DefaultBuilderGasLimit = originBeaconChainGasLimit
	}()

	globalDefaultGasLimit := validator.Uint64(0xbbdd)

	type want struct {
		pubkey   []byte
		gaslimit validator.Uint64
	}

	tests := []struct {
		name             string
		pubkey           []byte
		proposerSettings *validatorserviceconfig.ProposerSettings
		wantError        error
		w                []want
	}{
		{
			name:   "delete existing gas limit with default config",
			pubkey: pubkey1,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(pubkey1): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: validator.Uint64(987654321)},
					},
					bytesutil.ToBytes48(pubkey2): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: validator.Uint64(123456789)},
					},
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: validator.Uint64(5555)},
				},
			},
			wantError: nil,
			w: []want{
				{
					pubkey: pubkey1,
					// After deletion, use DefaultConfig.BuilderConfig.GasLimitMetaData.
					gaslimit: validator.Uint64(5555),
				},
				{
					pubkey:   pubkey2,
					gaslimit: validator.Uint64(123456789),
				},
			},
		},
		{
			name:   "delete existing gas limit with no default config",
			pubkey: pubkey1,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(pubkey1): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: validator.Uint64(987654321)},
					},
					bytesutil.ToBytes48(pubkey2): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: validator.Uint64(123456789)},
					},
				},
			},
			wantError: nil,
			w: []want{
				{
					pubkey: pubkey1,
					// After deletion, use global default, because DefaultConfig is not set at all.
					gaslimit: globalDefaultGasLimit,
				},
				{
					pubkey:   pubkey2,
					gaslimit: validator.Uint64(123456789),
				},
			},
		},
		{
			name:   "delete nonexist gas limit",
			pubkey: pubkey2,
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(pubkey1): {
						BuilderConfig: &validatorserviceconfig.BuilderConfig{GasLimit: validator.Uint64(987654321)},
					},
				},
			},
			wantError: fmt.Errorf("%d", http.StatusNotFound),
			w: []want{
				// pubkey1's gaslimit is unaffected
				{
					pubkey:   pubkey1,
					gaslimit: validator.Uint64(987654321),
				},
			},
		},
		{
			name:      "delete nonexist gas limit 2",
			pubkey:    pubkey2,
			wantError: fmt.Errorf("%d", http.StatusNotFound),
			w:         []want{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mock.Validator{}
			err := m.SetProposerSettings(ctx, tt.proposerSettings)
			require.NoError(t, err)
			validatorDB := dbtest.SetupDB(t, [][fieldparams.BLSPubkeyLength]byte{})
			vs, err := client.NewValidatorService(ctx, &client.Config{
				Validator: m,
				ValDB:     validatorDB,
			})
			require.NoError(t, err)
			s := &Server{
				validatorService: vs,
				valDB:            validatorDB,
			}
			// Set up global default value for builder gas limit.
			params.BeaconConfig().DefaultBuilderGasLimit = uint64(globalDefaultGasLimit)

			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/eth/v1/validator/{pubkey}/gas_limit"), nil)
			req = mux.SetURLVars(req, map[string]string{"pubkey": hexutil.Encode(tt.pubkey)})
			w := httptest.NewRecorder()
			w.Body = &bytes.Buffer{}

			s.DeleteGasLimit(w, req)

			if tt.wantError != nil {
				assert.StringContains(t, tt.wantError.Error(), w.Body.String())
			} else {
				assert.Equal(t, http.StatusNoContent, w.Code)
			}
			for _, wantedObj := range tt.w {
				assert.Equal(t, wantedObj.gaslimit, s.validatorService.ProposerSettings().ProposeConfig[bytesutil.ToBytes48(wantedObj.pubkey)].BuilderConfig.GasLimit)
			}
		})
	}
}

func TestServer_ListRemoteKeys(t *testing.T) {
	t.Run("wallet not ready", func(t *testing.T) {
		s := Server{}
		_, err := s.ListKeystores(context.Background(), &empty.Empty{})
		require.ErrorContains(t, "Prysm Wallet not initialized. Please create a new wallet.", err)
	})
	ctx := context.Background()
	w := wallet.NewWalletForWeb3Signer()
	root := make([]byte, fieldparams.RootLength)
	root[0] = 1
	bytevalue, err := hexutil.Decode("0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a")
	require.NoError(t, err)
	pubkeys := [][fieldparams.BLSPubkeyLength]byte{bytesutil.ToBytes48(bytevalue)}
	config := &remoteweb3signer.SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		ProvidedPublicKeys:    pubkeys,
	}
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false, Web3SignerConfig: config})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.Validator{
			Km: km,
		},
		Web3SignerConfig: config,
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}
	expectedKeys, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)

	t.Run("returns proper data with existing pub keystores", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/eth/v1/remotekeys", nil)
		w := httptest.NewRecorder()
		w.Body = &bytes.Buffer{}
		s.ListRemoteKeys(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		resp := &ListRemoteKeysResponse{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), resp))
		for i := 0; i < len(resp.Data); i++ {
			require.DeepEqual(t, hexutil.Encode(expectedKeys[i][:]), resp.Data[i].Pubkey)
		}
	})
}

func TestServer_ImportRemoteKeys(t *testing.T) {
	t.Run("wallet not ready", func(t *testing.T) {
		s := Server{}
		_, err := s.ListKeystores(context.Background(), &empty.Empty{})
		require.ErrorContains(t, "Prysm Wallet not initialized. Please create a new wallet.", err)
	})
	ctx := context.Background()
	w := wallet.NewWalletForWeb3Signer()
	root := make([]byte, fieldparams.RootLength)
	root[0] = 1
	config := &remoteweb3signer.SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		ProvidedPublicKeys:    nil,
	}
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false, Web3SignerConfig: config})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.Validator{
			Km: km,
		},
		Web3SignerConfig: config,
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}
	pubkey := "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"
	remoteKeys := []*RemoteKey{
		{
			Pubkey: pubkey,
		},
	}

	t.Run("returns proper data with existing pub keystores", func(t *testing.T) {
		var body bytes.Buffer
		b, err := json.Marshal(&ImportRemoteKeysRequest{RemoteKeys: remoteKeys})
		require.NoError(t, err)
		body.Write(b)
		req := httptest.NewRequest("GET", "/eth/v1/remotekeys", &body)
		w := httptest.NewRecorder()
		w.Body = &bytes.Buffer{}
		s.ImportRemoteKeys(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		expectedStatuses := []*keymanager.KeyStatus{
			{
				Status:  remoteweb3signer.StatusImported,
				Message: fmt.Sprintf("Successfully added pubkey: %v", pubkey),
			},
		}
		resp := &RemoteKeysResponse{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), resp))
		for i := 0; i < len(resp.Data); i++ {
			require.DeepEqual(t, expectedStatuses[i], resp.Data[i])
		}
	})
}

func TestServer_DeleteRemoteKeys(t *testing.T) {
	t.Run("wallet not ready", func(t *testing.T) {
		s := Server{}
		_, err := s.ListKeystores(context.Background(), &empty.Empty{})
		require.ErrorContains(t, "Prysm Wallet not initialized. Please create a new wallet.", err)
	})
	ctx := context.Background()
	w := wallet.NewWalletForWeb3Signer()
	root := make([]byte, fieldparams.RootLength)
	root[0] = 1
	pkey := "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"
	bytevalue, err := hexutil.Decode(pkey)
	require.NoError(t, err)
	pubkeys := [][fieldparams.BLSPubkeyLength]byte{bytesutil.ToBytes48(bytevalue)}
	config := &remoteweb3signer.SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		ProvidedPublicKeys:    pubkeys,
	}
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false, Web3SignerConfig: config})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.Validator{
			Km: km,
		},
		Web3SignerConfig: config,
	})
	require.NoError(t, err)
	s := &Server{
		walletInitialized: true,
		wallet:            w,
		validatorService:  vs,
	}

	t.Run("returns proper data with existing pub keystores", func(t *testing.T) {
		var body bytes.Buffer
		b, err := json.Marshal(&DeleteRemoteKeysRequest{
			Pubkeys: []string{pkey},
		})
		require.NoError(t, err)
		body.Write(b)
		req := httptest.NewRequest("DELETE", "/eth/v1/remotekeys", &body)
		w := httptest.NewRecorder()
		w.Body = &bytes.Buffer{}
		s.DeleteRemoteKeys(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		expectedStatuses := []*keymanager.KeyStatus{
			{
				Status:  remoteweb3signer.StatusDeleted,
				Message: fmt.Sprintf("Successfully deleted pubkey: %v", pkey),
			},
		}
		resp := &RemoteKeysResponse{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), resp))
		for i := 0; i < len(resp.Data); i++ {
			require.DeepEqual(t, expectedStatuses[i], resp.Data[i])

		}
		expectedKeys, err := km.FetchValidatingPublicKeys(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, len(expectedKeys))
	})
}
