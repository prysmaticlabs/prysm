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
	validatorserviceconfig "github.com/prysmaticlabs/prysm/v4/config/validator/service"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
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
		req := httptest.NewRequest("GET", "/eth/v1/remotekeys", nil)
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

func TestServer_ListFeeRecipientByPubkey(t *testing.T) {
	ctx := context.Background()
	pubkey := "0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493"
	byteval, err := hexutil.Decode(pubkey)
	require.NoError(t, err)

	type want struct {
		EthAddress string
	}

	tests := []struct {
		name   string
		args   *validatorserviceconfig.ProposerSettings
		want   *want
		cached *eth.FeeRecipientByPubKeyResponse
	}{
		{
			name: "ProposerSettings.ProposeConfig.FeeRecipientConfig defined for pubkey (and ProposerSettings.DefaultConfig.FeeRecipientConfig defined)",
			args: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(byteval): {
						FeeRecipientConfig: &validatorserviceconfig.FeeRecipientConfig{
							FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9"),
						},
					},
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					FeeRecipientConfig: &validatorserviceconfig.FeeRecipientConfig{
						FeeRecipient: common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
					},
				},
			},
			want: &want{
				EthAddress: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			},
		},
		{
			name: "ProposerSettings.ProposeConfig.FeeRecipientConfig NOT defined for pubkey and ProposerSettings.DefaultConfig.FeeRecipientConfig defined",
			args: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					FeeRecipientConfig: &validatorserviceconfig.FeeRecipientConfig{
						FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9"),
					},
				},
			},
			want: &want{
				EthAddress: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			},
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
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/eth/v1/validator/{pubkey}/feerecipient"), nil)
			req = mux.SetURLVars(req, map[string]string{"pubkey": pubkey})
			w := httptest.NewRecorder()
			w.Body = &bytes.Buffer{}
			s.ListFeeRecipientByPubkey(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			resp := &GetFeeRecipientByPubkeyResponse{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), resp))
			assert.Equal(t, tt.want.EthAddress, resp.Data.Ethaddress)
		})
	}
}

func TestServer_ListFeeRecipientByPubKey_NoFeeRecipientSet(t *testing.T) {
	ctx := context.Background()

	vs, err := client.NewValidatorService(ctx, &client.Config{
		Validator: &mock.Validator{},
	})
	require.NoError(t, err)

	s := &Server{
		validatorService: vs,
	}
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/eth/v1/validator/{pubkey}/feerecipient"), nil)
	req = mux.SetURLVars(req, map[string]string{"pubkey": "0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493"})
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	s.ListFeeRecipientByPubkey(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code)
	require.StringContains(t, "No fee recipient set", w.Body.String())
}

func TestServer_ListFeeRecipientByPubkey_ValidatorServiceNil(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/eth/v1/validator/{pubkey}/feerecipient"), nil)
	req = mux.SetURLVars(req, map[string]string{"pubkey": "0x00"})
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	s.SetFeeRecipientByPubkey(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code)
	require.StringContains(t, "Validator service not ready", w.Body.String())
}

func TestServer_ListFeeRecipientByPubkey_InvalidPubKey(t *testing.T) {
	s := &Server{
		validatorService: &client.ValidatorService{},
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/eth/v1/validator/{pubkey}/feerecipient"), nil)
	req = mux.SetURLVars(req, map[string]string{"pubkey": "0x00"})
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	s.SetFeeRecipientByPubkey(w, req)
	assert.NotEqual(t, http.StatusOK, w.Code)
	require.StringContains(t, "Invalid pubkey", w.Body.String())
}

func TestServer_FeeRecipientByPubkey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	beaconClient := validatormock.NewMockValidatorClient(ctrl)
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})
	pubkey := "0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493"
	byteval, err := hexutil.Decode(pubkey)
	require.NoError(t, err)

	type want struct {
		valEthAddress     string
		defaultEthaddress string
	}
	type beaconResp struct {
		resp  *eth.FeeRecipientByPubKeyResponse
		error error
	}
	tests := []struct {
		name             string
		args             string
		proposerSettings *validatorserviceconfig.ProposerSettings
		want             *want
		wantErr          bool
		beaconReturn     *beaconResp
	}{
		{
			name:             "ProposerSetting is nil",
			args:             "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			proposerSettings: nil,
			want: &want{
				valEthAddress: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			},
			wantErr: false,
			beaconReturn: &beaconResp{
				resp:  nil,
				error: nil,
			},
		},
		{
			name: "ProposerSetting.ProposeConfig is nil",
			args: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: nil,
			},
			want: &want{
				valEthAddress: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			},
			wantErr: false,
			beaconReturn: &beaconResp{
				resp:  nil,
				error: nil,
			},
		},
		{
			name: "ProposerSetting.ProposeConfig is nil AND ProposerSetting.Defaultconfig is defined",
			args: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: nil,
				DefaultConfig: &validatorserviceconfig.ProposerOption{},
			},
			want: &want{
				valEthAddress: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			},
			wantErr: false,
			beaconReturn: &beaconResp{
				resp:  nil,
				error: nil,
			},
		},
		{
			name: "ProposerSetting.ProposeConfig is defined for pubkey",
			args: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(byteval): {},
				},
			},
			want: &want{
				valEthAddress: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			},
			wantErr: false,
			beaconReturn: &beaconResp{
				resp:  nil,
				error: nil,
			},
		},
		{
			name: "ProposerSetting.ProposeConfig not defined for pubkey",
			args: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{},
			},
			want: &want{
				valEthAddress: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			},
			wantErr: false,
			beaconReturn: &beaconResp{
				resp:  nil,
				error: nil,
			},
		},
		{
			name: "ProposerSetting.ProposeConfig is nil for pubkey",
			args: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(byteval): nil,
				},
			},
			want: &want{
				valEthAddress: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			},
			wantErr: false,
			beaconReturn: &beaconResp{
				resp:  nil,
				error: nil,
			},
		},
		{
			name: "ProposerSetting.ProposeConfig is nil for pubkey AND DefaultConfig is not nil",
			args: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(byteval): nil,
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{},
			},
			want: &want{
				valEthAddress: "0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9",
			},
			wantErr: false,
			beaconReturn: &beaconResp{
				resp:  nil,
				error: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mock.Validator{}
			err := m.SetProposerSettings(ctx, tt.proposerSettings)
			require.NoError(t, err)
			validatorDB := dbtest.SetupDB(t, [][fieldparams.BLSPubkeyLength]byte{})

			// save a default here
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
			request := &SetFeeRecipientByPubkeyRequest{
				Ethaddress: tt.args,
			}

			var buf bytes.Buffer
			err = json.NewEncoder(&buf).Encode(request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/eth/v1/validator/{pubkey}/feerecipient"), &buf)
			req = mux.SetURLVars(req, map[string]string{"pubkey": pubkey})
			w := httptest.NewRecorder()
			w.Body = &bytes.Buffer{}
			s.SetFeeRecipientByPubkey(w, req)
			assert.Equal(t, http.StatusAccepted, w.Code)

			assert.Equal(t, tt.want.valEthAddress, s.validatorService.ProposerSettings().ProposeConfig[bytesutil.ToBytes48(byteval)].FeeRecipientConfig.FeeRecipient.Hex())
		})
	}
}

func TestServer_SetFeeRecipientByPubkey_InvalidPubKey(t *testing.T) {
	s := &Server{
		validatorService: &client.ValidatorService{},
	}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/eth/v1/validator/{pubkey}/feerecipient"), nil)
	req = mux.SetURLVars(req, map[string]string{"pubkey": "0x00"})
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	s.SetFeeRecipientByPubkey(w, req)
	assert.NotEqual(t, http.StatusAccepted, w.Code)
	require.StringContains(t, "Invalid pubkey", w.Body.String())
}

func TestServer_SetFeeRecipientByPubkey_InvalidFeeRecipient(t *testing.T) {
	pubkey := "0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493"

	s := &Server{
		validatorService: &client.ValidatorService{},
	}
	request := &SetFeeRecipientByPubkeyRequest{
		Ethaddress: "0x00",
	}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/eth/v1/validator/{pubkey}/feerecipient"), &buf)
	req = mux.SetURLVars(req, map[string]string{"pubkey": pubkey})
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	s.SetFeeRecipientByPubkey(w, req)
	assert.NotEqual(t, http.StatusAccepted, w.Code)

	require.StringContains(t, "Invalid Ethereum Address", w.Body.String())
}

func TestServer_DeleteFeeRecipientByPubkey(t *testing.T) {
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})
	pubkey := "0xaf2e7ba294e03438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493"
	byteval, err := hexutil.Decode(pubkey)
	require.NoError(t, err)
	type want struct {
		EthAddress string
	}
	tests := []struct {
		name             string
		proposerSettings *validatorserviceconfig.ProposerSettings
		want             *want
		wantErr          bool
	}{
		{
			name: "Happy Path Test",
			proposerSettings: &validatorserviceconfig.ProposerSettings{
				ProposeConfig: map[[48]byte]*validatorserviceconfig.ProposerOption{
					bytesutil.ToBytes48(byteval): {
						FeeRecipientConfig: &validatorserviceconfig.FeeRecipientConfig{
							FeeRecipient: common.HexToAddress("0x055Fb65722E7b2455012BFEBf6177F1D2e9738D5"),
						},
					},
				},
				DefaultConfig: &validatorserviceconfig.ProposerOption{
					FeeRecipientConfig: &validatorserviceconfig.FeeRecipientConfig{
						FeeRecipient: common.HexToAddress("0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9"),
					},
				},
			},
			want: &want{
				EthAddress: common.HexToAddress("0x046Fb65722E7b2455012BFEBf6177F1D2e9738D9").Hex(),
			},
			wantErr: false,
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
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/eth/v1/validator/{pubkey}/feerecipient"), nil)
			req = mux.SetURLVars(req, map[string]string{"pubkey": pubkey})
			w := httptest.NewRecorder()
			w.Body = &bytes.Buffer{}
			s.DeleteFeeRecipientByPubkey(w, req)
			assert.Equal(t, http.StatusNoContent, w.Code)
			assert.Equal(t, true, s.validatorService.ProposerSettings().ProposeConfig[bytesutil.ToBytes48(byteval)].FeeRecipientConfig == nil)
		})
	}
}

func TestServer_DeleteFeeRecipientByPubkey_ValidatorServiceNil(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/eth/v1/validator/{pubkey}/feerecipient"), nil)
	req = mux.SetURLVars(req, map[string]string{"pubkey": "0x1234567878903438ea819bd4033c6c1bf6b04320ee2075b77273c08d02f8a61bcc303c2c06bd3713cb442072ae591493"})
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	s.DeleteFeeRecipientByPubkey(w, req)
	assert.NotEqual(t, http.StatusNoContent, w.Code)
	require.StringContains(t, "Validator service not ready", w.Body.String())
}

func TestServer_DeleteFeeRecipientByPubkey_InvalidPubKey(t *testing.T) {
	s := &Server{
		validatorService: &client.ValidatorService{},
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/eth/v1/validator/{pubkey}/feerecipient"), nil)
	req = mux.SetURLVars(req, map[string]string{"pubkey": "0x123"})
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	s.DeleteFeeRecipientByPubkey(w, req)
	assert.NotEqual(t, http.StatusNoContent, w.Code)

	require.StringContains(t, "pubkey is invalid", w.Body.String())
}
