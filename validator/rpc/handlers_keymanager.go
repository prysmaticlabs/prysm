package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	validatorServiceConfig "github.com/prysmaticlabs/prysm/v4/config/validator/service"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	"github.com/prysmaticlabs/prysm/v4/validator/client"
	"github.com/prysmaticlabs/prysm/v4/validator/keymanager"
	remote_web3signer "github.com/prysmaticlabs/prysm/v4/validator/keymanager/remote-web3signer"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SetVoluntaryExit creates a signed voluntary exit message and returns a VoluntaryExit object.
func (s *Server) SetVoluntaryExit(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.keymanagerAPI.SetVoluntaryExit")
	defer span.End()

	if s.validatorService == nil {
		http2.HandleError(w, "Validator service not ready", http.StatusServiceUnavailable)
		return
	}

	if !s.walletInitialized {
		http2.HandleError(w, "No wallet found", http.StatusServiceUnavailable)
		return
	}

	km, err := s.validatorService.Keymanager()
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rawPubkey := mux.Vars(r)["pubkey"]
	if rawPubkey == "" {
		http2.HandleError(w, "pubkey is required in URL params", http.StatusBadRequest)
		return
	}

	pubkey, valid := shared.ValidateHex(w, "pubkey", rawPubkey, fieldparams.BLSPubkeyLength)
	if !valid {
		return
	}

	var epoch primitives.Epoch
	ok, _, e := shared.UintFromQuery(w, r, "epoch")
	if !ok {
		http2.HandleError(w, "Invalid epoch", http.StatusBadRequest)
		return
	}
	epoch = primitives.Epoch(e)

	if epoch == 0 {
		genesisResponse, err := s.beaconNodeClient.GetGenesis(ctx, &emptypb.Empty{})
		if err != nil {
			http2.HandleError(w, errors.Wrap(err, "Failed to get genesis time").Error(), http.StatusInternalServerError)
			return
		}
		currentEpoch, err := client.CurrentEpoch(genesisResponse.GenesisTime)
		if err != nil {
			http2.HandleError(w, errors.Wrap(err, "Failed to get current epoch").Error(), http.StatusInternalServerError)
			return
		}
		epoch = currentEpoch
	}
	sve, err := client.CreateSignedVoluntaryExit(
		ctx,
		s.beaconNodeValidatorClient,
		km.Sign,
		pubkey,
		epoch,
	)
	if err != nil {
		http2.HandleError(w, errors.Wrap(err, "Could not create voluntary exit").Error(), http.StatusInternalServerError)
		return
	}

	response := &SetVoluntaryExitResponse{
		Data: &shared.SignedVoluntaryExit{
			Message: &shared.VoluntaryExit{
				Epoch:          fmt.Sprintf("%d", sve.Exit.Epoch),
				ValidatorIndex: fmt.Sprintf("%d", sve.Exit.ValidatorIndex),
			},
			Signature: hexutil.Encode(sve.Signature),
		},
	}
	http2.WriteJson(w, response)
}

// ListRemoteKeys returns a list of all public keys defined for web3signer keymanager type.
func (s *Server) ListRemoteKeys(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.keymanagerAPI.ListRemoteKeys")
	defer span.End()

	if s.validatorService == nil {
		http2.HandleError(w, "Validator service not ready.", http.StatusServiceUnavailable)
		return
	}
	if !s.walletInitialized {
		http2.HandleError(w, "Prysm Wallet not initialized. Please create a new wallet.", http.StatusServiceUnavailable)
		return
	}
	km, err := s.validatorService.Keymanager()
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.wallet.KeymanagerKind() != keymanager.Web3Signer {
		http2.HandleError(w, "Prysm Wallet is not of type Web3Signer. Please execute validator client with web3signer flags.", http.StatusInternalServerError)
		return
	}
	pubKeys, err := km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		http2.HandleError(w, errors.Errorf("Could not retrieve public keys: %v", err).Error(), http.StatusInternalServerError)
		return
	}
	keystoreResponse := make([]*RemoteKey, len(pubKeys))
	for i := 0; i < len(pubKeys); i++ {
		keystoreResponse[i] = &RemoteKey{
			Pubkey:   hexutil.Encode(pubKeys[i][:]),
			Url:      s.validatorService.Web3SignerConfig.BaseEndpoint,
			Readonly: true,
		}
	}

	response := &ListRemoteKeysResponse{
		Data: keystoreResponse,
	}
	http2.WriteJson(w, response)
}

// ImportRemoteKeys imports a list of public keys defined for web3signer keymanager type.
func (s *Server) ImportRemoteKeys(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "validator.keymanagerAPI.ImportRemoteKeys")
	defer span.End()

	if s.validatorService == nil {
		http2.HandleError(w, "Validator service not ready.", http.StatusServiceUnavailable)
		return
	}
	if !s.walletInitialized {
		http2.HandleError(w, "Prysm Wallet not initialized. Please create a new wallet.", http.StatusServiceUnavailable)
		return
	}
	km, err := s.validatorService.Keymanager()
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.wallet.KeymanagerKind() != keymanager.Web3Signer {
		http2.HandleError(w, "Prysm Wallet is not of type Web3Signer. Please execute validator client with web3signer flags.", http.StatusInternalServerError)
		return
	}

	var req ImportRemoteKeysRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		http2.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	adder, ok := km.(keymanager.PublicKeyAdder)
	if !ok {
		statuses := make([]*keymanager.KeyStatus, len(req.RemoteKeys))
		for i := 0; i < len(req.RemoteKeys); i++ {
			statuses[i] = &keymanager.KeyStatus{
				Status:  remote_web3signer.StatusError,
				Message: "Keymanager kind cannot import public keys for web3signer keymanager type.",
			}
		}
		http2.WriteJson(w, &RemoteKeysResponse{Data: statuses})
		return
	}

	remoteKeys := make([]string, len(req.RemoteKeys))
	isUrlUsed := false
	for i, obj := range req.RemoteKeys {
		remoteKeys[i] = obj.Pubkey
		if obj.Url != "" {
			isUrlUsed = true
		}
	}
	if isUrlUsed {
		log.Warnf("Setting web3signer base url for imported keys is not supported. Prysm only uses the url from --validators-external-signer-url flag for web3signer.")
	}

	http2.WriteJson(w, &RemoteKeysResponse{Data: adder.AddPublicKeys(remoteKeys)})
}

// DeleteRemoteKeys deletes a list of public keys defined for web3signer keymanager type.
func (s *Server) DeleteRemoteKeys(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "validator.keymanagerAPI.DeleteRemoteKeys")
	defer span.End()

	if s.validatorService == nil {
		http2.HandleError(w, "Validator service not ready.", http.StatusServiceUnavailable)
		return
	}
	if !s.walletInitialized {
		http2.HandleError(w, "Prysm Wallet not initialized. Please create a new wallet.", http.StatusServiceUnavailable)
		return
	}
	km, err := s.validatorService.Keymanager()
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.wallet.KeymanagerKind() != keymanager.Web3Signer {
		http2.HandleError(w, "Prysm Wallet is not of type Web3Signer. Please execute validator client with web3signer flags.", http.StatusInternalServerError)
		return
	}
	var req DeleteRemoteKeysRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		http2.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	deleter, ok := km.(keymanager.PublicKeyDeleter)
	if !ok {
		statuses := make([]*keymanager.KeyStatus, len(req.Pubkeys))
		for i := 0; i < len(req.Pubkeys); i++ {
			statuses[i] = &keymanager.KeyStatus{
				Status:  remote_web3signer.StatusError,
				Message: "Keymanager kind cannot delete public keys for web3signer keymanager type.",
			}
		}
		http2.WriteJson(w, &RemoteKeysResponse{Data: statuses})
		return
	}

	http2.WriteJson(w, RemoteKeysResponse{Data: deleter.DeletePublicKeys(req.Pubkeys)})
}

// ListFeeRecipientByPubkey returns the public key to eth address mapping object to the end user.
func (s *Server) ListFeeRecipientByPubkey(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "validator.keymanagerAPI.DeleteRemoteKeys")
	defer span.End()

	if s.validatorService == nil {
		http2.HandleError(w, "Validator service not ready.", http.StatusServiceUnavailable)
		return
	}

	rawPubkey := mux.Vars(r)["pubkey"]
	if rawPubkey == "" {
		http2.HandleError(w, "pubkey is required in URL params", http.StatusBadRequest)
		return
	}

	pubkey, valid := shared.ValidateHex(w, "pubkey", rawPubkey, fieldparams.BLSPubkeyLength)
	if !valid {
		return
	}

	finalResp := &GetFeeRecipientByPubkeyResponse{
		Data: &FeeRecipient{
			Pubkey: rawPubkey,
		},
	}

	proposerSettings := s.validatorService.ProposerSettings()

	// If fee recipient is defined for this specific pubkey in proposer configuration, use it
	if proposerSettings != nil && proposerSettings.ProposeConfig != nil {
		proposerOption, found := proposerSettings.ProposeConfig[bytesutil.ToBytes48(pubkey)]

		if found && proposerOption.FeeRecipientConfig != nil {
			finalResp.Data.Ethaddress = proposerOption.FeeRecipientConfig.FeeRecipient.String()
			http2.WriteJson(w, finalResp)
			return
		}
	}

	// If fee recipient is defined in default configuration, use it
	if proposerSettings != nil && proposerSettings.DefaultConfig != nil && proposerSettings.DefaultConfig.FeeRecipientConfig != nil {
		finalResp.Data.Ethaddress = proposerSettings.DefaultConfig.FeeRecipientConfig.FeeRecipient.String()
		http2.WriteJson(w, finalResp)
		return
	}

	http2.HandleError(w, "No fee recipient set", http.StatusBadRequest)
}

// SetFeeRecipientByPubkey updates the eth address mapped to the public key.
func (s *Server) SetFeeRecipientByPubkey(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.keymanagerAPI.SetFeeRecipientByPubkey")
	defer span.End()

	if s.validatorService == nil {
		http2.HandleError(w, "Validator service not ready.", http.StatusServiceUnavailable)
		return
	}
	rawPubkey := mux.Vars(r)["pubkey"]
	if rawPubkey == "" {
		http2.HandleError(w, "pubkey is required in URL params", http.StatusBadRequest)
		return
	}

	pubkey, valid := shared.ValidateHex(w, "pubkey", rawPubkey, fieldparams.BLSPubkeyLength)
	if !valid {
		return
	}

	var req SetFeeRecipientByPubkeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http2.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	ethAddress, valid := shared.ValidateHex(w, "Ethereum Address", req.Ethaddress, fieldparams.FeeRecipientLength)
	if !valid {
		return
	}

	feeRecipient := common.BytesToAddress(ethAddress)
	settings := s.validatorService.ProposerSettings()
	switch {
	case settings == nil:
		settings = &validatorServiceConfig.ProposerSettings{
			ProposeConfig: map[[fieldparams.BLSPubkeyLength]byte]*validatorServiceConfig.ProposerOption{
				bytesutil.ToBytes48(pubkey): {
					FeeRecipientConfig: &validatorServiceConfig.FeeRecipientConfig{
						FeeRecipient: feeRecipient,
					},
					BuilderConfig: nil,
				},
			},
			DefaultConfig: nil,
		}
	case settings.ProposeConfig == nil:
		var builderConfig *validatorServiceConfig.BuilderConfig
		if settings.DefaultConfig != nil && settings.DefaultConfig.BuilderConfig != nil {
			builderConfig = settings.DefaultConfig.BuilderConfig.Clone()
		}
		settings.ProposeConfig = map[[fieldparams.BLSPubkeyLength]byte]*validatorServiceConfig.ProposerOption{
			bytesutil.ToBytes48(pubkey): {
				FeeRecipientConfig: &validatorServiceConfig.FeeRecipientConfig{
					FeeRecipient: feeRecipient,
				},
				BuilderConfig: builderConfig,
			},
		}
	default:
		proposerOption, found := settings.ProposeConfig[bytesutil.ToBytes48(pubkey)]
		if found && proposerOption != nil {
			proposerOption.FeeRecipientConfig = &validatorServiceConfig.FeeRecipientConfig{
				FeeRecipient: feeRecipient,
			}
		} else {
			var builderConfig = &validatorServiceConfig.BuilderConfig{}
			if settings.DefaultConfig != nil && settings.DefaultConfig.BuilderConfig != nil {
				builderConfig = settings.DefaultConfig.BuilderConfig.Clone()
			}
			settings.ProposeConfig[bytesutil.ToBytes48(pubkey)] = &validatorServiceConfig.ProposerOption{
				FeeRecipientConfig: &validatorServiceConfig.FeeRecipientConfig{
					FeeRecipient: feeRecipient,
				},
				BuilderConfig: builderConfig,
			}
		}
	}
	// save the settings
	if err := s.validatorService.SetProposerSettings(ctx, settings); err != nil {
		http2.HandleError(w, "Could not set proposer settings: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// override the 200 success with 202 according to the specs
	w.WriteHeader(http.StatusAccepted)
}

// DeleteFeeRecipientByPubkey updates the eth address mapped to the public key to the default fee recipient listed
func (s *Server) DeleteFeeRecipientByPubkey(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.keymanagerAPI.DeleteFeeRecipientByPubkey")
	defer span.End()

	if s.validatorService == nil {
		http2.HandleError(w, "Validator service not ready.", http.StatusServiceUnavailable)
		return
	}
	rawPubkey := mux.Vars(r)["pubkey"]
	if rawPubkey == "" {
		http2.HandleError(w, "pubkey is required in URL params", http.StatusBadRequest)
		return
	}

	pubkey, valid := shared.ValidateHex(w, "pubkey", rawPubkey, fieldparams.BLSPubkeyLength)
	if !valid {
		return
	}

	settings := s.validatorService.ProposerSettings()

	if settings != nil && settings.ProposeConfig != nil {
		proposerOption, found := settings.ProposeConfig[bytesutil.ToBytes48(pubkey)]
		if found {
			proposerOption.FeeRecipientConfig = nil
		}
	}

	// save the settings
	if err := s.validatorService.SetProposerSettings(ctx, settings); err != nil {
		http2.HandleError(w, "Could not set proposer settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// override the 200 success with 204 according to the specs
	w.WriteHeader(http.StatusNoContent)
}
