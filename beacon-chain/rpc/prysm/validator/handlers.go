package validator

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
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

	stateId := strings.ReplaceAll(r.URL.Query().Get("state_id"), " ", "")
	var epoch uint64
	switch stateId {
	case "head":
		e, err := s.CoreService.ChainInfoFetcher.ReceivedBlocksLastEpoch()
		if err != nil {
			httputil.HandleError(w, "Could not retrieve head root: "+err.Error(), http.StatusInternalServerError)
			return
		}
		epoch = e
	case "finalized":
		finalized := s.CoreService.ChainInfoFetcher.FinalizedCheckpt()
		epoch = uint64(finalized.Epoch)
	case "genesis":
		epoch = 0
	default:
		_, e, ok := shared.UintFromQuery(w, r, "epoch", true)
		if !ok {
			currentSlot := s.CoreService.GenesisTimeFetcher.CurrentSlot()
			currentEpoch := slots.ToEpoch(currentSlot)
			epoch = uint64(currentEpoch)
		} else {
			epoch = e
		}
	}
	as, err := s.CoreService.ValidatorActiveSetChanges(ctx, primitives.Epoch(epoch))
	if err != nil {
		httputil.HandleError(w, err.Err.Error(), core.ErrorReasonToHTTP(err.Reason))
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
