package validator

import (
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/mux"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/core"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"go.opencensus.io/trace"
)

// GetValidatorActiveSetChanges retrieves the active set changes for a given epoch.
//
// This data includes any activations, voluntary exits, and involuntary
// ejections.
func (s *Server) GetValidatorActiveSetChanges(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.GetValidatorActiveSetChanges")
	defer span.End()

	stateId := mux.Vars(r)["state_id"]
	if stateId == "" {
		httputil.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}

	st, err := s.Stater.State(ctx, []byte(stateId))
	if err != nil {
		shared.WriteStateFetchError(w, err)
		return
	}
	stEpoch := slots.ToEpoch(st.Slot())

	as, rpcError := s.CoreService.ValidatorActiveSetChanges(ctx, stEpoch)
	if rpcError != nil {
		httputil.HandleError(w, rpcError.Err.Error(), core.ErrorReasonToHTTP(rpcError.Reason))
		return
	}

	response := &structs.ActiveSetChanges{
		Epoch:               fmt.Sprintf("%d", as.Epoch),
		ActivatedPublicKeys: byteArrayToString(as.ActivatedPublicKeys),
		ActivatedIndices:    uint64ArrayToString(as.ActivatedIndices),
		ExitedPublicKeys:    byteArrayToString(as.ExitedPublicKeys),
		ExitedIndices:       uint64ArrayToString(as.ExitedIndices),
		SlashedPublicKeys:   byteArrayToString(as.SlashedPublicKeys),
		SlashedIndices:      uint64ArrayToString(as.SlashedIndices),
		EjectedPublicKeys:   byteArrayToString(as.EjectedPublicKeys),
		EjectedIndices:      uint64ArrayToString(as.EjectedIndices),
	}
	httputil.WriteJson(w, response)
}

func byteArrayToString(byteArrays [][]byte) []string {
	s := make([]string, len(byteArrays))
	for i, b := range byteArrays {
		s[i] = hexutil.Encode(b)
	}
	return s
}

func uint64ArrayToString(indices []primitives.ValidatorIndex) []string {
	s := make([]string, len(indices))
	for i, u := range indices {
		s[i] = fmt.Sprintf("%d", u)
	}
	return s
}
