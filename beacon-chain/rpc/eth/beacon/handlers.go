package beacon

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/network"
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
	validate := validator.New()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "could not read request body",
			Code:    http.StatusInternalServerError,
		}
		network.WriteError(w, errJson)
		return
	}

	var capellaBlock *SignedBeaconBlockCapella
	if err = json.Unmarshal(body, &capellaBlock); err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "could not decode request body into block",
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return
	}
	if err = validate.Struct(capellaBlock); err == nil {
		panic("capella")
	}
	var bellatrixBlock *SignedBeaconBlockCapella
	if err = json.Unmarshal(body, &bellatrixBlock); err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "could not decode request body into block",
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return
	}
	if err = validate.Struct(bellatrixBlock); err == nil {
		panic("bellatrix")
	}
	var altairBlock *SignedBeaconBlockAltair
	if err = json.Unmarshal(body, &altairBlock); err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "could not decode request body into block",
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return
	}
	if err = validate.Struct(altairBlock); err == nil {
		panic("altair")
	}
	var phase0Block *SignedBeaconBlock
	if err = json.Unmarshal(body, &phase0Block); err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "could not decode request body into block",
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return
	}
	if err = validate.Struct(phase0Block); err == nil {
		consensusBlock, err := phase0Block.ToConsensusReadOnly()
		if err != nil {
			errJson := &network.DefaultErrorJson{
				Message: "could not decode request body into consensus block: " + err.Error(),
				Code:    http.StatusBadRequest,
			}
			network.WriteError(w, errJson)
			return
		}
		parentState, err := bs.StateGenService.StateByRoot(r.Context(), consensusBlock.Block().ParentRoot())
		if err != nil {
			errJson := &network.DefaultErrorJson{
				Message: "could not get parent state: " + err.Error(),
				Code:    http.StatusInternalServerError,
			}
			network.WriteError(w, errJson)
			return
		}
		if _, err := transition.ExecuteStateTransition(r.Context(), parentState, consensusBlock); err != nil {
			errJson := &network.DefaultErrorJson{
				Message: "could not execute state transition: " + err.Error(),
				Code:    http.StatusBadRequest,
			}
			network.WriteError(w, errJson)
			return
		}
	}

	errJson := &network.DefaultErrorJson{
		Message: "invalid block",
		Code:    http.StatusBadRequest,
	}
	network.WriteError(w, errJson)
	return
}
