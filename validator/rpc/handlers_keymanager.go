package rpc

import (
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	validatorServiceConfig "github.com/prysmaticlabs/prysm/v4/config/validator/service"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/validator"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	ethpbservice "github.com/prysmaticlabs/prysm/v4/proto/eth/service"
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
	ctx, span := trace.StartSpan(r.Context(), "validator.keymanagerAPI.GetGasLimit")
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

	resp := &ethpbservice.GetGasLimitResponse{
		Data: &ethpbservice.GetGasLimitResponse_GasLimit{
			Pubkey: validatorKey,
		},
	}
	settings := s.validatorService.ProposerSettings()
	if settings != nil {
		proposerOption, found := settings.ProposeConfig[bytesutil.ToBytes48(validatorKey)]
		if found {
			if proposerOption.BuilderConfig != nil {
				resp.Data.GasLimit = uint64(proposerOption.BuilderConfig.GasLimit)
				return resp, nil
			}
		} else if settings.DefaultConfig != nil && settings.DefaultConfig.BuilderConfig != nil {
			resp.Data.GasLimit = uint64(s.validatorService.ProposerSettings().DefaultConfig.BuilderConfig.GasLimit)
			return resp, nil
		}
	}
	resp.Data.GasLimit = params.BeaconConfig().DefaultBuilderGasLimit
	return resp, nil
}

// SetGasLimit updates GasLimt of the public key.
func (s *Server) SetGasLimit(w http.ResponseWriter, r *http.Request) {
	if s.validatorService == nil {
		return nil, status.Error(codes.FailedPrecondition, "Validator service not ready")
	}
	validatorKey := req.Pubkey

	if err := validatePublicKey(validatorKey); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	settings := s.validatorService.ProposerSettings()
	if settings == nil {
		return &empty.Empty{}, status.Errorf(codes.FailedPrecondition, "no proposer settings were found to update")
	} else if settings.ProposeConfig == nil {
		if settings.DefaultConfig == nil || settings.DefaultConfig.BuilderConfig == nil || !settings.DefaultConfig.BuilderConfig.Enabled {
			return &empty.Empty{}, status.Errorf(codes.FailedPrecondition, "gas limit changes only apply when builder is enabled")
		}
		settings.ProposeConfig = make(map[[fieldparams.BLSPubkeyLength]byte]*validatorServiceConfig.ProposerOption)
		option := settings.DefaultConfig.Clone()
		option.BuilderConfig.GasLimit = validator.Uint64(req.GasLimit)
		settings.ProposeConfig[bytesutil.ToBytes48(validatorKey)] = option
	} else {
		proposerOption, found := settings.ProposeConfig[bytesutil.ToBytes48(validatorKey)]
		if found {
			if proposerOption.BuilderConfig == nil || !proposerOption.BuilderConfig.Enabled {
				return &empty.Empty{}, status.Errorf(codes.FailedPrecondition, "gas limit changes only apply when builder is enabled")
			} else {
				proposerOption.BuilderConfig.GasLimit = validator.Uint64(req.GasLimit)
			}
		} else {
			if settings.DefaultConfig == nil {
				return &empty.Empty{}, status.Errorf(codes.FailedPrecondition, "gas limit changes only apply when builder is enabled")
			}
			option := settings.DefaultConfig.Clone()
			option.BuilderConfig.GasLimit = validator.Uint64(req.GasLimit)
			settings.ProposeConfig[bytesutil.ToBytes48(validatorKey)] = option
		}
	}
	// save the settings
	if err := s.validatorService.SetProposerSettings(ctx, settings); err != nil {
		return &empty.Empty{}, status.Errorf(codes.Internal, "Could not set proposer settings: %v", err)
	}
	// override the 200 success with 202 according to the specs
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", "202")); err != nil {
		return &empty.Empty{}, status.Errorf(codes.Internal, "Could not set custom success code header: %v", err)
	}

	return &empty.Empty{}, nil
}

// DeleteGasLimit deletes the gas limit by public key
func (s *Server) DeleteGasLimit(w http.ResponseWriter, r *http.Request) {
	if s.validatorService == nil {
		return nil, status.Error(codes.FailedPrecondition, "Validator service not ready")
	}
	validatorKey := req.Pubkey
	if err := validatePublicKey(validatorKey); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	proposerSettings := s.validatorService.ProposerSettings()
	if proposerSettings != nil && proposerSettings.ProposeConfig != nil {
		proposerOption, found := proposerSettings.ProposeConfig[bytesutil.ToBytes48(validatorKey)]
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
				return &empty.Empty{}, status.Errorf(codes.Internal, "Could not set proposer settings: %v", err)
			}
			// Successfully deleted gas limit (reset to proposer config default or global default).
			// Return with success http code "204".
			if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", "204")); err != nil {
				return &empty.Empty{}, status.Errorf(codes.Internal, "Could not set custom http code 204 header: %v", err)
			}
			return &empty.Empty{}, nil
		}
	}
	// Otherwise, either no proposerOption is found for the pubkey or proposerOption.BuilderConfig is not enabled at all,
	// we response "not found".
	return nil, status.Error(codes.NotFound, fmt.Sprintf("no gaslimt found for pubkey: %q", hexutil.Encode(validatorKey)))
}
