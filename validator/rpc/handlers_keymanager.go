package rpc

import (
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
