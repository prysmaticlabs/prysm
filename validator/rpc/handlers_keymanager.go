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
	"github.com/prysmaticlabs/prysm/v4/config/params"
	validatorServiceConfig "github.com/prysmaticlabs/prysm/v4/config/validator/service"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/validator"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	"github.com/prysmaticlabs/prysm/v4/validator/client"
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

	if s.wallet == nil {
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

// GetGasLimit returns the gas limit measured in gwei defined for the custom mev builder by public key
func (s *Server) GetGasLimit(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "validator.keymanagerAPI.GetGasLimit")
	defer span.End()

	if s.validatorService == nil {
		http2.HandleError(w, "Validator service not ready", http.StatusServiceUnavailable)
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

	resp := &GetGasLimitResponse{
		Data: &GasLimitMetaData{
			Pubkey: rawPubkey,
		},
	}
	settings := s.validatorService.ProposerSettings()
	if settings != nil {
		proposerOption, found := settings.ProposeConfig[bytesutil.ToBytes48(pubkey)]
		if found {
			if proposerOption.BuilderConfig != nil {
				resp.Data.GasLimit = fmt.Sprintf("%d", proposerOption.BuilderConfig.GasLimit)
				http2.WriteJson(w, resp)
				return
			}
		} else if settings.DefaultConfig != nil && settings.DefaultConfig.BuilderConfig != nil {
			resp.Data.GasLimit = fmt.Sprintf("%d", s.validatorService.ProposerSettings().DefaultConfig.BuilderConfig.GasLimit)
			http2.WriteJson(w, resp)
			return
		}
	}
	resp.Data.GasLimit = fmt.Sprintf("%d", params.BeaconConfig().DefaultBuilderGasLimit)
	http2.WriteJson(w, resp)
}

// SetGasLimit updates GasLimt of the public key.
func (s *Server) SetGasLimit(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.keymanagerAPI.SetGasLimit")
	defer span.End()

	if s.validatorService == nil {
		http2.HandleError(w, "Validator service not ready", http.StatusServiceUnavailable)
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

	var req SetGasLimitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http2.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	gasLimit, valid := shared.ValidateUint(w, "Gas Limit", req.GasLimit)
	if !valid {
		return
	}

	settings := s.validatorService.ProposerSettings()
	if settings == nil {
		http2.HandleError(w, "no proposer settings were found to update", http.StatusInternalServerError)
		return
	} else if settings.ProposeConfig == nil {
		if settings.DefaultConfig == nil || settings.DefaultConfig.BuilderConfig == nil || !settings.DefaultConfig.BuilderConfig.Enabled {
			http2.HandleError(w, "gas limit changes only apply when builder is enabled", http.StatusInternalServerError)
			return
		}
		settings.ProposeConfig = make(map[[fieldparams.BLSPubkeyLength]byte]*validatorServiceConfig.ProposerOption)
		option := settings.DefaultConfig.Clone()
		option.BuilderConfig.GasLimit = validator.Uint64(gasLimit)
		settings.ProposeConfig[bytesutil.ToBytes48(pubkey)] = option
	} else {
		proposerOption, found := settings.ProposeConfig[bytesutil.ToBytes48(pubkey)]
		if found {
			if proposerOption.BuilderConfig == nil || !proposerOption.BuilderConfig.Enabled {
				http2.HandleError(w, "gas limit changes only apply when builder is enabled", http.StatusInternalServerError)
				return
			} else {
				proposerOption.BuilderConfig.GasLimit = validator.Uint64(gasLimit)
			}
		} else {
			if settings.DefaultConfig == nil {
				http2.HandleError(w, "gas limit changes only apply when builder is enabled", http.StatusInternalServerError)
				return
			}
			option := settings.DefaultConfig.Clone()
			option.BuilderConfig.GasLimit = validator.Uint64(gasLimit)
			settings.ProposeConfig[bytesutil.ToBytes48(pubkey)] = option
		}
	}
	// save the settings
	if err := s.validatorService.SetProposerSettings(ctx, settings); err != nil {
		http2.HandleError(w, "Could not set proposer settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// DeleteGasLimit deletes the gas limit by public key
func (s *Server) DeleteGasLimit(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.keymanagerAPI.SetGasLimit")
	defer span.End()

	if s.validatorService == nil {
		http2.HandleError(w, "Validator service not ready", http.StatusServiceUnavailable)
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

	proposerSettings := s.validatorService.ProposerSettings()
	if proposerSettings != nil && proposerSettings.ProposeConfig != nil {
		proposerOption, found := proposerSettings.ProposeConfig[bytesutil.ToBytes48(pubkey)]
		if found && proposerOption.BuilderConfig != nil {
			// If proposerSettings has default value, use it.
			if proposerSettings.DefaultConfig != nil && proposerSettings.DefaultConfig.BuilderConfig != nil {
				proposerOption.BuilderConfig.GasLimit = proposerSettings.DefaultConfig.BuilderConfig.GasLimit
			} else {
				// Fallback to using global default.
				proposerOption.BuilderConfig.GasLimit = validator.Uint64(params.BeaconConfig().DefaultBuilderGasLimit)
			}
			// save the settings
			if err := s.validatorService.SetProposerSettings(ctx, proposerSettings); err != nil {
				http2.HandleError(w, "Could not set proposer settings: "+err.Error(), http.StatusBadRequest)
				return
			}
			// Successfully deleted gas limit (reset to proposer config default or global default).
			// Return with success http code "204".
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	// Otherwise, either no proposerOption is found for the pubkey or proposerOption.BuilderConfig is not enabled at all,
	// we respond "not found".
	http2.HandleError(w, fmt.Sprintf("no gaslimt found for pubkey: %q", rawPubkey), http.StatusNotFound)
}
