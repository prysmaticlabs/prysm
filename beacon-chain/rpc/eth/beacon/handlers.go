package beacon

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/prysm/v1alpha1/types"
	"github.com/prysmaticlabs/prysm/v4/network"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// PublishBlindedBlockV2 instructs the beacon node to use the components of the `SignedBlindedBeaconBlock` to construct and publish a
// `SignedBeaconBlock` by swapping out the `transactions_root` for the corresponding full list of `transactions`.
// The beacon node should broadcast a newly constructed `SignedBeaconBlock` to the beacon network,
// to be included in the beacon chain. The beacon node is not required to validate the signed
// `BeaconBlock`, and a successful response (20X) only indicates that the broadcast has been
// successful. The beacon node is expected to integrate the new block into its state, and
// therefore validate the block internally, however blocks which fail the validation are still
// broadcast but a different status code is returned (202). Pre-Bellatrix, this endpoint will accept
// a `SignedBeaconBlock`. The broadcast behaviour may be adjusted via the `broadcast_validation`
// query parameter.
func (bs *Server) PublishBlindedBlockV2(w http.ResponseWriter, r *http.Request) {
	isSyncing, syncDetails, err := helpers.ValidateSyncHTTP(r.Context(), bs.SyncChecker, bs.HeadFetcher, bs.TimeFetcher, bs.OptimisticModeFetcher)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not check if node is syncing: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	if isSyncing {
		msg := "Beacon node is currently syncing and not serving request on that endpoint"
		details, err := json.Marshal(syncDetails)
		if err == nil {
			msg += " Details: " + string(details)
		}
		errJson := &network.DefaultErrorJson{
			Message: msg,
			Code:    http.StatusServiceUnavailable,
		}
		network.WriteError(w, errJson)
		return
	}

	var broadcastValidation types.BroadcastValidation
	switch r.URL.Query().Get("broadcast_validation") {
	case "consensus":
		broadcastValidation = types.Consensus
	case "consensus_and_equivocation":
		broadcastValidation = types.ConsensusAndEquivocation
	default:
		broadcastValidation = types.Gossip
	}

	validate := validator.New()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not read request body",
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}

	var capellaBlock *SignedBlindedBeaconBlockCapella
	if err = unmarshalStrict(body, &capellaBlock); err == nil {
		if err = validate.Struct(capellaBlock); err == nil {
			consensusBlock, err := capellaBlock.ToGeneric()
			if err != nil {
				errJson := &network.DefaultErrorJson{
					Message: "Could not decode request body into consensus block: " + err.Error(),
					Code:    http.StatusBadRequest,
				}
				network.WriteError(w, errJson)
				return
			}
			bs.proposeBlock(r.Context(), w, consensusBlock, broadcastValidation)
			return
		}
	}

	var bellatrixBlock *SignedBlindedBeaconBlockBellatrix
	if err = unmarshalStrict(body, &bellatrixBlock); err == nil {
		if err = validate.Struct(bellatrixBlock); err == nil {
			consensusBlock, err := bellatrixBlock.ToGeneric()
			if err != nil {
				errJson := &network.DefaultErrorJson{
					Message: "Could not decode request body into consensus block: " + err.Error(),
					Code:    http.StatusBadRequest,
				}
				network.WriteError(w, errJson)
				return
			}
			bs.proposeBlock(r.Context(), w, consensusBlock, broadcastValidation)
			return
		}
	}
	var altairBlock *SignedBeaconBlockAltair
	if err = unmarshalStrict(body, &altairBlock); err == nil {
		if err = validate.Struct(altairBlock); err == nil {
			consensusBlock, err := altairBlock.ToGeneric()
			if err != nil {
				errJson := &network.DefaultErrorJson{
					Message: "Could not decode request body into consensus block: " + err.Error(),
					Code:    http.StatusBadRequest,
				}
				network.WriteError(w, errJson)
				return
			}
			bs.proposeBlock(r.Context(), w, consensusBlock, broadcastValidation)
			return
		}
	}
	var phase0Block *SignedBeaconBlock
	if err = unmarshalStrict(body, &phase0Block); err == nil {
		if err = validate.Struct(phase0Block); err == nil {
			consensusBlock, err := phase0Block.ToGeneric()
			if err != nil {
				errJson := &network.DefaultErrorJson{
					Message: "Could not decode request body into consensus block: " + err.Error(),
					Code:    http.StatusBadRequest,
				}
				network.WriteError(w, errJson)
				return
			}
			bs.proposeBlock(r.Context(), w, consensusBlock, broadcastValidation)
			return
		}
	}

	errJson := &network.DefaultErrorJson{
		Message: "Body does not represent a valid block type",
		Code:    http.StatusBadRequest,
	}
	network.WriteError(w, errJson)
	return
}

// PublishBlockV2 instructs the beacon node to broadcast a newly signed beacon block to the beacon network,
// to be included in the beacon chain. A success response (20x) indicates that the block
// passed gossip validation and was successfully broadcast onto the network.
// The beacon node is also expected to integrate the block into the state, but may broadcast it
// before doing so, so as to aid timely delivery of the block. Should the block fail full
// validation, a separate success response code (202) is used to indicate that the block was
// successfully broadcast but failed integration. The broadcast behaviour may be adjusted via the
// `broadcast_validation` query parameter.
func (bs *Server) PublishBlockV2(w http.ResponseWriter, r *http.Request) {
	isSyncing, syncDetails, err := helpers.ValidateSyncHTTP(r.Context(), bs.SyncChecker, bs.HeadFetcher, bs.TimeFetcher, bs.OptimisticModeFetcher)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not check if node is syncing: " + err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
	if isSyncing {
		msg := "Beacon node is currently syncing and not serving request on that endpoint"
		details, err := json.Marshal(syncDetails)
		if err == nil {
			msg += " Details: " + string(details)
		}
		errJson := &network.DefaultErrorJson{
			Message: msg,
			Code:    http.StatusServiceUnavailable,
		}
		network.WriteError(w, errJson)
		return
	}

	var broadcastValidation types.BroadcastValidation
	switch r.URL.Query().Get("broadcast_validation") {
	case "consensus":
		broadcastValidation = types.Consensus
	case "consensus_and_equivocation":
		broadcastValidation = types.ConsensusAndEquivocation
	default:
		broadcastValidation = types.Gossip
	}

	validate := validator.New()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not read request body",
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}

	var capellaBlock *SignedBeaconBlockCapella
	if err = unmarshalStrict(body, &capellaBlock); err == nil {
		if err = validate.Struct(capellaBlock); err == nil {
			consensusBlock, err := capellaBlock.ToGeneric()
			if err != nil {
				errJson := &network.DefaultErrorJson{
					Message: "Could not decode request body into consensus block: " + err.Error(),
					Code:    http.StatusBadRequest,
				}
				network.WriteError(w, errJson)
				return
			}
			bs.proposeBlock(r.Context(), w, consensusBlock, broadcastValidation)
			return
		}
	}
	var bellatrixBlock *SignedBeaconBlockBellatrix
	if err = unmarshalStrict(body, &bellatrixBlock); err == nil {
		if err = validate.Struct(bellatrixBlock); err == nil {
			consensusBlock, err := bellatrixBlock.ToGeneric()
			if err != nil {
				errJson := &network.DefaultErrorJson{
					Message: "Could not decode request body into consensus block: " + err.Error(),
					Code:    http.StatusBadRequest,
				}
				network.WriteError(w, errJson)
				return
			}
			bs.proposeBlock(r.Context(), w, consensusBlock, broadcastValidation)
			return
		}
	}
	var altairBlock *SignedBeaconBlockAltair
	if err = unmarshalStrict(body, &altairBlock); err == nil {
		if err = validate.Struct(altairBlock); err == nil {
			consensusBlock, err := altairBlock.ToGeneric()
			if err != nil {
				errJson := &network.DefaultErrorJson{
					Message: "Could not decode request body into consensus block: " + err.Error(),
					Code:    http.StatusBadRequest,
				}
				network.WriteError(w, errJson)
				return
			}
			bs.proposeBlock(r.Context(), w, consensusBlock, broadcastValidation)
			return
		}
	}
	var phase0Block *SignedBeaconBlock
	if err = unmarshalStrict(body, &phase0Block); err == nil {
		if err = validate.Struct(phase0Block); err == nil {
			consensusBlock, err := phase0Block.ToGeneric()
			if err != nil {
				errJson := &network.DefaultErrorJson{
					Message: "Could not decode request body into consensus block: " + err.Error(),
					Code:    http.StatusBadRequest,
				}
				network.WriteError(w, errJson)
				return
			}
			bs.proposeBlock(r.Context(), w, consensusBlock, broadcastValidation)
			return
		}
	}

	errJson := &network.DefaultErrorJson{
		Message: "Body does not represent a valid block type",
		Code:    http.StatusBadRequest,
	}
	network.WriteError(w, errJson)
	return
}

func (bs *Server) proposeBlock(
	ctx context.Context,
	w http.ResponseWriter,
	blk *eth.GenericSignedBeaconBlock,
	broadcastValidation types.BroadcastValidation,
) {
	_, err := bs.V1Alpha1ValidatorServer.ProposeGenericBeaconBlock(ctx, blk, broadcastValidation)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, types.ErrConsensusValidationFailed.Error()) ||
			strings.Contains(errMsg, types.ErrEquivocationValidationFailed.Error()) {
			errJson := &network.DefaultErrorJson{
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			}
			network.WriteError(w, errJson)
			return
		}
		errJson := &network.DefaultErrorJson{
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}
}

func unmarshalStrict(data []byte, v interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
