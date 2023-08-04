package validator

import (
	"encoding/json"
	"net/http"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/core"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
)

type ProduceSyncCommitteeContributionRequest struct {
	Slot              primitives.Slot `json:"slot,omitempty"`
	SubcommitteeIndex uint64          `json:"subcommittee_index,omitempty"`
	BeaconBlockRoot   []byte          `json:"beacon_block_root,omitempty"`
}

type ProduceSyncCommitteeContributionResponse struct {
	Data *SyncCommitteeContribution
}

type SyncCommitteeContribution struct {
	Slot              primitives.Slot       `json:"slot,omitempty"`
	BeaconBlockRoot   []byte                `json:"beacon_block_root,omitempty"`
	SubcommitteeIndex uint64                `json:"subcommittee_index,omitempty"`
	AggregationBits   bitfield.Bitvector128 `json:"aggregation_bits,omitempty"`
	Signature         []byte                `json:"signature,omitempty"`
}

func (vs *Server) ProduceSyncCommitteeContribution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req ProduceSyncCommitteeContributionRequest
	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handleHTTPError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	syncCommitteeResp, err := core.ProduceSyncCommitteeContribution(
		ctx,
		&ethpbv2.ProduceSyncCommitteeContributionRequest{},
		vs.SyncCommitteePool,
		vs.V1Alpha1Server,
	)
	if err != nil {
		handleHTTPError(w, "Could not compute validator performance: "+err.Err.Error(), core.ErrorReasonToHTTP(err.Reason))
		return
	}
	response := &ProduceSyncCommitteeContributionResponse{
		Data: &SyncCommitteeContribution{
			Slot:              syncCommitteeResp.Data.Slot,
			BeaconBlockRoot:   syncCommitteeResp.Data.BeaconBlockRoot,
			SubcommitteeIndex: syncCommitteeResp.Data.SubcommitteeIndex,
			AggregationBits:   syncCommitteeResp.Data.AggregationBits,
			Signature:         syncCommitteeResp.Data.Signature,
		},
	}
	http2.WriteJson(w, response)
}
