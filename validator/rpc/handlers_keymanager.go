package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
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
